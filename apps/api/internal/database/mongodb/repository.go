// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package mongodb

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/observability"
	"github.com/qolzam/telar/apps/api/internal/database/utils"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoRepository implements the Repository interface for MongoDB
type MongoRepository struct {
	client   *mongo.Client
	database *mongo.Database
	dbName   string
}

// MongoQueryResult implements QueryResult for MongoDB
type MongoQueryResult struct {
	cursor *mongo.Cursor
	ctx    context.Context
	err    error
}

// MongoSingleResult implements QuerySingleResult for MongoDB
type MongoSingleResult struct {
	result   *mongo.SingleResult
	err      error
	noResult bool
}

// MongoTransactionContext implements TransactionContext for MongoDB (legacy)
type MongoTransactionContext struct {
	session mongo.Session
	ctx     context.Context
}

// NewMongoRepository creates a new MongoDB repository
func NewMongoRepository(ctx context.Context, config *interfaces.MongoDBConfig, databaseName string) (*MongoRepository, error) {
	// Build connection URI
	uri := buildConnectionURI(config)
	
	// Set client options
	clientOptions := options.Client().ApplyURI(uri)
	
	if config.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(uint64(config.MaxPoolSize))
	}
	
	if config.MinPoolSize > 0 {
		clientOptions.SetMinPoolSize(uint64(config.MinPoolSize))
	}
	
	if config.ConnectTimeout > 0 {
		clientOptions.SetConnectTimeout(time.Duration(config.ConnectTimeout) * time.Second)
	}
	
	if config.SocketTimeout > 0 {
		clientOptions.SetSocketTimeout(time.Duration(config.SocketTimeout) * time.Second)
	}
	
	if config.MaxIdleTime > 0 {
		clientOptions.SetMaxConnIdleTime(time.Duration(config.MaxIdleTime) * time.Second)
	}
	
	if config.ServerSelectionTimeout > 0 {
		clientOptions.SetServerSelectionTimeout(time.Duration(config.ServerSelectionTimeout) * time.Second)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(databaseName)

	return &MongoRepository{
		client:   client,
		database: database,
		dbName:   databaseName,
	}, nil
}

// buildConnectionURI builds MongoDB connection URI from config
func buildConnectionURI(config *interfaces.MongoDBConfig) string {
	uri := "mongodb://"
	
	if config.Username != "" && config.Password != "" {
		uri += fmt.Sprintf("%s:%s@", config.Username, config.Password)
	}
	
	uri += fmt.Sprintf("%s:%d", config.Host, config.Port)
	
	if config.AuthDatabase != "" {
		uri += fmt.Sprintf("/?authSource=%s", config.AuthDatabase)
	}
	
	if config.ReplicaSet != "" {
		if config.AuthDatabase != "" {
			uri += fmt.Sprintf("&replicaSet=%s", config.ReplicaSet)
		} else {
			uri += fmt.Sprintf("/?replicaSet=%s", config.ReplicaSet)
		}
	}
	
	if config.SSL {
		if config.AuthDatabase != "" || config.ReplicaSet != "" {
			uri += "&ssl=true"
		} else {
			uri += "/?ssl=true"
		}
	}
	
	return uri
}

