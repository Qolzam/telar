// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

//go:build integration
// +build integration

package database_test

import (
	"context"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/postgresql"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTransactionIntegration tests transactions with real database connections
// This file contains integration tests that require actual database instances
// Run with: go test -tags=integration -v ./internal/database -run TestTransactionIntegration
func TestTransactionIntegration(t *testing.T) {
	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Test PostgreSQL if available
	if suite.Config() != nil {
		t.Run("PostgreSQL_Integration", func(t *testing.T) {
			testTransactionIntegration(t, "postgresql")
		})
	}
	

	if suite.Config() == nil {
		t.Skip("No databases available for integration testing")
	}
}

func testTransactionIntegration(t *testing.T, dbType string) {
	ctx := context.Background()
	
	// Setup repository
	repo := setupRepository(t, dbType)
	require.NotNil(t, repo, "Repository should not be nil")
	
	// Test basic transaction lifecycle
	t.Run("BasicLifecycle", func(t *testing.T) {
		testBasicLifecycle(t, repo)
	})
	
	// Test transaction configuration
	t.Run("ConfiguredTransaction", func(t *testing.T) {
		testConfiguredTransaction(t, repo)
	})
	
	// Test transaction operations
	t.Run("TransactionOperations", func(t *testing.T) {
		testTransactionOperations(t, repo)
	})
	
	// Test rollback functionality
	t.Run("TransactionRollback", func(t *testing.T) {
		testTransactionRollback(t, repo)
	})
	
	// Test concurrent transactions
	t.Run("ConcurrentTransactions", func(t *testing.T) {
		testConcurrentTransactions(t, repo)
	})
	
	// Test ownership operations
	t.Run("OwnershipOperations", func(t *testing.T) {
		testOwnershipOperations(t, repo)
	})
}

func setupRepository(t *testing.T, dbType string) interfaces.Repository {
	if dbType != "postgresql" {
		t.Fatalf("Only PostgreSQL is supported, got: %s", dbType)
	}
	return setupPostgreSQLRepo(t)
}

func setupPostgreSQLRepo(t *testing.T) interfaces.Repository {
	// Use Config-First pattern instead of direct environment access
	cfg, err := platformconfig.LoadFromEnv()
	if err != nil {
		t.Skipf("Failed to load config: %v", err)
	}
	
	config := &interfaces.PostgreSQLConfig{
		Host:     cfg.Database.Postgres.Host,
		Port:     cfg.Database.Postgres.Port,
		Username: cfg.Database.Postgres.Username,
		Password: cfg.Database.Postgres.Password,
		Database: cfg.Database.Postgres.Database,
		SSLMode:  cfg.Database.Postgres.SSLMode,
	}
	
	repo, err := postgresql.NewPostgreSQLRepository(context.Background(), config, config.Database)
	if err != nil {
		t.Skipf("PostgreSQL not available: %v", err)
	}
	
	return repo
}



func testBasicLifecycle(t *testing.T, repo interfaces.Repository) {
	ctx := context.Background()
	
	// Begin transaction
	tx, err := repo.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	require.NotNil(t, tx, "Transaction should not be nil")
	
	// Verify transaction properties
	assert.True(t, tx.IsActive(), "Transaction should be active")
	assert.NotEmpty(t, tx.GetTransactionID(), "Transaction ID should not be empty")
	
	metrics := tx.GetMetrics()
	require.NotNil(t, metrics, "Metrics should not be nil")
	assert.Equal(t, "active", metrics.Status, "Transaction status should be active")
	
	// Commit transaction
	err = tx.Commit()
	assert.NoError(t, err, "Failed to commit transaction")
	assert.False(t, tx.IsActive(), "Transaction should not be active after commit")
	
	// Verify final metrics
	finalMetrics := tx.GetMetrics()
	assert.Equal(t, "committed", finalMetrics.Status, "Transaction status should be committed")
	assert.Greater(t, finalMetrics.Duration, time.Duration(0), "Transaction duration should be positive")
}

func testConfiguredTransaction(t *testing.T, repo interfaces.Repository) {
	ctx := context.Background()
	
	config := &interfaces.TransactionConfig{
		Timeout:        5 * time.Second,
		ReadOnly:       false,
		IsolationLevel: interfaces.IsolationLevelReadCommitted,
	}
	
	tx, err := repo.BeginWithConfig(ctx, config)
	require.NoError(t, err, "Failed to begin transaction with config")
	require.NotNil(t, tx, "Transaction should not be nil")
	
	// Verify configuration was applied
	txConfig := tx.GetConfig()
	require.NotNil(t, txConfig, "Transaction config should not be nil")
	assert.Equal(t, config.Timeout, txConfig.Timeout, "Timeout should match")
	assert.Equal(t, config.ReadOnly, txConfig.ReadOnly, "ReadOnly should match")
	assert.Equal(t, config.IsolationLevel, txConfig.IsolationLevel, "IsolationLevel should match")
	
	err = tx.Commit()
	assert.NoError(t, err, "Failed to commit configured transaction")
}

func testTransactionOperations(t *testing.T, repo interfaces.Repository) {
	ctx := context.Background()
	collectionName := "test_tx_operations"
	
	tx, err := repo.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	
	// Create test data
	testID, _ := uuid.NewV4()
	testData := map[string]interface{}{
		"objectId":    testID,
		"name":        "test_item",
		"value":       100,
		"createdDate": time.Now().Unix(),
		"lastUpdated": time.Now().Unix(),
		"deleted":     false,
	}
	
	// Test Save operation
	saveResult := <-tx.Save(ctx, collectionName, testData)
	require.NoError(t, saveResult.Error, "Save operation should succeed")
	require.NotNil(t, saveResult.Result, "Save result should not be nil")
	
	// Verify operation count increased
	metrics := tx.GetMetrics()
	assert.Greater(t, metrics.OperationsCount, int64(0), "Operation count should be greater than 0")
	
	// Test FindOne operation
	filter := map[string]interface{}{"objectId": testID}
	findResult := <-tx.FindOne(ctx, collectionName, filter)
	require.NoError(t, findResult.Error(), "FindOne operation should succeed")
	
	var retrieved map[string]interface{}
	err = findResult.Decode(&retrieved)
	require.NoError(t, err, "Should decode retrieved data")
	assert.Equal(t, "test_item", retrieved["name"], "Retrieved name should match")
	
	// Test Update operation
	updates := map[string]interface{}{"value": 200}
	updateResult := <-tx.UpdateFields(ctx, collectionName, filter, updates)
	require.NoError(t, updateResult.Error, "Update operation should succeed")
	
	// Test Increment operation
	increments := map[string]interface{}{"value": 50}
	incResult := <-tx.IncrementFields(ctx, collectionName, filter, increments)
	require.NoError(t, incResult.Error, "Increment operation should succeed")
	
	// Verify final operation count
	finalMetrics := tx.GetMetrics()
	assert.Greater(t, finalMetrics.OperationsCount, metrics.OperationsCount, "Operation count should have increased")
	
	err = tx.Commit()
	require.NoError(t, err, "Failed to commit transaction")
	
	// Verify data persisted after commit
	verifyTx, err := repo.Begin(ctx)
	require.NoError(t, err, "Failed to begin verification transaction")
	
	verifyResult := <-verifyTx.FindOne(ctx, collectionName, filter)
	require.NoError(t, verifyResult.Error(), "Verification FindOne should succeed")
	
	var verifiedData map[string]interface{}
	err = verifyResult.Decode(&verifiedData)
	require.NoError(t, err, "Should decode verified data")
	
	// Value should be 200 + 50 = 250
	assert.Equal(t, float64(250), verifiedData["value"], "Value should be updated and incremented")
	
	err = verifyTx.Commit()
	assert.NoError(t, err, "Failed to commit verification transaction")
}

func testTransactionRollback(t *testing.T, repo interfaces.Repository) {
	ctx := context.Background()
	collectionName := "test_tx_rollback"
	
	tx, err := repo.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	
	// Create test data
	testID, _ := uuid.NewV4()
	testData := map[string]interface{}{
		"objectId":    testID,
		"name":        "rollback_test",
		"value":       999,
		"createdDate": time.Now().Unix(),
		"lastUpdated": time.Now().Unix(),
		"deleted":     false,
	}
	
	// Save data in transaction
	saveResult := <-tx.Save(ctx, collectionName, testData)
	require.NoError(t, saveResult.Error, "Save operation should succeed")
	
	// Rollback the transaction
	err = tx.Rollback()
	assert.NoError(t, err, "Failed to rollback transaction")
	assert.False(t, tx.IsActive(), "Transaction should not be active after rollback")
	
	// Verify rollback metrics
	finalMetrics := tx.GetMetrics()
	assert.Equal(t, "rolled_back", finalMetrics.Status, "Transaction status should be rolled_back")
	
	// Verify data was not persisted
	verifyTx, err := repo.Begin(ctx)
	require.NoError(t, err, "Failed to begin verification transaction")
	
	filter := map[string]interface{}{"objectId": testID}
	verifyResult := <-verifyTx.FindOne(ctx, collectionName, filter)
	
	// Should not find the rolled back data
	assert.True(t, verifyResult.NoResult() || verifyResult.Error() != nil, "Should not find rolled back data")
	
	err = verifyTx.Commit()
	assert.NoError(t, err, "Failed to commit verification transaction")
}

func testConcurrentTransactions(t *testing.T, repo interfaces.Repository) {
	ctx := context.Background()
	collectionName := "test_tx_concurrent"
	const numTransactions = 3
	
	results := make(chan error, numTransactions)
	
	// Run concurrent transactions
	for i := 0; i < numTransactions; i++ {
		go func(index int) {
			tx, err := repo.Begin(ctx)
			if err != nil {
				results <- err
				return
			}
			
			testID, _ := uuid.NewV4()
			testData := map[string]interface{}{
				"objectId":    testID,
				"name":        "concurrent_test",
				"index":       index,
				"createdDate": time.Now().Unix(),
				"lastUpdated": time.Now().Unix(),
				"deleted":     false,
			}
			
			saveResult := <-tx.Save(ctx, collectionName, testData)
			if saveResult.Error != nil {
				results <- saveResult.Error
				_ = tx.Rollback()
				return
			}
			
			err = tx.Commit()
			results <- err
		}(i)
	}
	
	// Collect results
	errorCount := 0
	for i := 0; i < numTransactions; i++ {
		if err := <-results; err != nil {
			errorCount++
			t.Logf("Concurrent transaction %d error: %v", i, err)
		}
	}
	
	// Allow some errors but not all should fail
	assert.Less(t, errorCount, numTransactions, "Not all concurrent transactions should fail")
}

func testOwnershipOperations(t *testing.T, repo interfaces.Repository) {
	ctx := context.Background()
	collectionName := "test_tx_ownership"
	
	tx, err := repo.Begin(ctx)
	require.NoError(t, err, "Failed to begin transaction")
	
	// Create test data with ownership
	testID, _ := uuid.NewV4()
	ownerID, _ := uuid.NewV4()
	testData := map[string]interface{}{
		"objectId":     testID,
		"name":         "ownership_test",
		"value":        100,
		"ownerUserId":  ownerID.String(),
		"createdDate":  time.Now().Unix(),
		"lastUpdated":  time.Now().Unix(),
		"deleted":      false,
	}
	
	// Save test data
	saveResult := <-tx.Save(ctx, collectionName, testData)
	require.NoError(t, saveResult.Error, "Save operation should succeed")
	
	// Test UpdateWithOwnership - valid owner
	updates := map[string]interface{}{"value": 200}
	updateResult := <-tx.UpdateWithOwnership(ctx, collectionName, testID, ownerID.String(), updates)
	require.NoError(t, updateResult.Error, "UpdateWithOwnership should succeed for valid owner")
	
	// Test UpdateWithOwnership - invalid owner
	fakeOwner, _ := uuid.NewV4()
	invalidResult := <-tx.UpdateWithOwnership(ctx, collectionName, testID, fakeOwner.String(), updates)
	assert.Error(t, invalidResult.Error, "UpdateWithOwnership should fail for invalid owner")
	
	// Test IncrementWithOwnership - valid owner
	increments := map[string]interface{}{"value": 50}
	incResult := <-tx.IncrementWithOwnership(ctx, collectionName, testID, ownerID.String(), increments)
	require.NoError(t, incResult.Error, "IncrementWithOwnership should succeed for valid owner")
	
	// Test DeleteWithOwnership - valid owner
	deleteResult := <-tx.DeleteWithOwnership(ctx, collectionName, testID, ownerID.String())
	require.NoError(t, deleteResult.Error, "DeleteWithOwnership should succeed for valid owner")
	
	err = tx.Commit()
	assert.NoError(t, err, "Failed to commit transaction")
}

// BenchmarkTransactionE2EPerformance benchmarks real transaction performance with live databases
// Run with: go test -tags=e2e -bench=BenchmarkTransactionE2EPerformance -v ./apps/api/internal/database
func BenchmarkTransactionE2EPerformance(b *testing.B) {
	suite := testutil.Setup(&testing.T{})
	
	var repo interfaces.Repository
	
	if suite.Config() != nil {
		repo = setupPostgreSQLRepo(&testing.T{})
	} else {
		b.Skip("No databases available for benchmarking")
	}
	
	ctx := context.Background()
	testID, _ := uuid.NewV4()
	testData := map[string]interface{}{
		"objectId":    testID,
		"name":        "benchmark_test",
		"value":       100,
		"createdDate": time.Now().Unix(),
		"lastUpdated": time.Now().Unix(),
		"deleted":     false,
	}
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			tx, err := repo.Begin(ctx)
			if err != nil {
				b.Fatalf("Failed to begin transaction: %v", err)
			}
			
			result := <-tx.Save(ctx, "benchmark_collection", testData)
			if result.Error != nil {
				b.Fatalf("Save operation failed: %v", result.Error)
			}
			
			err = tx.Commit()
			if err != nil {
				b.Fatalf("Failed to commit transaction: %v", err)
			}
		}
	})
}
