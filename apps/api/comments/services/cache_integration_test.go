package services

import (
	"context"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qolzam/telar/apps/api/comments/models"
)

func TestCommentCacheKeyGeneration(t *testing.T) {
	RunCacheTest(t, "KeyGeneration", func(helper *CacheTestHelper) {
		cacheService := helper.GetCacheService()

		filter := &models.CommentQueryFilter{
			Limit: 10,
			Page:  1,
		}

		key1 := cacheService.GenerateHashKey("query", map[string]interface{}{
			"operation": "query",
			"limit":     filter.Limit,
			"page":      filter.Page,
		})

		key2 := cacheService.GenerateHashKey("query", map[string]interface{}{
			"operation": "query",
			"limit":     filter.Limit,
			"page":      filter.Page,
		})

		assert.Equal(t, key1, key2, "identical filters should produce identical cache keys")

		key3 := cacheService.GenerateHashKey("query", map[string]interface{}{
			"operation": "query",
			"limit":     20,
			"page":      filter.Page,
		})
		assert.NotEqual(t, key1, key3, "different filters should produce different cache keys")
	})
}

func TestCommentCacheRoundTrip(t *testing.T) {
	RunCacheTest(t, "RoundTrip", func(helper *CacheTestHelper) {
		cacheService := helper.GetCacheService()
		ctx := context.Background()

		response := &models.CommentsListResponse{
			Comments: []models.CommentResponse{
				{
					ObjectId:         uuid.Must(uuid.NewV4()).String(),
					OwnerUserId:      uuid.Must(uuid.NewV4()).String(),
					OwnerDisplayName: "Tester",
					Text:             "A cached comment",
					CreatedDate:      time.Now().Unix(),
				},
			},
			Page:  1,
			Limit: 10,
		}

		key := "comments_query_key"
		err := cacheService.CacheData(ctx, key, response, time.Minute)
		require.NoError(t, err)

		var fetched models.CommentsListResponse
		err = cacheService.GetCached(ctx, key, &fetched)
		require.NoError(t, err)

		require.Len(t, fetched.Comments, 1)
		assert.Equal(t, response.Comments[0].ObjectId, fetched.Comments[0].ObjectId)
	})
}

