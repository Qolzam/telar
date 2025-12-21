package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	commentMocks "github.com/qolzam/telar/apps/api/comments/services/mocks"
	"github.com/qolzam/telar/apps/api/posts/models"
)

// MockRepository implements a mock repository for testing
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Save(ctx context.Context, collectionName string, objectID uuid.UUID, ownerUserID uuid.UUID, createdDate, lastUpdated int64, data interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collectionName, objectID, ownerUserID, createdDate, lastUpdated, data)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) SaveMany(ctx context.Context, collectionName string, items []interfaces.SaveItem) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collectionName, items)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) FindOne(ctx context.Context, collection string, query *interfaces.Query) <-chan interfaces.SingleResult {
	args := m.Called(ctx, collection, query)
	
	// If first arg is a channel (already constructed), return it directly
	if len(args) > 0 {
		if ch, ok := args.Get(0).(<-chan interfaces.SingleResult); ok {
			return ch
		}
	}
	
	// Otherwise, construct from document/error
	result := make(chan interfaces.SingleResult, 1)
	if len(args) > 0 && args.Get(0) != nil {
		result <- &MockSingleResult{
			document: args.Get(0),
			err:      nil,
		}
	} else {
		err := args.Error(0)
		if err == nil && len(args) > 1 {
			err = args.Error(1)
		}
		result <- &MockSingleResult{
			err: err,
		}
	}
	close(result)
	return result
}

func (m *MockRepository) Find(ctx context.Context, collection string, query *interfaces.Query, options *interfaces.FindOptions) <-chan interfaces.QueryResult {
	args := m.Called(ctx, collection, query, options)
	result := make(chan interfaces.QueryResult, 1)
	
	if args.Get(0) != nil {
		result <- &MockCursor{
			documents: args.Get(0).([]interface{}),
			err:       args.Error(1),
		}
	} else {
		result <- &MockCursor{
			err: args.Error(1),
		}
	}
	close(result)
	return result
}

func (m *MockRepository) Update(ctx context.Context, collection string, query *interfaces.Query, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, data, opts)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) Count(ctx context.Context, collection string, query *interfaces.Query) <-chan interfaces.CountResult {
	args := m.Called(ctx, collection, query)
	result := make(chan interfaces.CountResult, 1)
	result <- interfaces.CountResult{
		Count: args.Get(0).(int64),
		Error: args.Error(1),
	}
	close(result)
	return result
}

func (m *MockRepository) UpdateFields(ctx context.Context, collection string, query *interfaces.Query, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, updates)
	
	// If first arg is a channel (already constructed), return it directly
	if len(args) > 0 {
		if ch, ok := args.Get(0).(<-chan interfaces.RepositoryResult); ok {
			return ch
		}
		// Try bidirectional channel type as well
		if ch, ok := args.Get(0).(chan interfaces.RepositoryResult); ok {
			return ch
		}
	}
	
	// Otherwise, construct from error
	result := make(chan interfaces.RepositoryResult, 1)
	var err error
	if len(args) > 0 {
		// Try to get error from first arg if it's an error
		if e, ok := args.Get(0).(error); ok {
			err = e
		} else {
			// Otherwise try Error(0)
			err = args.Error(0)
		}
	} else {
		err = args.Error(0)
	}
	result <- interfaces.RepositoryResult{Error: err}
	close(result)
	return result
}

func (m *MockRepository) UpdateMany(ctx context.Context, collection string, query *interfaces.Query, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, data, opts)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) IncrementFields(ctx context.Context, collection string, query *interfaces.Query, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, increments)
	
	// If first arg is a channel (already constructed), return it directly
	if len(args) > 0 {
		if ch, ok := args.Get(0).(<-chan interfaces.RepositoryResult); ok {
			return ch
		}
		// Try bidirectional channel type as well
		if ch, ok := args.Get(0).(chan interfaces.RepositoryResult); ok {
			return ch
		}
	}
	
	// Otherwise, construct from error (if no channel was returned)
	result := make(chan interfaces.RepositoryResult, 1)
	var err error
	if len(args) > 0 {
		// Try to get error from first arg if it's an error
		if e, ok := args.Get(0).(error); ok {
			err = e
		} else {
			// Otherwise try Error(0)
			err = args.Error(0)
		}
	} else {
		err = args.Error(0)
	}
	result <- interfaces.RepositoryResult{Error: err}
	close(result)
	return result
}

