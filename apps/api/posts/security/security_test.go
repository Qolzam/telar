package security

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	service "github.com/qolzam/telar/apps/api/internal/platform"
	platformconfig "github.com/qolzam/telar/apps/api/internal/platform/config"
	"github.com/qolzam/telar/apps/api/internal/types"
	"github.com/qolzam/telar/apps/api/posts/models"
	"github.com/qolzam/telar/apps/api/posts/services"
)

// MockRepository implements a mock repository for security testing
type MockRepository struct {
	mock.Mock
}

// Implement all Repository interface methods (minimal for security tests)
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
		// Try bidirectional channel type as well
		if ch, ok := args.Get(0).(chan interfaces.SingleResult); ok {
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
	result <- &MockCursor{err: args.Error(0)}
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

func (m *MockRepository) UpdateMany(ctx context.Context, collection string, query *interfaces.Query, data interface{}, opts *interfaces.UpdateOptions) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, data, opts)
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

func (m *MockRepository) CreateIndex(ctx context.Context, collection string, indexes map[string]interface{}) <-chan error {
	args := m.Called(ctx, collection, indexes)
	result := make(chan error, 1)
	result <- args.Error(0)
	close(result)
	return result
}

func (m *MockRepository) Count(ctx context.Context, collection string, query *interfaces.Query) <-chan interfaces.CountResult {
	args := m.Called(ctx, collection, query)
	result := make(chan interfaces.CountResult, 1)
	result <- interfaces.CountResult{Count: 0, Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) BeginTransaction(ctx context.Context) (interfaces.TransactionContext, error) {
	args := m.Called(ctx)
	return nil, args.Error(0)
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

// Missing methods for new interface
func (m *MockRepository) FindWithCursor(ctx context.Context, collection string, query *interfaces.Query, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	args := m.Called(ctx, collection, query, opts)
	result := make(chan interfaces.QueryResult, 1)
	result <- &MockCursor{err: args.Error(0)}
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
		return interfaces.ErrNoDocuments
	}
	
	// Simple mock decode for Post
	if post, ok := m.document.(*models.Post); ok {
		if targetPost, ok := v.(*models.Post); ok {
			*targetPost = *post
			return nil
		}
	}
	return interfaces.ErrNoDocuments
}

// MockCursor implements QueryResult interface
type MockCursor struct {
	err error
}

func (m *MockCursor) Error() error {
	return m.err
}

func (m *MockCursor) Next() bool {
	return false
}

func (m *MockCursor) Decode(v interface{}) error {
	return m.err
}

func (m *MockCursor) Close() {
	// No-op for mock
}

// Test helper functions
func createMaliciousUserContext() *types.UserContext {
	return &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "malicious@example.com",
		DisplayName: "Malicious User",
		SocialName:  "malicioususer",
		Avatar:      "https://malicious.com/avatar.jpg",
		SystemRole:  "user",
		CreatedDate: time.Now().Unix(),
	}
}

func createValidUserContext() *types.UserContext {
	return &types.UserContext{
		UserID:      uuid.Must(uuid.NewV4()),
		Username:    "valid@example.com",
		DisplayName: "Valid User",
		SocialName:  "validuser",
		Avatar:      "https://example.com/avatar.jpg",
		SystemRole:  "user",
		CreatedDate: time.Now().Unix(),
	}
}

