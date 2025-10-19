// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"sort"
	"strconv"

	"github.com/lib/pq"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/observability"
	"github.com/qolzam/telar/apps/api/internal/database/utils"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
)

// PostgreSQLRepository implements the Repository interface for PostgreSQL
type PostgreSQLRepository struct {
	db     *sql.DB
	dbName string
	schema string
}

// PostgreSQLQueryResult implements QueryResult for PostgreSQL
type PostgreSQLQueryResult struct {
	rows    *sql.Rows
	err     error
	columns []string
	closed  bool
}

// PostgreSQLSingleResult implements QuerySingleResult for PostgreSQL
type PostgreSQLSingleResult struct {
	row      *sql.Row
	err      error
	noResult bool
	columns  []string
}

// PostgreSQLTransactionContext implements TransactionContext for PostgreSQL (legacy)
type PostgreSQLTransactionContext struct {
	tx  *sql.Tx
	ctx context.Context
}

// NewPostgreSQLRepository creates a new PostgreSQL repository
func NewPostgreSQLRepository(ctx context.Context, config *interfaces.PostgreSQLConfig, databaseName string) (*PostgreSQLRepository, error) {
	// Build connection string
	connStr := buildConnectionString(config, databaseName)

	// Open database connection
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open PostgreSQL connection: %w", err)
	}

	// Configure connection pool
	if config.MaxOpenConnections > 0 {
		db.SetMaxOpenConns(config.MaxOpenConnections)
	}
	if config.MaxIdleConnections > 0 {
		db.SetMaxIdleConns(config.MaxIdleConnections)
	}

	if config.MaxLifetime > 0 {
		db.SetConnMaxLifetime(time.Duration(config.MaxLifetime) * time.Second)
	}

	// Test the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping PostgreSQL: %w", err)
	}

	schema := "public"
	if config.Schema != "" {
		schema = config.Schema
	}

	repo := &PostgreSQLRepository{
		db:     db,
		dbName: databaseName,
		schema: schema,
	}

	// Initialize schema if needed
	if err := repo.initializeSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return repo, nil
}

// buildConnectionString builds PostgreSQL connection string from config
func buildConnectionString(config *interfaces.PostgreSQLConfig, databaseName string) string {
	var parts []string

	parts = append(parts, fmt.Sprintf("host=%s", config.Host))
	parts = append(parts, fmt.Sprintf("port=%d", config.Port))
	parts = append(parts, fmt.Sprintf("dbname=%s", databaseName))

	if config.Username != "" {
		parts = append(parts, fmt.Sprintf("user=%s", config.Username))
	}

	if config.Password != "" {
		parts = append(parts, fmt.Sprintf("password=%s", config.Password))
	}

	parts = append(parts, fmt.Sprintf("sslmode=%s", config.SSLMode))

	if config.ConnectTimeout > 0 {
		parts = append(parts, fmt.Sprintf("connect_timeout=%d", config.ConnectTimeout))
	}

	return strings.Join(parts, " ")
}

// initializeSchema creates necessary tables and indexes
func (r *PostgreSQLRepository) initializeSchema(ctx context.Context) error {
	// This would typically be handled by migrations, but for demo purposes:
	// Create a generic JSONB table structure that can handle document-like data

	queries := []string{
		fmt.Sprintf(`CREATE SCHEMA IF NOT EXISTS %s`, r.schema),
		// This is a flexible approach - each "collection" becomes a table with JSONB data
		// In production, you might want specific schemas for each entity type
	}

	for _, query := range queries {
		if _, err := r.db.ExecContext(ctx, query); err != nil {
			log.Error("PostgreSQL schema initialization error: %s", err.Error())
			return err
		}
	}

	return nil
}

// generateIndexName creates collection-specific index names to avoid conflicts
func (r *PostgreSQLRepository) generateIndexName(collectionName, indexType string) string {
	// Sanitize collection name for use in index names
	sanitized := strings.ReplaceAll(collectionName, "-", "_")
	sanitized = strings.ReplaceAll(sanitized, ".", "_")
	return fmt.Sprintf("idx_%s_%s", sanitized, indexType)
}

// ensureTable ensures the table exists
func (r *PostgreSQLRepository) ensureTable(ctx context.Context, collectionName string) error {
	tableName := r.getTableName(collectionName)

	// Check if table already exists first to make this function idempotent.
	var exists bool
	checkQuery := `SELECT EXISTS (
		SELECT FROM information_schema.tables 
		WHERE table_schema = $1 AND table_name = $2
	)`

	err := r.db.QueryRowContext(ctx, checkQuery, r.schema, collectionName).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check for existence of table %s: %w", tableName, err)
	}

	if exists {
		// Table already exists, our job is done.
		return nil
	}

	// If we get here, the table does not exist. Proceed with creation.
	createQuery := fmt.Sprintf(`
	CREATE TABLE %s (
		id BIGSERIAL PRIMARY KEY,
		object_id VARCHAR(255) UNIQUE NOT NULL,
		data JSONB NOT NULL,
		created_date BIGINT,
		last_updated BIGINT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`, tableName)

	_, err = r.db.ExecContext(ctx, createQuery)
	if err != nil {
		// Handle concurrent table creation errors
		if pgErr, ok := err.(*pq.Error); ok {
			switch pgErr.Code {
			case "42P07": // duplicate_table - standard duplicate table error
				return nil
			case "23505": // unique_violation - system catalog constraint violation (pg_class_relname_nsp_index)
				if strings.Contains(pgErr.Message, "pg_class_relname_nsp_index") {
					// Another process created the table between our check and create attempt
					return nil
				}
			}
		}
		return fmt.Errorf("failed to create table %s: %w", tableName, err)
	}

	// Create indexes for performance with collection-specific names
	indexQueries := []string{
		fmt.Sprintf("CREATE INDEX %s ON %s (object_id)",
			r.generateIndexName(collectionName, "object_id"), tableName),
		fmt.Sprintf("CREATE INDEX %s ON %s (created_date)",
			r.generateIndexName(collectionName, "created_date"), tableName),
		fmt.Sprintf("CREATE INDEX %s ON %s (last_updated)",
			r.generateIndexName(collectionName, "last_updated"), tableName),
		fmt.Sprintf("CREATE INDEX %s ON %s USING GIN (data)",
			r.generateIndexName(collectionName, "data_gin"), tableName),
		// Composite index to optimize cursor pagination and ordering
		fmt.Sprintf("CREATE INDEX %s ON %s (created_date, object_id)",
			r.generateIndexName(collectionName, "created_object"), tableName),
	}

	for _, indexQuery := range indexQueries {
		if _, err := r.db.ExecContext(ctx, indexQuery); err != nil {
			log.Warn("Failed to create index: %s", err.Error())
			// Continue with other indexes
		}
	}

	return nil
}