// Save stores a single document
func (r *MongoRepository) Save(ctx context.Context, collectionName string, data interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		insertResult, err := collection.InsertOne(ctx, data)
		if err != nil {
			log.Error("MongoDB Save error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: insertResult.InsertedID}
	}()
	
	return result
}

// SaveMany stores multiple documents
func (r *MongoRepository) SaveMany(ctx context.Context, collectionName string, data []interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		insertOptions := options.InsertMany().SetOrdered(false)
		insertResult, err := collection.InsertMany(ctx, data, insertOptions)
		if err != nil {
			log.Error("MongoDB SaveMany error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: insertResult.InsertedIDs}
	}()
	
	return result
}

// Find retrieves multiple documents
func (r *MongoRepository) Find(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.FindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		findOptions := options.Find()
		
		if opts != nil {
			if opts.Limit != nil {
				findOptions.SetLimit(*opts.Limit)
			}
			if opts.Skip != nil {
				findOptions.SetSkip(*opts.Skip)
			}
			if opts.Sort != nil {
				findOptions.SetSort(opts.Sort)
			}
			if opts.Select != nil {
				findOptions.SetProjection(opts.Select)
			}
		}
		
		cursor, err := collection.Find(ctx, filter, findOptions)
		if err != nil {
			log.Error("MongoDB Find error: %s", err.Error())
			result <- &MongoQueryResult{err: err}
			return
		}
		
		result <- &MongoQueryResult{cursor: cursor, ctx: ctx}
	}()
	
	return result
}

// FindOne retrieves a single document
func (r *MongoRepository) FindOne(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.SingleResult {
	result := make(chan interfaces.SingleResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		singleResult := collection.FindOne(ctx, filter)
		err := singleResult.Err()
		
		if err != nil {
			if err == mongo.ErrNoDocuments {
				result <- &MongoSingleResult{result: singleResult, noResult: true}
				return
			}
			log.Error("MongoDB FindOne error: %s", err.Error())
			result <- &MongoSingleResult{err: err}
			return
		}
		
		result <- &MongoSingleResult{result: singleResult}
	}()
	
	return result
}

// Update updates documents matching the filter
func (r *MongoRepository) Update(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Check if data already contains MongoDB operators
		var updateData interface{}
		if dataMap, ok := data.(map[string]interface{}); ok {
			hasOperators := false
			for key := range dataMap {
				if strings.HasPrefix(key, "$") {
					hasOperators = true
					break
				}
			}
			
			if !hasOperators {
				// Wrap clean field updates in $set operator for MongoDB
				updateData = map[string]interface{}{"$set": dataMap}
			} else {
				updateData = data
			}
		} else {
			updateData = data
		}
		
		// Build MongoDB update options
		mongoOpts := options.Update()
		if opts != nil {
			if opts.Upsert != nil {
				mongoOpts.SetUpsert(*opts.Upsert)
			}
			if opts.BypassDocumentValidation != nil {
				mongoOpts.SetBypassDocumentValidation(*opts.BypassDocumentValidation)
			}
			if opts.ArrayFilters != nil && opts.ArrayFilters.Filters != nil {
				mongoOpts.SetArrayFilters(options.ArrayFilters{
					Registry: nil,
					Filters:  opts.ArrayFilters.Filters,
				})
			}
		}
		
		_, err := collection.UpdateOne(ctx, filter, updateData, mongoOpts)
		if err != nil {
			log.Error("MongoDB Update error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

// UpdateMany updates multiple documents matching the filter
func (r *MongoRepository) UpdateMany(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Check if data already contains MongoDB operators
		var updateData interface{}
		if dataMap, ok := data.(map[string]interface{}); ok {
			hasOperators := false
			for key := range dataMap {
				if strings.HasPrefix(key, "$") {
					hasOperators = true
					break
				}
			}
			
			if !hasOperators {
				// Wrap clean field updates in $set operator for MongoDB
				updateData = map[string]interface{}{"$set": dataMap}
			} else {
				updateData = data
			}
		} else {
			updateData = data
		}
		
		// Build MongoDB update options
		mongoOpts := options.Update()
		if opts != nil {
			if opts.Upsert != nil {
				mongoOpts.SetUpsert(*opts.Upsert)
			}
			if opts.BypassDocumentValidation != nil {
				mongoOpts.SetBypassDocumentValidation(*opts.BypassDocumentValidation)
			}
			if opts.ArrayFilters != nil && opts.ArrayFilters.Filters != nil {
				mongoOpts.SetArrayFilters(options.ArrayFilters{
					Registry: nil,
					Filters:  opts.ArrayFilters.Filters,
				})
			}
		}
		
		updateResult, err := collection.UpdateMany(ctx, filter, updateData, mongoOpts)
		if err != nil {
			log.Error("MongoDB UpdateMany error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: updateResult.ModifiedCount}
	}()
	
	return result
}

// Delete deletes documents matching the filter
func (r *MongoRepository) Delete(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Delete all matching documents (equivalent to justOne = false in old interface)
		deleteResult, err := collection.DeleteMany(ctx, filter)
		if err != nil {
			log.Error("MongoDB Delete error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: deleteResult.DeletedCount}
	}()
	
	return result
}

// DeleteMany performs bulk delete operations for multiple individual filters
func (r *MongoRepository) DeleteMany(ctx context.Context, collectionName string, filters []interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		var models []mongo.WriteModel
		for _, filter := range filters {
			model := mongo.NewDeleteOneModel().SetFilter(filter)
			models = append(models, model)
		}
		
		bulkOptions := options.BulkWrite().SetOrdered(false)
		bulkResult, err := collection.BulkWrite(ctx, models, bulkOptions)
		if err != nil {
			log.Error("MongoDB DeleteMany error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: bulkResult.DeletedCount}
	}()
	
	return result
}

// Aggregate performs aggregation pipeline operations
func (r *MongoRepository) Aggregate(ctx context.Context, collectionName string, pipeline interface{}) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		cursor, err := collection.Aggregate(ctx, pipeline)
		if err != nil {
			log.Error("MongoDB Aggregate error: %s", err.Error())
			result <- &MongoQueryResult{err: err}
			return
		}
		
		result <- &MongoQueryResult{cursor: cursor, ctx: ctx}
	}()
	
	return result
}

// Count counts documents matching filter
func (r *MongoRepository) Count(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		count, err := collection.CountDocuments(ctx, filter)
		if err != nil {
			log.Error("MongoDB Count error: %s", err.Error())
			result <- interfaces.CountResult{Error: err}
			return
		}
		
		result <- interfaces.CountResult{Count: count}
	}()
	
	return result
}

// Distinct gets distinct values for a field
func (r *MongoRepository) Distinct(ctx context.Context, collectionName string, field string, filter interface{}) <-chan interfaces.DistinctResult {
	result := make(chan interfaces.DistinctResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		values, err := collection.Distinct(ctx, field, filter)
		if err != nil {
			log.Error("MongoDB Distinct error: %s", err.Error())
			result <- interfaces.DistinctResult{Error: err}
			return
		}
		
		result <- interfaces.DistinctResult{Values: values}
	}()
	
	return result
}

// BulkWrite performs bulk operations
func (r *MongoRepository) BulkWrite(ctx context.Context, collectionName string, operations []interfaces.BulkOperation) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		var models []mongo.WriteModel
		
		for _, op := range operations {
			switch op.Type {
			case interfaces.BulkInsert:
				model := mongo.NewInsertOneModel().SetDocument(op.Data)
				models = append(models, model)
			case interfaces.BulkUpdate:
				model := mongo.NewUpdateOneModel().SetFilter(op.Filter).SetUpdate(op.Data)
				if op.Upsert {
					model.SetUpsert(true)
				}
				models = append(models, model)
			case interfaces.BulkDelete:
				model := mongo.NewDeleteOneModel().SetFilter(op.Filter)
				models = append(models, model)
			case interfaces.BulkReplace:
				model := mongo.NewReplaceOneModel().SetFilter(op.Filter).SetReplacement(op.Data)
				if op.Upsert {
					model.SetUpsert(true)
				}
				models = append(models, model)
			}
		}
		
		bulkOptions := options.BulkWrite().SetOrdered(false)
		bulkResult, err := collection.BulkWrite(ctx, models, bulkOptions)
		if err != nil {
			log.Error("MongoDB BulkWrite error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: bulkResult}
	}()
	
	return result
}

// CreateIndex creates indexes
func (r *MongoRepository) CreateIndex(ctx context.Context, collectionName string, indexes map[string]interface{}) <-chan error {
	result := make(chan error)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		var indexModels []mongo.IndexModel
		
		for key, value := range indexes {
			indexOption := options.Index().SetBackground(true)
			index := mongo.IndexModel{
				Keys:    bson.M{key: value},
				Options: indexOption,
			}
			indexModels = append(indexModels, index)
		}
		
		_, err := collection.Indexes().CreateMany(ctx, indexModels)
		result <- err
	}()
	
	return result
}

// DropIndex drops an index
func (r *MongoRepository) DropIndex(ctx context.Context, collectionName string, indexName string) <-chan error {
	result := make(chan error)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		_, err := collection.Indexes().DropOne(ctx, indexName)
		result <- err
	}()
	
	return result
}

// ListIndexes lists all indexes
func (r *MongoRepository) ListIndexes(ctx context.Context, collectionName string) <-chan interfaces.IndexResult {
	result := make(chan interfaces.IndexResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		cursor, err := collection.Indexes().List(ctx)
		if err != nil {
			result <- interfaces.IndexResult{Error: err}
			return
		}
		defer cursor.Close(ctx)
		
		var indexes []interfaces.IndexInfo
		for cursor.Next(ctx) {
			var index bson.M
			if err := cursor.Decode(&index); err != nil {
				result <- interfaces.IndexResult{Error: err}
				return
			}
			
			indexInfo := interfaces.IndexInfo{
				Name: index["name"].(string),
				Keys: index["key"].(bson.M),
			}
			
			if unique, ok := index["unique"]; ok {
				indexInfo.Unique = unique.(bool)
			}
			
			indexes = append(indexes, indexInfo)
		}
		
		result <- interfaces.IndexResult{Indexes: indexes}
	}()
	
	return result
}

// WithTransaction executes a function within a transaction
func (r *MongoRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	session, err := r.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)
	
	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})
	
	return err
}

