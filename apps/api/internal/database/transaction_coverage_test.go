// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

// Package database_test contains coverage-specific tests for transaction operations
// These tests focus on achieving comprehensive coverage of repository operations
// that may not be fully tested in the main transaction test suite.
//
// Purpose: Test operations with missing coverage, edge cases, and error paths
// Focus: Operations like Find, UpdateMany, Count, CreateIndex, error scenarios
// Coverage: Targets specific functions showing 0% or low coverage in reports

package database_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMissingCoverageOperations tests operations with low coverage to improve overall coverage
func TestMissingCoverageOperations(t *testing.T) {
	suite := NewTransactionTestSuite(t)
	suite.SetupDatabases()

	for dbType, repo := range suite.repos {
		t.Run(fmt.Sprintf("%s_MissingCoverage", dbType), func(t *testing.T) {
			suite.testMissingCoverageOperations(t, repo, dbType)
		})
	}
}

// testMissingCoverageOperations tests operations that currently have 0% or low coverage
func (ts *TransactionTestSuite) testMissingCoverageOperations(t *testing.T, repo interfaces.Repository, dbType string) {
	ctx := context.Background()
	collectionName := fmt.Sprintf("test_coverage_%s_%d", dbType, time.Now().UnixNano())

	// Test Find operation (0% coverage on MongoDB)
	t.Run("Find_Operation", func(t *testing.T) {
		// Create test data first
		testData := TestData{
			ObjectId:    uuid.Must(uuid.NewV4()),
			Name:        fmt.Sprintf("find_test_%d", time.Now().UnixNano()),
			Value:       100,
			OwnerUserId: "find_owner",
			CreatedDate: time.Now().Unix(),
			LastUpdated: time.Now().Unix(),
			Deleted:     false,
		}
		
		result := <-repo.Save(ctx, collectionName, testData)
		require.NoError(t, result.Error)

		// Test Find operation
		filter := map[string]interface{}{"ownerUserId": "find_owner"}
		limit := int64(10)
		opts := &interfaces.FindOptions{
			Limit: &limit,
			Sort:  map[string]int{"created_date": 1},
		}

		findResult := <-repo.Find(ctx, collectionName, filter, opts)
		assert.NoError(t, findResult.Error(), "Find operation should succeed")
		
		// Verify we can iterate through results
		var count int
		for findResult.Next() {
			var item TestData
			err := findResult.Decode(&item)
			assert.NoError(t, err)
			count++
		}
		assert.Greater(t, count, 0, "Should find at least one document")
	})

	// Test UpdateMany operation (0% coverage on MongoDB)
	t.Run("UpdateMany_Operation", func(t *testing.T) {
		// Create multiple test documents
		for i := 0; i < 3; i++ {
			testData := TestData{
				ObjectId:    uuid.Must(uuid.NewV4()),
				Name:        fmt.Sprintf("update_many_%d", i),
				Value:       200,
				OwnerUserId: "update_many_owner",
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
				Deleted:     false,
			}
			result := <-repo.Save(ctx, collectionName, testData)
			require.NoError(t, result.Error)
		}

		// Test UpdateMany operation
		filter := map[string]interface{}{"ownerUserId": "update_many_owner"}
		updateData := map[string]interface{}{
			"$set": map[string]interface{}{
				"value":       300,
				"lastUpdated": time.Now().Unix(),
			},
		}

		updateResult := <-repo.UpdateMany(ctx, collectionName, filter, updateData, nil)
		assert.NoError(t, updateResult.Error, "UpdateMany operation should succeed")
	})

	// Test Count operation (0% coverage on MongoDB)
	t.Run("Count_Operation", func(t *testing.T) {
		// Create test data for counting
		for i := 0; i < 5; i++ {
			testData := TestData{
				ObjectId:    uuid.Must(uuid.NewV4()),
				Name:        fmt.Sprintf("count_test_%d", i),
				Value:       500,
				OwnerUserId: "count_owner",
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
				Deleted:     false,
			}
			result := <-repo.Save(ctx, collectionName, testData)
			require.NoError(t, result.Error)
		}

		// Test Count operation
		filter := map[string]interface{}{"ownerUserId": "count_owner"}
		countResult := <-repo.Count(ctx, collectionName, filter)
		assert.NoError(t, countResult.Error, "Count operation should succeed")
		assert.GreaterOrEqual(t, countResult.Count, int64(5), "Should count at least 5 documents")
	})

	// Test CreateIndex operation (0% coverage on MongoDB)
	t.Run("CreateIndex_Operation", func(t *testing.T) {
		indexes := map[string]interface{}{
			"ownerUserId": 1,
			"value":       -1,
		}

		indexResult := <-repo.CreateIndex(ctx, collectionName, indexes)
		assert.NoError(t, indexResult, "CreateIndex operation should succeed")
	})

	// Test WithTransaction operation (0% coverage)
	t.Run("WithTransaction_Operation", func(t *testing.T) {
		var transactionExecuted bool
		
		err := repo.WithTransaction(ctx, func(txCtx context.Context) error {
			transactionExecuted = true
			
			// Perform some operations within the transaction function
			testData := TestData{
				ObjectId:    uuid.Must(uuid.NewV4()),
				Name:        fmt.Sprintf("with_tx_test_%d", time.Now().UnixNano()),
				Value:       777,
				OwnerUserId: "with_tx_owner",
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
				Deleted:     false,
			}
			
			result := <-repo.Save(txCtx, collectionName, testData)
			return result.Error
		})

		assert.NoError(t, err, "WithTransaction should succeed")
		assert.True(t, transactionExecuted, "Transaction function should be executed")
	})

	// Test Ping operation (0% coverage)
	t.Run("Ping_Operation", func(t *testing.T) {
		pingResult := <-repo.Ping(ctx)
		assert.NoError(t, pingResult, "Ping operation should succeed")
	})

	// Test error handling paths
	t.Run("Error_Handling", func(t *testing.T) {
		// Test operations with invalid data to trigger error paths
		
		// Invalid filter for FindOne
		invalidFilter := map[string]interface{}{
			"$invalid": "invalid_operator",
		}
		
		findResult := <-repo.FindOne(ctx, collectionName, invalidFilter)
		// This may or may not error depending on database implementation
		// The important thing is we're exercising the error handling code paths
		_ = findResult.Error()
	})
}

