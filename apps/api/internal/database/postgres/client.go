// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
)

// Client wraps sqlx.DB and provides connection pooling, health checks, and transaction management
type Client struct {
	db *sqlx.DB
}

// NewClient creates a new PostgreSQL client wrapper
func NewClient(ctx context.Context, config *dbi.PostgreSQLConfig, databaseName string) (*Client, error) {
	connStr := buildConnectionString(config, databaseName)

	db, err := sqlx.ConnectContext(ctx, "postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %w", err)
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

	return &Client{db: db}, nil
}

// buildConnectionString builds PostgreSQL connection string from config
func buildConnectionString(config *dbi.PostgreSQLConfig, databaseName string) string {
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

// DB returns the underlying *sqlx.DB connection
func (c *Client) DB() *sqlx.DB {
	return c.db
}

// Ping tests the database connection
func (c *Client) Ping(ctx context.Context) error {
	return c.db.PingContext(ctx)
}

// BeginTxx starts a new transaction with the given context
func (c *Client) BeginTxx(ctx context.Context, opts *sql.TxOptions) (*sqlx.Tx, error) {
	return c.db.BeginTxx(ctx, opts)
}

// Close closes the database connection
func (c *Client) Close() error {
	return c.db.Close()
}

// HealthCheck performs a health check on the database connection
func (c *Client) HealthCheck(ctx context.Context) error {
	return c.Ping(ctx)
}