// StartTransaction starts a new transaction
func (r *MongoRepository) StartTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	session, err := r.client.StartSession()
	if err != nil {
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	
	err = session.StartTransaction()
	if err != nil {
		session.EndSession(ctx)
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	
	sessionCtx := mongo.NewSessionContext(ctx, session)
	
	return &MongoTransactionContext{
		session: session,
		ctx:     sessionCtx,
	}, nil
}

// Begin starts a new transaction that implements the Transaction interface
func (r *MongoRepository) Begin(ctx context.Context) (interfaces.Transaction, error) {
	return r.BeginWithConfig(ctx, utils.DefaultTransactionConfig())
}

// BeginWithConfig starts a new transaction with enterprise configuration
func (r *MongoRepository) BeginWithConfig(ctx context.Context, config *interfaces.TransactionConfig) (interfaces.Transaction, error) {
	// Validate and merge configuration
	if err := utils.ValidateTransactionConfig(config); err != nil {
		return nil, fmt.Errorf("invalid transaction config: %w", err)
	}
	
	finalConfig := utils.MergeTransactionConfig(config)
	
	// Create timeout context
	timeoutCtx, cancel := utils.CreateTimeoutContext(ctx, finalConfig)
	
	session, err := r.client.StartSession()
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to start session: %w", err)
	}
	
	// Set transaction options based on configuration
	var transactionOpts *options.TransactionOptions
	if finalConfig.ReadOnly {
		// For read-only transactions, we can set read preference
		transactionOpts = options.Transaction().SetReadPreference(readpref.Primary())
	}
	
	err = session.StartTransaction(transactionOpts)
	if err != nil {
		session.EndSession(timeoutCtx)
		cancel()
		return nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	
	// Generate transaction ID and create metrics
	txID := utils.GenerateTransactionID()
	metrics := observability.GetGlobalMetrics().StartTransaction(txID, "mongodb", finalConfig)
	
	// Create session context that combines the caller's context with the session
	sessionCtx := mongo.NewSessionContext(timeoutCtx, session)
	
	// Create a new transaction repository that uses the session context
	txRepo := &MongoRepository{
		client:   r.client,
		database: r.database,
		dbName:   r.dbName,
	}
	
	return &MongoTransaction{
		MongoRepository: txRepo,
		session:         session,
		ctx:             sessionCtx, // This ctx is used for commit/rollback and all operations
		cancel:          cancel,
		config:          finalConfig,
		metrics:         metrics,
		transactionID:   txID,
		isActive:        1, // Set as active
		operationCount:  0,
	}, nil
}

// BeginTransaction starts a new database transaction (legacy method)
func (r *MongoRepository) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	return r.StartTransaction(ctx)
}

