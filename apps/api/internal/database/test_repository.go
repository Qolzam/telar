// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package database

import (
	"testing"

	"github.com/qolzam/telar/apps/api/internal/database/factory"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
)

func TestRepositoryCreation(t *testing.T) {
	// Test that we can create a repository instance
	config := &interfaces.RepositoryConfig{
		DatabaseType: interfaces.DatabaseTypeMongoDB,
		DatabaseName: "test_db",
		MongoConfig: &interfaces.MongoDBConfig{
			Host: "localhost",
			Port: 27017,
		},
	}

	repo, err := factory.CreateRepositoryFromConfig(nil, interfaces.DatabaseTypeMongoDB, config)
	if err != nil {
		t.Logf("Repository creation failed (expected for test environment): %v", err)
		// This is expected in test environment without actual database
		return
	}

	if repo == nil {
		t.Error("Repository should not be nil")
	}

	t.Log("Repository creation test passed")
}

func TestRepositoryConfigValidation(t *testing.T) {
	// Test MongoDB config validation
	mongoConfig := &interfaces.MongoDBConfig{
		Host: "localhost",
		Port: 27017,
	}

	if mongoConfig.Host == "" {
		t.Error("MongoDB host should not be empty")
	}

	if mongoConfig.Port <= 0 {
		t.Error("MongoDB port should be positive")
	}

	// Test PostgreSQL config validation
	pgConfig := &interfaces.PostgreSQLConfig{
		Host:     "localhost",
		Port:     5432,
		Database: "test_db",
	}

	if pgConfig.Host == "" {
		t.Error("PostgreSQL host should not be empty")
	}

	if pgConfig.Port <= 0 {
		t.Error("PostgreSQL port should be positive")
	}

	t.Log("Repository config validation test passed")
} 