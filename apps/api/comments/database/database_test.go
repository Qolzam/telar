package database

import (
	"context"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Database configuration tests
func TestDatabaseConfiguration_PostgreSQL(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}

	// Get the shared connection pool
	suite := testutil.Setup(t)
	
	// Create isolated test environment with transaction
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()
	
	// Test with valid configuration
	t.Run("ValidConfiguration", func(t *testing.T) {
		
		base, err := platform.NewBaseService(ctx, iso.Config)
		
		assert.NoError(t, err)
		assert.NotNil(t, base)
		assert.NotNil(t, base.Repository)
	})
	
	// Test with invalid configuration
	t.Run("InvalidConfiguration", func(t *testing.T) {
		platformCfg := &platformconfig.Config{
			Database: platformconfig.DatabaseConfig{
				Type: dbi.DatabaseTypePostgreSQL,
				Postgres: platformconfig.PostgreSQLConfig{
					Host:     "invalid-host-12345",
					Port:     5432,
					Username: "invalid",
					Password: "invalid",
					Database: "test",
					SSLMode:  "disable",
				},
			},
		}
		_, err := platform.NewBaseService(ctx, platformCfg)
		
		// Should fail to connect to invalid host
		assert.Error(t, err)
	})
}

// Database operations tests for Comments - MongoDB tests removed, PostgreSQL only

func TestDatabaseOperations_PostgreSQL_Comments(t *testing.T) {
	t.Parallel()
	
	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}

	ctx := context.Background()
	baseService := platform.NewBaseServiceWithRepo(iso.Repo, iso.Config.ToPlatformConfig(dbi.DatabaseTypePostgreSQL))
	
	const testCollection = "comments_database_test"
	
	// Test basic CRUD operations for comments in PostgreSQL
	t.Run("BasicCRUDOperations", func(t *testing.T) {
		objectID := uuid.Must(uuid.NewV4())
		postID := uuid.Must(uuid.NewV4())
		userID := uuid.Must(uuid.NewV4())
		now := time.Now().Unix()
		
		testComment := map[string]interface{}{
			"objectId":     objectID,
			"postId":       postID,
			"ownerUserId":  userID,
			"text":         "This is a test comment",
			"score":        0,
			"createdDate":  now,
			"lastUpdated":  now,
		}
		
		// Create comment - using new Save signature
		saveResult := <-baseService.Repository.Save(ctx, testCollection, objectID, userID, now, now, testComment)
		assert.NoError(t, saveResult.Error)
		
		// Read comment - using Query object
		query := &dbi.Query{
			Conditions: []dbi.Field{
				{Name: "object_id", Value: objectID, Operator: "="},
			},
		}
		findResult := <-baseService.Repository.FindOne(ctx, testCollection, query)
		assert.NoError(t, findResult.Error())
		assert.False(t, findResult.NoResult())
		
		// Update comment - using UpdateFields
		updateQuery := &dbi.Query{
			Conditions: []dbi.Field{
				{Name: "object_id", Value: objectID, Operator: "="},
			},
		}
		updates := map[string]interface{}{
			"text":        "Updated test comment",
			"lastUpdated": time.Now().Unix(),
		}
		updateResult := <-baseService.Repository.UpdateFields(ctx, testCollection, updateQuery, updates)
		assert.NoError(t, updateResult.Error)
		
		// Verify update
		findUpdatedResult := <-baseService.Repository.FindOne(ctx, testCollection, query)
		assert.NoError(t, findUpdatedResult.Error())
		
		// Delete comment - using Query object
		deleteQuery := &dbi.Query{
			Conditions: []dbi.Field{
				{Name: "object_id", Value: objectID, Operator: "="},
			},
		}
		deleteResult := <-baseService.Repository.Delete(ctx, testCollection, deleteQuery)
		assert.NoError(t, deleteResult.Error)
	})
}

// Connection pooling and concurrent access tests for comments - MongoDB tests removed
// All MongoDB tests have been removed - PostgreSQL only

