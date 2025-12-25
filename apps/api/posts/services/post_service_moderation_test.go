package services

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"errors"

	commentMocks "github.com/qolzam/telar/apps/api/comments/services/mocks"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/packages/clients/aiengine"
)

// TestCreatePost_AsyncModeration_FlaggedContent tests that posts flagged by AI are moved to moderation queue
func TestCreatePost_AsyncModeration_FlaggedContent(t *testing.T) {
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

	// Create test user
	user := &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		DisplayName: "Test User",
		Avatar:      "avatar.jpg",
		SocialName:  "testuser",
	}

	// Create test post request
	req := &models.CreatePostRequest{
		PostTypeId: 1,
		Body:        "This is toxic hate speech content that should be flagged",
	}

	// Mock AI Engine to return flagged result (must be set BEFORE CreatePost triggers async goroutine)
	flaggedResult := &aiengine.AnalysisResult{
		IsFlagged:       true,
		FlagReason:      "Contains hate speech",
		Scores:          map[string]float64{"toxicity": 0.85, "sexual": 0.1, "violence": 0.2, "spam": 0.05, "misinformation": 0.1},
		Confidence:      0.92,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		SuggestedAction: "review_needed",
	}
	mockAIEngine.On("AnalyzeContent", mock.Anything, mock.MatchedBy(func(req aiengine.AnalysisRequest) bool {
		return req.Content == "This is toxic hate speech content that should be flagged"
	})).Return(flaggedResult, nil)

	// Mock repository Create call
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil).Run(func(args mock.Arguments) {
		post := args.Get(1).(*models.Post)
		assert.Equal(t, "published", post.Status, "Post should be created with 'published' status initially")
	})

	// Mock repository FindByID (called by UpdateFields to load post) - will be called by async goroutine
	// Return a post that will be updated - we'll verify the Update call instead
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&models.Post{
		Body:   "This is toxic hate speech content that should be flagged",
		Status: "published",
	}, nil)

	// Mock repository Update (called by UpdateFields) - will be called by async goroutine
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *models.Post) bool {
		return p.Status == "needs_moderation" && p.ModerationDetails != nil
	})).Return(nil)

	// Create the post (this triggers async goroutine)
	post, err := svc.CreatePost(ctx, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, post)
	assert.Equal(t, "published", post.Status, "Post should be created with 'published' status")

	// Wait for async goroutine to complete
	time.Sleep(500 * time.Millisecond)

	// Verify AI Engine was called
	mockAIEngine.AssertExpectations(t)

	// Verify repository Update was called with correct status
	mockRepo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(p *models.Post) bool {
		return p.Status == "needs_moderation"
	}))
}

// TestCreatePost_AsyncModeration_SafeContent tests that safe content remains published
func TestCreatePost_AsyncModeration_SafeContent(t *testing.T) {
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
		Body:        "This is a friendly post about technology",
	}

	// Mock AI Engine to return safe result (must be set BEFORE CreatePost)
	safeResult := &aiengine.AnalysisResult{
		IsFlagged:       false,
		FlagReason:      "",
		Scores:          map[string]float64{"toxicity": 0.05, "sexual": 0.02, "violence": 0.01, "spam": 0.03, "misinformation": 0.02},
		Confidence:      0.95,
		Timestamp:       time.Now().UTC().Format(time.RFC3339),
		SuggestedAction: "approve",
	}
	mockAIEngine.On("AnalyzeContent", mock.Anything, mock.Anything).Return(safeResult, nil)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	// Mock repository Create call
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	// Mock repository FindByID and Update for storing moderation details
	mockRepo.On("FindByID", mock.Anything, mock.AnythingOfType("uuid.UUID")).Return(&models.Post{
		Body:   "This is a friendly post about technology",
		Status: "published",
	}, nil)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *models.Post) bool {
		// Should store moderation details but keep status as published
		return p.Status == "published" && p.ModerationDetails != nil
	})).Return(nil)

	post, err := svc.CreatePost(ctx, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, post)

	// Wait for async goroutine to complete
	time.Sleep(500 * time.Millisecond)

	// Verify AI Engine was called
	mockAIEngine.AssertExpectations(t)

	// Verify repository Update was called to store moderation details
	mockRepo.AssertCalled(t, "Update", mock.Anything, mock.MatchedBy(func(p *models.Post) bool {
		return p.ModerationDetails != nil
	}))
}

// TestCreatePost_AsyncModeration_AIEngineError tests graceful handling when AI Engine fails
func TestCreatePost_AsyncModeration_AIEngineError(t *testing.T) {
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
		Body:        "Test content",
	}

	// Mock AI Engine to return a retryable error (Service Unavailable) - should trigger retries
	retryableErr := &aiengine.ClientError{
		Type:        aiengine.ErrServiceUnavailable,
		StatusCode:  503,
		Message:     "Service temporarily unavailable",
		Retryable:   true,
		OriginalErr: errors.New("connection refused"),
	}
	// Expect 4 calls (initial + 3 retries)
	mockAIEngine.On("AnalyzeContent", mock.Anything, mock.Anything).Return(nil, retryableErr).Times(4)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	post, err := svc.CreatePost(ctx, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, post)

	// Wait for async goroutine to complete (including retries with exponential backoff)
	// Retries: 1s + 2s + 4s = ~7 seconds, add buffer for safety
	time.Sleep(10 * time.Second)

	// Verify AI Engine was called multiple times (retries)
	mockAIEngine.AssertExpectations(t)

	// Verify repository Update was NOT called (error should be logged but not crash)
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

