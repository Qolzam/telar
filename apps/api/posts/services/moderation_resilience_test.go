// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package services

import (
	"context"
	"errors"
	"testing"
	"time"

	uuid "github.com/gofrs/uuid"
	commentMocks "github.com/qolzam/telar/apps/api/comments/services/mocks"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/packages/clients/aiengine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// TestModerationResilience_AIEngineUnavailable tests the fail-safe behavior
// when AI Engine is unavailable (503 Service Unavailable).
// This test verifies that toxic content is NOT published when AI analysis fails.
func TestModerationResilience_AIEngineUnavailable(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockPostRepository{}
	mockAIEngine := &MockAIEngineClient{}
	mockCommentRepo := &commentMocks.MockCommentRepository{}

	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		Cache: platformconfig.CacheConfig{
			Enabled: false,
		},
	}

	svc := &postService{
		repo:           mockRepo,
		commentRepo:    mockCommentRepo,
		config:         cfg,
		aiEngineClient: mockAIEngine,
	}

	user := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		DisplayName: "Test User",
		Avatar:      "avatar.jpg",
		SocialName:  "testuser",
	}

	// Create a post with toxic content
	req := &models.CreatePostRequest{
		PostTypeId: 1,
		Body:       "Fuck you, this is toxic hate speech content that should be flagged!",
	}

	// Mock AI Engine to return Service Unavailable (503) - simulates DDoS or maintenance
	retryableErr := &aiengine.ClientError{
		Type:        aiengine.ErrServiceUnavailable,
		StatusCode:  503,
		Message:     "Service temporarily unavailable",
		Retryable:   true,
		OriginalErr: errors.New("connection refused"),
	}

	// Expect 4 calls (initial + 3 retries)
	mockAIEngine.On("AnalyzeContent", mock.Anything, mock.Anything).Return(nil, retryableErr).Times(4)

	// Mock Create to capture the post that's initially created
	var createdPost *models.Post
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Run(func(args mock.Arguments) {
		createdPost = args.Get(1).(*models.Post)
		// Verify post is initially created as 'published' (optimistic approach)
		assert.Equal(t, "published", createdPost.Status, "Post should be created with 'published' status initially")
	}).Return(nil)

	// Mock FindByID to return the created post
	// Called twice: once for race condition check, once in UpdateFields
	// We need to use a closure to capture createdPost after it's set
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(func() *models.Post {
		// Return a copy to avoid race conditions
		if createdPost != nil {
			postCopy := *createdPost
			return &postCopy
		}
		// Fallback if called before Create
		return &models.Post{
			Body:   req.Body,
			Status: "published",
		}
	}(), nil).Twice() // Called twice: race check + UpdateFields
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(post *models.Post) bool {
		// Verify the post status is updated to 'analysis_failed' (fail-safe)
		return post.Status == "analysis_failed"
	})).Return(nil).Once()

	// Create the post
	post, err := svc.CreatePost(ctx, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, "published", post.Status, "Post should be initially published (optimistic)")

	// Wait for async goroutine to complete (including retries with exponential backoff)
	// Retries: 1s + 2s + 4s = ~7 seconds, add buffer for safety
	time.Sleep(10 * time.Second)

	// Verify AI Engine was called multiple times (retries)
	mockAIEngine.AssertExpectations(t)

	// CRITICAL ASSERTION: Verify that UpdateFields was called to change status to 'analysis_failed'
	// This proves the fail-safe mechanism works
	mockRepo.AssertExpectations(t)
}

// TestModerationResilience_AIEngineTimeout tests the fail-safe behavior
// when AI Engine times out (network timeout error).
func TestModerationResilience_AIEngineTimeout(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockPostRepository{}
	mockAIEngine := &MockAIEngineClient{}
	mockCommentRepo := &commentMocks.MockCommentRepository{}

	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		Cache: platformconfig.CacheConfig{
			Enabled: false,
		},
	}

	svc := &postService{
		repo:           mockRepo,
		commentRepo:    mockCommentRepo,
		config:         cfg,
		aiEngineClient: mockAIEngine,
	}

	user := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		DisplayName: "Test User",
		Avatar:      "avatar.jpg",
		SocialName:  "testuser",
	}

	req := &models.CreatePostRequest{
		PostTypeId: 1,
		Body:       "This content should be reviewed when AI times out",
	}

	// Mock AI Engine to return timeout error (retryable)
	timeoutErr := &aiengine.ClientError{
		Type:        aiengine.ErrTimeout,
		StatusCode:  0,
		Message:     "Request timeout",
		Retryable:   true,
		OriginalErr: errors.New("context deadline exceeded"),
	}

	// Expect 4 calls (initial + 3 retries)
	mockAIEngine.On("AnalyzeContent", mock.Anything, mock.Anything).Return(nil, timeoutErr).Times(4)

	var createdPost *models.Post
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Run(func(args mock.Arguments) {
		createdPost = args.Get(1).(*models.Post)
	}).Return(nil)

	// Mock FindByID and Update for the fail-safe status update
	// Called twice: once for race condition check, once in UpdateFields
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(func() *models.Post {
		if createdPost != nil {
			postCopy := *createdPost
			return &postCopy
		}
		return &models.Post{
			Body:   req.Body,
			Status: "published",
		}
	}(), nil).Twice() // Called twice: race check + UpdateFields
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(post *models.Post) bool {
		// Verify the post status is updated to 'analysis_failed' (fail-safe)
		return post.Status == "analysis_failed"
	})).Return(nil).Once()

	post, err := svc.CreatePost(ctx, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, post)

	// Wait for async goroutine to complete
	time.Sleep(10 * time.Second)

	// Verify fail-safe mechanism was triggered
	mockAIEngine.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

