package services

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/qolzam/telar/apps/api/comments/models"
)

func TestCommentCachePerformance(t *testing.T) {
	RunCacheTest(t, "Performance", func(helper *CacheTestHelper) {
		cacheService := helper.GetCacheService()
		ctx := context.Background()

		payload := &models.CommentsListResponse{
			Comments: []models.CommentResponse{},
			Page:     1,
			Limit:    10,
		}

		key := "comments_performance_suite"

		startCold := time.Now()
		err := cacheService.CacheData(ctx, key, payload, time.Minute)
		require.NoError(t, err)
		coldDuration := time.Since(startCold)

		total := time.Duration(0)
		for i := 0; i < 20; i++ {
			start := time.Now()
			var out models.CommentsListResponse
			err := cacheService.GetCached(ctx, key, &out)
			require.NoError(t, err)
			total += time.Since(start)
		}

		avg := total / 20
		assert.True(t, avg < coldDuration, "expected cache hit latency to be less than initial write")
	})
}

