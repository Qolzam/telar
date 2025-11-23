package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	commentsErrors "github.com/qolzam/telar/apps/api/comments/errors"
	"github.com/qolzam/telar/apps/api/comments/models"
	dbi "github.com/qolzam/telar/apps/api/internal/database/interfaces"
	"github.com/qolzam/telar/apps/api/internal/database/interfaces"
	service "github.com/qolzam/telar/apps/api/internal/platform"
	"github.com/qolzam/telar/apps/api/internal/types"
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

	// Allow callers to provide a pre-built channel
	if len(args) > 0 {
		if ch, ok := args.Get(0).(<-chan interfaces.SingleResult); ok {
			return ch
		}
	}

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
		result <- &MockSingleResult{err: err}
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
		result <- &MockCursor{err: args.Error(1)}
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
	result <- interfaces.CountResult{Count: args.Get(0).(int64), Error: args.Error(1)}
	close(result)
	return result
}

func (m *MockRepository) UpdateFields(ctx context.Context, collection string, query *interfaces.Query, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, updates)

	if len(args) > 0 {
		if ch, ok := args.Get(0).(<-chan interfaces.RepositoryResult); ok {
			return ch
		}
		if ch, ok := args.Get(0).(chan interfaces.RepositoryResult); ok {
			return ch
		}
	}

	result := make(chan interfaces.RepositoryResult, 1)
	var err error
	if len(args) > 0 {
		if e, ok := args.Get(0).(error); ok {
			err = e
		} else {
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

	if len(args) > 0 {
		if ch, ok := args.Get(0).(<-chan interfaces.RepositoryResult); ok {
			return ch
		}
		if ch, ok := args.Get(0).(chan interfaces.RepositoryResult); ok {
			return ch
		}
	}

	result := make(chan interfaces.RepositoryResult, 1)
	var err error
	if len(args) > 0 {
		if e, ok := args.Get(0).(error); ok {
			err = e
		} else {
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

func (m *MockRepository) UpdateWithOwnership(ctx context.Context, collection string, entityID interface{}, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, entityID, ownerID, updates)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) DeleteWithOwnership(ctx context.Context, collection string, entityID interface{}, ownerID interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, entityID, ownerID)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
	close(result)
	return result
}

func (m *MockRepository) IncrementWithOwnership(ctx context.Context, collection string, entityID interface{}, ownerID interface{}, increments map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, entityID, ownerID, increments)
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

func (m *MockRepository) FindWithCursor(ctx context.Context, collection string, query *interfaces.Query, opts *interfaces.CursorFindOptions) <-chan interfaces.QueryResult {
	args := m.Called(ctx, collection, query, opts)
	result := make(chan interfaces.QueryResult, 1)

	if args.Get(0) != nil {
		result <- &MockCursor{
			documents: args.Get(0).([]interface{}),
			err:       args.Error(1),
		}
	} else {
		result <- &MockCursor{err: args.Error(1)}
	}
	close(result)
	return result
}

func (m *MockRepository) CountWithFilter(ctx context.Context, collection string, query *interfaces.Query) <-chan interfaces.CountResult {
	args := m.Called(ctx, collection, query)
	result := make(chan interfaces.CountResult, 1)
	result <- interfaces.CountResult{Count: args.Get(0).(int64), Error: args.Error(1)}
	close(result)
	return result
}

func (m *MockRepository) UpdateFieldsWithOwnership(ctx context.Context, collection string, query *interfaces.Query, ownerID interface{}, updates map[string]interface{}) <-chan interfaces.RepositoryResult {
	args := m.Called(ctx, collection, query, ownerID, updates)
	result := make(chan interfaces.RepositoryResult, 1)
	result <- interfaces.RepositoryResult{Error: args.Error(0)}
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
	
	// Simple mock decode
	if comment, ok := m.document.(*models.Comment); ok {
		if targetComment, ok := v.(*models.Comment); ok {
			*targetComment = *comment
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
	if comment, ok := m.documents[m.current].(*models.Comment); ok {
		if targetComment, ok := v.(*models.Comment); ok {
			*targetComment = *comment
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

func createTestCreateCommentRequest() *models.CreateCommentRequest {
	postID := uuid.Must(uuid.NewV4())
	return &models.CreateCommentRequest{
		PostId: postID,
		Text:   "This is a test comment",
	}
}

func createTestComment() *models.Comment {
	commentID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())
	postID := uuid.Must(uuid.NewV4())
	
	return &models.Comment{
		ObjectId:         commentID,
		PostId:           postID,
		Score:            0,
		Text:             "Test comment text",
		OwnerUserId:      userID,
		OwnerDisplayName: "Test User",
		OwnerAvatar:      "https://example.com/avatar.jpg",
		Deleted:          false,
		DeletedDate:      0,
		CreatedDate:      time.Now().Unix(),
		LastUpdated:      0,
	}
}

func setupTestService() (*commentService, *MockRepository) {
	mockRepo := &MockRepository{}
	baseService := &service.BaseService{
		Repository: mockRepo,
	}
	svc := &commentService{
		base: baseService,
	}
	return svc, mockRepo
}

// Test CreateComment with valid request
func TestCreateComment_ValidRequest_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateCommentRequest()

	// Setup mock expectations
	mockRepo.On("Save", ctx, commentCollectionName, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.AnythingOfType("*models.Comment")).Return(nil)

	// Execute
	result, err := service.CreateComment(ctx, req, user)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, req.PostId, result.PostId)
	assert.Equal(t, req.Text, result.Text)
	assert.Equal(t, user.UserID, result.OwnerUserId)
	assert.Equal(t, user.DisplayName, result.OwnerDisplayName)
	assert.Equal(t, user.Avatar, result.OwnerAvatar)
	assert.Equal(t, int64(0), result.Score)
	assert.False(t, result.Deleted)
	assert.NotEqual(t, uuid.Nil, result.ObjectId)
	assert.Greater(t, result.CreatedDate, int64(0))

	mockRepo.AssertExpectations(t)
}

// Test CreateComment with nil request
func TestCreateComment_NilRequest_ReturnsError(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()

	// Execute
	result, err := service.CreateComment(ctx, nil, user)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "create comment request is required")
}

// Test CreateComment with nil user context
func TestCreateComment_NilUserContext_ReturnsError(t *testing.T) {
	service, _ := setupTestService()
	ctx := context.Background()
	req := createTestCreateCommentRequest()

	// Execute
	result, err := service.CreateComment(ctx, req, nil)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "user context is required")
}

// Test CreateComment with database error
func TestCreateComment_DatabaseError_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateCommentRequest()

	// Setup mock expectations - simulate database error
	mockRepo.On("Save", ctx, commentCollectionName, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.AnythingOfType("*models.Comment")).Return(errors.New("database connection failed"))

	// Execute
	result, err := service.CreateComment(ctx, req, user)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to save comment")

	mockRepo.AssertExpectations(t)
}

// Test GetComment with valid ID
func TestGetComment_ValidId_ReturnsComment(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	testComment := createTestComment()
	commentID := testComment.ObjectId

	// Setup mock expectations
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)

	// Execute
	result, err := service.GetComment(ctx, commentID)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, testComment.ObjectId, result.ObjectId)
	assert.Equal(t, testComment.Text, result.Text)

	mockRepo.AssertExpectations(t)
}