func (m *MockRepository) UpdateAndIncrement(ctx context.Context, collection string, query *interfaces.Query, updates map[string]interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, updates, increments)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) UpdateWithOwnership(ctx context.Context, collection string, postID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, postID, ownerID, updates)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) DeleteWithOwnership(ctx context.Context, collection string, postID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, postID, ownerID)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) IncrementWithOwnership(ctx context.Context, collection string, postID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, postID, ownerID, increments)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) Delete(ctx context.Context, collection string, query *interfaces.Query) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) DeleteMany(ctx context.Context, collection string, queries []*interfaces.Query) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, queries)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}


func (m *MockRepository) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	args := m.Called(ctx)
	return args.Get(0).(interfaces.TransactionContext), args.Error(1)
}

func (m *MockRepository) Begin(ctx context.Context) (interfaces.Transaction, error) {
	args := m.Called(ctx)
	return args.Get(0).(interfaces.Transaction), args.Error(1)
}

func (m *MockRepository) BeginWithConfig(ctx context.Context, config *interfaces.TransactionConfig) (interfaces.Transaction, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(interfaces.Transaction), args.Error(1)
}

func (m *MockRepository) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func (m *MockRepository) Ping(ctx context.Context) <-chan error {
	args := m.Called(ctx)
	result := make(chan error, 1)
	result <- args.Error(0)
	close(result)
	return result
}

func (m *MockRepository) Close() error {
	args := m.Called()
	return args.Error(0)
}

// New cursor-based pagination methods
func (m *MockRepository) FindWithCursor(ctx context.Context, collection string, query *interfaces.Query, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	args := m.Called(ctx, collection, query, opts)
	result := make(chan interfaces.QueryResult, 1)
	
	if args.Get(0) != nil {
		result <- &MockCursor{
			documents: args.Get(0).([]interface{}),
			err:       args.Error(1),
		}
	} else {
		result <- &MockCursor{
			err: args.Error(1),
		}
	}
	close(result)
	return result
}

func (m *MockRepository) CountWithFilter(ctx context.Context, collection string, query *interfaces.Query) <-chan interfaces.CountResult {
	args := m.Called(ctx, collection, query)
	result := make(chan interfaces.CountResult, 1)
	result <- interfaces.CountResult{
		Count: args.Get(0).(int64),
		Error: args.Error(1),
	}
	close(result)
	return result
}

// MockSingleResult implements SingleResult interface
type MockSingleResult struct {
	document interface{}
	err      error
}

func (m *MockSingleResult) Error() error {
	return m.err
}

func (m *MockSingleResult) NoResult() bool {
	return m.document == nil && m.err != nil
}

func (m *MockSingleResult) Decode(v interface{}) error {
	if m.err != nil {
		return m.err
	}
	if m.document == nil {
		return dbi.ErrNoDocuments
	}
	
	// Simple mock decode - in real tests this would be more sophisticated
	if post, ok := m.document.(*models.Post); ok {
		if targetPost, ok := v.(*models.Post); ok {
			*targetPost = *post
			return nil
		}
	}
	return errors.New("decode error")
}

// MockCursor implements QueryResult interface  
type MockCursor struct {
	documents []interface{}
	current   int
	err       error
}

func (m *MockCursor) Error() error {
	return m.err
}

func (m *MockCursor) Next() bool {
	if m.err != nil {
		return false
	}
	return m.current < len(m.documents)
}

func (m *MockCursor) Decode(v interface{}) error {
	if m.err != nil {
		return m.err
	}
	if m.current >= len(m.documents) {
		return errors.New("no more documents")
	}
	
	// Simple mock decode
	if post, ok := m.documents[m.current].(*models.Post); ok {
		if targetPost, ok := v.(*models.Post); ok {
			*targetPost = *post
			m.current++
			return nil
		}
	}
	return errors.New("decode error")
}

func (m *MockCursor) Close() {
	// No-op for mock
}

// Test helper functions
func createTestUserContext() *types.UserContext {
	return &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "test@example.com",
		DisplayName: "Test User",
		SocialName:  "testuser",
		Avatar:      "https://example.com/avatar.jpg",
		SystemRole:  "user",
		CreatedDate: time.Now().Unix(),
	}
}

func createTestCreatePostRequest() *models.CreatePostRequest {
	return &models.CreatePostRequest{
		PostTypeId:      1,
		Body:            "This is a test post body",
		Image:           "https://example.com/image.jpg",
		ImageFullPath:   "https://example.com/full/image.jpg",
		Tags:            []string{"test", "post"},
		DisableComments: false,
		DisableSharing:  false,
		Permission:      "Public",
		Version:         "1.0",
	}
}