// getTableName returns the table name for a collection
func (r *PostgreSQLRepository) getTableName(collectionName string) string {
	return fmt.Sprintf("%s.%s", r.schema, collectionName)
}

// Save stores a single document
func (r *PostgreSQLRepository) Save(ctx context.Context, collectionName string, data interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Convert data to JSON
		jsonData, err := json.Marshal(data)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to marshal data: %w", err)}
			return
		}

		// Extract common fields if they exist
		objectID, createdDate, lastUpdated := r.extractCommonFields(data)

		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf(`
			INSERT INTO %s (object_id, data, created_date, last_updated) 
			VALUES ($1, $2, $3, $4) 
			RETURNING id`, tableName)

		var id int64
		err = r.db.QueryRowContext(ctx, query, objectID, jsonData, createdDate, lastUpdated).Scan(&id)
		if err != nil {
			if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" { // Unique violation
				result <- interfaces.RepositoryResult{Error: interfaces.ErrDuplicateKey}
				return
			}
			log.Error("PostgreSQL Save error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		result <- interfaces.RepositoryResult{Result: id}
	}()

	return result
}

// SaveMany stores multiple documents
func (r *PostgreSQLRepository) SaveMany(ctx context.Context, collectionName string, data []interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		if len(data) == 0 {
			result <- interfaces.RepositoryResult{Result: []int64{}}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build bulk insert query
		valueStrings := make([]string, 0, len(data))
		valueArgs := make([]interface{}, 0, len(data)*4)

		for i, item := range data {
			jsonData, err := json.Marshal(item)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to marshal data at index %d: %w", i, err)}
				return
			}

			objectID, createdDate, lastUpdated := r.extractCommonFields(item)

			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4))
			valueArgs = append(valueArgs, objectID, jsonData, createdDate, lastUpdated)
		}

		query := fmt.Sprintf(`
			INSERT INTO %s (object_id, data, created_date, last_updated) 
			VALUES %s 
			RETURNING id`, tableName, strings.Join(valueStrings, ","))

		rows, err := r.db.QueryContext(ctx, query, valueArgs...)
		if err != nil {
			log.Error("PostgreSQL SaveMany error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		defer rows.Close()

		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				result <- interfaces.RepositoryResult{Error: err}
				return
			}
			ids = append(ids, id)
		}

		result <- interfaces.RepositoryResult{Result: ids}
	}()

	return result
}

// Find retrieves multiple documents
func (r *PostgreSQLRepository) Find(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.FindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build query
		query := fmt.Sprintf("SELECT data FROM %s", tableName)
		whereClause, args, err := r.buildWhereClause(filter)
		if err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		// Add sorting
		if opts != nil && opts.Sort != nil {
			orderBy := r.buildOrderByClause(opts.Sort)
			if orderBy != "" {
				query += " ORDER BY " + orderBy
			}
		}

		// Add limit and offset
		if opts != nil {
			if opts.Limit != nil {
				query += fmt.Sprintf(" LIMIT %d", *opts.Limit)
			}
			if opts.Skip != nil {
				query += fmt.Sprintf(" OFFSET %d", *opts.Skip)
			}
		}

		// Log the actual SQL query being executed (for debugging)
		// log.Info("PostgreSQL Find query: %s", query)
		// log.Info("PostgreSQL Find args: %v", args)
		
		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL Find error: %s", err.Error())
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		result <- &PostgreSQLQueryResult{rows: rows, columns: []string{"data"}}
	}()

	return result
}

// FindOne retrieves a single document
func (r *PostgreSQLRepository) FindOne(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.SingleResult {
	result := make(chan interfaces.SingleResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- &PostgreSQLSingleResult{err: err}
			return
		}

		tableName := r.getTableName(collectionName)

		whereClause, args, err := r.buildWhereClause(filter)
		if err != nil {
			result <- &PostgreSQLSingleResult{err: err}
			return
		}

		query := fmt.Sprintf("SELECT data FROM %s", tableName)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}
		query += " LIMIT 1"

		row := r.db.QueryRowContext(ctx, query, args...)
		result <- &PostgreSQLSingleResult{row: row, columns: []string{"data"}}
	}()

	return result
}

// Update updates documents matching the filter
func (r *PostgreSQLRepository) Update(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		if opts != nil && opts.Upsert != nil && *opts.Upsert {
			result <- r.upsertOperation(ctx, collectionName, filter, data)
			return
		}

		// Build UPDATE clause first
		updateClause, updateArgs, err := r.buildUpdateClause(data)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause with offset after UPDATE parameters
		whereClause, args, err := r.buildWhereClauseWithOffset(filter, len(updateArgs)+1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Combine all arguments
		allArgs := append(updateArgs, args...)

		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, updateClause, whereClause)

		_, err = r.db.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Update error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// upsertOperation implements UPSERT using PostgreSQL's INSERT ... ON CONFLICT
func (r *PostgreSQLRepository) upsertOperation(ctx context.Context, collectionName string, filter interface{}, data interface{}) interfaces.RepositoryResult {
	tableName := r.getTableName(collectionName)
	
	filterMap, ok := filter.(map[string]interface{})
	if !ok {
		jsonData, err := json.Marshal(filter)
		if err != nil {
			return interfaces.RepositoryResult{Error: fmt.Errorf("upsert: failed to marshal filter: %w", err)}
		}
		filterMap = make(map[string]interface{})
		if err := json.Unmarshal(jsonData, &filterMap); err != nil {
			return interfaces.RepositoryResult{Error: fmt.Errorf("upsert: failed to unmarshal filter: %w", err)}
		}
	}
	
	objectId, ok := filterMap["objectId"]
	if !ok {
		return interfaces.RepositoryResult{Error: fmt.Errorf("upsert: objectId not found in filter")}
	}
	
	// Extract data from $set operator or use data directly
	dataMap := make(map[string]interface{})
	if m, ok := data.(map[string]interface{}); ok {
		if setVal, hasSet := m["$set"]; hasSet {
			if setMap, ok2 := setVal.(map[string]interface{}); ok2 {
				dataMap = setMap
			}
		} else {
			dataMap = m
		}
	} else {
		return interfaces.RepositoryResult{Error: fmt.Errorf("upsert: data must be a map")}
	}
	
	// Ensure objectId is in the data
	dataMap["objectId"] = objectId
	
	// Marshal to JSON
	jsonData, err := json.Marshal(dataMap)
	if err != nil {
		return interfaces.RepositoryResult{Error: fmt.Errorf("upsert: failed to marshal data: %w", err)}
	}
	
	now := time.Now().Unix()
	
	// PostgreSQL UPSERT using INSERT ... ON CONFLICT
	query := fmt.Sprintf(`
		INSERT INTO %s (object_id, data, created_date, last_updated)
		VALUES ($1, $2::jsonb, $3, $4)
		ON CONFLICT (object_id) DO UPDATE
		SET data = $2::jsonb, last_updated = $4
	`, tableName)
	
	_, err = r.db.ExecContext(ctx, query, objectId, string(jsonData), now, now)
	if err != nil {
		log.Error("PostgreSQL Upsert error: %s", err.Error())
		return interfaces.RepositoryResult{Error: err}
	}
	
	log.Info("PostgreSQL: Upsert operation completed for objectId: %v", objectId)
	return interfaces.RepositoryResult{Result: "OK"}
}

// UpdateMany updates multiple documents
func (r *PostgreSQLRepository) UpdateMany(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	// For PostgreSQL, UpdateMany is the same as Update since we don't have document-level operations
	return r.Update(ctx, collectionName, filter, data, opts)
}

// Delete deletes documents matching the filter
func (r *PostgreSQLRepository) Delete(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause
		whereClause, args, err := r.buildWhereClause(filter)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)

		_, err = r.db.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL Delete error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// DeleteMany performs bulk delete operations for multiple individual filters
func (r *PostgreSQLRepository) DeleteMany(ctx context.Context, collectionName string, filters []interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Use a transaction for bulk delete operations
		tx, err := r.db.BeginTx(ctx, nil)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		defer tx.Rollback()

		tableName := r.getTableName(collectionName)
		var totalDeleted int64

		for _, filter := range filters {
			// Build WHERE clause for each filter
			whereClause, args, err := r.buildWhereClause(filter)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: err}
				return
			}

			query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)
			res, err := tx.ExecContext(ctx, query, args...)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: err}
				return
			}

			deleted, err := res.RowsAffected()
			if err != nil {
				result <- interfaces.RepositoryResult{Error: err}
				return
			}
			totalDeleted += deleted
		}

		// Commit the transaction
		if err := tx.Commit(); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		result <- interfaces.RepositoryResult{Result: totalDeleted}
	}()

	return result
}

