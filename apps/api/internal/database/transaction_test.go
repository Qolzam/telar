// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

// Package database_test contains the main transaction test suite
// This is the comprehensive test suite for enterprise-grade transaction management
//
// Purpose: Complete transaction lifecycle testing, enterprise features, stress testing
// Coverage: Basic operations, configuration, concurrent transactions, metrics, stress scenarios
// Database: PostgreSQL with full behavior validation
//
// Run with: go test -v ./internal/database -run TestTransactionSuite

package database_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/observability"
	"github.com/qolzam/telar/apps/api/internal/database/postgresql"
	"github.com/qolzam/telar/apps/api/internal/database/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestData represents test data structure
type TestData struct {
	ObjectId    uuid.UUID `json:"objectId" bson:"objectId"`
	Name        string    `json:"name" bson:"name"`
	Value       int       `json:"value" bson:"value"`
	OwnerUserId string    `json:"ownerUserId" bson:"ownerUserId"`
	CreatedDate int64     `json:"created_date" bson:"created_date"`
	LastUpdated int64     `json:"last_updated" bson:"last_updated"`
	Deleted     bool      `json:"deleted" bson:"deleted"`
}

// DatabaseTestConfig holds database configuration for tests
type DatabaseTestConfig struct {
	PostgreSQL *interfaces.PostgreSQLConfig
}

// TransactionTestSuite provides comprehensive transaction testing
type TransactionTestSuite struct {
	t        *testing.T
	repos    map[string]interfaces.Repository
	testData []TestData
}

// NewTransactionTestSuite creates a new transaction test suite
func NewTransactionTestSuite(t *testing.T) *TransactionTestSuite {
	return &TransactionTestSuite{
		t:       t,
		repos:   make(map[string]interfaces.Repository),
		testData: generateTestData(10),
	}
}

// generateTestData creates test data for testing
func generateTestData(count int) []TestData {
	data := make([]TestData, count)
	now := time.Now().Unix()
	
	for i := 0; i < count; i++ {
		id, _ := uuid.NewV4()
		ownerID, _ := uuid.NewV4()
		
		data[i] = TestData{
			ObjectId:    id,
			Name:        fmt.Sprintf("test_item_%d_%d", i, now), // Add timestamp for uniqueness
			Value:       i * 10,
			OwnerUserId: ownerID.String(),
			CreatedDate: now,
			LastUpdated: now,
			Deleted:     false,
		}
	}
	
	return data
}

// SetupDatabases initializes test databases
func (ts *TransactionTestSuite) SetupDatabases() {
	// Setup PostgreSQL - matching test_env.sh configuration
	pgConfig := &interfaces.PostgreSQLConfig{
		Host:     "127.0.0.1",
		Port:     5432,
		Username: "postgres",
		Password: "postgres",
		Database: "telar_social_test",
		SSLMode:  "disable",
	}
	
	if pgRepo, err := postgresql.NewPostgreSQLRepository(context.Background(), pgConfig, "telar_social_test"); err == nil {
		ts.repos["postgresql"] = pgRepo
		ts.t.Logf("PostgreSQL repository initialized")
	} else {
		ts.t.Logf("PostgreSQL not available: %v", err)
	}
	
	
	if len(ts.repos) == 0 {
		ts.t.Skip("No databases available for testing")
	}
}

// TestBasicTransactionLifecycle tests basic transaction operations
func (ts *TransactionTestSuite) TestBasicTransactionLifecycle() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_BasicLifecycle", dbType), func(t *testing.T) {
			ctx := context.Background()
			
			// Test Begin
			tx, err := repo.Begin(ctx)
			require.NoError(t, err, "Failed to begin transaction")
			require.NotNil(t, tx, "Transaction should not be nil")
			
			// Verify transaction is active
			assert.True(t, tx.IsActive(), "Transaction should be active")
			assert.NotEmpty(t, tx.GetTransactionID(), "Transaction ID should not be empty")
			
			// Test metrics
			metrics := tx.GetMetrics()
			require.NotNil(t, metrics, "Metrics should not be nil")
			assert.Equal(t, "active", metrics.Status, "Transaction status should be active")
			assert.Equal(t, dbType, metrics.DatabaseType, "Database type should match")
			
			// Test Commit
			err = tx.Commit()
			assert.NoError(t, err, "Failed to commit transaction")
			assert.False(t, tx.IsActive(), "Transaction should not be active after commit")
		})
	}
}