func createTestPost(ownerID uuid.UUID) *models.Post {
	postID := uuid.Must(uuid.NewV4())
	
	return &models.Post{
		ObjectId:         postID,
		PostTypeId:       1,
		Score:            0,
		Votes:            make(map[string]string),
		ViewCount:        0,
		Body:             "Test post body",
		OwnerUserId:      ownerID,
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

func setupSecurityTestService() (services.PostService, *MockRepository) {
	mockRepo := &MockRepository{}
	baseService := &service.BaseService{
		Repository: mockRepo,
	}
	// Create platform config for the service using test config
	cfg := &platformconfig.Config{
		JWT: platformconfig.JWTConfig{
			PublicKey:  "test-public-key",
			PrivateKey: "test-private-key",
		},
		HMAC: platformconfig.HMACConfig{
			Secret: "test-secret",
		},
		App: platformconfig.AppConfig{
			WebDomain: "http://localhost:3000",
		},
	}
	svc := services.NewPostService(baseService, cfg)
	return svc, mockRepo
}

// Test unauthorized access to another user's post
func TestSecurityUnauthorizedPostAccess(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	
	validUser := createValidUserContext()
	maliciousUser := createMaliciousUserContext()
	
	// Create a post owned by validUser
	testPost := createTestPost(validUser.UserID)
	
	// Malicious user tries to update validUser's post
	req := &models.UpdatePostRequest{
		Body: func() *string { s := "Hacked content"; return &s }(),
	}

	// Setup mock expectations - should fail ownership validation
	resultChan := make(chan interfaces.SingleResult, 1)
	resultChan <- &MockSingleResult{err: interfaces.ErrNoDocuments}
	close(resultChan)
	mockRepo.On("FindOne", ctx, "post", mock.AnythingOfType("*interfaces.Query")).Return(resultChan)

	// Execute - should fail due to ownership validation
	err := postService.UpdatePost(ctx, testPost.ObjectId, req, maliciousUser)

	// Assert
	assert.Error(t, err)
	// Error message changed to be more accurate - check for either pattern
	assert.True(t,
		strings.Contains(err.Error(), "post not found") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "document not found"),
		"Expected error about post not found, got: %s", err.Error())

	mockRepo.AssertExpectations(t)
}

// Test unauthorized post deletion
func TestSecurityUnauthorizedPostDeletion(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	
	validUser := createValidUserContext()
	maliciousUser := createMaliciousUserContext()
	
	// Create a post owned by validUser
	testPost := createTestPost(validUser.UserID)
	
	// Malicious user tries to delete validUser's post
	resultChan := make(chan interfaces.SingleResult, 1)
	resultChan <- &MockSingleResult{err: interfaces.ErrNoDocuments}
	close(resultChan)
	mockRepo.On("FindOne", ctx, "post", mock.AnythingOfType("*interfaces.Query")).Return(resultChan)

	// Execute - should fail due to ownership validation
	err := postService.DeletePost(ctx, testPost.ObjectId, maliciousUser)

	// Assert
	assert.Error(t, err)
	// Error message changed to be more accurate - check for either pattern
	assert.True(t,
		strings.Contains(err.Error(), "post not found") ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "document not found"),
		"Expected error about post not found, got: %s", err.Error())

	mockRepo.AssertExpectations(t)
}

// Test unauthorized score manipulation
func TestSecurityUnauthorizedScoreManipulation(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	
	validUser := createValidUserContext()
	maliciousUser := createMaliciousUserContext()
	
	// Create a post owned by validUser
	testPost := createTestPost(validUser.UserID)
	
	// Malicious user tries to manipulate score of validUser's post
	// This should be blocked by ownership validation in IncrementScore
	
	// Mock expectation: IncrementScore now uses IncrementFields with Query object
	expectedIncrements := map[string]interface{}{"score": 1000} // Large score manipulation
	resultChan := make(chan interfaces.RepositoryResult, 1)
	resultChan <- interfaces.RepositoryResult{Error: interfaces.ErrNoDocuments}
	close(resultChan)
	mockRepo.On("IncrementFields", mock.Anything, "post", mock.AnythingOfType("*interfaces.Query"), expectedIncrements).Return(resultChan)

	// Execute - should fail due to ownership validation
	err := postService.IncrementScore(ctx, testPost.ObjectId, 1000, maliciousUser)

	// Assert
	assert.Error(t, err)

	mockRepo.AssertExpectations(t)
}

// Test SQL injection prevention in post body
func TestSecuritySQLInjectionPrevention(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	user := createValidUserContext()

	// Create post request with SQL injection attempt
	maliciousBody := "'; DROP TABLE posts; --"
	req := &models.CreatePostRequest{
		PostTypeId: 1,
		Body:       maliciousBody,
		Permission: "Public",
	}

	// The service should accept the content as-is (it's the repository's job to handle SQL injection)
	// But we should ensure the content is stored exactly as provided without interpretation
	mockRepo.On("Save", ctx, "post", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.MatchedBy(func(post *models.Post) bool {
		return post.Body == maliciousBody // Body should be stored exactly as provided
	})).Return(nil)

	// Execute
	result, err := postService.CreatePost(ctx, req, user)

	// Assert
	assert.NoError(t, err) // Service should not fail, but content should be stored safely
	assert.NotNil(t, result)
	assert.Equal(t, maliciousBody, result.Body) // Content stored exactly as provided

	mockRepo.AssertExpectations(t)
}