// Aggregate performs aggregation operations (simplified for PostgreSQL)
func (r *PostgreSQLRepository) Aggregate(ctx context.Context, collectionName string, pipeline interface{}) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)

	go func() {
		defer close(result)

		// This is a simplified implementation
		// In a real implementation, you'd need to translate MongoDB aggregation pipeline to SQL
		result <- &PostgreSQLQueryResult{err: interfaces.ErrUnsupportedOperation}
	}()

	return result
}

// Count counts documents matching filter
func (r *PostgreSQLRepository) Count(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.CountResult{Error: err}
			return
		}

		// Build WHERE clause
		whereClause, args, err := r.buildWhereClause(filter)
		if err != nil {
			result <- interfaces.CountResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		var count int64
		err = r.db.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			log.Error("PostgreSQL Count error: %s", err.Error())
			result <- interfaces.CountResult{Error: err}
			return
		}

		result <- interfaces.CountResult{Count: count}
	}()

	return result
}

// Distinct gets distinct values for a field
func (r *PostgreSQLRepository) Distinct(ctx context.Context, collectionName string, field string, filter interface{}) <-chan interfaces.DistinctResult {
	result := make(chan interfaces.DistinctResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.DistinctResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Use JSONB operators to extract distinct values
		query := fmt.Sprintf("SELECT DISTINCT data->>'%s' FROM %s", field, tableName)
		whereClause, args, err := r.buildWhereClause(filter)
		if err != nil {
			result <- interfaces.DistinctResult{Error: err}
			return
		}

		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL Distinct error: %s", err.Error())
			result <- interfaces.DistinctResult{Error: err}
			return
		}
		defer rows.Close()

		var values []interface{}
		for rows.Next() {
			var value interface{}
			if err := rows.Scan(&value); err != nil {
				result <- interfaces.DistinctResult{Error: err}
				return
			}
			values = append(values, value)
		}

		result <- interfaces.DistinctResult{Values: values}
	}()

	return result
}

// BulkWrite performs bulk operations
func (r *PostgreSQLRepository) BulkWrite(ctx context.Context, collectionName string, operations []interfaces.BulkOperation) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		// This would require implementing a complex bulk operation handler
		// For now, return unsupported
		result <- interfaces.RepositoryResult{Error: interfaces.ErrUnsupportedOperation}
	}()

	return result
}

// CreateIndex creates indexes for the collection
func (r *PostgreSQLRepository) CreateIndex(ctx context.Context, collectionName string, indexes map[string]interface{}) <-chan error {
	result := make(chan error)

	go func() {
		defer close(result)

		// PostgreSQL doesn't support dynamic index creation like MongoDB
		// This is a no-op for PostgreSQL as indexes are created during table creation
		result <- nil
	}()

	return result
}

// DropIndex drops an index
func (r *PostgreSQLRepository) DropIndex(ctx context.Context, collectionName string, indexName string) <-chan error {
	result := make(chan error)

	go func() {
		defer close(result)

		query := fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
		_, err := r.db.ExecContext(ctx, query)
		result <- err
	}()

	return result
}

// ListIndexes lists all indexes
func (r *PostgreSQLRepository) ListIndexes(ctx context.Context, collectionName string) <-chan interfaces.IndexResult {
	result := make(chan interfaces.IndexResult)

	go func() {
		defer close(result)

		tableName := r.getTableName(collectionName)

		query := `
                        SELECT indexname, indexdef 
                        FROM pg_indexes 
                        WHERE tablename = $1 AND schemaname = $2`

		rows, err := r.db.QueryContext(ctx, query, tableName, r.schema)
		if err != nil {
			result <- interfaces.IndexResult{Error: err}
			return
		}
		defer rows.Close()

		var indexes []interfaces.IndexInfo
		for rows.Next() {
			var name, def string
			if err := rows.Scan(&name, &def); err != nil {
				result <- interfaces.IndexResult{Error: err}
				return
			}

			indexes = append(indexes, interfaces.IndexInfo{
				Name: name,
				Keys: map[string]interface{}{"definition": def},
			})
		}

		result <- interfaces.IndexResult{Indexes: indexes}
	}()

	return result
}

// WithTransaction executes a function within a transaction
func (r *PostgreSQLRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create transaction context
	txCtx := &PostgreSQLTransactionContext{
		tx:  tx,
		ctx: ctx,
	}

	// Execute the function
	if err := fn(txCtx.Context()); err != nil {
		// Rollback on error
		if rollbackErr := tx.Rollback(); rollbackErr != nil {
			return fmt.Errorf("failed to rollback transaction: %w (original error: %w)", rollbackErr, err)
		}
		return err
	}

	// Commit on success
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// StartTransaction starts a new transaction
func (r *PostgreSQLRepository) StartTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	return &PostgreSQLTransactionContext{
		tx:  tx,
		ctx: ctx,
	}, nil
}