// Test GetComment with non-existent ID
func TestGetComment_NonExistentId_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	commentID := uuid.Must(uuid.NewV4())

	// Setup mock expectations - simulate not found
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(nil, errors.New("document not found"))

	// Execute
	result, err := service.GetComment(ctx, commentID)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode comment")

	mockRepo.AssertExpectations(t)
}

// Test UpdateComment with valid request
func TestUpdateComment_ValidRequest_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID // Ensure ownership
	
	newText := "Updated comment text"
	req := &models.UpdateCommentRequest{
		ObjectId: testComment.ObjectId,
		Text:     newText,
	}

	// Setup mock expectations for ownership validation
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)
	
	// Setup mock expectations for update
	mockRepo.On("UpdateFields", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), mock.MatchedBy(func(updates map[string]interface{}) bool {
		return updates["text"] == newText
	})).Return(nil)
	
	// Mock the GetComment call that happens after update for cache invalidation
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)

	// Execute
	err := service.UpdateComment(ctx, testComment.ObjectId, req, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test UpdateComment with ownership validation failure
func TestUpdateComment_UnauthorizedUser_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	// Different user ID to simulate unauthorized access
	
	newText := "Updated comment text"
	req := &models.UpdateCommentRequest{
		ObjectId: testComment.ObjectId,
		Text:     newText,
	}

	// Setup mock expectations for ownership validation failure
	// fetchOwnedComment returns ErrCommentNotFound when document not found
	// When document is nil, Decode returns dbi.ErrNoDocuments
	// Pass nil as document and nil as error to create MockSingleResult{document: nil, err: nil}
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(nil, nil)

	// Execute
	err := service.UpdateComment(ctx, testComment.ObjectId, req, user)

	// Assert
	assert.Error(t, err)
	// fetchOwnedComment returns ErrCommentNotFound which wraps to "comment not found"
	assert.Contains(t, err.Error(), "not found")

	mockRepo.AssertExpectations(t)
}