// Test XSS prevention in post content
func TestSecurityXSSPrevention(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	user := createValidUserContext()

	// Create post request with XSS attempt
	maliciousBody := `<script>alert('XSS')</script>Hello World`
	req := &models.CreatePostRequest{
		PostTypeId: 1,
		Body:       maliciousBody,
		Permission: "Public",
	}

	// The service should store the content as-is (XSS prevention happens at the presentation layer)
	mockRepo.On("Save", ctx, "post", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.MatchedBy(func(post *models.Post) bool {
		return post.Body == maliciousBody
	})).Return(nil)

	// Execute
	result, err := postService.CreatePost(ctx, req, user)

	// Assert
	assert.NoError(t, err) // Service stores content as-is
	assert.NotNil(t, result)
	assert.Equal(t, maliciousBody, result.Body)

	mockRepo.AssertExpectations(t)
}

// Test unauthorized access to sensitive operations
func TestSecurityUnauthorizedSensitiveOperations(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	
	validUser := createValidUserContext()
	maliciousUser := createMaliciousUserContext()
	testPost := createTestPost(validUser.UserID)

	// Test unauthorized comment disabling
	t.Run("Unauthorized comment disabling", func(t *testing.T) {
		// Service uses UpdateFields with Query object that includes ownership check
		// When user doesn't own the post, UpdateFields should return no rows affected (error)
		resultChan := make(chan interfaces.RepositoryResult, 1)
		resultChan <- interfaces.RepositoryResult{Error: interfaces.ErrNoDocuments}
		close(resultChan)
		mockRepo.On("UpdateFields", mock.Anything, "post", mock.AnythingOfType("*interfaces.Query"), 
			map[string]interface{}{"disableComments": true}).Return(resultChan)

		err := postService.SetCommentDisabled(ctx, testPost.ObjectId, true, maliciousUser)
		assert.Error(t, err)
	})

	// Test unauthorized sharing disabling
	t.Run("Unauthorized sharing disabling", func(t *testing.T) {
		// Service uses UpdateFields with Query object that includes ownership check
		// When user doesn't own the post, UpdateFields should return no rows affected (error)
		resultChan := make(chan interfaces.RepositoryResult, 1)
		resultChan <- interfaces.RepositoryResult{Error: interfaces.ErrNoDocuments}
		close(resultChan)
		mockRepo.On("UpdateFields", mock.Anything, "post", mock.AnythingOfType("*interfaces.Query"), 
			map[string]interface{}{"disableSharing": true}).Return(resultChan)

		err := postService.SetSharingDisabled(ctx, testPost.ObjectId, true, maliciousUser)
		assert.Error(t, err)
	})

	mockRepo.AssertExpectations(t)
}

// Test privilege escalation prevention
func TestSecurityPrivilegeEscalationPrevention(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	
	regularUser := createValidUserContext()
	regularUser.SystemRole = "user" // Regular user
	
	// Try to create a post with admin-like privileges
	req := &models.CreatePostRequest{
		PostTypeId: 999, // Assume this is an admin-only post type
		Body:       "Admin announcement",
		Permission: "Public",
	}

	// Service should allow creation (authorization happens at the handler level)
	// But we test that the user context is preserved correctly
	mockRepo.On("Save", ctx, "post", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.MatchedBy(func(post *models.Post) bool {
		return post.OwnerUserId == regularUser.UserID && post.PostTypeId == 999
	})).Return(nil)

	// Execute
	result, err := postService.CreatePost(ctx, req, regularUser)

	// Assert - service allows creation but maintains user context
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, regularUser.UserID, result.OwnerUserId)
	assert.Equal(t, 999, result.PostTypeId) // PostTypeId validation should happen at handler level

	mockRepo.AssertExpectations(t)
}