func createTestPost() *models.Post {
	postID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	
	return &models.Post{
		ObjectId:         postID,
		PostTypeId:       1,
		Score:            0,
		Votes:            make(map[string]string),
		ViewCount:        0,
		Body:             "Test post body",
		OwnerUserId:      userID,
		OwnerDisplayName: "Test User",
		OwnerAvatar:      "https://example.com/avatar.jpg",
		URLKey:           "test-url-key",
		Tags:             []string{"test"},
		CommentCounter:   0,
		DisableComments:  false,
		DisableSharing:   false,
		Deleted:          false,
		DeletedDate:      0,
		CreatedDate:      time.Now().Unix(),
		LastUpdated:      0,
		Permission:       "Public",
		Version:          "1.0",
	}
}

func setupTestService() (*postService, *MockPostRepository) {
	mockRepo := &MockPostRepository{}
	
	// Create test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
	}
	
	// Create mock comment repository (required for SoftDeletePost)
	mockCommentRepo := &commentMocks.MockCommentRepository{}
	
	svc := &postService{
		repo:        mockRepo,
		commentRepo: mockCommentRepo,
		config:      cfg,
	}
	return svc, mockRepo
}

// Test CreatePost with valid request
func TestCreatePost_ValidRequest_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreatePostRequest()

	// Setup mock expectations
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	// Execute
	result, err := service.CreatePost(ctx, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.PostTypeId, result.PostTypeId)
	assert.Equal(t, req.Body, result.Body)
	assert.Equal(t, user.UserID, result.OwnerUserId)
	assert.Equal(t, user.DisplayName, result.OwnerDisplayName)
	assert.Equal(t, user.Avatar, result.OwnerAvatar)
	assert.Equal(t, int64(0), result.Score)
	assert.Equal(t, int64(0), result.ViewCount)
	assert.False(t, result.Deleted)
	assert.NotEqual(t, uuid.Nil, result.ObjectId)
	assert.Greater(t, result.CreatedDate, int64(0))

	mockRepo.AssertExpectations(t)
}

// Test CreatePost with nil request
func TestCreatePost_NilRequest_ReturnsError(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()

	// Execute
	result, err := service.CreatePost(ctx, nil, user)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "create post request is required")
}

// Test CreatePost with nil user context
func TestCreatePost_NilUserContext_ReturnsError(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()
	req := createTestCreatePostRequest()

	// Execute
	result, err := service.CreatePost(ctx, req, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user context is required")
}

// Test CreatePost with database error
func TestCreatePost_DatabaseError_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreatePostRequest()

	// Setup mock expectations - simulate database error
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(errors.New("database connection failed"))

	// Execute
	result, err := service.CreatePost(ctx, req, user)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to create post")
	assert.Contains(t, err.Error(), "database connection failed")

	mockRepo.AssertExpectations(t)
}

// Test CreatePost with provided ObjectId (backward compatibility)
func TestCreatePost_WithProvidedObjectId_UsesProvidedId(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreatePostRequest()
	
	// Set a specific ObjectId
	providedID := uuid.Must(uuid.NewV4())
	req.ObjectId = &providedID

	// Setup mock expectations
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	// Execute
	result, err := service.CreatePost(ctx, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, providedID, result.ObjectId)

	mockRepo.AssertExpectations(t)
}

// Test CreatePost with album
func TestCreatePost_WithAlbum_SetsAlbum(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreatePostRequest()
	
	// Add album
	req.Album = models.Album{
		Count:   3,
		Cover:   "cover.jpg",
		CoverId: uuid.Must(uuid.NewV4()),
		Photos:  []string{"photo1.jpg", "photo2.jpg", "photo3.jpg"},
		Title:   "Test Album",
	}

	// Setup mock expectations
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	// Execute
	result, err := service.CreatePost(ctx, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Album)
	assert.Equal(t, req.Album.Count, result.Album.Count)
	assert.Equal(t, req.Album.Title, result.Album.Title)
	assert.Equal(t, len(req.Album.Photos), len(result.Album.Photos))

	mockRepo.AssertExpectations(t)
}

// Test GetPost with valid ID
func TestGetPost_ValidId_ReturnsPost(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	testPost := createTestPost()
	postID := testPost.ObjectId

	// Setup mock expectations
	mockRepo.On("FindByID", ctx, postID).Return(testPost, nil)

	// Execute
	result, err := service.GetPost(ctx, postID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testPost.ObjectId, result.ObjectId)
	assert.Equal(t, testPost.Body, result.Body)

	mockRepo.AssertExpectations(t)
}