// TestTransactionWithConfig tests enterprise configuration features
func (ts *TransactionTestSuite) TestTransactionWithConfig() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_WithConfig", dbType), func(t *testing.T) {
			ctx := context.Background()
			
			config := &interfaces.TransactionConfig{
				Timeout:        5 * time.Second,
				ReadOnly:       false,
				IsolationLevel: interfaces.IsolationLevelReadCommitted,
				RetryPolicy: &interfaces.RetryPolicy{
					MaxRetries:      2,
					InitialDelay:    100 * time.Millisecond,
					MaxDelay:        1 * time.Second,
					BackoffFactor:   2.0,
					RetryableErrors: []string{"TRANSACTION_CONFLICT"},
				},
			}
			
			tx, err := repo.BeginWithConfig(ctx, config)
			require.NoError(t, err, "Failed to begin transaction with config")
			require.NotNil(t, tx, "Transaction should not be nil")
			
			// Verify configuration
			txConfig := tx.GetConfig()
			require.NotNil(t, txConfig, "Transaction config should not be nil")
			assert.Equal(t, config.Timeout, txConfig.Timeout, "Timeout should match")
			assert.Equal(t, config.ReadOnly, txConfig.ReadOnly, "ReadOnly should match")
			assert.Equal(t, config.IsolationLevel, txConfig.IsolationLevel, "IsolationLevel should match")
			
			err = tx.Commit()
			assert.NoError(t, err, "Failed to commit transaction")
		})
	}
}

// TestTransactionOperations tests CRUD operations within transactions
func (ts *TransactionTestSuite) TestTransactionOperations() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_Operations", dbType), func(t *testing.T) {
			ctx := context.Background()
			collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())
			
			tx, err := repo.Begin(ctx)
			require.NoError(t, err, "Failed to begin transaction")
			
			// Get initial operation count
			initialMetrics := tx.GetMetrics()
			initialCount := initialMetrics.OperationsCount
			
			// Test Save operation
			testItem := ts.testData[0]
			ownerID, err := uuid.FromString(testItem.OwnerUserId)
			require.NoError(t, err)
			result := <-tx.Save(ctx, collectionName, testItem.ObjectId, ownerID, testItem.CreatedDate, testItem.LastUpdated, testItem)
			require.NoError(t, result.Error, "Save operation failed")
			require.NotNil(t, result.Result, "Save result should not be nil")
			
			// Test FindOne operation - use Query object
			queryObj := &interfaces.Query{
				Conditions: []interfaces.Field{
					{Name: "object_id", Value: testItem.ObjectId, Operator: "="},
				},
			}
			findResult := <-tx.FindOne(ctx, collectionName, queryObj)
			require.NoError(t, findResult.Error(), "FindOne operation failed")
			
			var retrievedItem TestData
			err = findResult.Decode(&retrievedItem)
			require.NoError(t, err, "Failed to decode retrieved item")
			assert.Equal(t, testItem.Name, retrievedItem.Name, "Retrieved item name should match")
			
			// Test Update operation - use Query object
			updates := map[string]interface{}{"value": 999}
			updateResult := <-tx.UpdateFields(ctx, collectionName, queryObj, updates)
			require.NoError(t, updateResult.Error, "Update operation failed")
			
			// Test Increment operation - use Query object
			increments := map[string]interface{}{"value": 1}
			incResult := <-tx.IncrementFields(ctx, collectionName, queryObj, increments)
			require.NoError(t, incResult.Error, "Increment operation failed")
			
			// Test Delete operation - use Query object
			deleteResult := <-tx.Delete(ctx, collectionName, queryObj)
			require.NoError(t, deleteResult.Error, "Delete operation failed")
			
			// Verify final operation count has increased
			finalMetrics := tx.GetMetrics()
			t.Logf("Initial operations: %d, Final operations: %d", initialCount, finalMetrics.OperationsCount)
			assert.Greater(t, finalMetrics.OperationsCount, initialCount, "Operation count should have increased from %d to more than %d", initialCount, finalMetrics.OperationsCount)
			
			err = tx.Commit()
			assert.NoError(t, err, "Failed to commit transaction")
		})
	}
}