// Ping tests the database connection
func (r *MongoRepository) Ping(ctx context.Context) <-chan error {
	result := make(chan error)
	
	go func() {
		defer close(result)
		result <- r.client.Ping(ctx, nil)
	}()
	
	return result
}

// Close closes the database connection
func (r *MongoRepository) Close() error {
	return r.client.Disconnect(context.Background())
}

// Client returns the underlying mongo.Client.
// This is useful for administrative operations in tests, like dropping a database.
func (r *MongoRepository) Client() *mongo.Client {
	return r.client
}

// MongoQueryResult implementation
func (r *MongoQueryResult) Next() bool {
	if r.cursor == nil {
		return false
	}
	return r.cursor.Next(r.ctx)
}

func (r *MongoQueryResult) Decode(v interface{}) error {
	if r.cursor == nil {
		return fmt.Errorf("cursor is nil")
	}
	return r.cursor.Decode(v)
}

func (r *MongoQueryResult) Close() {
	if r.cursor != nil {
		r.cursor.Close(r.ctx)
	}
}

func (r *MongoQueryResult) Error() error {
	return r.err
}

// MongoSingleResult implementation
func (r *MongoSingleResult) Decode(v interface{}) error {
	if r.result == nil {
		return fmt.Errorf("result is nil")
	}
	// Normalize backend-specific no-docs error to interfaces.ErrNoDocuments
	if err := r.result.Decode(v); err != nil {
		if err == mongo.ErrNoDocuments {
			r.noResult = true
			return interfaces.ErrNoDocuments
		}
		return err
	}
	return nil
}

