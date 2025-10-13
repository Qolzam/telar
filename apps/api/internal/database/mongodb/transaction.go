package mongodb

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/observability"
	"go.mongodb.org/mongo-driver/mongo"
)

// MongoTransaction implements Transaction interface for MongoDB
// It embeds MongoRepository and adds transaction-specific operations
type MongoTransaction struct {
	*MongoRepository
	session       mongo.Session
	ctx           context.Context
	cancel        context.CancelFunc
	config        *interfaces.TransactionConfig
	metrics       *interfaces.TransactionMetrics
	transactionID string
	isActive      int32 // atomic flag for thread-safe status checking
	operationCount int64 // atomic counter for operations
}

// isValidSession checks if the transaction session is still valid
func (t *MongoTransaction) isValidSession() bool {
	return t.session != nil && t.ctx != nil && atomic.LoadInt32(&t.isActive) == 1
}

// incrementOperationCount safely increments the operation counter
func (t *MongoTransaction) incrementOperationCount() {
	atomic.AddInt64(&t.operationCount, 1)
	if t.metrics != nil {
		observability.GetGlobalMetrics().IncrementOperations(t.transactionID)
	}
}

// Enterprise methods implementation
func (t *MongoTransaction) GetMetrics() *interfaces.TransactionMetrics {
	if t.metrics != nil {
		// Return current metrics with updated operation count
		metrics := *t.metrics
		metrics.OperationsCount = atomic.LoadInt64(&t.operationCount)
		if metrics.Status == "active" {
			metrics.Duration = time.Since(metrics.StartTime)
		}
		return &metrics
	}
	return nil
}

func (t *MongoTransaction) GetConfig() *interfaces.TransactionConfig {
	return t.config
}

func (t *MongoTransaction) IsActive() bool {
	return atomic.LoadInt32(&t.isActive) == 1
}

func (t *MongoTransaction) GetTransactionID() string {
	return t.transactionID
}

// Commit commits the transaction and ends the session
func (t *MongoTransaction) Commit() error {
	if !t.isValidSession() {
		return interfaces.ErrTransactionInactive
	}
	
	// Use atomic compare-and-swap to ensure idempotency
	// Only proceed if the transaction is still active (isActive == 1)
	if !atomic.CompareAndSwapInt32(&t.isActive, 1, 0) {
		// Transaction was already committed or rolled back
		return interfaces.ErrTransactionInactive
	}
	
	// Cancel the timeout context
	if t.cancel != nil {
		defer t.cancel()
	}
	
	err := t.session.CommitTransaction(t.ctx)
	// Always end the session after commit, regardless of commit result
	t.session.EndSession(t.ctx)
	
	// Record metrics
	if t.metrics != nil {
		if err != nil {
			observability.GetGlobalMetrics().FailTransaction(t.transactionID, err)
		} else {
			observability.GetGlobalMetrics().CommitTransaction(t.transactionID)
		}
	}
	
	return err
}

// Rollback rolls back the transaction and ends the session
func (t *MongoTransaction) Rollback() error {
	if !t.isValidSession() {
		return interfaces.ErrTransactionInactive
	}
	
	// Use atomic compare-and-swap to ensure idempotency
	// Only proceed if the transaction is still active (isActive == 1)
	if !atomic.CompareAndSwapInt32(&t.isActive, 1, 0) {
		// Transaction was already committed or rolled back
		return interfaces.ErrTransactionInactive
	}
	
	// Cancel the timeout context
	if t.cancel != nil {
		defer t.cancel()
	}
	
	err := t.session.AbortTransaction(t.ctx)
	// Always end the session after rollback, regardless of abort result
	t.session.EndSession(t.ctx)
	
	// Record metrics
	if t.metrics != nil {
		observability.GetGlobalMetrics().RollbackTransaction(t.transactionID, err)
	}
	
	return err
}

// Override repository methods to use transaction context
func (t *MongoTransaction) Save(ctx context.Context, collectionName string, data interface{}) <-chan interfaces.RepositoryResult {
	// Check if transaction is still active and increment operation counter
	if !t.isValidSession() {
		result := make(chan interfaces.RepositoryResult, 1)
		result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
		close(result)
		return result
	}
	
	t.incrementOperationCount()
	// Always use the transaction context to ensure operations are part of the transaction
	return t.MongoRepository.Save(t.ctx, collectionName, data)
}

func (t *MongoTransaction) SaveMany(ctx context.Context, collectionName string, data []interface{}) <-chan interfaces.RepositoryResult {
	if !t.isValidSession() {
		result := make(chan interfaces.RepositoryResult, 1)
		result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
		close(result)
		return result
	}
	t.incrementOperationCount()
	return t.MongoRepository.SaveMany(t.ctx, collectionName, data)
}