// Begin starts a new transaction that implements the Transaction interface
func (r *PostgreSQLRepository) Begin(ctx context.Context) (interfaces.Transaction, error) {
	return r.BeginWithConfig(ctx, utils.DefaultTransactionConfig())
}

// BeginWithConfig starts a new transaction with enterprise configuration
func (r *PostgreSQLRepository) BeginWithConfig(ctx context.Context, config *interfaces.TransactionConfig) (interfaces.Transaction, error) {
	// Validate and merge configuration
	if err := utils.ValidateTransactionConfig(config); err != nil {
		return nil, fmt.Errorf("invalid transaction config: %w", err)
	}

	finalConfig := utils.MergeTransactionConfig(config)

	// Create timeout context
	timeoutCtx, cancel := utils.CreateTimeoutContext(ctx, finalConfig)

	// Set transaction options based on configuration
	opts := &sql.TxOptions{
		Isolation: utils.ConvertIsolationLevel(finalConfig.IsolationLevel),
		ReadOnly:  finalConfig.ReadOnly,
	}

	// Begin transaction with options
	tx, err := r.db.BeginTx(timeoutCtx, opts)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Generate transaction ID and create metrics
	txID := utils.GenerateTransactionID()
	metrics := observability.GetGlobalMetrics().StartTransaction(txID, "postgresql", finalConfig)

	// Create a new transaction repository that uses the transaction
	txRepo := &PostgreSQLRepository{
		db:     r.db,
		dbName: r.dbName,
		schema: r.schema,
	}

	return &PostgreSQLTransaction{
		PostgreSQLRepository: txRepo,
		tx:                   tx,
		ctx:                  timeoutCtx,
		cancel:               cancel,
		config:               finalConfig,
		metrics:              metrics,
		transactionID:        txID,
		isActive:             1, // Set as active
		operationCount:       0,
	}, nil
}

// BeginTransaction starts a new database transaction (legacy method)
func (r *PostgreSQLRepository) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}

	return &PostgreSQLTransactionContext{
		tx:  tx,
		ctx: ctx,
	}, nil
}

// Ping tests the database connection
func (r *PostgreSQLRepository) Ping(ctx context.Context) <-chan error {
	result := make(chan error)

	go func() {
		defer close(result)
		result <- r.db.PingContext(ctx)
	}()

	return result
}

// Close closes the database connection
func (r *PostgreSQLRepository) Close() error {
	return r.db.Close()
}

// Helper methods

// mapFieldToColumn maps camelCase application field names to snake_case database column names
func (r *PostgreSQLRepository) mapFieldToColumn(fieldName string) string {
	switch fieldName {
	case "objectId":
		return "object_id"
	case "createdDate":
		return "created_date"
	case "lastUpdated":
		return "last_updated"
	default:
		return fieldName
	}
}

// extractCommonFields extracts common fields from data using reflection
func (r *PostgreSQLRepository) extractCommonFields(data interface{}) (interface{}, interface{}, interface{}) {
	var objectID, createdDate, lastUpdated interface{}

	// Use reflection to extract common fields
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() == reflect.Struct {
		// Map camelCase application fields to snake_case database columns
		if field := val.FieldByName("ObjectId"); field.IsValid() {
			rawValue := field.Interface()
			// Convert UUID types to string for PostgreSQL compatibility
			if uuidVal, ok := rawValue.(fmt.Stringer); ok {
				objectID = uuidVal.String()
			} else {
				objectID = rawValue
			}
		}
		if field := val.FieldByName("CreatedDate"); field.IsValid() {
			createdDate = field.Interface()
		}
		if field := val.FieldByName("LastUpdated"); field.IsValid() {
			lastUpdated = field.Interface()
		}
	} else if val.Kind() == reflect.Map {
		// Handle map[string]interface{} which is common in tests
		if mapData, ok := data.(map[string]interface{}); ok {
			if value, exists := mapData["objectId"]; exists {
				// Convert UUID types to string for PostgreSQL compatibility
				if uuidVal, ok := value.(fmt.Stringer); ok {
					objectID = uuidVal.String()
				} else {
					objectID = value
				}
			}
			if value, exists := mapData["created"]; exists {
				createdDate = value
			}
			if value, exists := mapData["last_updated"]; exists {
				lastUpdated = value
			}
		}
	}

	return objectID, createdDate, lastUpdated
}

// getMapKeys returns the keys of a map for debugging
func getMapKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// buildWhereClause builds a WHERE clause from filter
func (r *PostgreSQLRepository) buildWhereClause(filter interface{}) (string, []interface{}, error) {
	return r.buildWhereClauseWithOffset(filter, 1)
}