func (r *MongoSingleResult) Error() error {
	if r.noResult {
		return interfaces.ErrNoDocuments
	}
	return r.err
}

func (r *MongoSingleResult) NoResult() bool {
	return r.noResult
}

// MongoTransactionContext implementation
func (t *MongoTransactionContext) Commit() error {
	return t.session.CommitTransaction(t.ctx)
}

func (t *MongoTransactionContext) Rollback() error {
	return t.session.AbortTransaction(t.ctx)
}

func (t *MongoTransactionContext) Context() context.Context {
	return t.ctx
}



// UpdateFields updates specific fields using clean syntax (no $set, $inc operators)
func (r *MongoRepository) UpdateFields(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Convert clean syntax to MongoDB $set operation
		data := map[string]interface{}{
			"$set": updates,
		}
		
		_, err := collection.UpdateMany(ctx, filter, data)
		if err != nil {
			log.Error("MongoDB UpdateFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

// IncrementFields increments numeric fields using clean syntax (no $inc operators)
func (r *MongoRepository) IncrementFields(ctx context.Context, collectionName string, filter interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Convert clean syntax to MongoDB $inc operation
		data := map[string]interface{}{
			"$inc": increments,
		}
		
		_, err := collection.UpdateMany(ctx, filter, data)
		if err != nil {
			log.Error("MongoDB IncrementFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

// UpdateAndIncrement performs both update and increment operations
func (r *MongoRepository) UpdateAndIncrement(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Convert clean syntax to MongoDB $set and $inc operations
		data := map[string]interface{}{
			"$set": updates,
			"$inc": increments,
		}
		
		_, err := collection.UpdateMany(ctx, filter, data)
		if err != nil {
			log.Error("MongoDB UpdateAndIncrement error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

// UpdateWithOwnership performs atomic update with ownership validation (optimized)
func (r *MongoRepository) UpdateWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Build filter with ownership validation
		filter := map[string]interface{}{
			"objectId":    entityID,
			"ownerUserId": ownerID,
			"deleted":     false,
		}
		
		// Add lastUpdated timestamp
		updates["lastUpdated"] = time.Now().Unix()
		
		// Execute atomic update with ownership validation
		updateResult, err := collection.UpdateOne(ctx, filter, map[string]interface{}{
			"$set": updates,
		})
		if err != nil {
			log.Error("MongoDB UpdateWithOwnership error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Check if any documents were affected (entity existed and belonged to owner)
		if updateResult.MatchedCount == 0 {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("entity not found or unauthorized")}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: updateResult.ModifiedCount}
	}()
	
	return result
}

// DeleteWithOwnership performs atomic delete with ownership validation (optimized)
func (r *MongoRepository) DeleteWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Build filter with ownership validation
		filter := map[string]interface{}{
			"objectId":    entityID,
			"ownerUserId": ownerID,
			"deleted":     false,
		}
		
		// Execute atomic soft delete with ownership validation
		updateResult, err := collection.UpdateOne(ctx, filter, map[string]interface{}{
			"$set": map[string]interface{}{
				"deleted":     true,
				"deletedDate": time.Now().Unix(),
			},
		})
		if err != nil {
			log.Error("MongoDB DeleteWithOwnership error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Check if any documents were affected (entity existed and belonged to owner)
		if updateResult.MatchedCount == 0 {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("entity not found or unauthorized")}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: updateResult.ModifiedCount}
	}()
	
	return result
}

// IncrementWithOwnership performs atomic increment with ownership validation (optimized)
func (r *MongoRepository) IncrementWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		// Build filter with ownership validation
		filter := map[string]interface{}{
			"objectId":    entityID,
			"ownerUserId": ownerID,
			"deleted":     false,
		}
		
		// Execute atomic increment with ownership validation
		updateResult, err := collection.UpdateOne(ctx, filter, map[string]interface{}{
			"$inc": increments,
		})
		if err != nil {
			log.Error("MongoDB IncrementWithOwnership error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Check if any documents were affected (entity existed and belonged to owner)
		if updateResult.MatchedCount == 0 {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("entity not found or unauthorized")}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: updateResult.ModifiedCount}
	}()
	
	return result
}

// FindWithCursor retrieves documents with cursor-based pagination
func (r *MongoRepository) FindWithCursor(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		findOptions := options.Find()
		
		if opts != nil {
			if opts.Limit != nil {
				findOptions.SetLimit(*opts.Limit)
			}
			
			// Build sort option based on sort field and direction
			sortField := opts.SortField
			if sortField == "" {
				sortField = "createdDate" // Default sort field
			}
			
			sortDirection := -1 // Default desc
			if opts.SortDirection == "asc" {
				sortDirection = 1
			}
			
			// For compound sorting, always include objectId as tiebreaker for cross-DB consistency
			sortOrder := bson.D{
				{Key: sortField, Value: sortDirection},
				{Key: "objectId", Value: sortDirection},
			}
			findOptions.SetSort(sortOrder)
		}
		
		cursor, err := collection.Find(ctx, filter, findOptions)
		if err != nil {
			log.Error("MongoDB FindWithCursor error: %s", err.Error())
			result <- &MongoQueryResult{err: err}
			return
		}
		
		result <- &MongoQueryResult{cursor: cursor, ctx: ctx}
	}()
	
	return result
}

// CountWithFilter counts documents matching the filter
func (r *MongoRepository) CountWithFilter(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)
	
	go func() {
		defer close(result)
		
		collection := r.database.Collection(collectionName)
		
		count, err := collection.CountDocuments(ctx, filter)
		if err != nil {
			log.Error("MongoDB CountWithFilter error: %s", err.Error())
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}
		
		result <- interfaces.CountResult{Count: count, Error: nil}
	}()
	
	return result
}