// Test ValidateCommentOwnership with valid ownership
func TestValidateCommentOwnership_ValidOwner_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID

	// Setup mock expectations
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)

	// Execute
	err := service.ValidateCommentOwnership(ctx, testComment.ObjectId, user.UserID)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test ValidateCommentOwnership with invalid ownership
func TestValidateCommentOwnership_InvalidOwner_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	commentID := uuid.Must(uuid.NewV4())
	userID := uuid.Must(uuid.NewV4())

	// Setup mock expectations - simulate not found
	// fetchOwnedComment returns ErrCommentNotFound when document not found
	// When document is nil, Decode returns dbi.ErrNoDocuments
	// Pass nil as document and nil as error to create MockSingleResult{document: nil, err: nil}
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(nil, nil)

	// Execute
	err := service.ValidateCommentOwnership(ctx, commentID, userID)

	// Assert
	assert.Error(t, err)
	// ValidateCommentOwnership just returns the error from fetchOwnedComment
	assert.ErrorIs(t, err, commentsErrors.ErrCommentNotFound)

	mockRepo.AssertExpectations(t)
}

// Test IncrementScore with valid user
func TestIncrementScore_ValidUser_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	commentID := uuid.Must(uuid.NewV4())
	delta := 5

	// Setup mock expectations
	// IncrementScore -> IncrementFields doesn't call FindOne, it just builds a query and calls IncrementFields
	mockRepo.On("IncrementFields", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), mock.MatchedBy(func(increments map[string]interface{}) bool {
		return increments["score"] == delta
	})).Return(nil)

	// Execute
	err := service.IncrementScore(ctx, commentID, delta, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test IncrementScore with database error
func TestIncrementScore_DatabaseError_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	commentID := uuid.Must(uuid.NewV4())
	delta := 5

	// Setup mock expectations with error
	// IncrementScore -> IncrementFields doesn't call FindOne, it just builds a query and calls IncrementFields
	mockRepo.On("IncrementFields", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), mock.MatchedBy(func(increments map[string]interface{}) bool {
		return increments["score"] == delta
	})).Return(errors.New("database error"))

	// Execute
	err := service.IncrementScore(ctx, commentID, delta, user)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to increment fields")

	mockRepo.AssertExpectations(t)
}

// Test DeleteComment with valid ownership
func TestDeleteComment_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID
	postID := testComment.PostId

	// Setup mock expectations for ownership validation
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)
	
	// Setup mock expectations for delete
	mockRepo.On("Delete", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(nil)

	// Execute
	err := service.DeleteComment(ctx, testComment.ObjectId, postID, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test SoftDeleteComment with valid ownership
func TestSoftDeleteComment_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID

	// Setup mock expectations for ownership validation
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)
	
	// Setup mock expectations for update
	mockRepo.On("UpdateFields", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), mock.MatchedBy(func(updates map[string]interface{}) bool {
		deleted, hasDeleted := updates["deleted"]
		deletedDate, hasDeletedDate := updates["deletedDate"]
		lastUpdated, hasLastUpdated := updates["lastUpdated"]
		return hasDeleted && deleted == true && hasDeletedDate && deletedDate.(int64) > 0 && hasLastUpdated && lastUpdated.(int64) > 0
	})).Return(nil)
	
	// Setup mock expectations for GetComment call (used internally by SoftDeleteComment)
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)

	// Execute
	err := service.SoftDeleteComment(ctx, testComment.ObjectId, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test CreateIndex
func TestCreateIndex_ValidIndexes_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	indexes := map[string]interface{}{
		"objectId": 1,
		"postId":   1,
	}

	// Setup mock expectations
	mockRepo.On("CreateIndex", ctx, commentCollectionName, indexes).Return(nil)

	// Execute
	err := service.CreateIndex(ctx, indexes)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test CreateIndex with database error
func TestCreateIndex_DatabaseError_ReturnsError(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	indexes := map[string]interface{}{
		"objectId": 1,
	}

	// Setup mock expectations with error
	mockRepo.On("CreateIndex", ctx, commentCollectionName, indexes).Return(errors.New("index creation failed"))

	// Execute
	err := service.CreateIndex(ctx, indexes)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "index creation failed")

	mockRepo.AssertExpectations(t)
}