// buildWhereClauseWithOffset builds a WHERE clause using a starting placeholder index
func (r *PostgreSQLRepository) buildWhereClauseWithOffset(filter interface{}, startIndex int) (string, []interface{}, error) {
	if filter == nil {
		return "", nil, nil
	}

	// Convert filter to map[string]interface{} if it's not already
	// This allows supporting both map filters (Profile service) and struct filters (Auth service)
	var filterMap map[string]interface{}
	
	switch f := filter.(type) {
	case map[string]interface{}:
		// Already a map - use directly (zero overhead)
		filterMap = f
	default:
		// Convert struct (or other type) to map via JSON marshaling
		// This handles structs with json/bson tags commonly used in MongoDB-compatible code
		jsonData, err := json.Marshal(filter)
		if err != nil {
			return "", nil, fmt.Errorf("failed to marshal filter to JSON: %w", err)
		}
		
		filterMap = make(map[string]interface{})
		if err := json.Unmarshal(jsonData, &filterMap); err != nil {
			return "", nil, fmt.Errorf("failed to unmarshal filter to map: %w", err)
		}
		
		// Log for debugging (helps track struct filter usage)
		log.Info("PostgreSQL: Converted struct filter to map for WHERE clause")
	}

	// Support Mongo-like filters including $or and comparison operators for known fields
	conditions := make([]string, 0)
	args := make([]interface{}, 0)
	argIndex := startIndex

	// Helper to append a condition and its args
	appendCond := func(cond string, vals ...interface{}) {
		conditions = append(conditions, cond)
		args = append(args, vals...)
		argIndex += len(vals)
	}

		// First handle $or if present
		if orVal, hasOr := filterMap["$or"]; hasOr {
			var orConds []string
			var orItems []interface{}
			
			// Handle both []interface{} and []map[string]interface{}
			switch list := orVal.(type) {
			case []interface{}:
				orItems = list
			case []map[string]interface{}:
				// Convert []map[string]interface{} to []interface{}
				orItems = make([]interface{}, len(list))
				for i, item := range list {
					orItems[i] = item
				}
			default:
				return "", nil, fmt.Errorf("unsupported $or type: %T", orVal)
			}
			
			orConds = make([]string, 0, len(orItems))
			for _, item := range orItems {
				if sub, ok2 := item.(map[string]interface{}); ok2 {
					clause, subArgs, err := r.buildWhereClauseWithOffset(sub, argIndex)
					if err != nil {
						return "", nil, err
					}
					if clause != "" {
						orConds = append(orConds, "("+clause+")")
						args = append(args, subArgs...)
						argIndex += len(subArgs)
					}
				}
			}
			
			if len(orConds) > 0 {
				conditions = append(conditions, "("+strings.Join(orConds, " OR ")+")")
			}
		}

		// Process objectId (supports equality and { $gt/$lt/$gte/$lte })
		if value, hasObjectId := filterMap["objectId"]; hasObjectId {
			if opMap, ok2 := value.(map[string]interface{}); ok2 {
				for op, v := range opMap {
					column := "object_id"
					switch op {
					case "$gt":
						appendCond(fmt.Sprintf("%s > $%d", column, argIndex), fmt.Sprint(v))
					case "$lt":
						appendCond(fmt.Sprintf("%s < $%d", column, argIndex), fmt.Sprint(v))
					case "$gte":
						appendCond(fmt.Sprintf("%s >= $%d", column, argIndex), fmt.Sprint(v))
					case "$lte":
						appendCond(fmt.Sprintf("%s <= $%d", column, argIndex), fmt.Sprint(v))
					case "$ne":
						appendCond(fmt.Sprintf("%s <> $%d", column, argIndex), fmt.Sprint(v))
					}
				}
			} else {
				appendCond(fmt.Sprintf("object_id = $%d", argIndex), fmt.Sprint(value))
			}
		}

		// Process remaining fields
		for key, value := range filterMap {
			if key == "$or" || key == "objectId" {
				continue
			}

			// Special handling for deleted boolean
			if key == "deleted" {
				if boolVal, ok := value.(bool); ok {
					appendCond(fmt.Sprintf("(data->>'%s')::boolean = $%d", key, argIndex), boolVal)
				} else {
					appendCond(fmt.Sprintf("data->>'%s' = $%d", key, argIndex), value)
				}
				continue
			}


			// Comparison and advanced operators support for known fields and JSONB values
			if opMap, ok2 := value.(map[string]interface{}); ok2 {
				for op, v := range opMap {
					var column string
					switch key {
					case "createdDate":
						column = "created_date"
					case "lastUpdated":
						column = "last_updated"
					case "score", "viewCount", "commentCounter":
						column = fmt.Sprintf("(data->>'%s')::bigint", key)
					default:
						column = fmt.Sprintf("(data->>'%s')", key)
					}

					// Handle $regex with optional $options (e.g., case-insensitive)
					if op == "$regex" {
						pattern := fmt.Sprint(v)
						operator := "~" // case-sensitive by default
						if opt, okOpt := opMap["$options"].(string); okOpt && strings.Contains(opt, "i") {
							operator = "~*"
						}
						appendCond(fmt.Sprintf("(data->>'%s') %s $%d", key, operator, argIndex), pattern)
						continue
					}

					// Handle $in for JSONB array fields (expects array of strings)
					if op == "$in" {
						switch vals := v.(type) {
						case []string:
							appendCond(fmt.Sprintf("data->'%s' ?| $%d", key, argIndex), pq.Array(vals))
						case []interface{}:
							strValues := make([]string, 0, len(vals))
							for _, item := range vals {
								strValues = append(strValues, fmt.Sprint(item))
							}
							appendCond(fmt.Sprintf("data->'%s' ?| $%d", key, argIndex), pq.Array(strValues))
						default:
							return "", nil, fmt.Errorf("invalid type for $in operator on field '%s', expected a slice", key)
						}
						continue
					}

					// Handle $all for JSONB array fields (expects array of strings)
					if op == "$all" {
						switch vals := v.(type) {
						case []string:
							// Convert to JSONB array for PostgreSQL
							jsonArray, err := json.Marshal(vals)
							if err != nil {
								return "", nil, fmt.Errorf("failed to marshal $all values: %w", err)
							}
							appendCond(fmt.Sprintf("data->'%s' @> $%d", key, argIndex), string(jsonArray))
						case []interface{}:
							strValues := make([]string, 0, len(vals))
							for _, item := range vals {
								strValues = append(strValues, fmt.Sprint(item))
							}
							// Convert to JSONB array for PostgreSQL
							jsonArray, err := json.Marshal(strValues)
							if err != nil {
								return "", nil, fmt.Errorf("failed to marshal $all values: %w", err)
							}
							appendCond(fmt.Sprintf("data->'%s' @> $%d", key, argIndex), string(jsonArray))
						default:
							return "", nil, fmt.Errorf("invalid type for $all operator on field '%s', expected a slice", key)
						}
						continue
					}

					param := fmt.Sprint(v)
					switch op {
					case "$gt":
						appendCond(fmt.Sprintf("%s > $%d", column, argIndex), param)
					case "$lt":
						appendCond(fmt.Sprintf("%s < $%d", column, argIndex), param)
					case "$gte":
						appendCond(fmt.Sprintf("%s >= $%d", column, argIndex), param)
					case "$lte":
						appendCond(fmt.Sprintf("%s <= $%d", column, argIndex), param)
					case "$ne":
						appendCond(fmt.Sprintf("%s <> $%d", column, argIndex), param)
					}
				}
				continue
			}

			// Handle __in suffix for array-based IN queries (PostgreSQL-specific convention)
			if strings.HasSuffix(key, "__in") {
				realKey := strings.TrimSuffix(key, "__in")
				// This is our convention for an IN clause
				if values, ok := value.([]string); ok {
					appendCond(fmt.Sprintf("(data->>'%s') = ANY($%d)", realKey, argIndex), pq.Array(values))
				}
				continue
			}

			// Handle nested fields with dot notation (e.g., "leftUser.userId")
			if strings.Contains(key, ".") {
				// Convert dot notation to JSONB path (e.g., "leftUser.userId" -> "leftUser"->>'userId')
				segments := strings.Split(key, ".")
				if len(segments) == 2 {
					// Simple case: "leftUser.userId" -> data->'leftUser'->>'userId'
					appendCond(fmt.Sprintf("data->'%s'->>'%s' = $%d", segments[0], segments[1], argIndex), value)
				} else {
					// Complex case: "a.b.c" -> data->'a'->'b'->>'c'
					path := fmt.Sprintf("data->'%s'", segments[0])
					for i := 1; i < len(segments)-1; i++ {
						path = fmt.Sprintf("%s->'%s'", path, segments[i])
					}
					path = fmt.Sprintf("%s->>'%s'", path, segments[len(segments)-1])
					appendCond(fmt.Sprintf("%s = $%d", path, argIndex), value)
				}
			} else {
				// Equality fallback for non-nested fields
				appendCond(fmt.Sprintf("data->>'%s' = $%d", key, argIndex), value)
			}
		}

		if len(conditions) > 0 {
			return strings.Join(conditions, " AND "), args, nil
		}

	return "", nil, nil
}