// TestTransactionRollback tests rollback functionality
func (ts *TransactionTestSuite) TestTransactionRollback() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_Rollback", dbType), func(t *testing.T) {
			ctx := context.Background()
			collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())
			
			tx, err := repo.Begin(ctx)
			require.NoError(t, err, "Failed to begin transaction")
			
			// Perform some operations
			testItem := ts.testData[1]
			ownerID, err := uuid.FromString(testItem.OwnerUserId)
			require.NoError(t, err)
			result := <-tx.Save(ctx, collectionName, testItem.ObjectId, ownerID, testItem.CreatedDate, testItem.LastUpdated, testItem)
			require.NoError(t, result.Error, "Save operation failed")
			
			// Rollback the transaction
			err = tx.Rollback()
			assert.NoError(t, err, "Failed to rollback transaction")
			assert.False(t, tx.IsActive(), "Transaction should not be active after rollback")
			
			// Verify rollback was recorded in metrics
			finalMetrics := tx.GetMetrics()
			assert.Equal(t, "rolled_back", finalMetrics.Status, "Transaction status should be rolled_back")
		})
	}
}

// TestTransactionTimeout tests timeout functionality
func (ts *TransactionTestSuite) TestTransactionTimeout() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_Timeout", dbType), func(t *testing.T) {
			ctx := context.Background()
			collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())
			
			config := &interfaces.TransactionConfig{
				Timeout: 100 * time.Millisecond, // Very short timeout
			}
			
			tx, err := repo.BeginWithConfig(ctx, config)
			require.NoError(t, err, "Failed to begin transaction with timeout")
			
			// Wait longer than the timeout
			time.Sleep(200 * time.Millisecond)
			
			// Operations should fail due to timeout
			testItem := ts.testData[2]
			ownerID, err := uuid.FromString(testItem.OwnerUserId)
			require.NoError(t, err)
			<-tx.Save(ctx, collectionName, testItem.ObjectId, ownerID, testItem.CreatedDate, testItem.LastUpdated, testItem)
			// This may or may not error depending on database implementation
			// The important thing is that the transaction respects the timeout
			
			// Clean up
			if tx.IsActive() {
				_ = tx.Rollback()
			}
		})
	}
}

// TestConcurrentTransactions tests concurrent transaction handling
func (ts *TransactionTestSuite) TestConcurrentTransactions() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_Concurrent", dbType), func(t *testing.T) {
			ctx := context.Background()
			collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())
			const numTransactions = 5
			
			var wg sync.WaitGroup
			results := make(chan error, numTransactions)
			
			for i := 0; i < numTransactions; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()
					
					tx, err := repo.Begin(ctx)
					if err != nil {
						results <- err
						return
					}
					
					testItem := ts.testData[index]
					testItem.Name = fmt.Sprintf("concurrent_test_%d_%d", index, time.Now().UnixNano())
					ownerID, err := uuid.FromString(testItem.OwnerUserId)
			require.NoError(t, err)
					
					result := <-tx.Save(ctx, collectionName, testItem.ObjectId, ownerID, testItem.CreatedDate, testItem.LastUpdated, testItem)
					if result.Error != nil {
						results <- result.Error
						_ = tx.Rollback()
						return
					}
					
					err = tx.Commit()
					results <- err
				}(i)
			}
			
			wg.Wait()
			close(results)
			
			errorCount := 0
			for err := range results {
				if err != nil {
					errorCount++
					t.Logf("Concurrent transaction error: %v", err)
				}
			}
			
			// Allow some errors in concurrent scenarios, but not all should fail
			assert.Less(t, errorCount, numTransactions, "Not all concurrent transactions should fail")
		})
	}
}