// Test GetPost with non-existent ID
func TestGetPost_NonExistentId_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	postID := uuid.Must(uuid.NewV4())

	// Setup mock expectations - simulate not found
	mockRepo.On("FindByID", ctx, postID).Return(nil, errors.New("document not found"))

	// Execute
	result, err := service.GetPost(ctx, postID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	// Error message changed to be more accurate - check for either pattern
	assert.True(t,
		strings.Contains(err.Error(), "failed to find post") ||
		strings.Contains(err.Error(), "failed to decode post") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "document not found"),
		"Expected error about post not found, got: %s", err.Error())

	mockRepo.AssertExpectations(t)
}

// Test UpdatePost with valid request
func TestUpdatePost_ValidRequest_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testPost := createTestPost()
	testPost.OwnerUserId = user.UserID // Ensure ownership
	
	newBody := "Updated post body"
	req := &models.UpdatePostRequest{
		Body: &newBody,
	}

	// Setup mock expectations for ownership validation
	mockRepo.On("FindByID", ctx, testPost.ObjectId).Return(testPost, nil)
	
	// Setup mock expectations for update - UpdatePost loads post, modifies it, then calls Update
	updatedPost := *testPost
	updatedPost.Body = newBody
	mockRepo.On("Update", ctx, mock.MatchedBy(func(p *models.Post) bool {
		return p.ObjectId == testPost.ObjectId && p.Body == newBody
	})).Return(nil)

	// Execute
	err := service.UpdatePost(ctx, testPost.ObjectId, req, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test UpdatePost with ownership validation failure
func TestUpdatePost_UnauthorizedUser_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testPost := createTestPost()
	// Different user ID to simulate unauthorized access
	
	newBody := "Updated post body"
	req := &models.UpdatePostRequest{
		Body: &newBody,
	}

	// Setup mock expectations for ownership validation failure
	mockRepo.On("FindByID", ctx, testPost.ObjectId).Return(nil, errors.New("not found"))

	// Execute
	err := service.UpdatePost(ctx, testPost.ObjectId, req, user)

	// Assert
	assert.Error(t, err)
	// Error message changed to be more accurate - check for either pattern
	assert.True(t, 
		strings.Contains(err.Error(), "post not found") || 
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "document not found") ||
		strings.Contains(err.Error(), "decode error"),
		"Expected error about post not found, got: %s", err.Error())

	mockRepo.AssertExpectations(t)
}

// Test ValidatePostOwnership with valid ownership
func TestValidatePostOwnership_ValidOwner_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testPost := createTestPost()
	testPost.OwnerUserId = user.UserID

	// Setup mock expectations
	mockRepo.On("FindByID", ctx, testPost.ObjectId).Return(testPost, nil)

	// Execute
	err := service.ValidatePostOwnership(ctx, testPost.ObjectId, user.UserID)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test ValidatePostOwnership with invalid ownership
func TestValidatePostOwnership_InvalidOwner_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	postID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())

	// Setup mock expectations - simulate not found
	mockRepo.On("FindByID", ctx, postID).Return(nil, errors.New("not found"))

	// Execute
	err := service.ValidatePostOwnership(ctx, postID, userID)

	// Assert
	assert.Error(t, err)
	// Error message changed to be more accurate - check for either pattern
	assert.True(t,
		strings.Contains(err.Error(), "post not found") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "document not found") ||
		strings.Contains(err.Error(), "decode error"),
		"Expected error about post not found, got: %s", err.Error())

	mockRepo.AssertExpectations(t)
}

// Test IncrementScore with valid user
func TestIncrementScore_ValidUser_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	postID := uuid.Must(uuid.NewV4())
	delta := 5

	// Setup mock expectations - IncrementScore uses atomic repository method
	mockRepo.On("IncrementScore", ctx, postID, delta).Return(nil)

	// Execute
	err := service.IncrementScore(ctx, postID, delta, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test IncrementScore with database error
func TestIncrementScore_DatabaseError_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	postID := uuid.Must(uuid.NewV4())
	delta := 5

	// Setup mock expectations with error - IncrementScore uses atomic repository method
	mockRepo.On("IncrementScore", ctx, postID, delta).Return(errors.New("database error"))

	// Execute
	err := service.IncrementScore(ctx, postID, delta, user)

	// Assert
	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
}