// buildUpdateClause builds an UPDATE clause from data
func (r *PostgreSQLRepository) buildUpdateClause(data interface{}) (string, []interface{}, error) {
	// Handle plain field updates (cleaner syntax)
	if fieldMap, ok := data.(map[string]interface{}); ok {
		// Check if this is a plain field update (no $set, $inc, etc.)
		hasOperators := false
		for key := range fieldMap {
			if len(key) > 0 && key[0] == '$' {
				hasOperators = true
				break
			}
		}

		if !hasOperators {
			// Plain field update - treat as $set operation
			return r.buildSetOperation(fieldMap)
		}
	}

	// Handle MongoDB-style operators for backward compatibility
	if m, ok := data.(map[string]interface{}); ok {
		// Handle $set operations
		if setVal, hasSet := m["$set"]; hasSet {
			if setMap, ok2 := setVal.(map[string]interface{}); ok2 {
				return r.buildSetOperation(setMap)
			}
		}

		// Handle $inc operations
		if incVal, hasInc := m["$inc"]; hasInc {
			if incMap, ok2 := incVal.(map[string]interface{}); ok2 {
				return r.buildIncrementOperation(incMap)
			}
		}

		// Handle mixed operations ($set + $inc)
		if setVal, hasSet := m["$set"]; hasSet {
			if incVal, hasInc := m["$inc"]; hasInc {
				if setMap, ok2 := setVal.(map[string]interface{}); ok2 {
					if incMap, ok3 := incVal.(map[string]interface{}); ok3 {
						return r.buildMixedOperation(setMap, incMap)
					}
				}
			}
		}
	}

	// Default: replace the entire JSON document
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal update data: %w", err)
	}
	clause := "data = $1, last_updated = $2"
	args := []interface{}{jsonData, time.Now().Unix()}
	return clause, args, nil
}

// buildSetOperation builds a SET operation for plain field updates
func (r *PostgreSQLRepository) buildSetOperation(setMap map[string]interface{}) (string, []interface{}, error) {
	// Build: data = jsonb_set(jsonb_set(data, '{k1}', $1::jsonb, true), '{k2}', $2::jsonb, true) ... , last_updated = $N
	clause := "data = "
	args := make([]interface{}, 0, len(setMap)+1)
	idx := 1
	nested := "data"

	// Chain jsonb_set for each field
	for k, v := range setMap {
		// Support nested paths using dot notation e.g., "votes.123" -> '{votes,123}'
		segments := strings.Split(k, ".")
		for i := range segments {
			segments[i] = strings.TrimSpace(segments[i])
		}
		path := "'{" + strings.Join(segments, ",") + "}'"
		nested = fmt.Sprintf("jsonb_set(%s, %s, $%d::jsonb, true)", nested, path, idx)

		// Marshal value to JSON so we can bind as jsonb
		jsonVal, err := json.Marshal(v)
		if err != nil {
			return "", nil, err
		}
		args = append(args, string(jsonVal))
		idx++
	}

	clause += nested + ", last_updated = $" + fmt.Sprint(idx)
	args = append(args, time.Now().Unix())
	return clause, args, nil
}

// buildIncrementOperation builds an INCREMENT operation
func (r *PostgreSQLRepository) buildIncrementOperation(incMap map[string]interface{}) (string, []interface{}, error) {
	// Build: data = jsonb_set(jsonb_set(data, '{k1}', to_jsonb(COALESCE((data->>'k1')::numeric, 0) + $1), true), ... , last_updated = $N
	clause := "data = "
	args := make([]interface{}, 0, len(incMap)+1)
	idx := 1
	nested := "data"

	// Chain jsonb_set for each increment field
	for k, v := range incMap {
		// Support nested paths using dot notation e.g., "votes.123" -> '{votes,123}'
		segments := strings.Split(k, ".")
		for i := range segments {
			segments[i] = strings.TrimSpace(segments[i])
		}
		path := "'{" + strings.Join(segments, ",") + "}'"

		// Ensure the increment value is numeric
		var numericValue interface{}
		switch val := v.(type) {
		case int, int8, int16, int32, int64:
			numericValue = val
		case uint, uint8, uint16, uint32, uint64:
			numericValue = val
		case float32, float64:
			numericValue = val
		case string:
			// Try to parse string as numeric
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				numericValue = f
			} else {
				return "", nil, fmt.Errorf("cannot convert string '%s' to numeric for increment", val)
			}
		default:
			return "", nil, fmt.Errorf("unsupported type for increment: %T", v)
		}

		// Build increment expression: Handle both numeric and string representations
		// Use CASE to safely convert from JSON to numeric, defaulting to 0 if conversion fails
		incrementExpr := fmt.Sprintf(`to_jsonb(
            COALESCE(
                CASE 
                    WHEN jsonb_typeof(%s->'%s') = 'number' THEN (%s->>'%s')::numeric
                    WHEN jsonb_typeof(%s->'%s') = 'string' AND (%s->>'%s') ~ '^-?[0-9]+\.?[0-9]*$' THEN (%s->>'%s')::numeric
                    ELSE 0
                END, 0
            ) + $%d
        )`, nested, k, nested, k, nested, k, nested, k, nested, k, idx)
		nested = fmt.Sprintf("jsonb_set(%s, %s, %s, true)", nested, path, incrementExpr)

		args = append(args, numericValue)
		idx++
	}

	clause += nested + ", last_updated = $" + fmt.Sprint(idx)
	args = append(args, time.Now().Unix())
	return clause, args, nil
}