func (t *MongoTransaction) Find(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.FindOptions) <-chan interfaces.QueryResult {
	return t.MongoRepository.Find(t.ctx, collectionName, filter, opts)
}

func (t *MongoTransaction) FindOne(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.SingleResult {
	return t.MongoRepository.FindOne(t.ctx, collectionName, filter)
}

func (t *MongoTransaction) Update(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	if !t.isValidSession() {
		result := make(chan interfaces.RepositoryResult, 1)
		result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
		close(result)
		return result
	}
	t.incrementOperationCount()
	return t.MongoRepository.Update(t.ctx, collectionName, filter, data, opts)
}

func (t *MongoTransaction) UpdateMany(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	if !t.isValidSession() {
		result := make(chan interfaces.RepositoryResult, 1)
		result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
		close(result)
		return result
	}
	t.incrementOperationCount()
	return t.MongoRepository.UpdateMany(t.ctx, collectionName, filter, data, opts)
}

func (t *MongoTransaction) Delete(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.RepositoryResult {
	if !t.isValidSession() {
		result := make(chan interfaces.RepositoryResult, 1)
		result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
		close(result)
		return result
	}
	t.incrementOperationCount()
	return t.MongoRepository.Delete(t.ctx, collectionName, filter)
}

func (t *MongoTransaction) DeleteMany(ctx context.Context, collectionName string, filters []interface{}) <-chan interfaces.RepositoryResult {
	if !t.isValidSession() {
		result := make(chan interfaces.RepositoryResult, 1)
		result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
		close(result)
		return result
	}
	t.incrementOperationCount()
	return t.MongoRepository.DeleteMany(t.ctx, collectionName, filters)
}

func (t *MongoTransaction) Count(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	if !t.isValidSession() {
		result := make(chan interfaces.CountResult, 1)
		result <- interfaces.CountResult{Error: interfaces.ErrTransactionInactive}
		close(result)
		return result
	}
	t.incrementOperationCount()
	return t.MongoRepository.Count(t.ctx, collectionName, filter)
}

func (t *MongoTransaction) FindWithCursor(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	return t.MongoRepository.FindWithCursor(t.ctx, collectionName, filter, opts)
}

func (t *MongoTransaction) CountWithFilter(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	return t.MongoRepository.CountWithFilter(t.ctx, collectionName, filter)
}

func (t *MongoTransaction) UpdateFields(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	return t.MongoRepository.UpdateFields(t.ctx, collectionName, filter, updates)
}

func (t *MongoTransaction) IncrementFields(ctx context.Context, collectionName string, filter interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	return t.MongoRepository.IncrementFields(t.ctx, collectionName, filter, increments)
}

func (t *MongoTransaction) UpdateAndIncrement(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	return t.MongoRepository.UpdateAndIncrement(t.ctx, collectionName, filter, updates, increments)
}

func (t *MongoTransaction) UpdateWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	return t.MongoRepository.UpdateWithOwnership(t.ctx, collectionName, entityID, ownerID, updates)
}

func (t *MongoTransaction) DeleteWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult {
	return t.MongoRepository.DeleteWithOwnership(t.ctx, collectionName, entityID, ownerID)
}

func (t *MongoTransaction) IncrementWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	return t.MongoRepository.IncrementWithOwnership(t.ctx, collectionName, entityID, ownerID, increments)
}

// Implement remaining interface methods for transaction
func (t *MongoTransaction) CreateIndex(ctx context.Context, collectionName string, indexes map[string]interface{}) <-chan error {
	return t.MongoRepository.CreateIndex(t.ctx, collectionName, indexes)
}

// Transaction methods that don't support nesting
func (t *MongoTransaction) Begin(ctx context.Context) (interfaces.Transaction, error) {
	return nil, interfaces.ErrNestedTransaction
}

func (t *MongoTransaction) BeginWithConfig(ctx context.Context, config *interfaces.TransactionConfig) (interfaces.Transaction, error) {
	return nil, interfaces.ErrNestedTransaction
}

func (t *MongoTransaction) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	return nil, interfaces.ErrNestedTransaction
}

func (t *MongoTransaction) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return interfaces.ErrNestedTransaction
}

func (t *MongoTransaction) Ping(ctx context.Context) <-chan error {
	return t.MongoRepository.Ping(t.ctx)
}

func (t *MongoTransaction) Close() error {
	// Don't close the underlying connection, just rollback if still active
	if t.IsActive() {
		return t.Rollback()
	}
	return nil
}