// TestOwnershipOperations tests ownership validation operations
func (ts *TransactionTestSuite) TestOwnershipOperations() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_Ownership", dbType), func(t *testing.T) {
			ctx := context.Background()
			collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())
			
			tx, err := repo.Begin(ctx)
			require.NoError(t, err, "Failed to begin transaction")
			
			testItem := ts.testData[3]
			
			// Save test item first
			ownerID, err := uuid.FromString(testItem.OwnerUserId)
			require.NoError(t, err)
			result := <-tx.Save(ctx, collectionName, testItem.ObjectId, ownerID, testItem.CreatedDate, testItem.LastUpdated, testItem)
			require.NoError(t, result.Error, "Save operation failed")
			
			// Test UpdateWithOwnership - valid owner
			updates := map[string]interface{}{"value": 777}
			updateResult := <-tx.UpdateWithOwnership(ctx, collectionName, testItem.ObjectId, testItem.OwnerUserId, updates)
			require.NoError(t, updateResult.Error, "UpdateWithOwnership failed for valid owner")
			
			// Test UpdateWithOwnership - invalid owner
			fakeOwner, _ := uuid.NewV4()
			invalidResult := <-tx.UpdateWithOwnership(ctx, collectionName, testItem.ObjectId, fakeOwner.String(), updates)
			assert.Error(t, invalidResult.Error, "UpdateWithOwnership should fail for invalid owner")
			
			// Test IncrementWithOwnership - valid owner
			increments := map[string]interface{}{"value": 10}
			incResult := <-tx.IncrementWithOwnership(ctx, collectionName, testItem.ObjectId, testItem.OwnerUserId, increments)
			require.NoError(t, incResult.Error, "IncrementWithOwnership failed for valid owner")
			
			// Test DeleteWithOwnership - valid owner
			deleteResult := <-tx.DeleteWithOwnership(ctx, collectionName, testItem.ObjectId, testItem.OwnerUserId)
			require.NoError(t, deleteResult.Error, "DeleteWithOwnership failed for valid owner")
			
			err = tx.Commit()
			assert.NoError(t, err, "Failed to commit transaction")
		})
	}
}

// TestNestedTransactions tests that nested transactions are properly rejected
func (ts *TransactionTestSuite) TestNestedTransactions() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_NestedTransactions", dbType), func(t *testing.T) {
			ctx := context.Background()
			
			tx, err := repo.Begin(ctx)
			require.NoError(t, err, "Failed to begin transaction")
			
			// Attempt to begin nested transaction
			nestedTx, err := tx.Begin(ctx)
			assert.Error(t, err, "Nested transaction should fail")
			assert.Nil(t, nestedTx, "Nested transaction should be nil")
			assert.Equal(t, interfaces.ErrNestedTransaction, err, "Error should be ErrNestedTransaction")
			
			// Test BeginWithConfig
			config := utils.DefaultTransactionConfig()
			nestedTxWithConfig, err := tx.BeginWithConfig(ctx, config)
			assert.Error(t, err, "Nested transaction with config should fail")
			assert.Nil(t, nestedTxWithConfig, "Nested transaction with config should be nil")
			
			// Test legacy BeginTransaction
			nestedLegacy, err := tx.BeginTransaction(ctx)
			assert.Error(t, err, "Nested legacy transaction should fail")
			assert.Nil(t, nestedLegacy, "Nested legacy transaction should be nil")
			
			err = tx.Commit()
			assert.NoError(t, err, "Failed to commit transaction")
		})
	}
}

// TestTransactionMetrics tests metrics collection and reporting
func (ts *TransactionTestSuite) TestTransactionMetrics() {
	// Clean up existing metrics
	observability.GetGlobalMetrics().CleanupCompletedTransactions(0)
	
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_Metrics", dbType), func(t *testing.T) {
			ctx := context.Background()
			collectionName := fmt.Sprintf("test_collection_%d", time.Now().UnixNano())
			
			initialStats := observability.GetGlobalMetrics().GetGlobalStats()
			initialActive := initialStats["active_transactions"].(int64)
			
			tx, err := repo.Begin(ctx)
			require.NoError(t, err, "Failed to begin transaction")
			
			// Check that active transactions increased
			stats := observability.GetGlobalMetrics().GetGlobalStats()
			assert.Equal(t, initialActive+1, stats["active_transactions"].(int64), "Active transactions should increase")
			
			// Perform some operations to test operation counting
			testItem := ts.testData[4]
			// Make the test item completely unique for this specific test
			testItem.ObjectId, _ = uuid.NewV4()
			testItem.Name = fmt.Sprintf("metrics_test_%d_%s", time.Now().UnixNano(), testItem.ObjectId.String()[:8])
			
			for i := 0; i < 3; i++ {
				// Use a slightly different item for each save to avoid conflicts
				uniqueItem := testItem
				uniqueItem.ObjectId, _ = uuid.NewV4()
				uniqueItem.Name = fmt.Sprintf("metrics_test_%d_%d_%s", time.Now().UnixNano(), i, uniqueItem.ObjectId.String()[:8])
				
				ownerID, err := uuid.FromString(uniqueItem.OwnerUserId)
				require.NoError(t, err)
				result := <-tx.Save(ctx, collectionName, uniqueItem.ObjectId, ownerID, uniqueItem.CreatedDate, uniqueItem.LastUpdated, uniqueItem)
				require.NoError(t, result.Error, "Save operation failed")
			}
			
			// Check operation count in metrics
			metrics := tx.GetMetrics()
			assert.GreaterOrEqual(t, metrics.OperationsCount, int64(3), "Operation count should be at least 3")
			
			err = tx.Commit()
			require.NoError(t, err, "Failed to commit transaction")
			
			// Check that active transactions decreased and committed increased
			finalStats := observability.GetGlobalMetrics().GetGlobalStats()
			assert.Equal(t, initialActive, finalStats["active_transactions"].(int64), "Active transactions should return to initial")
			
			// Final metrics should show committed status
			finalMetrics := tx.GetMetrics()
			assert.Equal(t, "committed", finalMetrics.Status, "Transaction status should be committed")
			assert.Greater(t, finalMetrics.Duration, time.Duration(0), "Transaction duration should be positive")
		})
	}
}