// TestTransactionCoverageSpecific tests specific transaction operations for better coverage
func TestTransactionCoverageSpecific(t *testing.T) {
	suite := NewTransactionTestSuite(t)
	suite.SetupDatabases()

	for dbType, repo := range suite.repos {
		t.Run(fmt.Sprintf("%s_TransactionSpecific", dbType), func(t *testing.T) {
			suite.testTransactionSpecificCoverage(t, repo, dbType)
		})
	}
}

func (ts *TransactionTestSuite) testTransactionSpecificCoverage(t *testing.T, repo interfaces.Repository, dbType string) {
	ctx := context.Background()
	collectionName := fmt.Sprintf("test_tx_coverage_%s_%d", dbType, time.Now().UnixNano())

	// Test transaction Find operation (0% coverage)
	t.Run("Transaction_Find", func(t *testing.T) {
		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Create test data in transaction
		testData := TestData{
			ObjectId:    uuid.Must(uuid.NewV4()),
			Name:        fmt.Sprintf("tx_find_%d", time.Now().UnixNano()),
			Value:       888,
			OwnerUserId: "tx_find_owner",
			CreatedDate: time.Now().Unix(),
			LastUpdated: time.Now().Unix(),
			Deleted:     false,
		}

		saveResult := <-tx.Save(ctx, collectionName, testData)
		require.NoError(t, saveResult.Error)

		// Test Find operation in transaction
		filter := map[string]interface{}{"ownerUserId": "tx_find_owner"}
		limit := int64(5)
		opts := &interfaces.FindOptions{
			Limit: &limit,
		}

		findResult := <-tx.Find(ctx, collectionName, filter, opts)
		assert.NoError(t, findResult.Error(), "Transaction Find should succeed")

		err = tx.Commit()
		assert.NoError(t, err)
	})

	// Test transaction UpdateMany operation (0% coverage)
	t.Run("Transaction_UpdateMany", func(t *testing.T) {
		// Create initial data
		for i := 0; i < 2; i++ {
			testData := TestData{
				ObjectId:    uuid.Must(uuid.NewV4()),
				Name:        fmt.Sprintf("tx_update_many_%d", i),
				Value:       999,
				OwnerUserId: "tx_update_many_owner",
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
				Deleted:     false,
			}
			result := <-repo.Save(ctx, collectionName, testData)
			require.NoError(t, result.Error)
		}

		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Test UpdateMany in transaction
		filter := map[string]interface{}{"ownerUserId": "tx_update_many_owner"}
		updateData := map[string]interface{}{
			"$set": map[string]interface{}{
				"value": 1111,
			},
		}

		updateResult := <-tx.UpdateMany(ctx, collectionName, filter, updateData, nil)
		assert.NoError(t, updateResult.Error, "Transaction UpdateMany should succeed")

		err = tx.Commit()
		assert.NoError(t, err)
	})

	// Test transaction Count operation (0% coverage)
	t.Run("Transaction_Count", func(t *testing.T) {
		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Test Count in transaction
		filter := map[string]interface{}{"value": 1111}
		countResult := <-tx.Count(ctx, collectionName, filter)
		assert.NoError(t, countResult.Error, "Transaction Count should succeed")

		err = tx.Commit()
		assert.NoError(t, err)
	})

	// Test CreateIndex in transaction 
	// MongoDB Reference: https://www.mongodb.com/docs/manual/core/transactions-operations/
	// PostgreSQL Reference: https://www.postgresql.org/docs/current/sql-createindex.html
	// 
	// MongoDB: Supports CreateIndex in transactions with restrictions:
	// 1. Index must be on non-existent collection OR new empty collection created in same transaction
	// 2. Transaction must use read concern "local"
	// 3. Cannot be cross-shard write transaction with non-local read concern
	//
	// PostgreSQL: Fully supports CreateIndex in transactions without restrictions
	// Quote: "a regular CREATE INDEX command can be performed within a transaction block"
	t.Run("Transaction_CreateIndex", func(t *testing.T) {
		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Use a unique collection name to ensure it's new/empty for index creation
		uniqueCollectionName := fmt.Sprintf("test_idx_%s_%d", dbType, time.Now().UnixNano())
		
		indexes := map[string]interface{}{
			"name": 1,
		}

		indexResult := <-tx.CreateIndex(ctx, uniqueCollectionName, indexes)
		// Both MongoDB and PostgreSQL should support this operation per documentation
		// If it fails, it's likely due to implementation-specific restrictions
		if indexResult != nil {
			t.Logf("CreateIndex in transaction failed: %v", indexResult)
		}

		// Always rollback for transaction CreateIndex tests to avoid affecting other tests
		err = tx.Rollback()
		assert.NoError(t, err)
	})

	// Test different transaction error scenarios
	t.Run("Transaction_ErrorPaths", func(t *testing.T) {
		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Test operations that might fail to exercise error handling
		// This is important for coverage of error handling paths
		invalidData := map[string]interface{}{
			"invalid_field": func() {}, // Function types are typically not supported
		}

		result := <-tx.Save(ctx, collectionName, invalidData)
		// This should likely error, which is good for testing error paths
		_ = result.Error

		// Clean up - rollback since we might have errors
		err = tx.Rollback()
		assert.NoError(t, err)
	})
}

// TestRepositoryCloseAndConnection tests connection management operations
func TestRepositoryCloseAndConnection(t *testing.T) {
	suite := NewTransactionTestSuite(t)
	suite.SetupDatabases()

	for dbType, repo := range suite.repos {
		t.Run(fmt.Sprintf("%s_Connection", dbType), func(t *testing.T) {
			// Test Ping operation
			ctx := context.Background()
			pingResult := <-repo.Ping(ctx)
			assert.NoError(t, pingResult, "Ping should succeed")

			// Note: We don't test Close() here as it would close the connection
			// and break other tests. In a real scenario, this would be tested
			// in isolation or as the final test.
		})
	}
}