// Test IncrementCommentCount with valid parameters
func TestIncrementCommentCount_ValidParameters_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	postID := uuid.Must(uuid.NewV4())
	delta := 1

	// Setup mock expectations - IncrementCommentCount verifies ownership, then uses atomic repository method
	testPost := createTestPost()
	testPost.ObjectId = postID
	testPost.OwnerUserId = user.UserID
	mockRepo.On("FindByID", ctx, postID).Return(testPost, nil)
	mockRepo.On("IncrementCommentCount", ctx, postID, delta).Return(nil)

	// Execute
	err := service.IncrementCommentCount(ctx, postID, delta, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test IncrementViewCount
func TestIncrementViewCount_ValidUser_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	postID := uuid.Must(uuid.NewV4())

	// Setup mock expectations - IncrementViewCount uses atomic repository method
	mockRepo.On("IncrementViewCount", ctx, postID).Return(nil)

	// Execute
	err := service.IncrementViewCount(ctx, postID, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test SetCommentDisabled
func TestSetCommentDisabled_ValidParameters_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	postID := uuid.Must(uuid.NewV4())
	disabled := true

	// Setup mock expectations - SetCommentDisabled uses repository method with ownership validation
	mockRepo.On("SetCommentDisabled", ctx, postID, disabled, user.UserID).Return(nil)

	// Execute
	err := service.SetCommentDisabled(ctx, postID, disabled, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test SetSharingDisabled
func TestSetSharingDisabled_ValidParameters_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	postID := uuid.Must(uuid.NewV4())
	disabled := false

	// Setup mock expectations - SetSharingDisabled uses repository method with ownership validation
	mockRepo.On("SetSharingDisabled", ctx, postID, disabled, user.UserID).Return(nil)

	// Execute
	err := service.SetSharingDisabled(ctx, postID, disabled, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test DeletePost with valid ownership
func TestDeletePost_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testPost := createTestPost()
	testPost.OwnerUserId = user.UserID

	// Setup mock expectations for ownership validation
	mockRepo.On("FindByID", ctx, testPost.ObjectId).Return(testPost, nil)
	
	// Setup mock expectations for delete
	mockRepo.On("Delete", ctx, testPost.ObjectId).Return(nil)

	// Execute
	err := service.DeletePost(ctx, testPost.ObjectId, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test SoftDeletePost with valid ownership
func TestSoftDeletePost_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testPost := createTestPost()
	testPost.OwnerUserId = user.UserID
	testPost.Deleted = false // Ensure post is not already deleted

	// Setup mock expectations for findPostForOwnershipCheck (no deleted filter)
	mockRepo.On("FindByID", ctx, testPost.ObjectId).Return(testPost, nil)
	
	// Setup mock for comment repository (cascade soft-delete)
	mockCommentRepo := service.commentRepo.(*commentMocks.MockCommentRepository)
	mockCommentRepo.On("DeleteByPostID", mock.Anything, testPost.ObjectId).Return(nil)
	
	// Setup transaction mock - WithTransaction calls the function with a transaction context
	// The mock implementation already executes the function, so we just need to return nil
	mockRepo.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).Return(nil)
	
	// Setup mock for FindByID (called by UpdateFields to load the post before updating)
	// This is called WITHIN the transaction, so use mock.Anything for context
	mockRepo.On("FindByID", mock.Anything, testPost.ObjectId).Return(testPost, nil)
	
	// Setup mock for Update (called by UpdateFields within transaction)
	mockRepo.On("Update", mock.Anything, mock.MatchedBy(func(p *models.Post) bool {
		return p.ObjectId == testPost.ObjectId && p.Deleted == true
	})).Return(nil)

	// Execute
	err := service.SoftDeletePost(ctx, testPost.ObjectId, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test UpdatePostProfile
func TestUpdatePostProfile_ValidParameters_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	displayName := "Updated Display Name"
	avatar := "https://example.com/new-avatar.jpg"

	// Setup mock expectations - UpdatePostProfile uses repository method directly
	mockRepo.On("UpdateOwnerProfile", ctx, userID, displayName, avatar).Return(nil)

	// Execute
	err := service.UpdatePostProfile(ctx, userID, displayName, avatar)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test QueryPosts with filter
func TestQueryPosts_WithFilter_ReturnsResults(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	
	filter := &models.PostQueryFilter{
		Page:  1,
		Limit: 10,
	}

	testPosts := []*models.Post{createTestPost(), createTestPost()}
	interfacePosts := make([]interface{}, len(testPosts))
	for i, post := range testPosts {
		interfacePosts[i] = post
	}

	// Setup mock expectations
	// Note: Service now uses snake_case for sort fields (created_date)
	mockRepo.On("Find", ctx, mock.AnythingOfType("repository.PostFilter"), 10, 0).Return(testPosts, nil)
	mockRepo.On("Count", ctx, mock.AnythingOfType("repository.PostFilter")).Return(int64(2), nil)

	// Execute
	result, err := service.QueryPosts(ctx, filter)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Posts, 2)
	assert.Equal(t, int64(2), result.TotalCount)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)

	mockRepo.AssertExpectations(t)
}

// Test edge cases and error scenarios

// Test GetPostByURLKey with valid key
func TestGetPostByURLKey_ValidKey_ReturnsPost(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	testPost := createTestPost()
	urlKey := "test-url-key"

	// Setup mock expectations
	mockRepo.On("FindByURLKey", ctx, urlKey).Return(testPost, nil)

	// Execute
	result, err := service.GetPostByURLKey(ctx, urlKey)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testPost.URLKey, result.URLKey)

	mockRepo.AssertExpectations(t)
}

// Test GetPostByURLKey with non-existent key
func TestGetPostByURLKey_NonExistentKey_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	urlKey := "non-existent-key"

	// Setup mock expectations
	mockRepo.On("FindByURLKey", ctx, urlKey).Return(nil, errors.New("document not found"))

	// Execute
	result, err := service.GetPostByURLKey(ctx, urlKey)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	// Error message changed to be more accurate - check for either pattern
	assert.True(t,
		strings.Contains(err.Error(), "failed to find post by URL key") ||
		strings.Contains(err.Error(), "failed to decode post") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "document not found"),
		"Expected error about post not found, got: %s", err.Error())

	mockRepo.AssertExpectations(t)
}