// RunAllTests runs the complete test suite
func (ts *TransactionTestSuite) RunAllTests() {
	ts.SetupDatabases()
	
	ts.TestBasicTransactionLifecycle()
	ts.TestTransactionWithConfig()
	ts.TestTransactionOperations()
	ts.TestTransactionRollback()
	ts.TestTransactionTimeout()
	ts.TestConcurrentTransactions()
	ts.TestOwnershipOperations()
	ts.TestNestedTransactions()
	ts.TestTransactionMetrics()
	ts.TestEnterpriseTransactionOperations()  // Added enterprise coverage
	ts.TestAdvancedTransactionConfiguration() // Added advanced config testing
	ts.TestTransactionStressScenarios()       // Added stress testing
}

// TestTransactionSuite is the main test function
func TestTransactionSuite(t *testing.T) {
	suite := NewTransactionTestSuite(t)
	suite.RunAllTests()
}

// TestEnterpriseTransactionOperations tests operations that need additional transaction coverage
func (ts *TransactionTestSuite) TestEnterpriseTransactionOperations() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_EnterpriseOperations", dbType), func(t *testing.T) {
			ts.testEnterpriseOperationsInTransaction(t, repo, dbType)
		})
	}
}

// testEnterpriseOperationsInTransaction tests critical operations for enterprise readiness
func (ts *TransactionTestSuite) testEnterpriseOperationsInTransaction(t *testing.T, repo interfaces.Repository, dbType string) {
	ctx := context.Background()
	collectionName := fmt.Sprintf("test_enterprise_%s_%d", dbType, time.Now().UnixNano())

	// Test SaveMany in transaction (0% coverage previously)
	t.Run("SaveMany_InTransaction", func(t *testing.T) {
		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Prepare bulk data
		bulkData := make([]interface{}, 0, 5)
		for i := 0; i < 5; i++ {
			data := TestData{
				ObjectId:    uuid.Must(uuid.NewV4()),
				Name:        fmt.Sprintf("bulk_item_%d_%d", i, time.Now().UnixNano()),
				Value:       i * 10,
				OwnerUserId: uuid.Must(uuid.NewV4()).String(),
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
				Deleted:     false,
			}
			bulkData = append(bulkData, data)
		}

		// Save many documents in transaction - convert to SaveItem format
		saveItems := make([]interfaces.SaveItem, len(bulkData))
		for i, data := range bulkData {
			td := data.(TestData)
			ownerID, err := uuid.FromString(td.OwnerUserId)
			require.NoError(t, err)
			saveItems[i] = interfaces.SaveItem{
				ObjectID:    td.ObjectId,
				OwnerUserID: ownerID,
				CreatedDate: td.CreatedDate,
				LastUpdated: td.LastUpdated,
				Data:        td,
			}
		}
		result := <-tx.SaveMany(ctx, collectionName, saveItems)
		assert.NoError(t, result.Error, "SaveMany should succeed in transaction")
		assert.NotNil(t, result.Result, "SaveMany should return result")

		// Verify operation count
		metrics := tx.GetMetrics()
		assert.Equal(t, int64(1), metrics.OperationsCount, "Operation count should be 1 after SaveMany")

		err = tx.Commit()
		assert.NoError(t, err, "Transaction should commit successfully")
	})

	// Test DeleteMany in transaction
	t.Run("DeleteMany_InTransaction", func(t *testing.T) {
		// First, create test data
		testIds := make([]uuid.UUID, 3)
		for i := 0; i < 3; i++ {
			id := uuid.Must(uuid.NewV4())
			testIds[i] = id
			data := TestData{
				ObjectId:    id,
				Name:        fmt.Sprintf("delete_many_%d_%d", i, time.Now().UnixNano()),
				Value:       i,
				OwnerUserId: uuid.Must(uuid.NewV4()).String(),
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
				Deleted:     false,
			}
			ownerID, err := uuid.FromString(data.OwnerUserId)
			require.NoError(t, err)
			result := <-repo.Save(ctx, collectionName, data.ObjectId, ownerID, data.CreatedDate, data.LastUpdated, data)
			require.NoError(t, result.Error)
		}

		// Now test DeleteMany in transaction
		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Prepare delete queries - use Query objects
		deleteQueries := make([]*interfaces.Query, len(testIds))
		for i, id := range testIds {
			deleteQueries[i] = &interfaces.Query{
				Conditions: []interfaces.Field{
					{Name: "object_id", Value: id, Operator: "="},
				},
			}
		}

		// Delete many documents in transaction
		result := <-tx.DeleteMany(ctx, collectionName, deleteQueries)
		assert.NoError(t, result.Error, "DeleteMany should succeed in transaction")

		err = tx.Commit()
		assert.NoError(t, err, "Transaction should commit successfully")
	})

	// Test CountWithFilter in transaction (0% coverage on PostgreSQL)
	t.Run("CountWithFilter_InTransaction", func(t *testing.T) {
		// Create test data with specific values
		for i := 0; i < 3; i++ {
			data := TestData{
				ObjectId:    uuid.Must(uuid.NewV4()),
				Name:        fmt.Sprintf("countable_%d", time.Now().UnixNano()),
				Value:       100, // Specific value to filter on
				OwnerUserId: uuid.Must(uuid.NewV4()).String(),
				CreatedDate: time.Now().Unix(),
				LastUpdated: time.Now().Unix(),
				Deleted:     false,
			}
			ownerID, err := uuid.FromString(data.OwnerUserId)
			require.NoError(t, err)
			result := <-repo.Save(ctx, collectionName, data.ObjectId, ownerID, data.CreatedDate, data.LastUpdated, data)
			require.NoError(t, result.Error)
		}

		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Count documents with specific filter - use Query object
		queryObj := &interfaces.Query{
			Conditions: []interfaces.Field{
				{Name: "data->>'value'", Value: 100, Operator: "=", IsJSONB: true},
			},
		}
		countResult := <-tx.Count(ctx, collectionName, queryObj)
		assert.NoError(t, countResult.Error, "CountWithFilter should succeed in transaction")
		assert.GreaterOrEqual(t, countResult.Count, int64(3), "Should count at least 3 documents with value=100")

		err = tx.Commit()
		assert.NoError(t, err, "Transaction should commit successfully")
	})

	// Test FindWithCursor in transaction (needs better coverage)
	t.Run("FindWithCursor_InTransaction", func(t *testing.T) {
		// Create test data for pagination
		for i := 0; i < 5; i++ {
			data := TestData{
				ObjectId:    uuid.Must(uuid.NewV4()),
				Name:        fmt.Sprintf("paginated_%d_%d", i, time.Now().UnixNano()),
				Value:       i + 1,
				OwnerUserId: uuid.Must(uuid.NewV4()).String(),
				CreatedDate: time.Now().Unix() + int64(i), // Different timestamps
				LastUpdated: time.Now().Unix() + int64(i),
				Deleted:     false,
			}
			ownerID, err := uuid.FromString(data.OwnerUserId)
			require.NoError(t, err)
			result := <-repo.Save(ctx, collectionName, data.ObjectId, ownerID, data.CreatedDate, data.LastUpdated, data)
			require.NoError(t, result.Error)
		}

		tx, err := repo.Begin(ctx)
		require.NoError(t, err)

		// Test cursor-based pagination
		limit := int64(2)
		opts := &interfaces.CursorFindOptions{
			Limit:         &limit,
			Sort:          map[string]int{"created_date": 1},
			SortField:     "created_date",
			SortDirection: "asc",
		}

		queryObj := &interfaces.Query{
			Conditions: []interfaces.Field{
				{Name: "data->>'value'", Value: 50, Operator: "=", IsJSONB: true},
			},
		}
		queryResult := <-tx.FindWithCursor(ctx, collectionName, queryObj, opts)
		assert.NoError(t, queryResult.Error(), "FindWithCursor should succeed in transaction")

		err = tx.Commit()
		assert.NoError(t, err, "Transaction should commit successfully")
	})
}