func setupPostgreSQLTestService(t *testing.T, ctx context.Context) *platform.BaseService {
	suite := testutil.Setup(t)
	iso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, suite.Config())
	if iso.Repo == nil {
		t.Skip("PostgreSQL not available, skipping test")
	}
	
	baseService := platform.NewBaseServiceWithRepo(iso.Repo, iso.Config.ToPlatformConfig(dbi.DatabaseTypePostgreSQL))
	return baseService
}

// All MongoDB tests removed - PostgreSQL only
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("set RUN_DB_TESTS=1 to run database tests")
	}
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}
	
	ctx := context.Background()
	base := setupMongoDBTestService(t, ctx)
	
	const testCollection = "comments_performance_test"
	
	t.Run("BulkCommentOperations", func(t *testing.T) {
		const numComments = 100
		var commentIDs []uuid.UUID
		postID := uuid.Must(uuid.NewV4())
		userID := uuid.Must(uuid.NewV4())
		
		start := time.Now()
		
		// Bulk insert comments
		for i := 0; i < numComments; i++ {
			commentID := uuid.Must(uuid.NewV4())
			commentIDs = append(commentIDs, commentID)
			
			testComment := map[string]interface{}{
				"objectId":     commentID,
				"postId":       postID,
				"ownerUserId":  userID,
				"text":         "Bulk comment test",
				"score":        0,
				"createdDate":  time.Now().Unix(),
				"index":        i,
			}
			
			saveResult := <-base.Repository.Save(ctx, testCollection, testComment)
			if saveResult.Error != nil {
				t.Fatalf("bulk comment insert error at index %d: %v", i, saveResult.Error)
			}
		}
		
		insertDuration := time.Since(start)
		t.Logf("Inserted %d comments in %v (%.2f comments/sec)", numComments, insertDuration, float64(numComments)/insertDuration.Seconds())
		
		// Bulk read comments
		start = time.Now()
		for _, commentID := range commentIDs {
			findResult := <-base.Repository.FindOne(ctx, testCollection, map[string]interface{}{"objectId": commentID})
			if findResult.Error() != nil {
				t.Fatalf("bulk comment read error: %v", findResult.Error())
			}
		}
		readDuration := time.Since(start)
		t.Logf("Read %d comments in %v (%.2f comments/sec)", numComments, readDuration, float64(numComments)/readDuration.Seconds())
		
		// Test comment query performance
		start = time.Now()
		limit := int64(100)
		findByPostResult := <-base.Repository.Find(ctx, testCollection,
			map[string]interface{}{"postId": postID},
			&dbi.FindOptions{
				Limit: &limit,
				Sort:  map[string]int{"createdDate": -1},
			})
		queryDuration := time.Since(start)
		assert.NoError(t, findByPostResult.Error())
		t.Logf("Queried comments by postId in %v", queryDuration)
		
		// Bulk delete comments (single filter for determinism) and verify cleanup
		start = time.Now()
		deleteResult := <-base.Repository.Delete(ctx, testCollection, map[string]interface{}{})
		require.NoError(t, deleteResult.Error)
		deleteDuration := time.Since(start)
		t.Logf("Deleted %d comments in %v (%.2f comments/sec)", numComments, deleteDuration, float64(numComments)/deleteDuration.Seconds())

		// Verify cleanup
		countAfter := <-base.Repository.Count(ctx, testCollection, map[string]interface{}{})
		require.NoError(t, countAfter.Error)
		require.Equal(t, int64(0), countAfter.Count)
	})
}

// BenchmarkBulkOperations_MongoDB_Comments - removed (MongoDB tests removed)
// Error recovery tests for comments - MongoDB tests removed
// Helper functions for test setup
// setupMongoDBTestService - removed (MongoDB tests removed)
// Duplicate setupPostgreSQLTestService - removed (keeping first one)
// Index performance tests for comments - MongoDB tests removed