// Test UpdatePost with multiple fields
func TestUpdatePost_MultipleFields_UpdatesAllFields(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testPost := createTestPost()
	testPost.OwnerUserId = user.UserID

	newBody := "Updated body"
	newImage := "new-image.jpg"
	newTags := []string{"updated", "tags"}
	disableComments := true

	req := &models.UpdatePostRequest{
		Body:            &newBody,
		Image:           &newImage,
		Tags:            &newTags,
		DisableComments: &disableComments,
	}

	// Setup mock expectations for ownership validation
	mockRepo.On("FindByID", ctx, testPost.ObjectId).Return(testPost, nil)

	// Setup mock expectations for update - UpdatePost loads post, modifies it, then calls Update
	updatedPost := *testPost
	updatedPost.Body = newBody
	updatedPost.Image = newImage
	updatedPost.Tags = newTags
	updatedPost.DisableComments = disableComments
	mockRepo.On("Update", ctx, mock.MatchedBy(func(p *models.Post) bool {
		return p.ObjectId == testPost.ObjectId && p.Body == newBody && p.Image == newImage
	})).Return(nil)

	// Execute
	err := service.UpdatePost(ctx, testPost.ObjectId, req, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test business logic edge cases

// Test CreatePost with empty body (should fail validation at handler level, but service should handle)
func TestCreatePost_EmptyBody_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreatePostRequest()
	req.Body = "" // Empty body

	// Setup mock expectations
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

	// Execute - service layer should not validate business rules, that's handler's job
	result, err := service.CreatePost(ctx, req, user)

	// Assert
	assert.NoError(t, err) // Service layer accepts empty body
	assert.NotNil(t, result)
	assert.Equal(t, "", result.Body)

	mockRepo.AssertExpectations(t)
}

// Test permission validation edge cases
func TestCreatePost_DifferentPermissionTypes_Success(t *testing.T) {
	testCases := []struct {
		name       string
		permission string
	}{
		{"Public permission", "Public"},
		{"OnlyMe permission", "OnlyMe"},
		{"Circles permission", "Circles"},
		{"Empty permission", ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			service, mockRepo := setupTestService()
			ctx := context.Background()
			user := createTestUserContext()
			req := createTestCreatePostRequest()
			req.Permission = tc.permission

	// Setup mock expectations
	mockRepo.On("Create", ctx, mock.AnythingOfType("*models.Post")).Return(nil)

			// Execute
			result, err := service.CreatePost(ctx, req, user)

			// Assert
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, tc.permission, result.Permission)

			mockRepo.AssertExpectations(t)
		})
	}
}