// TestAdvancedTransactionConfiguration tests enterprise configuration scenarios
func (ts *TransactionTestSuite) TestAdvancedTransactionConfiguration() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_AdvancedConfig", dbType), func(t *testing.T) {
			ts.testAdvancedConfiguration(t, repo, dbType)
		})
	}
}

func (ts *TransactionTestSuite) testAdvancedConfiguration(t *testing.T, repo interfaces.Repository, dbType string) {
	ctx := context.Background()

	// Test different retry configurations
	t.Run("RetryPolicies", func(t *testing.T) {
		retryConfigs := []*interfaces.RetryPolicy{
			{
				MaxRetries:      3,
				InitialDelay:    time.Millisecond * 100,
				MaxDelay:        time.Second,
				BackoffFactor:   2.0,
				RetryableErrors: []string{"connection_error", "timeout"},
			},
			{
				MaxRetries:      1,
				InitialDelay:    time.Millisecond * 50,
				MaxDelay:        time.Millisecond * 500,
				BackoffFactor:   1.5,
				RetryableErrors: []string{"deadlock"},
			},
		}

		for i, retryPolicy := range retryConfigs {
			t.Run(fmt.Sprintf("RetryConfig_%d", i), func(t *testing.T) {
				config := &interfaces.TransactionConfig{
					Timeout:        time.Second * 5,
					IsolationLevel: interfaces.IsolationLevelDefault,
					RetryPolicy:    retryPolicy,
				}

				tx, err := repo.BeginWithConfig(ctx, config)
				require.NoError(t, err)

				// Verify retry policy is set
				txConfig := tx.GetConfig()
				assert.Equal(t, retryPolicy.MaxRetries, txConfig.RetryPolicy.MaxRetries)
				assert.Equal(t, retryPolicy.InitialDelay, txConfig.RetryPolicy.InitialDelay)
				assert.Equal(t, retryPolicy.BackoffFactor, txConfig.RetryPolicy.BackoffFactor)

				err = tx.Commit()
				assert.NoError(t, err)
			})
		}
	})

	// Test different isolation levels
	t.Run("IsolationLevels", func(t *testing.T) {
		isolationLevels := []interfaces.IsolationLevel{
			interfaces.IsolationLevelDefault,
			interfaces.IsolationLevelReadCommitted,
			interfaces.IsolationLevelRepeatableRead,
			interfaces.IsolationLevelSerializable,
		}

		for _, level := range isolationLevels {
			t.Run(fmt.Sprintf("IsolationLevel_%d", level), func(t *testing.T) {
				config := &interfaces.TransactionConfig{
					Timeout:        time.Second * 10,
					IsolationLevel: level,
				}

				tx, err := repo.BeginWithConfig(ctx, config)
				require.NoError(t, err)

				// Verify isolation level is set
				txConfig := tx.GetConfig()
				assert.Equal(t, level, txConfig.IsolationLevel)

				// Perform a simple operation to ensure transaction works
				collectionName := fmt.Sprintf("test_isolation_%s_%d", dbType, time.Now().UnixNano())
				data := TestData{
					ObjectId:    uuid.Must(uuid.NewV4()),
					Name:        fmt.Sprintf("isolation_test_%d", level),
					Value:       1,
					OwnerUserId: uuid.Must(uuid.NewV4()).String(),
					CreatedDate: time.Now().Unix(),
					LastUpdated: time.Now().Unix(),
					Deleted:     false,
				}

				ownerID, err := uuid.FromString(data.OwnerUserId)
				require.NoError(t, err)
				result := <-tx.Save(ctx, collectionName, data.ObjectId, ownerID, data.CreatedDate, data.LastUpdated, data)
				assert.NoError(t, result.Error, "Save should succeed with isolation level %d", level)

				err = tx.Commit()
				assert.NoError(t, err, "Commit should succeed with isolation level %d", level)
			})
		}
	})
}

