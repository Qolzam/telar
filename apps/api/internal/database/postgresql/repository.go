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
	"sort"
	"strconv"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/observability"
	"github.com/qolzam/telar/apps/api/internal/database/utils"
	"github.com/qolzam/telar/apps/api/internal/pkg/log"
)

// PostgreSQLRepository implements the Repository interface for PostgreSQL
type PostgreSQLRepository struct {
	db     *sqlx.DB
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

	db, err := sqlx.ConnectContext(ctx, "postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL with sqlx: %w", err)
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
		// Note: For test environments with outdated schemas, developers should run `make clean-dbs`
		// to reset their database to a clean state. This ensures the latest schema is created.
		return nil
	}

	// If we get here, the table does not exist. Proceed with creation.
	createQuery := fmt.Sprintf(`
	CREATE TABLE %s (
		id BIGSERIAL PRIMARY KEY,
		object_id VARCHAR(255) UNIQUE NOT NULL,
		owner_user_id UUID,
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

// Save stores a single document with indexed columns explicitly provided
func (r *PostgreSQLRepository) Save(ctx context.Context, collectionName string, objectID uuid.UUID, ownerUserID uuid.UUID, createdDate, lastUpdated int64, data interface{}) <-chan interfaces.RepositoryResult {
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

		
		tableName := r.getTableName(collectionName)
		query := fmt.Sprintf(`
			INSERT INTO %s (object_id, owner_user_id, data, created_date, last_updated) 
			VALUES ($1, $2, $3, $4, $5) 
			RETURNING id`, tableName)

		var id int64
		err = r.db.QueryRowContext(ctx, query, objectID, ownerUserID, jsonData, createdDate, lastUpdated).Scan(&id)
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
func (r *PostgreSQLRepository) SaveMany(ctx context.Context, collectionName string, items []interfaces.SaveItem) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		if len(items) == 0 {
			result <- interfaces.RepositoryResult{Result: []int64{}}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build bulk insert query
		valueStrings := make([]string, 0, len(items))
		valueArgs := make([]interface{}, 0, len(items)*5)

		for i, item := range items {
			jsonData, err := json.Marshal(item.Data)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to marshal data at index %d: %w", i, err)}
				return
			}

			
			valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)", i*5+1, i*5+2, i*5+3, i*5+4, i*5+5))
			valueArgs = append(valueArgs, item.ObjectID, item.OwnerUserID, jsonData, item.CreatedDate, item.LastUpdated)
		}

		query := fmt.Sprintf(`
			INSERT INTO %s (object_id, owner_user_id, data, created_date, last_updated) 
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
func (r *PostgreSQLRepository) Find(ctx context.Context, collectionName string, query *interfaces.Query, opts *interfaces.FindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build the WHERE clause using the hybrid approach (named + positional params)
		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		// Build the full SQL query
		fullQuery := fmt.Sprintf("SELECT data FROM %s", tableName)
		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}

		// Add sorting
		if opts != nil && opts.Sort != nil {
			orderBy := r.buildOrderByClause(opts.Sort)
			if orderBy != "" {
				fullQuery += " ORDER BY " + orderBy
			}
		}

		// Add limit and offset
		if opts != nil {
			if opts.Limit != nil {
				fullQuery += fmt.Sprintf(" LIMIT %d", *opts.Limit)
			}
			if opts.Skip != nil {
				fullQuery += fmt.Sprintf(" OFFSET %d", *opts.Skip)
			}
		}

		// Use sqlx to bind named parameters
		// IMPORTANT: sqlx.BindNamed() matches :paramName patterns, which can incorrectly match ::type casts.
		// We need to temporarily escape ::type casts before sqlx.BindNamed(), then unescape them after.
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "__CAST__")

		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(namedArgs) > 0 {
			// Use BindNamed to convert named parameters to positional
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
			if err2 != nil {
				result <- &PostgreSQLQueryResult{err: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			// No named parameters - just rebind the query as-is
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders with correct positional numbers
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts after sqlx processing
		finalQuery = strings.ReplaceAll(finalQuery, "__CAST__", "::")

		// Combine argument slices: named args first, then positional args (arrays)
		allArgs := append(namedArgsSlice, positionalArgs...)

		// Execute the query with the combined arguments
		rows, err := r.db.QueryContext(ctx, finalQuery, allArgs...)
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
func (r *PostgreSQLRepository) FindOne(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.SingleResult {
	result := make(chan interfaces.SingleResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- &PostgreSQLSingleResult{err: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build the WHERE clause using the hybrid approach (named + positional params)
		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- &PostgreSQLSingleResult{err: err}
			return
		}

		// Build the full SQL query
		fullQuery := fmt.Sprintf("SELECT data FROM %s", tableName)
		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}
		fullQuery += " LIMIT 1"

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(namedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
			if err2 != nil {
				result <- &PostgreSQLSingleResult{err: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders with correct positional numbers
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, positionalArgs...)

		row := r.db.QueryRowContext(ctx, finalQuery, allArgs...)
		result <- &PostgreSQLSingleResult{row: row, columns: []string{"data"}}
	}()

	return result
}

// Update updates documents matching the query
func (r *PostgreSQLRepository) Update(ctx context.Context, collectionName string, query *interfaces.Query, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Handle upsert operation: INSERT ... ON CONFLICT ... DO UPDATE
		if opts != nil && opts.Upsert != nil && *opts.Upsert {
			// Extract objectId from query for upsert
			var objectID uuid.UUID
			var ownerUserID uuid.UUID
			var createdDate, lastUpdated int64

			// Find object_id in query conditions
			for _, field := range query.Conditions {
				if field.Name == "object_id" && !field.IsJSONB {
					if id, ok := field.Value.(uuid.UUID); ok {
						objectID = id
					}
				}
			}

			if objectID == uuid.Nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("upsert requires object_id in query conditions")}
				return
			}

			// Extract indexed fields from data map if available
			if dataMap, ok := data.(map[string]interface{}); ok {
				if oid, ok := dataMap["objectId"].(uuid.UUID); ok && oid != uuid.Nil {
					objectID = oid
				}
				if ownerID, ok := dataMap["ownerUserID"].(uuid.UUID); ok {
					ownerUserID = ownerID
				} else if objectID != uuid.Nil {
					ownerUserID = objectID // Default to objectID as owner
				}
				if cd, ok := dataMap["createdDate"].(int64); ok {
					createdDate = cd
				} else {
					createdDate = time.Now().Unix()
				}
				if lu, ok := dataMap["lastUpdated"].(int64); ok {
					lastUpdated = lu
				} else {
					lastUpdated = time.Now().Unix()
				}
			} else {
				// Default values if not in data map
				ownerUserID = objectID
				createdDate = time.Now().Unix()
				lastUpdated = time.Now().Unix()
			}

			// Convert data to JSONB
			jsonData, err := json.Marshal(data)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to marshal data: %w", err)}
				return
			}

			tableName := r.getTableName(collectionName)

			// Build INSERT ... ON CONFLICT ... DO UPDATE query
			upsertQuery := fmt.Sprintf(`
				INSERT INTO %s (object_id, owner_user_id, data, created_date, last_updated)
				VALUES ($1, $2, $3::jsonb, $4, $5)
				ON CONFLICT (object_id) 
				DO UPDATE SET 
					data = $3::jsonb,
					last_updated = $5
				RETURNING id`, tableName)

			var id int64
			err = r.db.QueryRowContext(ctx, upsertQuery, objectID, ownerUserID, jsonData, createdDate, lastUpdated).Scan(&id)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("upsert failed: %w", err)}
				return
			}

			result <- interfaces.RepositoryResult{Result: "OK"}
			return
		}

		// Build UPDATE clause first
		updateClause, updateArgs, err := r.buildUpdateClause(data)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause using the hybrid approach
		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build the full UPDATE query
		fullQuery := fmt.Sprintf("UPDATE %s SET %s", tableName, updateClause)
		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}

		// Combine all argument maps for sqlx (named params from WHERE clause + SET clause)
		allNamedArgs := make(map[string]interface{})
		for k, v := range updateArgs {
			allNamedArgs[k] = v
		}
		for k, v := range namedArgs {
			allNamedArgs[k] = v
		}

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(allNamedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, allNamedArgs)
			if err2 != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders with correct positional numbers
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, positionalArgs...)

		// Execute the query with the combined arguments
		_, err = r.db.ExecContext(ctx, finalQuery, allArgs...)
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

