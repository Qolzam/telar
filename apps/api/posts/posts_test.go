package posts_test

import (
	"context"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platform "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/testutil"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/services"
)

func TestPostsOperations(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("RUN_DB_TESTS not set, skipping database tests")
	}

	testCases := []struct {
		name   string
		dbType string
	}{
		{name: "PostgreSQL", dbType: dbi.DatabaseTypePostgreSQL},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			suite := testutil.Setup(t)

			baseConfig := suite.Config()
			iso := testutil.NewIsolatedTest(t, tc.dbType, baseConfig)
			if iso.Repo == nil {
				t.Skip("Database not available")
			}

			ctx := context.Background()
			base, err := platform.NewBaseService(ctx, iso.Config)
			require.NoError(t, err)

			postService := services.NewPostService(base, iso.Config)
			testData := createTestData(t, postService, base)
			defer cleanupTestData(t, postService, testData)

			t.Run("ArrayOperations", func(t *testing.T) {
				testArrayOperations(t, postService)
			})

			t.Run("SearchOperations", func(t *testing.T) {
				testSearchOperations(t, postService)
			})

			t.Run("DateRangeOperations", func(t *testing.T) {
				testDateRangeOperations(t, postService)
			})

			t.Run("ComplexFiltering", func(t *testing.T) {
				testComplexFiltering(t, postService)
			})

			t.Run("PaginationOperations", func(t *testing.T) {
				testPaginationOperations(t, postService)
			})

			t.Run("UpdateOperations", func(t *testing.T) {
				testUpdateOperations(t, postService, testData)
			})

			t.Run("ErrorHandling", func(t *testing.T) {
				testErrorHandling(t, postService, testData)
			})
		})
	}
}

func TestDatabaseCompatibility(t *testing.T) {
	if !testutil.ShouldRunDatabaseTests() {
		t.Skip("RUN_DB_TESTS not set, skipping compatibility tests")
	}

	suite := testutil.Setup(t)

	baseConfig := suite.Config()
	postgresIso := testutil.NewIsolatedTest(t, dbi.DatabaseTypePostgreSQL, baseConfig)

	if postgresIso.Repo == nil {
		t.Skip("PostgreSQL not available")
	}

	ctx := context.Background()

	postgresCfg := postgresIso.Config
	postgresBase, err := platform.NewBaseService(ctx, postgresCfg)
	require.NoError(t, err)
	postgresService := services.NewPostService(postgresBase, postgresCfg)

	postgresData := createTestData(t, postgresService, postgresBase)
	defer cleanupTestData(t, postgresService, postgresData)

	t.Run("BasicQueryPosts", func(t *testing.T) {
		postgresResult, err := postgresService.QueryPosts(ctx, &models.PostQueryFilter{Limit: 10, Page: 1})
		require.NoError(t, err)

		// PostgreSQL should return results
		assert.Greater(t, len(postgresResult.Posts), 0, "PostgreSQL should return results")
	})

	t.Run("TagsFiltering", func(t *testing.T) {
		filter := &models.PostQueryFilter{
			Tags:  []string{"golang", "database"},
			Limit: 10,
			Page:  1,
		}

		postgresResult, err := postgresService.QueryPosts(ctx, filter)
		require.NoError(t, err)

		// PostgreSQL should return results
		assert.Greater(t, len(postgresResult.Posts), 0, "PostgreSQL should return results")
	})

	t.Run("SearchFunctionality", func(t *testing.T) {
		searchQuery := "golang"
		filter := &models.PostQueryFilter{Limit: 10, Page: 1}

		postgresResult, err := postgresService.SearchPosts(ctx, searchQuery, filter)
		require.NoError(t, err)

		// PostgreSQL should return results
		assert.Greater(t, len(postgresResult.Posts), 0, "PostgreSQL should return results")
	})
}