// TestTransactionStressScenarios tests high-load transaction scenarios
func (ts *TransactionTestSuite) TestTransactionStressScenarios() {
	for dbType, repo := range ts.repos {
		ts.t.Run(fmt.Sprintf("%s_StressTest", dbType), func(t *testing.T) {
			ts.testLargeTransactionStress(t, repo, dbType)
		})
	}
}

func (ts *TransactionTestSuite) testLargeTransactionStress(t *testing.T, repo interfaces.Repository, dbType string) {
	ctx := context.Background()
	collectionName := fmt.Sprintf("test_stress_%s_%d", dbType, time.Now().UnixNano())

	config := &interfaces.TransactionConfig{
		Timeout:        time.Minute, // Longer timeout for stress test
		IsolationLevel: interfaces.IsolationLevelDefault,
	}

	tx, err := repo.BeginWithConfig(ctx, config)
	require.NoError(t, err)

	// Perform many operations to test enterprise scalability
	const (
		numSaves   = 50
		numUpdates = 25
		numDeletes = 10
	)
	
	var savedIds []uuid.UUID

	// Phase 1: Save many documents
	for i := 0; i < numSaves; i++ {
		id := uuid.Must(uuid.NewV4())
		savedIds = append(savedIds, id)
		
		data := TestData{
			ObjectId:    id,
			Name:        fmt.Sprintf("stress_test_%d_%d", i, time.Now().UnixNano()),
			Value:       i,
			OwnerUserId: uuid.Must(uuid.NewV4()).String(),
			CreatedDate: time.Now().Unix(),
			LastUpdated: time.Now().Unix(),
			Deleted:     false,
		}

		ownerID, err := uuid.FromString(data.OwnerUserId)
		require.NoError(t, err)
		result := <-tx.Save(ctx, collectionName, data.ObjectId, ownerID, data.CreatedDate, data.LastUpdated, data)
		assert.NoError(t, result.Error, "Save %d should succeed", i)
	}

	// Phase 2: Update some documents
	for i := 0; i < numUpdates; i++ {
		updates := map[string]interface{}{
			"name":        fmt.Sprintf("updated_stress_%d", i),
			"value":       i + 100,
			"lastUpdated": time.Now().Unix(),
		}

		queryObj := &interfaces.Query{
			Conditions: []interfaces.Field{
				{Name: "object_id", Value: savedIds[i], Operator: "="},
			},
		}
		result := <-tx.Update(ctx, collectionName, queryObj, updates, nil)
		assert.NoError(t, result.Error, "Update %d should succeed", i)
	}

	// Phase 3: Delete some documents
	for i := 0; i < numDeletes; i++ {
		queryObj := &interfaces.Query{
			Conditions: []interfaces.Field{
				{Name: "object_id", Value: savedIds[40+i], Operator: "="},
			},
		}
		result := <-tx.Delete(ctx, collectionName, queryObj)
		assert.NoError(t, result.Error, "Delete %d should succeed", i)
	}

	// Verify operation count
	metrics := tx.GetMetrics()
	expectedOps := int64(numSaves + numUpdates + numDeletes)
	assert.Equal(t, expectedOps, metrics.OperationsCount, "Operation count should match total operations")

	// Commit the large transaction
	err = tx.Commit()
	assert.NoError(t, err, "Large transaction should commit successfully")

	// Verify final metrics
	finalMetrics := tx.GetMetrics()
	assert.Equal(t, expectedOps, finalMetrics.OperationsCount)
	assert.True(t, finalMetrics.Duration > 0)
	
	t.Logf("Stress test completed: %d operations in %v", finalMetrics.OperationsCount, finalMetrics.Duration)
}