// Test data validation bypass attempts
func TestSecurityDataValidationBypass(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	user := createValidUserContext()

	testCases := []struct {
		name        string
		requestBody string
		shouldPass  bool
	}{
		{
			name:        "Extremely long body",
			requestBody: string(make([]byte, 100000)), // 100KB of data
			shouldPass:  true, // Service layer doesn't validate length
		},
		{
			name:        "Empty body",
			requestBody: "",
			shouldPass:  true, // Service layer allows empty body
		},
		{
			name:        "Special characters",
			requestBody: "Special chars: 💀☠️🔥💯 and unicode: 你好世界",
			shouldPass:  true, // Should handle unicode correctly
		},
		{
			name:        "Null bytes",
			requestBody: "Hello\x00World",
			shouldPass:  true, // Service stores as-is
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := &models.CreatePostRequest{
				PostTypeId: 1,
				Body:       tc.requestBody,
				Permission: "Public",
			}

			if tc.shouldPass {
				mockRepo.On("Save", ctx, "post", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.MatchedBy(func(post *models.Post) bool {
					return post.Body == tc.requestBody
				})).Return(nil).Once()
			}

			result, err := postService.CreatePost(ctx, req, user)

			if tc.shouldPass {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tc.requestBody, result.Body)
			} else {
				assert.Error(t, err)
				assert.Nil(t, result)
			}
		})
	}

	mockRepo.AssertExpectations(t)
}

// Test rate limiting bypass prevention (conceptual - actual rate limiting happens at handler level)
func TestSecurityRateLimitingConcepts(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	user := createValidUserContext()

	// Simulate rapid-fire post creation attempts
	for i := 0; i < 5; i++ {
		req := &models.CreatePostRequest{
			PostTypeId: 1,
			Body:       "Rapid post " + string(rune(i)),
			Permission: "Public",
		}

		mockRepo.On("Save", ctx, "post", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.AnythingOfType("*models.Post")).Return(nil).Once()

		result, err := postService.CreatePost(ctx, req, user)
		
		// Service layer allows all requests (rate limiting is middleware responsibility)
		assert.NoError(t, err)
		assert.NotNil(t, result)
	}

	mockRepo.AssertExpectations(t)
}

// Test permission validation edge cases
func TestSecurityPermissionValidation(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	user := createValidUserContext()

	// Test with various permission values
	permissions := []string{"Public", "OnlyMe", "Circles", "", "Invalid", "admin", "ADMIN"}

	for _, perm := range permissions {
		t.Run("Permission: "+perm, func(t *testing.T) {
			req := &models.CreatePostRequest{
				PostTypeId: 1,
				Body:       "Test post with " + perm + " permission",
				Permission: perm,
			}

			mockRepo.On("Save", ctx, "post", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.MatchedBy(func(post *models.Post) bool {
				return post.Permission == perm
			})).Return(nil).Once()

			result, err := postService.CreatePost(ctx, req, user)

			// Service layer accepts all permission values (validation at handler level)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, perm, result.Permission)
		})
	}

	mockRepo.AssertExpectations(t)
}

// Test context injection and user context integrity
func TestSecurityUserContextIntegrity(t *testing.T) {
	postService, mockRepo := setupSecurityTestService()
	ctx := context.Background()
	user := createValidUserContext()

	// Verify that user context data is preserved correctly and not corrupted
	req := &models.CreatePostRequest{
		PostTypeId: 1,
		Body:       "Test post for user context integrity",
		Permission: "Public",
	}

	mockRepo.On("Save", ctx, "post", mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.MatchedBy(func(post *models.Post) bool {
		// Verify all user context fields are correctly set
		return post.OwnerUserId == user.UserID &&
			post.OwnerDisplayName == user.DisplayName &&
			post.OwnerAvatar == user.Avatar
	})).Return(nil)

	result, err := postService.CreatePost(ctx, req, user)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, user.UserID, result.OwnerUserId)
	assert.Equal(t, user.DisplayName, result.OwnerDisplayName)
	assert.Equal(t, user.Avatar, result.OwnerAvatar)

	mockRepo.AssertExpectations(t)
}