func testArrayOperations(t *testing.T, postService services.PostService) {
	t.Run("InOperator_SingleTag", func(t *testing.T) {
		filter := &models.PostQueryFilter{
			Tags:  []string{"golang"},
			Limit: 10,
			Page:  1,
		}

		result, err := postService.QueryPosts(context.Background(), filter)
		require.NoError(t, err)
		assert.NotNil(t, result)

		for _, post := range result.Posts {
			hasMatchingTag := false
			for _, postTag := range post.Tags {
				for _, filterTag := range filter.Tags {
					if postTag == filterTag {
						hasMatchingTag = true
						break
					}
				}
			}
			assert.True(t, hasMatchingTag, "Post should have matching tag")
		}
	})

	t.Run("InOperator_MultipleTags", func(t *testing.T) {
		filter := &models.PostQueryFilter{
			Tags:  []string{"golang", "database"},
			Limit: 10,
			Page:  1,
		}

		result, err := postService.QueryPosts(context.Background(), filter)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("AllOperator_SearchWithCursor", func(t *testing.T) {
		searchQuery := "test"
		filter := &models.PostQueryFilter{
			Tags:  []string{"golang", "test"},
			Limit: 10,
			Page:  1,
		}

		result, err := postService.SearchPostsWithCursor(context.Background(), searchQuery, filter)
		require.NoError(t, err)
		assert.NotNil(t, result)

		for _, post := range result.Posts {
			for _, requiredTag := range filter.Tags {
				hasRequiredTag := false
				for _, postTag := range post.Tags {
					if postTag == requiredTag {
						hasRequiredTag = true
						break
					}
				}
				assert.True(t, hasRequiredTag, "Post should have required tag '%s'", requiredTag)
			}
		}
	})
}

func testSearchOperations(t *testing.T, postService services.PostService) {
	t.Run("SearchPosts_CaseInsensitive", func(t *testing.T) {
		searchQuery := "golang"
		filter := &models.PostQueryFilter{Limit: 10, Page: 1}

		result, err := postService.SearchPosts(context.Background(), searchQuery, filter)
		require.NoError(t, err)
		assert.NotNil(t, result)

		for _, post := range result.Posts {
			containsSearchTerm := false
			if containsIgnoreCase(post.Body, searchQuery) ||
				containsIgnoreCase(post.OwnerDisplayName, searchQuery) {
				containsSearchTerm = true
			}
			for _, tag := range post.Tags {
				if containsIgnoreCase(tag, searchQuery) {
					containsSearchTerm = true
					break
				}
			}
			assert.True(t, containsSearchTerm, "Post should contain search term")
		}
	})

	t.Run("SearchPosts_PartialMatch", func(t *testing.T) {
		searchQuery := "database"
		filter := &models.PostQueryFilter{Limit: 10, Page: 1}

		result, err := postService.SearchPosts(context.Background(), searchQuery, filter)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("SearchPostsWithCursor_ComplexQuery", func(t *testing.T) {
		searchQuery := "test"
		filter := &models.PostQueryFilter{
			Tags:  []string{"golang", "test"},
			Limit: 5,
			Page:  1,
		}

		result, err := postService.SearchPostsWithCursor(context.Background(), searchQuery, filter)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func testDateRangeOperations(t *testing.T, postService services.PostService) {
	t.Run("CreatedAfter_Filter", func(t *testing.T) {
		createdAfter := time.Now().Add(-24 * time.Hour)
		filter := &models.PostQueryFilter{
			CreatedAfter: &createdAfter,
			Limit:        10,
			Page:         1,
		}

		result, err := postService.QueryPosts(context.Background(), filter)
		require.NoError(t, err)
		assert.NotNil(t, result)

		for _, post := range result.Posts {
			assert.True(t, post.CreatedDate >= filter.CreatedAfter.Unix(),
				"Post should be created after filter time")
		}
	})

	t.Run("DateRange_WithTags", func(t *testing.T) {
		createdAfter := time.Now().Add(-48 * time.Hour)
		filter := &models.PostQueryFilter{
			Tags:         []string{"golang"},
			CreatedAfter: &createdAfter,
			Limit:        10,
			Page:         1,
		}

		result, err := postService.QueryPosts(context.Background(), filter)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func testComplexFiltering(t *testing.T, postService services.PostService) {
	t.Run("MultipleFilters_Combined", func(t *testing.T) {
		filter := &models.PostQueryFilter{
			Tags:  []string{"golang", "database"},
			Limit: 5,
			Page:  1,
		}

		result, err := postService.QueryPosts(context.Background(), filter)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})

	t.Run("SearchWithComplexFilters", func(t *testing.T) {
		searchQuery := "database"
		createdAfter := time.Now().Add(-24 * time.Hour)
		filter := &models.PostQueryFilter{
			Tags:         []string{"golang", "database"},
			CreatedAfter: &createdAfter,
			Limit:        5,
			Page:         1,
		}

		result, err := postService.SearchPostsWithCursor(context.Background(), searchQuery, filter)
		require.NoError(t, err)
		assert.NotNil(t, result)

		for _, post := range result.Posts {
			for _, requiredTag := range filter.Tags {
				hasRequiredTag := false
				for _, postTag := range post.Tags {
					if postTag == requiredTag {
						hasRequiredTag = true
						break
					}
				}
				assert.True(t, hasRequiredTag, "Post should have required tag '%s'", requiredTag)
			}

			containsSearchTerm := false
			if containsIgnoreCase(post.Body, searchQuery) ||
				containsIgnoreCase(post.OwnerDisplayName, searchQuery) {
				containsSearchTerm = true
			}
			for _, tag := range post.Tags {
				if containsIgnoreCase(tag, searchQuery) {
					containsSearchTerm = true
					break
				}
			}
			assert.True(t, containsSearchTerm, "Post should contain search term")

			assert.True(t, post.CreatedDate >= filter.CreatedAfter.Unix(),
				"Post should be created after filter time")
		}
	})
}

func testPaginationOperations(t *testing.T, postService services.PostService) {
	t.Run("BasicPagination", func(t *testing.T) {
		filter := &models.PostQueryFilter{
			Limit: 5,
			Page:  1,
		}

		result, err := postService.QueryPosts(context.Background(), filter)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.LessOrEqual(t, len(result.Posts), filter.Limit, "Should not exceed limit")
	})

	t.Run("CursorPagination", func(t *testing.T) {
		filter1 := &models.PostQueryFilter{
			Limit: 2,
			Page:  1,
		}

		result1, err := postService.QueryPosts(context.Background(), filter1)
		require.NoError(t, err)
		assert.NotNil(t, result1)

		filter2 := &models.PostQueryFilter{
			Limit: 2,
			Page:  2,
		}

		result2, err := postService.QueryPosts(context.Background(), filter2)
		require.NoError(t, err)
		assert.NotNil(t, result2)

		page1IDs := make(map[string]bool)
		for _, post := range result1.Posts {
			page1IDs[post.ObjectId] = true
		}

		for _, post := range result2.Posts {
			if page1IDs[post.ObjectId] {
				require.Falsef(t, true, "duplicate post %s detected. page1=%+v page2=%+v", post.ObjectId, result1.Posts, result2.Posts)
			}
			assert.False(t, page1IDs[post.ObjectId], "Post should not appear on both pages")
		}
	})
}

func testUpdateOperations(t *testing.T, postService services.PostService, testData *TestData) {
	if len(testData.PostIDs) == 0 {
		t.Skip("No test posts available")
		return
	}

	postID := testData.PostIDs[0]

	t.Run("UpdatePost_Basic", func(t *testing.T) {
		updateReq := &models.UpdatePostRequest{
			ObjectId: &postID,
			Body:     stringPtr("Updated post content"),
			Tags:     &[]string{"updated", "test"},
		}

		user := &types.UserContext{
			UserID: testData.UserID,
		}

		err := postService.UpdatePost(context.Background(), postID, updateReq, user)
		require.NoError(t, err, "UpdatePost should not fail")
	})

	t.Run("IncrementScore", func(t *testing.T) {
		post, err := postService.GetPost(context.Background(), postID)
		require.NoError(t, err)
		originalScore := post.Score

		user := &types.UserContext{
			UserID: testData.UserID,
		}

		err = postService.IncrementScore(context.Background(), postID, 10, user)
		require.NoError(t, err, "IncrementScore should not fail")

		updatedPost, err := postService.GetPost(context.Background(), postID)
		require.NoError(t, err)
		assert.Equal(t, originalScore+10, updatedPost.Score, "Score should be incremented by 10")
	})
}

func testErrorHandling(t *testing.T, postService services.PostService, testData *TestData) {
	t.Run("GetPost_NonExistent", func(t *testing.T) {
		nonExistentID := uuid.Must(uuid.NewV4())
		_, err := postService.GetPost(context.Background(), nonExistentID)
		require.Error(t, err, "GetPost should fail for non-existent post")
	})

	t.Run("UpdatePost_NonExistent", func(t *testing.T) {
		nonExistentID := uuid.Must(uuid.NewV4())
		updateReq := &models.UpdatePostRequest{
			ObjectId: &nonExistentID,
			Body:     stringPtr("Updated content"),
		}

		user := &types.UserContext{
			UserID: testData.UserID,
		}

		err := postService.UpdatePost(context.Background(), nonExistentID, updateReq, user)
		require.Error(t, err, "UpdatePost should fail for non-existent post")
	})

	t.Run("InvalidPagination", func(t *testing.T) {
		filter := &models.PostQueryFilter{
			Limit: 0,
			Page:  1,
		}

		result, err := postService.QueryPosts(context.Background(), filter)
		require.NoError(t, err, "QueryPosts should handle invalid limit gracefully")
		assert.NotNil(t, result, "Result should not be nil")
	})

	t.Run("EmptySearchQuery", func(t *testing.T) {
		filter := &models.PostQueryFilter{
			Limit: 10,
			Page:  1,
		}

		result, err := postService.SearchPosts(context.Background(), "", filter)
		require.NoError(t, err, "SearchPosts should handle empty query gracefully")
		assert.NotNil(t, result, "Result should not be nil")
	})
}

type TestData struct {
	UserID  uuid.UUID
	PostIDs []uuid.UUID
}

func createTestData(t *testing.T, postService services.PostService, base *platform.BaseService) *TestData {
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	now := time.Now().Unix()

	testPosts := []struct {
		body string
		tags []string
	}{
		{
			body: "This is a golang database tutorial",
			tags: []string{"golang", "database", "tutorial"},
		},
		{
			body: "Testing PostgreSQL with Go",
			tags: []string{"golang", "test", "postgresql"},
		},
		{
			body: "PostgreSQL integration patterns",
			tags: []string{"database", "postgresql", "patterns"},
		},
		{
			body: "Go microservices architecture",
			tags: []string{"golang", "microservices", "architecture"},
		},
		{
			body: "Database performance optimization",
			tags: []string{"database", "performance", "optimization"},
		},
	}

	var postIDs []uuid.UUID
	for i, testPost := range testPosts {
		postID := uuid.Must(uuid.NewV4())
		post := &models.Post{
			ObjectId:         postID,
			PostTypeId:       1,
			Score:            0,
			ViewCount:        0,
			CommentCounter:   0,
			Body:             testPost.body,
			OwnerUserId:      userID,
			OwnerDisplayName: "Test User",
			OwnerAvatar:      "",
			Tags:             testPost.tags,
			Deleted:          false,
			DeletedDate:      0,
			CreatedDate:      now - int64(i*3600),
			LastUpdated:      now - int64(i*3600),
		}

		result := <-base.Repository.Save(ctx, "post", post.ObjectId, post.OwnerUserId, post.CreatedDate, post.LastUpdated, post)
		require.NoError(t, result.Error, "Failed to save test post %d", i)
		postIDs = append(postIDs, postID)
	}

	return &TestData{
		UserID:  userID,
		PostIDs: postIDs,
	}
}

func cleanupTestData(t *testing.T, postService services.PostService, testData *TestData) {
	for _, postID := range testData.PostIDs {
		t.Logf("Would cleanup post: %s", postID.String())
	}
}

func stringPtr(s string) *string {
	return &s
}

func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}

	for i := 0; i <= len(s)-len(substr); i++ {
		match := true
		for j := 0; j < len(substr); j++ {
			if toLower(s[i+j]) != toLower(substr[j]) {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

func toLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + ('a' - 'A')
	}
	return c
}
