package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/lib/pq"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/observability"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
)

// PostgreSQLTransaction implements Transaction interface for PostgreSQL
// It embeds PostgreSQLRepository and adds transaction-specific operations
type PostgreSQLTransaction struct {
	*PostgreSQLRepository
	tx            *sql.Tx
	ctx           context.Context
	cancel        context.CancelFunc
	config        *interfaces.TransactionConfig
	metrics       *interfaces.TransactionMetrics
	transactionID string
	isActive      int32 // atomic flag for thread-safe status checking
	operationCount int64 // atomic counter for operations
}

// isValidTransaction checks if the transaction is still valid
func (t *PostgreSQLTransaction) isValidTransaction() bool {
	return t.tx != nil && t.ctx != nil && atomic.LoadInt32(&t.isActive) == 1
}

// incrementOperationCount safely increments the operation counter
func (t *PostgreSQLTransaction) incrementOperationCount() {
	atomic.AddInt64(&t.operationCount, 1)
	if t.metrics != nil {
		observability.GetGlobalMetrics().IncrementOperations(t.transactionID)
	}
}

// Enterprise methods implementation
func (t *PostgreSQLTransaction) GetMetrics() *interfaces.TransactionMetrics {
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

func (t *PostgreSQLTransaction) GetConfig() *interfaces.TransactionConfig {
	return t.config
}

func (t *PostgreSQLTransaction) IsActive() bool {
	return atomic.LoadInt32(&t.isActive) == 1
}

func (t *PostgreSQLTransaction) GetTransactionID() string {
	return t.transactionID
}

// Commit commits the transaction
func (t *PostgreSQLTransaction) Commit() error {
	// Use a compare-and-swap to ensure this block only runs once.
	if !atomic.CompareAndSwapInt32(&t.isActive, 1, 0) {
		// If the swap fails, the transaction is already inactive. Do nothing.
		return nil // Or return sql.ErrTxDone if you prefer
	}
	
	// If we get here, we were the first to make it inactive. Proceed with commit.
	if t.cancel != nil {
		defer t.cancel()
	}
	
	err := t.tx.Commit()
	
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

// Rollback rolls back the transaction
func (t *PostgreSQLTransaction) Rollback() error {
	// Use the same atomic compare-and-swap pattern.
	if !atomic.CompareAndSwapInt32(&t.isActive, 1, 0) {
		// Transaction is already inactive.
		return nil // Or return sql.ErrTxDone
	}

	// If we get here, we were the first to make it inactive. Proceed with rollback.
	if t.cancel != nil {
		defer t.cancel()
	}
	
	err := t.tx.Rollback()
	
	// Record metrics
	if t.metrics != nil {
		observability.GetGlobalMetrics().RollbackTransaction(t.transactionID, err)
	}
	
	return err
}

// Override repository methods to use transaction context
// For PostgreSQL, we need to override methods to use the transaction instead of the main DB connection
func (t *PostgreSQLTransaction) Save(ctx context.Context, collectionName string, data interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
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
		objectID, createdDate, lastUpdated := t.extractCommonFields(data)
		
		tableName := t.getTableName(collectionName)
		query := fmt.Sprintf(`
			INSERT INTO %s (object_id, data, created_date, last_updated) 
			VALUES ($1, $2, $3, $4) 
			RETURNING id`, tableName)
		
		var id int64
		err = t.tx.QueryRowContext(ctx, query, objectID, jsonData, createdDate, lastUpdated).Scan(&id)
		if err != nil {
			if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" { // Unique violation
				result <- interfaces.RepositoryResult{Error: interfaces.ErrDuplicateKey}
				return
			}
			log.Error("PostgreSQL Transaction Save error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: id}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) Delete(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build WHERE clause
		whereClause, args, err := t.buildWhereClause(filter)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)
		
		_, err = t.tx.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL Transaction Delete error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) Count(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- interfaces.CountResult{Error: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.CountResult{Error: err}
			return
		}
		
		// Build WHERE clause
		whereClause, args, err := t.buildWhereClause(filter)
		if err != nil {
			result <- interfaces.CountResult{Error: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}
		
		var count int64
		err = t.tx.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			log.Error("PostgreSQL Transaction Count error: %s", err.Error())
			result <- interfaces.CountResult{Error: err}
			return
		}
		
		result <- interfaces.CountResult{Count: count}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) FindOne(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.SingleResult {
	result := make(chan interfaces.SingleResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- &PostgreSQLSingleResult{err: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- &PostgreSQLSingleResult{err: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
		whereClause, args, err := t.buildWhereClause(filter)
		if err != nil {
			result <- &PostgreSQLSingleResult{err: err}
			return
		}
		
		query := fmt.Sprintf("SELECT data FROM %s", tableName)
		if whereClause != "" {
			query += " WHERE " + whereClause
		}
		query += " LIMIT 1"
		
		row := t.tx.QueryRowContext(ctx, query, args...)
		result <- &PostgreSQLSingleResult{row: row, columns: []string{"data"}}
	}()
	
	return result
}

// Implement remaining interface methods for transaction
func (t *PostgreSQLTransaction) CreateIndex(ctx context.Context, collectionName string, indexes map[string]interface{}) <-chan error {
	t.incrementOperationCount()
	return t.PostgreSQLRepository.CreateIndex(t.ctx, collectionName, indexes)
}

// Transaction methods that don't support nesting
func (t *PostgreSQLTransaction) Begin(ctx context.Context) (interfaces.Transaction, error) {
	return nil, interfaces.ErrNestedTransaction
}

func (t *PostgreSQLTransaction) BeginWithConfig(ctx context.Context, config *interfaces.TransactionConfig) (interfaces.Transaction, error) {
	return nil, interfaces.ErrNestedTransaction
}

func (t *PostgreSQLTransaction) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	return nil, interfaces.ErrNestedTransaction
}

func (t *PostgreSQLTransaction) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	return interfaces.ErrNestedTransaction
}

func (t *PostgreSQLTransaction) Ping(ctx context.Context) <-chan error {
	t.incrementOperationCount()
	return t.PostgreSQLRepository.Ping(t.ctx)
}

func (t *PostgreSQLTransaction) Close() error {
	// Don't close the underlying connection, just rollback if still active
	if t.IsActive() {
		return t.Rollback()
	}
	return nil
}

// Implement remaining methods to use transaction context properly
func (t *PostgreSQLTransaction) SaveMany(ctx context.Context, collectionName string, data []interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		if len(data) == 0 {
			result <- interfaces.RepositoryResult{Result: []int64{}}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
		// Build bulk insert query
		valueStrings := make([]string, 0, len(data))
		valueArgs := make([]interface{}, 0, len(data)*4)
		
		for i, item := range data {
			jsonData, err := json.Marshal(item)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to marshal data at index %d: %w", i, err)}
				return
			}
			
			objectID, createdDate, lastUpdated := t.extractCommonFields(item)
			
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d)", i*4+1, i*4+2, i*4+3, i*4+4))
			valueArgs = append(valueArgs, objectID, jsonData, createdDate, lastUpdated)
		}
		
		query := fmt.Sprintf(`
			INSERT INTO %s (object_id, data, created_date, last_updated) 
			VALUES %s 
			RETURNING id`, tableName, strings.Join(valueStrings, ","))
		
		rows, err := t.tx.QueryContext(ctx, query, valueArgs...)
		if err != nil {
			log.Error("PostgreSQL Transaction SaveMany error: %s", err.Error())
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

func (t *PostgreSQLTransaction) Find(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.FindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
		// Build query
		query := fmt.Sprintf("SELECT data FROM %s", tableName)
		whereClause, args, err := t.buildWhereClause(filter)
		if err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}
		
		if whereClause != "" {
			query += " WHERE " + whereClause
		}
		
		// Add sorting
		if opts != nil && opts.Sort != nil {
			orderBy := t.buildOrderByClause(opts.Sort)
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
		
		rows, err := t.tx.QueryContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL Transaction Find error: %s", err.Error())
			result <- &PostgreSQLQueryResult{err: err}
			return
		}
		
		result <- &PostgreSQLQueryResult{rows: rows, columns: []string{"data"}}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) Update(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build UPDATE clause first
		updateClause, updateArgs, err := t.buildUpdateClause(data)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build WHERE clause with offset after UPDATE parameters
		whereClause, args, err := t.buildWhereClauseWithOffset(filter, len(updateArgs)+1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Combine all arguments
		allArgs := append(updateArgs, args...)
		
		tableName := t.getTableName(collectionName)
		query := fmt.Sprintf("UPDATE %s SET %s WHERE %s", tableName, updateClause, whereClause)
		
		_, err = t.tx.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Transaction Update error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) UpdateMany(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	// For PostgreSQL, UpdateMany is the same as Update since we don't have document-level operations
	return t.Update(ctx, collectionName, filter, data, opts)
}

func (t *PostgreSQLTransaction) DeleteMany(ctx context.Context, collectionName string, filters []interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		var totalDeleted int64
		
		for _, filter := range filters {
			// Build WHERE clause for each filter
			whereClause, args, err := t.buildWhereClause(filter)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: err}
				return
			}
			
			query := fmt.Sprintf("DELETE FROM %s WHERE %s", tableName, whereClause)
			res, err := t.tx.ExecContext(ctx, query, args...)
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
		
		result <- interfaces.RepositoryResult{Result: totalDeleted}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) FindWithCursor(ctx context.Context, collectionName string, filter interface{}, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
		// Build query
		query := fmt.Sprintf("SELECT data FROM %s", tableName)
		whereClause, args, err := t.buildWhereClause(filter)
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
				primary = fmt.Sprintf("(data->>'%s')::text", sortField)
			}
			
			// For compound sorting, always include object_id as tiebreaker (indexed)
			orderBy := fmt.Sprintf("%s %s, object_id %s", primary, sortDirection, sortDirection)
			query += " ORDER BY " + orderBy
		}
		
		// Add limit
		if opts != nil && opts.Limit != nil {
			query += fmt.Sprintf(" LIMIT %d", *opts.Limit)
		}
		
		rows, err := t.tx.QueryContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL Transaction FindWithCursor error: %s", err.Error())
			result <- &PostgreSQLQueryResult{err: err}
			return
		}
		
		result <- &PostgreSQLQueryResult{rows: rows, columns: []string{"data"}}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) CountWithFilter(ctx context.Context, collectionName string, filter interface{}) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
		// Build count query
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		whereClause, args, err := t.buildWhereClause(filter)
		if err != nil {
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}
		
		if whereClause != "" {
			query += " WHERE " + whereClause
		}
		
		var count int64
		err = t.tx.QueryRowContext(ctx, query, args...).Scan(&count)
		if err != nil {
			log.Error("PostgreSQL Transaction CountWithFilter error: %s", err.Error())
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}
		
		result <- interfaces.CountResult{Count: count, Error: nil}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) UpdateFields(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build SET clause for plain field updates
		setClause, setArgs, err := t.buildSetOperation(updates)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build WHERE clause with offset after SET parameters
		whereClause, args, err := t.buildWhereClauseWithOffset(filter, len(setArgs)+1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Combine all arguments
		allArgs := append(setArgs, args...)
		
		tableName := t.getTableName(collectionName)
		query := fmt.Sprintf(`
			UPDATE %s 
			SET %s
			WHERE %s`, tableName, setClause, whereClause)
		
		_, err = t.tx.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Transaction UpdateFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) IncrementFields(ctx context.Context, collectionName string, filter interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		// Check if transaction is still active
		if !t.isValidTransaction() {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrTransactionInactive}
			return
		}
		
		// Increment operation counter
		t.incrementOperationCount()
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build increment clause first
		incClause, incArgs, err := t.buildIncrementOperation(increments)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build WHERE clause with correct parameter offset
		whereClause, args, err := t.buildWhereClauseWithOffset(filter, len(incArgs)+1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Combine all arguments
		allArgs := append(incArgs, args...)
		
		tableName := t.getTableName(collectionName)
		query := fmt.Sprintf(`
			UPDATE %s 
			SET %s
			WHERE %s`, tableName, incClause, whereClause)
		
		_, err = t.tx.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Transaction IncrementFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) UpdateAndIncrement(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build WHERE clause
		whereClause, args, err := t.buildWhereClauseWithOffset(filter, 1)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Build mixed operation clause
		mixedClause, mixedArgs, err := t.buildMixedOperation(updates, increments)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		// Combine all arguments
		allArgs := append(mixedArgs, args...)
		
		tableName := t.getTableName(collectionName)
		query := fmt.Sprintf(`
			UPDATE %s 
			SET %s, last_updated = $%d
			WHERE %s`, tableName, mixedClause, len(allArgs)+1, whereClause)
		
		// Add last_updated timestamp
		allArgs = append(allArgs, time.Now().Unix())
		
		_, err = t.tx.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Transaction UpdateAndIncrement error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		result <- interfaces.RepositoryResult{Result: "OK"}
	}()
	
	return result
}

func (t *PostgreSQLTransaction) UpdateWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
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
		
		sqlResult, err := t.tx.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Transaction UpdateWithOwnership error: %s", err.Error())
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

func (t *PostgreSQLTransaction) DeleteWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
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
		
		sqlResult, err := t.tx.ExecContext(ctx, query, args...)
		if err != nil {
			log.Error("PostgreSQL Transaction DeleteWithOwnership error: %s", err.Error())
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

func (t *PostgreSQLTransaction) IncrementWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)
	
	go func() {
		defer close(result)
		
		if err := t.ensureTable(t.ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}
		
		tableName := t.getTableName(collectionName)
		
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
		
		sqlResult, err := t.tx.ExecContext(ctx, query, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Transaction IncrementWithOwnership error: %s", err.Error())
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
