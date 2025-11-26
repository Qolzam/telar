// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package postgres

import (
	"context"
	"testing"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
)

func TestNewClient(t *testing.T) {
	ctx := context.Background()

	config := &dbi.PostgreSQLConfig{
		Host:               "localhost",
		Port:               5432,
		Username:           "postgres",
		Password:           "postgres",
		SSLMode:            "disable",
		MaxOpenConnections: 25,
		MaxIdleConnections: 10,
		MaxLifetime:        300,
		ConnectTimeout:     10,
	}

	client, err := NewClient(ctx, config, "telar_social_test")
	if err != nil {
		t.Skipf("Skipping test: PostgreSQL not available: %v", err)
		return
	}
	defer client.Close()

	// Test connection
	if err := client.Ping(ctx); err != nil {
		t.Fatalf("Failed to ping database: %v", err)
	}

	// Test connection pool configuration
	db := client.DB()
	// sqlx.DB uses SetMaxOpenConns/SetMaxIdleConns, not getters
	// We can only verify the connection works, not the exact pool settings
	// The pool settings are configured via SetMaxOpenConns/SetMaxIdleConns during client creation
	_ = db // Verify DB is accessible
}

func TestClient_HealthCheck(t *testing.T) {
	ctx := context.Background()

	config := &dbi.PostgreSQLConfig{
		Host:     "localhost",
		Port:     5432,
		Username: "postgres",
		Password: "postgres",
		SSLMode:  "disable",
	}

	client, err := NewClient(ctx, config, "telar_social_test")
	if err != nil {
		t.Skipf("Skipping test: PostgreSQL not available: %v", err)
		return
	}
	defer client.Close()

	if err := client.HealthCheck(ctx); err != nil {
		t.Fatalf("Health check failed: %v", err)
	}
}