// buildMixedOperation builds a mixed SET + INCREMENT operation
func (r *PostgreSQLRepository) buildMixedOperation(setMap, incMap map[string]interface{}) (string, []interface{}, error) {
	// Combine both operations
	clause := "data = "
	args := make([]interface{}, 0, len(setMap)+len(incMap)+1)
	idx := 1
	nested := "data"

	// Process $set operations first
	for k, v := range setMap {
		segments := strings.Split(k, ".")
		for i := range segments {
			segments[i] = strings.TrimSpace(segments[i])
		}
		path := "'{" + strings.Join(segments, ",") + "}'"
		nested = fmt.Sprintf("jsonb_set(%s, %s, $%d::jsonb, true)", nested, path, idx)
		jsonVal, err := json.Marshal(v)
		if err != nil {
			return "", nil, err
		}
		args = append(args, string(jsonVal))
		idx++
	}

	// Process $inc operations
	for k, v := range incMap {
		segments := strings.Split(k, ".")
		for i := range segments {
			segments[i] = strings.TrimSpace(segments[i])
		}
		path := "'{" + strings.Join(segments, ",") + "}'"

		// Ensure the increment value is numeric
		var numericValue interface{}
		switch val := v.(type) {
		case int, int8, int16, int32, int64:
			numericValue = val
		case uint, uint8, uint16, uint32, uint64:
			numericValue = val
		case float32, float64:
			numericValue = val
		case string:
			// Try to parse string as numeric
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				numericValue = f
			} else {
				return "", nil, fmt.Errorf("cannot convert string '%s' to numeric for increment", val)
			}
		default:
			return "", nil, fmt.Errorf("unsupported type for increment: %T", v)
		}

		incrementExpr := fmt.Sprintf("to_jsonb(COALESCE((%s->>'%s')::numeric, 0) + $%d)", nested, k, idx)
		nested = fmt.Sprintf("jsonb_set(%s, %s, %s, true)", nested, path, incrementExpr)

		args = append(args, numericValue)
		idx++
	}

	clause += nested + ", last_updated = $" + fmt.Sprint(idx)
	args = append(args, time.Now().Unix())
	return clause, args, nil
}

// buildOrderByClause builds an ORDER BY clause from sort options
func (r *PostgreSQLRepository) buildOrderByClause(sort map[string]int) string {
	if len(sort) == 0 {
		return ""
	}

	var clauses []string
	for field, direction := range sort {
		order := "ASC"
		if direction == -1 {
			order = "DESC"
		}

		// Map known fields to real columns and cast types where necessary
		switch field {
		case "objectId":
			clauses = append(clauses, fmt.Sprintf("object_id %s", order))
		case "createdDate":
			clauses = append(clauses, fmt.Sprintf("created_date %s", order))
		case "lastUpdated":
			clauses = append(clauses, fmt.Sprintf("last_updated %s", order))
		default:
			clauses = append(clauses, fmt.Sprintf("(data->>'%s') %s", field, order))
		}
	}

	return strings.Join(clauses, ", ")
}

// PostgreSQLQueryResult implementation
func (r *PostgreSQLQueryResult) Next() bool {
	if r.rows == nil || r.closed {
		return false
	}
	return r.rows.Next()
}

func (r *PostgreSQLQueryResult) Decode(v interface{}) error {
	if r.rows == nil {
		return fmt.Errorf("rows is nil")
	}

	var jsonData []byte
	if err := r.rows.Scan(&jsonData); err != nil {
		return err
	}

	return json.Unmarshal(jsonData, v)
}

func (r *PostgreSQLQueryResult) Close() {
	if r.rows != nil && !r.closed {
		r.rows.Close()
		r.closed = true
	}
}

func (r *PostgreSQLQueryResult) Error() error {
	return r.err
}

// PostgreSQLSingleResult implementation
func (r *PostgreSQLSingleResult) Decode(v interface{}) error {
	if r.row == nil {
		return fmt.Errorf("row is nil")
	}

	var jsonData []byte
	err := r.row.Scan(&jsonData)
	if err != nil {
		if err == sql.ErrNoRows {
			r.noResult = true
			return interfaces.ErrNoDocuments
		}
		return err
	}

	return json.Unmarshal(jsonData, v)
}

func (r *PostgreSQLSingleResult) Error() error {
	if r.noResult {
		return interfaces.ErrNoDocuments
	}
	return r.err
}

func (r *PostgreSQLSingleResult) NoResult() bool {
	return r.noResult
}

// PostgreSQLTransactionContext implementation
func (t *PostgreSQLTransactionContext) Commit() error {
	return t.tx.Commit()
}

func (t *PostgreSQLTransactionContext) Rollback() error {
	return t.tx.Rollback()
}

func (t *PostgreSQLTransactionContext) Context() context.Context {
	return t.ctx
}

// UpdateFields updates specific fields using clean syntax (no $set, $inc operators)
func (r *PostgreSQLRepository) UpdateFields(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build SET clause for plain field updates
		setClause, setArgs, err := r.buildSetOperation(updates)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause with offset after SET parameters
		whereClause, args, err := r.buildWhereClauseWithOffset(filter, len(setArgs)+1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Combine all arguments
		allArgs := append(setArgs, args...)

		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf(`
			UPDATE %s 
			SET %s
			WHERE %s`, tableName, setClause, whereClause)

		_, err = r.db.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL UpdateFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// IncrementFields increments numeric fields using clean syntax (no $inc operators)
func (r *PostgreSQLRepository) IncrementFields(ctx context.Context, collectionName string, filter interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build increment clause first
		incClause, incArgs, err := r.buildIncrementOperation(increments)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause with correct parameter offset
		whereClause, args, err := r.buildWhereClauseWithOffset(filter, len(incArgs)+1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Combine all arguments
		allArgs := append(incArgs, args...)

		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf(`
            UPDATE %s 
            SET %s
            WHERE %s`, tableName, incClause, whereClause)

		_, err = r.db.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL IncrementFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// UpdateAndIncrement performs both update and increment operations
func (r *PostgreSQLRepository) UpdateAndIncrement(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause
		whereClause, args, err := r.buildWhereClauseWithOffset(filter, 1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build mixed operation clause
		mixedClause, mixedArgs, err := r.buildMixedOperation(updates, increments)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Combine all arguments
		allArgs := append(mixedArgs, args...)

		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf(`
			UPDATE %s 
			SET %s, last_updated = $%d
			WHERE %s`, tableName, mixedClause, len(allArgs)+1, whereClause)

		// Add last_updated timestamp
		allArgs = append(allArgs, time.Now().Unix())

		_, err = r.db.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL UpdateAndIncrement error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// UpdateWithOwnership performs atomic update with ownership validation (optimized)
func (r *PostgreSQLRepository) UpdateWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build SET clause for updates
		var args []interface{}
		argIndex := 1

		// Add lastUpdated timestamp
		updates["lastUpdated"] = time.Now().Unix()

		// Build chained jsonb_set operations
		setClause := "data = "
		nested := "data"

		// Process updates in a specific order to ensure proper chaining
		updateKeys := make([]string, 0, len(updates))
		for key := range updates {
			updateKeys = append(updateKeys, key)
		}
		sort.Strings(updateKeys) // Ensure consistent ordering

		for _, key := range updateKeys {
			value := updates[key]
			// Cast the parameter to JSONB explicitly for PostgreSQL
			setClause = fmt.Sprintf("data = jsonb_set(%s, '{%s}', $%d::jsonb, true)", nested, key, argIndex)
			nested = fmt.Sprintf("jsonb_set(%s, '{%s}', $%d::jsonb, true)", nested, key, argIndex)
			args = append(args, value)
			argIndex++
		}

		// Build WHERE clause with ownership validation - use parameters after SET parameters
		whereClause := fmt.Sprintf("object_id = $%d AND (data->>'ownerUserId')::text = $%d AND (data->>'deleted')::boolean = false", argIndex, argIndex+1)

		// Execute atomic update with ownership validation
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, setClause, whereClause)

		// Add WHERE clause parameters after SET parameters
		allArgs := append(args, fmt.Sprint(entityID), fmt.Sprint(ownerID))

		sqlResult, err := r.db.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL UpdateWithOwnership error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Check if any rows were affected (entity existed and belonged to owner)
		rowsAffected, _ := sqlResult.RowsAffected()
		if rowsAffected == 0 {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("entity not found or unauthorized")}
			return
		}

		result <- interfaces.RepositoryResult{Result: rowsAffected}
	}()

	return result
}

// DeleteWithOwnership performs atomic delete with ownership validation (optimized)
func (r *PostgreSQLRepository) DeleteWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build SET clause for soft delete
		setClause := "data = jsonb_set(jsonb_set(data, '{deleted}', 'true'), '{deletedDate}', $1)"

		// Build WHERE clause with ownership validation
		whereClause := "object_id = $2 AND (data->>'ownerUserId')::text = $3 AND (data->>'deleted')::boolean = false"

		// Execute atomic soft delete with ownership validation
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, setClause, whereClause)

		args := []interface{}{
			time.Now().Unix(),    // deletedDate
			fmt.Sprint(entityID), // objectId
			fmt.Sprint(ownerID),  // ownerUserId
		}

		sqlResult, err := r.db.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL DeleteWithOwnership error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Check if any rows were affected (entity existed and belonged to owner)
		rowsAffected, _ := sqlResult.RowsAffected()
		if rowsAffected == 0 {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("entity not found or unauthorized")}
			return
		}

		result <- interfaces.RepositoryResult{Result: rowsAffected}
	}()

	return result
}