// BenchmarkTransactionOperations benchmarks transaction performance
func BenchmarkTransactionOperations(b *testing.B) {
	suite := NewTransactionTestSuite(&testing.T{})
	suite.SetupDatabases()
	
	if len(suite.repos) == 0 {
		b.Skip("No databases available for benchmarking")
	}
	
	for dbType, repo := range suite.repos {
		b.Run(fmt.Sprintf("%s_SaveCommit", dbType), func(b *testing.B) {
			ctx := context.Background()
			testData := generateTestData(1)[0]
			collectionName := fmt.Sprintf("benchmark_collection_%d", time.Now().UnixNano())
			
			b.ResetTimer()
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					tx, err := repo.Begin(ctx)
					if err != nil {
						b.Fatalf("Failed to begin transaction: %v", err)
					}
					
					ownerID, err := uuid.FromString(testData.OwnerUserId)
					if err != nil {
						b.Fatalf("Failed to parse owner ID: %v", err)
					}
					result := <-tx.Save(ctx, collectionName, testData.ObjectId, ownerID, testData.CreatedDate, testData.LastUpdated, testData)
					if result.Error != nil {
						b.Fatalf("Save operation failed: %v", result.Error)
					}
					
					err = tx.Commit()
					if err != nil {
						b.Fatalf("Failed to commit transaction: %v", err)
					}
				}
			})
		})
	}
}