// Test UpdateCommentProfile
func TestUpdateCommentProfile_ValidParameters_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	displayName := "Updated Display Name"
	avatar := "https://example.com/new-avatar.jpg"

	// Setup mock expectations
	expectedUpdates := mock.MatchedBy(func(updates map[string]interface{}) bool {
		displayNameMatch := updates["ownerDisplayName"] == displayName
		avatarMatch := updates["ownerAvatar"] == avatar
		lastUpdatedMatch := false
		if lastUpdated, ok := updates["lastUpdated"]; ok {
			if _, ok := lastUpdated.(int64); ok {
				lastUpdatedMatch = true
			}
		}
		return displayNameMatch && avatarMatch && lastUpdatedMatch
	})
	mockRepo.On("UpdateFields", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), expectedUpdates).Return(nil)

	// Execute
	err := service.UpdateCommentProfile(ctx, userID, displayName, avatar)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test GetCommentsByPost with filter
func TestGetCommentsByPost_WithFilter_ReturnsResults(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	postID := uuid.Must(uuid.NewV4())
	
	filter := &models.CommentQueryFilter{
		PostId: &postID,
		Page:   1,
		Limit:  10,
	}

	testComments := []*models.Comment{createTestComment(), createTestComment()}
	interfaceComments := make([]interface{}, len(testComments))
	for i, comment := range testComments {
		interfaceComments[i] = comment
	}

	// Setup mock expectations
	expectedOptions := &interfaces.FindOptions{
		Limit: func() *int64 { l := int64(10); return &l }(),
		Skip:  func() *int64 { s := int64(0); return &s }(),
		Sort:  map[string]int{"created_date": -1},
	}
	
	// GetCommentsByPost calls QueryComments which calls getCommentCount
	mockRepo.On("Count", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(int64(2), nil)
	mockRepo.On("Find", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), expectedOptions).Return(interfaceComments, nil)

	// Execute
	result, err := service.GetCommentsByPost(ctx, postID, filter)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Comments, 2)
	assert.Equal(t, 1, result.Page)
	assert.Equal(t, 10, result.Limit)
	assert.False(t, result.HasMore)

	mockRepo.AssertExpectations(t)
}

// Test QueryComments with cursor pagination
func TestQueryCommentsWithCursor_ValidFilter_ReturnsResults(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	
	filter := &models.CommentQueryFilter{
		Limit: 5,
		Page:  1,
	}

	testComments := []*models.Comment{createTestComment(), createTestComment()}
	interfaceComments := make([]interface{}, len(testComments))
	for i, comment := range testComments {
		interfaceComments[i] = comment
	}

	// Setup mock expectations for cursor-based query (delegates to regular query)
	expectedOptions := &interfaces.FindOptions{
		Limit: func() *int64 { l := int64(5); return &l }(),
		Skip:  func() *int64 { s := int64(0); return &s }(),
		Sort:  map[string]int{"created_date": -1},
	}
	
	// QueryCommentsWithCursor calls QueryComments which calls getCommentCount
	mockRepo.On("Count", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(int64(2), nil)
	mockRepo.On("Find", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), expectedOptions).Return(interfaceComments, nil)

	// Execute
	result, err := service.QueryCommentsWithCursor(ctx, filter)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Comments, 2)

	mockRepo.AssertExpectations(t)
}