// IncrementWithOwnership performs atomic increment with ownership validation (optimized)
func (r *PostgreSQLRepository) IncrementWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build SET clause for increments
		var setClauses []string
		var args []interface{}
		argIndex := 1

		for key, value := range increments {
			// Ensure the increment value is numeric
			var numericValue interface{}
			switch val := value.(type) {
			case int, int8, int16, int32, int64:
				numericValue = val
			case uint, uint8, uint16, uint32, uint64:
				numericValue = val
			case float32, float64:
				numericValue = val
			case string:
				// Try to parse string as numeric
				if f, err := strconv.ParseFloat(val, 64); err == nil {
					numericValue = f
				} else {
					result <- interfaces.RepositoryResult{Error: fmt.Errorf("cannot convert string '%s' to numeric for increment", val)}
					return
				}
			default:
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("unsupported type for increment: %T", value)}
				return
			}

			// Use COALESCE to handle NULL values and increment safely
			// Cast the parameter to numeric explicitly for PostgreSQL
			setClause := fmt.Sprintf("data = jsonb_set(data, '{%s}', to_jsonb(COALESCE((data->>'%s')::numeric, 0) + $%d::numeric))", key, key, argIndex)
			setClauses = append(setClauses, setClause)
			args = append(args, numericValue)
			argIndex++
		}

		// Build WHERE clause with ownership validation - use consecutive parameter numbers
		whereClause := fmt.Sprintf("object_id = $%d AND (data->>'ownerUserId')::text = $%d AND (data->>'deleted')::boolean = false", argIndex, argIndex+1)

		// Execute atomic increment with ownership validation
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, strings.Join(setClauses, ", "), whereClause)

		// Add WHERE clause parameters after SET parameters
		allArgs := append(args, fmt.Sprint(entityID), fmt.Sprint(ownerID))

		sqlResult, err := r.db.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL IncrementWithOwnership error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Check if any rows were affected (entity existed and belonged to owner)
		rowsAffected, _ := sqlResult.RowsAffected()
		if rowsAffected == 0 {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("entity not found or unauthorized")}
			return
		}

		result <- interfaces.RepositoryResult{Result: rowsAffected}
	}()

	return result
}

// FindWithCursor retrieves documents with cursor-based pagination
func (r *PostgreSQLRepository) FindWithCursor(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build query
		query := fmt.Sprintf("SELECT data FROM %s", tableName)
		whereClause, args, err := r.buildWhereClause(filter)
		if err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		// Add cursor-based sorting
		if opts != nil {
			sortField := opts.SortField
			if sortField == "" {
				sortField = "createdDate" // Default sort field
			}

			sortDirection := "DESC" // Default desc
			if opts.SortDirection == "asc" {
				sortDirection = "ASC"
			}

			// Build primary sort expression
			var primary string
			switch sortField {
			case "createdDate":
				primary = "created_date"
			case "lastUpdated":
				primary = "last_updated"
			default:
				// Use JSONB path with appropriate cast for other fields
				primary = fmt.Sprintf("(data->>'%s')::%s", sortField, r.getPostgreSQLType(sortField))
			}

			// For compound sorting, always include object_id as tiebreaker (indexed)
			orderBy := fmt.Sprintf("%s %s, object_id %s", primary, sortDirection, sortDirection)
			query += " ORDER BY " + orderBy
		}

		// Add limit
		if opts != nil && opts.Limit != nil {
			query += fmt.Sprintf(" LIMIT %d", *opts.Limit)
		}

		rows, err := r.db.QueryContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL FindWithCursor error: %s", err.Error())
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		result <- &PostgreSQLQueryResult{rows: rows, columns: []string{"data"}}
	}()

	return result
}

// CountWithFilter counts documents matching the filter
func (r *PostgreSQLRepository) CountWithFilter(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build count query
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		whereClause, args, err := r.buildWhereClause(filter)
		if err != nil {
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}

		if whereClause != "" {
			query += " WHERE " + whereClause
		}

		var count int64
		err = r.db.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			log.Error("PostgreSQL CountWithFilter error: %s", err.Error())
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}

		result <- interfaces.CountResult{Count: count, Error: nil}
	}()

	return result
}

// getPostgreSQLType returns the appropriate PostgreSQL cast type for sorting
func (r *PostgreSQLRepository) getPostgreSQLType(fieldName string) string {
	switch fieldName {
	case "createdDate", "lastUpdated", "deletedDate", "score", "viewCount", "commentCounter":
		return "bigint"
	case "objectId", "ownerUserId", "postTypeId":
		return "text"
	default:
		return "text"
	}
}

// DB returns the underlying *sql.DB connection for direct access
func (r *PostgreSQLRepository) DB() *sql.DB {
	return r.db
}
