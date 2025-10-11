// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package interfaces

import (
	"context"
	"time"

	uuid "github.com/gofrs/uuid"
)

// Repository defines the interface for database operations
type Repository interface {
	// Basic CRUD operations
	Save(ctx context.Context, collectionName string, data interface{}) <-chan RepositoryResult
	SaveMany(ctx context.Context, collectionName string, data []interface{}) <-chan RepositoryResult
	Find(ctx context.Context, collectionName string, filter interface{}, opts *FindOptions) <-chan QueryResult
	FindOne(ctx context.Context, collectionName string, filter interface{}) <-chan SingleResult
	Update(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *UpdateOptions) <-chan RepositoryResult
	UpdateMany(ctx context.Context, collectionName string, filter interface{}, data interface{}, opts *UpdateOptions) <-chan RepositoryResult
	Delete(ctx context.Context, collectionName string, filter interface{}) <-chan RepositoryResult
	DeleteMany(ctx context.Context, collectionName string, filters []interface{}) <-chan RepositoryResult // Bulk delete for multiple individual filters
	
	// Index operations
	CreateIndex(ctx context.Context, collectionName string, indexes map[string]interface{}) <-chan error
	
	// Aggregation operations
	Count(ctx context.Context, collectionName string, filter interface{}) <-chan CountResult
	
	// Cursor-based pagination operations
	FindWithCursor(ctx context.Context, collectionName string, filter interface{}, opts *CursorFindOptions) <-chan QueryResult
	CountWithFilter(ctx context.Context, collectionName string, filter interface{}) <-chan CountResult
	
	// Transaction support with enterprise features
	Begin(ctx context.Context) (Transaction, error) // New method for proper transactional isolation
	BeginWithConfig(ctx context.Context, config *TransactionConfig) (Transaction, error) // Enterprise transaction creation
	BeginTransaction(ctx context.Context) (TransactionContext, error) // Legacy method for backward compatibility
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
	
	// Connection management
	Ping(ctx context.Context) <-chan error
	Close() error
	
	// Clean abstraction methods for common operations
	UpdateFields(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}) <-chan RepositoryResult
	IncrementFields(ctx context.Context, collectionName string, filter interface{}, increments map[string]interface{}) <-chan RepositoryResult
	UpdateAndIncrement(ctx context.Context, collectionName string, filter interface{}, updates map[string]interface{}, increments map[string]interface{}) <-chan RepositoryResult
	
	// Atomic operations with ownership validation (for performance optimization)
	UpdateWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan RepositoryResult
	DeleteWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}) <-chan RepositoryResult
	IncrementWithOwnership(ctx context.Context, collectionName string, entityID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan RepositoryResult
}

// FindOptions represents options for find operations
type FindOptions struct {
	Limit  *int64
	Skip   *int64
	Sort   map[string]int
	Select map[string]int
}

// CursorFindOptions represents options for cursor-based find operations
type CursorFindOptions struct {
	Limit         *int64
	Sort          map[string]int // Field: 1 (asc) or -1 (desc)
	Select        map[string]int
	SortField     string
	SortDirection string // "asc" or "desc"
	CursorValue   interface{}
	CursorID      string
	IsAfter       bool // true for after cursor, false for before cursor
}

// UpdateOptions represents options for update operations
type UpdateOptions struct {
	Upsert                   *bool
	BypassDocumentValidation *bool
	ArrayFilters             *ArrayFilters
}

// ArrayFilters represents array filters for update operations
type ArrayFilters struct {
	Filters []interface{}
}

// BulkOperation represents a bulk operation
type BulkOperation struct {
	Type   BulkOperationType
	Filter interface{}
	Data   interface{}
	Upsert bool
}

// BulkOperationType represents the type of bulk operation
type BulkOperationType int

const (
	BulkInsert BulkOperationType = iota
	BulkUpdate
	BulkDelete
	BulkReplace
)

// RepositoryResult represents the result of a repository operation
type RepositoryResult struct {
	Result interface{}
	Error  error
}

// QueryResult represents a query result cursor
type QueryResult interface {
	Next() bool
	Decode(v interface{}) error
	Close()
	Error() error
}

// SingleResult represents a single document result
type SingleResult interface {
	Decode(v interface{}) error
	Error() error
	NoResult() bool
}

// CountResult represents the result of a count operation
type CountResult struct {
	Count int64
	Error error
}

// DistinctResult represents the result of a distinct operation
type DistinctResult struct {
	Values []interface{}
	Error  error
}

// IndexResult represents the result of index operations
type IndexResult struct {
	Indexes []IndexInfo
	Error   error
}

// IndexInfo represents information about an index
type IndexInfo struct {
	Name   string
	Keys   map[string]interface{}
	Unique bool
}