// Test UpdateComment with multiple fields
func TestUpdateComment_MultipleFields_UpdatesAllFields(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	testComment := createTestComment()
	testComment.OwnerUserId = user.UserID

	newText := "Updated text"

	req := &models.UpdateCommentRequest{
		ObjectId: testComment.ObjectId,
		Text:     newText,
	}

	// Setup mock expectations for ownership validation
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)

	// Setup mock expectations for GetComment call (used internally by UpdateComment for cache invalidation)
	mockRepo.On("FindOne", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(testComment, nil)

	// Setup mock expectations for update
	expectedUpdates := mock.MatchedBy(func(updates map[string]interface{}) bool {
		textMatch := updates["text"] == newText
		lastUpdatedMatch := false
		if lastUpdated, ok := updates["lastUpdated"]; ok {
			if _, ok := lastUpdated.(int64); ok {
				lastUpdatedMatch = true
			}
		}
		return textMatch && lastUpdatedMatch
	})
	mockRepo.On("UpdateFields", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), expectedUpdates).Return(nil)

	// Execute
	err := service.UpdateComment(ctx, testComment.ObjectId, req, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test business logic edge cases

// Test CreateComment with empty text (should pass service validation)
func TestCreateComment_EmptyText_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	req := createTestCreateCommentRequest()
	req.Text = "" // Empty text

	// Setup mock expectations
	mockRepo.On("Save", ctx, commentCollectionName, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.AnythingOfType("*models.Comment")).Return(nil)

	// Execute - service layer should not validate business rules, that's handler's job
	result, err := service.CreateComment(ctx, req, user)

	// Assert
	assert.NoError(t, err) // Service layer accepts empty text
	assert.NotNil(t, result)
	assert.Equal(t, "", result.Text)

	mockRepo.AssertExpectations(t)
}

// Test DeleteCommentsByPost
func TestDeleteCommentsByPost_ValidOwnership_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()
	postID := uuid.Must(uuid.NewV4())

	// Setup mock expectations for delete
	mockRepo.On("Delete", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(nil)

	// Execute
	err := service.DeleteCommentsByPost(ctx, postID, user)

	// Assert
	assert.NoError(t, err)

	mockRepo.AssertExpectations(t)
}

// Test GetCommentsByUser
func TestGetCommentsByUser_ValidUser_ReturnsComments(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	userID := uuid.Must(uuid.NewV4())
	
	filter := &models.CommentQueryFilter{
		OwnerUserId: &userID,
		Page:        1,
		Limit:       10,
	}

	testComments := []*models.Comment{createTestComment(), createTestComment()}
	interfaceComments := make([]interface{}, len(testComments))
	for i, comment := range testComments {
		interfaceComments[i] = comment
	}

	// Setup mock expectations
	expectedOptions := &interfaces.FindOptions{
		Limit: func() *int64 { l := int64(10); return &l }(),
		Skip:  func() *int64 { s := int64(0); return &s }(),
		Sort:  map[string]int{"created_date": -1},
	}
	
	// GetCommentsByUser calls QueryComments which calls getCommentCount
	mockRepo.On("Count", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query")).Return(int64(2), nil)
	mockRepo.On("Find", ctx, commentCollectionName, mock.AnythingOfType("*interfaces.Query"), expectedOptions).Return(interfaceComments, nil)

	// Execute
	result, err := service.GetCommentsByUser(ctx, userID, filter)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Comments, 2)

	mockRepo.AssertExpectations(t)
}

// Test concurrent validation scenarios
func TestCreateComment_ConcurrentRequests_Success(t *testing.T) {
	service, mockRepo := setupTestService()
	ctx := context.Background()
	user := createTestUserContext()

	// Simulate multiple concurrent requests
	numRequests := 5
	mockRepo.On("Save", ctx, commentCollectionName, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("int64"), mock.AnythingOfType("int64"), mock.AnythingOfType("*models.Comment")).Return(nil).Times(numRequests)

	// Execute concurrent requests
	done := make(chan bool, numRequests)
	for i := 0; i < numRequests; i++ {
		go func() {
			req := createTestCreateCommentRequest()
			result, err := service.CreateComment(ctx, req, user)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < numRequests; i++ {
		<-done
	}

	mockRepo.AssertExpectations(t)
}