// UpdateMany updates multiple documents matching the query
func (r *PostgreSQLRepository) UpdateMany(ctx context.Context, collectionName string, query *interfaces.Query, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	// For PostgreSQL, UpdateMany is the same as Update since we don't have document-level operations
	return r.Update(ctx, collectionName, query, data, opts)
}

// Delete deletes documents matching the query
func (r *PostgreSQLRepository) Delete(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build WHERE clause using the hybrid approach
		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build the full DELETE query
		fullQuery := fmt.Sprintf("DELETE FROM %s", tableName)
		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(namedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
			if err2 != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders with correct positional numbers
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, positionalArgs...)

		execResult, err := r.db.ExecContext(ctx, finalQuery, allArgs...)
		if err != nil {
			log.Error("PostgreSQL Delete error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		rowsAffected, err := execResult.RowsAffected()
		if err != nil {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to get rows affected: %w", err)}
			return
		}

		if rowsAffected == 0 {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrNoDocuments}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// DeleteMany performs bulk delete operations for multiple queries
func (r *PostgreSQLRepository) DeleteMany(ctx context.Context, collectionName string, queries []*interfaces.Query) <-chan interfaces.RepositoryResult {
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

		for _, query := range queries {
			// Build WHERE clause using the hybrid approach
			whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
			if err != nil {
				result <- interfaces.RepositoryResult{Error: err}
				return
			}

			// Build the full DELETE query
			fullQuery := fmt.Sprintf("DELETE FROM %s", tableName)
			if whereClause != "" && whereClause != "TRUE" {
				fullQuery += " WHERE " + whereClause
			}

			// Use sqlx to bind named parameters
			tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "__CAST__")
			var reboundQuery string
			var namedArgsSlice []interface{}
			if len(namedArgs) > 0 {
				var err2 error
				reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
				if err2 != nil {
					result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
					return
				}
			} else {
				reboundQuery = r.db.Rebind(tempEscapedQuery)
				namedArgsSlice = []interface{}{}
			}

			// Replace temporary array placeholders
			finalQuery := reboundQuery
			for i := range positionalArgs {
				tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
				finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
				finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
			}

			// Restore ::type casts
			finalQuery = strings.ReplaceAll(finalQuery, "__CAST__", "::")

			// Combine argument slices
			allArgs := append(namedArgsSlice, positionalArgs...)

			res, err := tx.ExecContext(ctx, finalQuery, allArgs...)
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
		// In a real implementation, you'd need to translate any legacy aggregation pipeline representation to SQL
		result <- &PostgreSQLQueryResult{err: interfaces.ErrUnsupportedOperation}
	}()

	return result
}

// Count counts documents matching filter
func (r *PostgreSQLRepository) Count(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.CountResult{Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build WHERE clause using the hybrid approach
		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- interfaces.CountResult{Error: err}
			return
		}

		// Build the full SQL query
		fullQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(namedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
			if err2 != nil {
				result <- interfaces.CountResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders with correct positional numbers
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, positionalArgs...)

		var count int64
		err = r.db.QueryRowContext(ctx, finalQuery, allArgs...).Scan(&count)
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
		baseQuery := fmt.Sprintf("SELECT DISTINCT data->>'%s' FROM %s", field, tableName)

		// Convert filter to Query object if it's not already
		var queryObj *interfaces.Query
		if q, ok := filter.(*interfaces.Query); ok {
			queryObj = q
		} else {
			// Legacy support: create a simple Query from filter map
			// This is a temporary bridge - service layer should use Query objects
			if filterMap, ok := filter.(map[string]interface{}); ok {
				queryObj = &interfaces.Query{
					Conditions: []interfaces.Field{},
				}
				for k, v := range filterMap {
					queryObj.Conditions = append(queryObj.Conditions, interfaces.Field{
						Name:     k,
						Value:    v,
						Operator: "=",
					})
				}
			} else {
				queryObj = nil
			}
		}

		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(queryObj)
		if err != nil {
			result <- interfaces.DistinctResult{Error: err}
			return
		}

		fullQuery := baseQuery
		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(namedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
			if err2 != nil {
				result <- interfaces.DistinctResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, positionalArgs...)

		rows, err := r.db.QueryContext(ctx, finalQuery, allArgs...)
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

		// PostgreSQL expects indexes to be defined ahead of time; runtime creation is not supported here
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

	// Begin transaction with options using sqlx
	tx, err := r.db.BeginTxx(timeoutCtx, opts)
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


// buildWhereClause is the single source of truth for building WHERE clauses.
// It uses a HYBRID approach: named parameters for scalars, and temporary placeholders
// for arrays to work around sqlx limitations with slice values.
// Returns: (WHERE clause, named args map, positional args slice, error)
func (r *PostgreSQLRepository) buildWhereClause(query *interfaces.Query) (string, map[string]interface{}, []interface{}, error) {
	if query == nil || (len(query.Conditions) == 0 && len(query.OrGroups) == 0) {
		return "TRUE", nil, nil, nil
	}

	conditions := []string{}
	namedArgs := make(map[string]interface{})
	positionalArgs := []interface{}{}
	paramCounter := 0

	nextNamedParam := func() string {
		p := fmt.Sprintf("p%d", paramCounter)
		paramCounter++
		return p
	}

	processField := func(field interfaces.Field) string {
		columnExpr := field.Name
		if field.JSONBCast != "" {
			columnExpr = fmt.Sprintf("(%s)%s", field.Name, field.JSONBCast)
		}

		// --- THE HYBRID LOGIC ---
		// Check if the value is a slice that needs special array handling.
		val := reflect.ValueOf(field.Value)
		if val.Kind() == reflect.Slice && val.Len() > 0 {
			// This is an array. Use a TEMPORARY placeholder that we'll replace later.
			arrayIndex := len(positionalArgs)
			placeholder := fmt.Sprintf("__ARRAY_PARAM_%d__", arrayIndex)
			positionalArgs = append(positionalArgs, pq.Array(field.Value))

			// Translate abstract array operators to PostgreSQL syntax.
			switch field.Operator {
			case "CONTAINS_ANY":
				// PostgreSQL JSONB array containment operator: column ?| array
				return fmt.Sprintf("%s ?| %s", columnExpr, placeholder)
			default:
				// For other array operators (e.g., @> for contains all)
				return fmt.Sprintf("%s = ANY(%s)", columnExpr, placeholder)
			}
		} else {
			// This is a scalar. Use a NAMED placeholder for sqlx.
			if field.Operator == "CURSOR_PAGINATION" {
				valMap, ok := field.Value.(map[string]interface{})
				if !ok {
					return fmt.Sprintf("/* invalid CURSOR_PAGINATION value for %s */ TRUE", field.Name)
				}

				sortValue := valMap["sortValue"]
				tieBreaker := valMap["tieBreaker"]

				primaryOp, _ := valMap["primaryOp"].(string)
				if primaryOp == "" {
					primaryOp = "<"
				}

				tieOp, _ := valMap["tieOp"].(string)
				if tieOp == "" {
					tieOp = primaryOp
				}

				sortParam := nextNamedParam()
				namedArgs[sortParam] = sortValue
				tieParam := nextNamedParam()
				namedArgs[tieParam] = tieBreaker

				return fmt.Sprintf("(%s %s :%s OR (%s = :%s AND object_id %s :%s))",
					columnExpr, primaryOp, sortParam,
					columnExpr, sortParam, tieOp, tieParam)
			}

			paramName := nextNamedParam()
			namedArgs[paramName] = field.Value

			// Translate abstract operators to PostgreSQL syntax.
			switch field.Operator {
			case "REGEX_I":
				// PostgreSQL case-insensitive regex: column ~* pattern
				return fmt.Sprintf("%s ~* :%s", columnExpr, paramName)
			default: // Standard operators (=, <, >, <=, >=, !=, etc.)
				return fmt.Sprintf("%s %s :%s", columnExpr, field.Operator, paramName)
			}
		}
	}

	// Process AND conditions
	for _, field := range query.Conditions {
		conditions = append(conditions, processField(field))
	}

	// Process OR groups
	for _, orGroup := range query.OrGroups {
		orConditions := []string{}
		for _, field := range orGroup {
			orConditions = append(orConditions, processField(field))
		}
		if len(orConditions) > 0 {
			conditions = append(conditions, fmt.Sprintf("(%s)", strings.Join(orConditions, " OR ")))
		}
	}

	if len(conditions) == 0 {
		return "TRUE", nil, nil, nil
	}

	return strings.Join(conditions, " AND "), namedArgs, positionalArgs, nil
}


func (r *PostgreSQLRepository) buildUpdateClause(data interface{}) (string, map[string]interface{}, error) {
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

	// Handle legacy NoSQL-style operators for backward compatibility
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
	clause := "data = :upd0, last_updated = :upd1"
	args := map[string]interface{}{
		"upd0": jsonData,
		"upd1": time.Now().Unix(),
	}
	return clause, args, nil
}

// buildSetOperation builds a SET operation for plain field updates
func (r *PostgreSQLRepository) buildSetOperation(setMap map[string]interface{}) (string, map[string]interface{}, error) {
	// Build: data = jsonb_set(jsonb_set(data, '{k1}', :set0::jsonb, true), '{k2}', :set1::jsonb, true) ... , last_updated = :setN
	clause := "data = "
	args := make(map[string]interface{})
	idx := 0
	nested := "data"

	// Chain jsonb_set for each field
	for k, v := range setMap {
		// Support nested paths using dot notation e.g., "votes.123" -> '{votes,123}'
		segments := strings.Split(k, ".")
		for i := range segments {
			segments[i] = strings.TrimSpace(segments[i])
		}
		path := "'{" + strings.Join(segments, ",") + "}'"
		paramName := fmt.Sprintf("set%d", idx)
		nested = fmt.Sprintf("jsonb_set(%s, %s, :%s::jsonb, true)", nested, path, paramName)

		// Marshal value to JSON so we can bind as jsonb
		jsonVal, err := json.Marshal(v)
		if err != nil {
			return "", nil, err
		}
		args[paramName] = string(jsonVal)
		idx++
	}

	lastUpdatedParam := fmt.Sprintf("set%d", idx)
	clause += nested + ", last_updated = :" + lastUpdatedParam
	args[lastUpdatedParam] = time.Now().Unix()
	return clause, args, nil
}

// buildIncrementOperation builds an INCREMENT operation
func (r *PostgreSQLRepository) buildIncrementOperation(incMap map[string]interface{}) (string, map[string]interface{}, error) {
	// Build: data = jsonb_set(jsonb_set(data, '{k1}', to_jsonb(COALESCE((data->>'k1')::numeric, 0) + :inc0), true), ... , last_updated = :incN
	clause := "data = "
	args := make(map[string]interface{})
	idx := 0
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

		paramName := fmt.Sprintf("inc%d", idx)
		// Build increment expression: Handle both numeric and string representations
		// Use CASE to safely convert from JSON to numeric, defaulting to 0 if conversion fails
		incrementExpr := fmt.Sprintf(`to_jsonb(
            COALESCE(
                CASE 
                    WHEN jsonb_typeof(%s->'%s') = 'number' THEN (%s->>'%s')::numeric
                    WHEN jsonb_typeof(%s->'%s') = 'string' AND (%s->>'%s') ~ '^-?[0-9]+\.?[0-9]*$' THEN (%s->>'%s')::numeric
                    ELSE 0
                END, 0
            ) + :%s
        )`, nested, k, nested, k, nested, k, nested, k, nested, k, paramName)
		nested = fmt.Sprintf("jsonb_set(%s, %s, %s, true)", nested, path, incrementExpr)

		args[paramName] = numericValue
		idx++
	}

	lastUpdatedParam := fmt.Sprintf("inc%d", idx)
	clause += nested + ", last_updated = :" + lastUpdatedParam
	args[lastUpdatedParam] = time.Now().Unix()
	return clause, args, nil
}

// buildMixedOperation builds a mixed SET + INCREMENT operation
func (r *PostgreSQLRepository) buildMixedOperation(setMap, incMap map[string]interface{}) (string, map[string]interface{}, error) {
	// Combine both operations
	clause := "data = "
	args := make(map[string]interface{})
	idx := 0
	nested := "data"

	// Process $set operations first
	for k, v := range setMap {
		segments := strings.Split(k, ".")
		for i := range segments {
			segments[i] = strings.TrimSpace(segments[i])
		}
		path := "'{" + strings.Join(segments, ",") + "}'"
		paramName := fmt.Sprintf("mixset%d", idx)
		nested = fmt.Sprintf("jsonb_set(%s, %s, :%s::jsonb, true)", nested, path, paramName)
		jsonVal, err := json.Marshal(v)
		if err != nil {
			return "", nil, err
		}
		args[paramName] = string(jsonVal)
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

		paramName := fmt.Sprintf("mixinc%d", idx)
		incrementExpr := fmt.Sprintf("to_jsonb(COALESCE((%s->>'%s')::numeric, 0) + :%s)", nested, k, paramName)
		nested = fmt.Sprintf("jsonb_set(%s, %s, %s, true)", nested, path, incrementExpr)

		args[paramName] = numericValue
		idx++
	}

	lastUpdatedParam := fmt.Sprintf("mix%d", idx)
	clause += nested + ", last_updated = :" + lastUpdatedParam
	args[lastUpdatedParam] = time.Now().Unix()
	return clause, args, nil
}

// buildOrderByClause builds an ORDER BY clause from sort options
// Service layer must provide correct snake_case column names (e.g., "created_date", "object_id")
// or JSONB paths (e.g., "data->>'score'") - repository has no schema knowledge
func (r *PostgreSQLRepository) buildOrderByClause(sortFields map[string]int) string {
	if len(sortFields) == 0 {
		return ""
	}

	keys := make([]string, 0, len(sortFields))
	for columnExpr := range sortFields {
		keys = append(keys, columnExpr)
	}
	sort.Strings(keys)

	var clauses []string
	for _, columnExpr := range keys {
		direction := sortFields[columnExpr]
		// columnExpr is now assumed to be a valid SQL expression (column name or JSONB path)
		// Service layer owns schema knowledge and provides correct expressions
		order := "ASC"
		if direction == -1 {
			order = "DESC"
		}
		clauses = append(clauses, fmt.Sprintf("%s %s", columnExpr, order))
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
func (r *PostgreSQLRepository) UpdateFields(ctx context.Context, collectionName string, query *interfaces.Query, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build SET clause for plain field updates (returns named parameters)
		setClause, setArgs, err := r.buildSetOperation(updates)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause using the hybrid approach
		whereClause, whereNamedArgs, wherePositionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Combine all named argument maps for sqlx
		allNamedArgs := make(map[string]interface{})
		for k, v := range setArgs {
			allNamedArgs[k] = v
		}
		for k, v := range whereNamedArgs {
			allNamedArgs[k] = v
		}

		tableName := r.getTableName(collectionName)
		fullQuery := fmt.Sprintf(`
			UPDATE %s 
			SET %s
			WHERE %s`, tableName, setClause, whereClause)

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(allNamedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, allNamedArgs)
			if err2 != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders
		finalQuery := reboundQuery
		for i := range wherePositionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, wherePositionalArgs...)

		// Execute the query with the combined arguments
		execResult, err := r.db.ExecContext(ctx, finalQuery, allArgs...)
		if err != nil {
			log.Error("PostgreSQL UpdateFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		rowsAffected, err := execResult.RowsAffected()
		if err != nil {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to get rows affected: %w", err)}
			return
		}

		if rowsAffected == 0 {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrNoDocuments}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// IncrementFields increments numeric fields using clean syntax (no $inc operators)
func (r *PostgreSQLRepository) IncrementFields(ctx context.Context, collectionName string, query *interfaces.Query, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build increment clause (returns named parameters)
		incClause, incArgs, err := r.buildIncrementOperation(increments)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause using the hybrid approach
		whereClause, whereNamedArgs, wherePositionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Combine all named argument maps for sqlx
		allNamedArgs := make(map[string]interface{})
		for k, v := range incArgs {
			allNamedArgs[k] = v
		}
		for k, v := range whereNamedArgs {
			allNamedArgs[k] = v
		}

		tableName := r.getTableName(collectionName)
		fullQuery := fmt.Sprintf(`
            UPDATE %s 
            SET %s
            WHERE %s`, tableName, incClause, whereClause)

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(allNamedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, allNamedArgs)
			if err2 != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders
		finalQuery := reboundQuery
		for i := range wherePositionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, wherePositionalArgs...)

		// Execute the query with the combined arguments
		execResult, err := r.db.ExecContext(ctx, finalQuery, allArgs...)
		if err != nil {
			log.Error("PostgreSQL IncrementFields error: %s", err.Error())
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		rowsAffected, err := execResult.RowsAffected()
		if err != nil {
			result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to get rows affected: %w", err)}
			return
		}

		if rowsAffected == 0 {
			result <- interfaces.RepositoryResult{Error: interfaces.ErrNoDocuments}
			return
		}

		result <- interfaces.RepositoryResult{Result: "OK"}
	}()

	return result
}

// UpdateAndIncrement performs both update and increment operations
func (r *PostgreSQLRepository) UpdateAndIncrement(ctx context.Context, collectionName string, query *interfaces.Query, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	result := make(chan interfaces.RepositoryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build WHERE clause using the hybrid approach
		whereClause, whereNamedArgs, wherePositionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Build mixed operation clause (returns named parameters)
		mixedClause, mixedArgs, err := r.buildMixedOperation(updates, increments)
		if err != nil {
			result <- interfaces.RepositoryResult{Error: err}
			return
		}

		// Combine all named argument maps for sqlx
		allNamedArgs := make(map[string]interface{})
		for k, v := range mixedArgs {
			allNamedArgs[k] = v
		}
		for k, v := range whereNamedArgs {
			allNamedArgs[k] = v
		}

		tableName := r.getTableName(collectionName)
		fullQuery := fmt.Sprintf(`
			UPDATE %s 
			SET %s
			WHERE %s`, tableName, mixedClause, whereClause)

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(allNamedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, allNamedArgs)
			if err2 != nil {
				result <- interfaces.RepositoryResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders
		finalQuery := reboundQuery
		for i := range wherePositionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, wherePositionalArgs...)

		// Execute the query with the combined arguments
		_, err = r.db.ExecContext(ctx, finalQuery, allArgs...)
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
func (r *PostgreSQLRepository) FindWithCursor(ctx context.Context, collectionName string, query *interfaces.Query, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	result := make(chan interfaces.QueryResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build WHERE clause using the hybrid approach
		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		// Determine sort field and direction for cursor conditions
		var sortField string
		var sortDirection string
		var primary string
		if opts != nil {
			sortField = opts.SortField
			if sortField == "" {
				sortField = "created_date" // Default sort field (snake_case)
			}

			sortDirection = "DESC" // Default desc
			if opts.SortDirection == "asc" {
				sortDirection = "ASC"
			}

			// Use snake_case column names directly (service layer should provide correct field names)
			// For indexed columns, use them directly; for others, use JSONB access
			if sortField == "object_id" || sortField == "created_date" || sortField == "last_updated" {
				primary = sortField
			} else {
				// Use JSONB path with type casting
				// Service layer can provide explicit type via SortFieldType, otherwise use safe default
				castType := opts.SortFieldType
				if castType == "" {
					// Default to numeric for most numeric sorts (works for integers, floats, timestamps)
					castType = "numeric"
				}
				primary = fmt.Sprintf("(data->>'%s')::%s", sortField, castType)
			}
		} else {
			sortField = "created_date"
			sortDirection = "DESC"
			primary = "created_date"
		}

		// NOTE: Cursor conditions are already in the Query object from the service layer
		// (via WhereCursorCondition). We do not add them here to avoid duplication.
		// The service layer is responsible for adding cursor pagination filters to the Query.

		// Build the full SELECT query
		fullQuery := fmt.Sprintf("SELECT data FROM %s", tableName)

		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}

		// Add cursor-based sorting
		if opts != nil {
			// For compound sorting, always include object_id as tiebreaker (indexed)
			orderBy := fmt.Sprintf("%s %s, object_id %s", primary, sortDirection, sortDirection)
			fullQuery += " ORDER BY " + orderBy
		}

		// Add limit
		if opts != nil && opts.Limit != nil {
			fullQuery += fmt.Sprintf(" LIMIT %d", *opts.Limit)
		}

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(namedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
			if err2 != nil {
				result <- &PostgreSQLQueryResult{err: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, positionalArgs...)

		rows, err := r.db.QueryContext(ctx, finalQuery, allArgs...)
		if err != nil {
			log.Error("PostgreSQL FindWithCursor error: %s", err.Error())
			result <- &PostgreSQLQueryResult{err: err}
			return
		}

		result <- &PostgreSQLQueryResult{rows: rows, columns: []string{"data"}}
	}()

	return result
}

// CountWithFilter counts documents matching the query
func (r *PostgreSQLRepository) CountWithFilter(ctx context.Context, collectionName string, query *interfaces.Query) <-chan interfaces.CountResult {
	result := make(chan interfaces.CountResult)

	go func() {
		defer close(result)

		if err := r.ensureTable(ctx, collectionName); err != nil {
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}

		tableName := r.getTableName(collectionName)

		// Build WHERE clause using the hybrid approach
		whereClause, namedArgs, positionalArgs, err := r.buildWhereClause(query)
		if err != nil {
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}

		// Build the full COUNT query
		fullQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
		if whereClause != "" && whereClause != "TRUE" {
			fullQuery += " WHERE " + whereClause
		}

		// Use sqlx to bind named parameters
		// Escape :: casts with a sequence that won't be confused with parameter names
		// Use #CAST# (with #) to avoid sqlx treating it as part of parameter name
		// sqlx matches :[a-zA-Z0-9_]+ for parameter names, so # breaks the pattern
		// Don't use __ prefix to avoid sqlx matching :param__ as parameter name
		tempEscapedQuery := strings.ReplaceAll(fullQuery, "::", "#CAST#")
		var reboundQuery string
		var namedArgsSlice []interface{}
		if len(namedArgs) > 0 {
			var err2 error
			reboundQuery, namedArgsSlice, err2 = r.db.BindNamed(tempEscapedQuery, namedArgs)
			if err2 != nil {
				result <- interfaces.CountResult{Error: fmt.Errorf("failed to bind named query: %w", err2)}
				return
			}
		} else {
			reboundQuery = r.db.Rebind(tempEscapedQuery)
			namedArgsSlice = []interface{}{}
		}

		// Replace temporary array placeholders with correct positional numbers
		finalQuery := reboundQuery
		for i := range positionalArgs {
			tempPlaceholder := fmt.Sprintf("__ARRAY_PARAM_%d__", i)
			finalPlaceholder := fmt.Sprintf("$%d", len(namedArgsSlice)+i+1)
			finalQuery = strings.Replace(finalQuery, tempPlaceholder, finalPlaceholder, 1)
		}

		// Restore ::type casts
		finalQuery = strings.ReplaceAll(finalQuery, "#CAST#", "::")

		// Combine argument slices
		allArgs := append(namedArgsSlice, positionalArgs...)

		var count int64
		err = r.db.QueryRowContext(ctx, finalQuery, allArgs...).Scan(&count)
		if err != nil {
			log.Error("PostgreSQL CountWithFilter error: %s", err.Error())
			result <- interfaces.CountResult{Count: 0, Error: err}
			return
		}

		result <- interfaces.CountResult{Count: count, Error: nil}
	}()

	return result
}



// DB returns the underlying *sql.DB connection for direct access
func (r *PostgreSQLRepository) DB() *sql.DB {
	return r.db.DB // Return underlying *sql.DB from *sqlx.DB
}