// TestCreatePost_AsyncModeration_NonRetryableError tests that non-retryable errors fail immediately
func TestCreatePost_AsyncModeration_NonRetryableError(t *testing.T) {
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
		Body:        "Test content",
	}

	// Mock AI Engine to return a non-retryable error (Unauthorized) - should NOT retry
	nonRetryableErr := &aiengine.ClientError{
		Type:        aiengine.ErrUnauthorized,
		StatusCode:  401,
		Message:     "Invalid API key",
		Retryable:   false,
		OriginalErr: errors.New("unauthorized"),
	}
	mockAIEngine.On("AnalyzeContent", mock.Anything, mock.Anything).Return(nil, nonRetryableErr).Once() // Only once, no retries

	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	post, err := svc.CreatePost(ctx, req, user)
	assert.NoError(t, err)
	assert.NotNil(t, post)

	// Wait for async goroutine (should fail immediately, no retries)
	time.Sleep(200 * time.Millisecond)

	// Verify AI Engine was called only once (no retries for non-retryable errors)
	mockAIEngine.AssertExpectations(t)

	// Verify repository Update was NOT called
	mockRepo.AssertNotCalled(t, "Update", mock.Anything, mock.Anything)
}

// TestGetModerationQueue tests retrieving posts in moderation queue
func TestGetModerationQueue(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockPostRepository{}
	mockCommentRepo := &commentMocks.MockCommentRepository{}

	cfg := &platformconfig.Config{
		Cache: platformconfig.CacheConfig{
			Enabled: false,
		},
	}

	svc := &postService{
		repo:        mockRepo,
		commentRepo: mockCommentRepo,
		config:      cfg,
	}

	// Create test posts in moderation queue
	post1 := &models.Post{
		ObjectId: uuid.Must(uuid.NewV4()),
		Body:     "Flagged content 1",
		Status:   "needs_moderation",
	}
	post2 := &models.Post{
		ObjectId: uuid.Must(uuid.NewV4()),
		Body:     "Flagged content 2",
		Status:   "needs_moderation",
	}

	mockRepo.On("FindByStatus", ctx, "needs_moderation", 20, 0).Return([]*models.Post{post1, post2}, nil)
	mockRepo.On("CountByStatus", ctx, "needs_moderation").Return(int64(2), nil)

	posts, totalCount, err := svc.GetModerationQueue(ctx, 20, 0)

	assert.NoError(t, err)
	assert.Equal(t, int64(2), totalCount)
	assert.Len(t, posts, 2)
	mockRepo.AssertExpectations(t)
}

// TestApprovePost tests approving a post in moderation queue
func TestApprovePost(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockPostRepository{}
	mockCommentRepo := &commentMocks.MockCommentRepository{}

	cfg := &platformconfig.Config{
		Cache: platformconfig.CacheConfig{
			Enabled: false,
		},
	}

	svc := &postService{
		repo:        mockRepo,
		commentRepo: mockCommentRepo,
		config:      cfg,
	}

	postID := uuid.Must(uuid.NewV4())
	post := &models.Post{
		ObjectId: postID,
		Body:     "Flagged content",
		Status:   "needs_moderation",
	}

	mockRepo.On("FindByID", ctx, postID).Return(post, nil)
	mockRepo.On("FindByID", ctx, postID).Return(post, nil) // Called again by UpdateFields
	mockRepo.On("Update", ctx, mock.MatchedBy(func(p *models.Post) bool {
		return p.Status == "published"
	})).Return(nil)

	err := svc.ApprovePost(ctx, postID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

// TestRejectPost tests rejecting a post in moderation queue
func TestRejectPost(t *testing.T) {
	ctx := context.Background()
	mockRepo := &MockPostRepository{}
	mockCommentRepo := &commentMocks.MockCommentRepository{}

	cfg := &platformconfig.Config{
		Cache: platformconfig.CacheConfig{
			Enabled: false,
		},
	}

	svc := &postService{
		repo:        mockRepo,
		commentRepo: mockCommentRepo,
		config:      cfg,
	}

	postID := uuid.Must(uuid.NewV4())
	post := &models.Post{
		ObjectId: postID,
		Body:     "Flagged content",
		Status:   "needs_moderation",
	}

	mockRepo.On("FindByID", ctx, postID).Return(post, nil)
	mockRepo.On("FindByID", ctx, postID).Return(post, nil) // Called again by UpdateFields
	mockRepo.On("Update", ctx, mock.MatchedBy(func(p *models.Post) bool {
		return p.Status == "rejected"
	})).Return(nil)

	err := svc.RejectPost(ctx, postID)

	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