// TransactionConfig represents configuration for database transactions
type TransactionConfig struct {
	Timeout        time.Duration     `json:"timeout,omitempty"`
	ReadOnly       bool              `json:"readOnly,omitempty"`
	IsolationLevel IsolationLevel    `json:"isolationLevel,omitempty"`
	RetryPolicy    *RetryPolicy      `json:"retryPolicy,omitempty"`
}

// IsolationLevel represents transaction isolation levels
type IsolationLevel int

const (
	IsolationLevelDefault IsolationLevel = iota
	IsolationLevelReadUncommitted
	IsolationLevelReadCommitted
	IsolationLevelRepeatableRead
	IsolationLevelSerializable
)

// RetryPolicy defines retry behavior for transaction operations
type RetryPolicy struct {
	MaxRetries      int           `json:"maxRetries"`
	InitialDelay    time.Duration `json:"initialDelay"`
	MaxDelay        time.Duration `json:"maxDelay"`
	BackoffFactor   float64       `json:"backoffFactor"`
	RetryableErrors []string      `json:"retryableErrors"`
}

// TransactionMetrics represents metrics for transaction operations
type TransactionMetrics struct {
	TransactionID    string        `json:"transactionId"`
	StartTime        time.Time     `json:"startTime"`
	Duration         time.Duration `json:"duration,omitempty"`
	OperationsCount  int64         `json:"operationsCount"`
	Status           string        `json:"status"` // "active", "committed", "rolled_back", "failed"
	DatabaseType     string        `json:"databaseType"`
	ErrorCode        string        `json:"errorCode,omitempty"`
	ErrorMessage     string        `json:"errorMessage,omitempty"`
}

// Transaction represents a database transaction that can perform repository operations
// It embeds Repository so it can be used anywhere a Repository is expected
type Transaction interface {
	Repository // A transaction IS a repository, so it has all the same methods
	
	// Transaction-specific operations
	Commit() error
	Rollback() error
	
	// Enterprise features
	GetMetrics() *TransactionMetrics
	GetConfig() *TransactionConfig
	IsActive() bool
	GetTransactionID() string
}

// TransactionContext represents a database transaction context (legacy interface)
type TransactionContext interface {
	Commit() error
	Rollback() error
	Context() context.Context
}

// SearchOperator represents a search operation
type SearchOperator struct {
	Search string
}

// BaseEntity represents the base entity with common fields
type BaseEntity struct {
	ObjectId    uuid.UUID `json:"objectId" bson:"objectId"`
	CreatedDate int64     `json:"created_date" bson:"created_date"`
	LastUpdated int64     `json:"last_updated" bson:"last_updated"`
}

// PaginationResult represents paginated results
type PaginationResult struct {
	Data       interface{} `json:"data"`
	TotalCount int64       `json:"totalCount"`
	Page       int64       `json:"page"`
	Limit      int64       `json:"limit"`
	HasNext    bool        `json:"hasNext"`
	HasPrev    bool        `json:"hasPrev"`
}

// CursorPaginationResult represents cursor-based paginated results
type CursorPaginationResult struct {
	Data       interface{} `json:"data"`
	NextCursor string      `json:"nextCursor,omitempty"`
	PrevCursor string      `json:"prevCursor,omitempty"`
	HasNext    bool        `json:"hasNext"`
	HasPrev    bool        `json:"hasPrev"`
	Limit      int64       `json:"limit"`
}

// Database configuration constants
const (
	DatabaseTypeMongoDB    = "mongodb"
	DatabaseTypePostgreSQL = "postgresql"
)

// Common errors
var (
	ErrNoDocuments        = NewRepositoryError("no documents found", "NOT_FOUND")
	ErrDuplicateKey       = NewRepositoryError("duplicate key error", "DUPLICATE_KEY")
	ErrInvalidFilter      = NewRepositoryError("invalid filter", "INVALID_FILTER")
	ErrConnectionFailed   = NewRepositoryError("database connection failed", "CONNECTION_FAILED")
	ErrTransactionFailed  = NewRepositoryError("transaction failed", "TRANSACTION_FAILED")
	ErrUnsupportedOperation = NewRepositoryError("unsupported operation", "UNSUPPORTED_OPERATION")
	ErrTransactionTimeout = NewRepositoryError("transaction timeout", "TRANSACTION_TIMEOUT")
	ErrTransactionInactive = NewRepositoryError("transaction is not active", "TRANSACTION_INACTIVE")
	ErrTransactionConflict = NewRepositoryError("transaction conflict detected", "TRANSACTION_CONFLICT")
	ErrNestedTransaction  = NewRepositoryError("nested transactions not supported", "NESTED_TRANSACTION")
)

// RepositoryError represents a repository specific error
type RepositoryError struct {
	Message string
	Code    string
	Time    time.Time
}

func (e *RepositoryError) Error() string {
	return e.Message
}

// NewRepositoryError creates a new repository error
func NewRepositoryError(message, code string) *RepositoryError {
	return &RepositoryError{
		Message: message,
		Code:    code,
		Time:    time.Now(),
	}
}