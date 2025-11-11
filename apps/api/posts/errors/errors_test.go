package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	postErrors "github.com/qolzam/telar/apps/api/posts/errors"
)

// Test PostError functionality
func TestPostError_Error(t *testing.T) {
	// Test PostError without cause
	err := postErrors.NewPostError("TEST_CODE", "Test message", nil)
	assert.Equal(t, "TEST_CODE: Test message", err.Error())

	// Test PostError with cause
	cause := errors.New("database connection failed")
	errWithCause := postErrors.NewPostError("DB_ERROR", "Database error", cause)
	assert.Contains(t, errWithCause.Error(), "DB_ERROR: Database error")
	assert.Contains(t, errWithCause.Error(), "database connection failed")
}

func TestPostError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := postErrors.NewPostError("TEST_CODE", "Test message", cause)
	
	unwrapped := errors.Unwrap(err)
	assert.Equal(t, cause, unwrapped)
}

// Test error wrapping functions
func TestWrapDatabaseError(t *testing.T) {
	originalErr := errors.New("connection timeout")
	wrappedErr := postErrors.WrapDatabaseError(originalErr)
	
	assert.Equal(t, postErrors.CodeDatabaseError, wrappedErr.Code)
	assert.Equal(t, "Database operation failed", wrappedErr.Message)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

func TestWrapValidationError(t *testing.T) {
	originalErr := errors.New("field required")
	details := "Body field is required"
	wrappedErr := postErrors.WrapValidationError(originalErr, details)
	
	assert.Equal(t, postErrors.CodeValidationFailed, wrappedErr.Code)
	assert.Equal(t, "Validation failed", wrappedErr.Message)
	assert.Equal(t, details, wrappedErr.Details)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

func TestWrapSystemError(t *testing.T) {
	originalErr := errors.New("memory allocation failed")
	wrappedErr := postErrors.WrapSystemError(originalErr)
	
	assert.Equal(t, postErrors.CodeSystemError, wrappedErr.Code)
	assert.Equal(t, "System error occurred", wrappedErr.Message)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

func TestWrapRequestError(t *testing.T) {
	originalErr := errors.New("malformed JSON")
	details := "Invalid JSON syntax"
	wrappedErr := postErrors.WrapRequestError(originalErr, details)
	
	assert.Equal(t, postErrors.CodeInvalidRequest, wrappedErr.Code)
	assert.Equal(t, "Invalid request", wrappedErr.Message)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

// Test error constants and codes
func TestErrorConstants(t *testing.T) {
	// Test that all error constants are defined
	assert.NotNil(t, postErrors.ErrPostNotFound)
	assert.NotNil(t, postErrors.ErrPostUnauthorized)
	assert.NotNil(t, postErrors.ErrInvalidPostData)
	assert.NotNil(t, postErrors.ErrPostAlreadyExists)
	assert.NotNil(t, postErrors.ErrPostOwnershipRequired)
	assert.NotNil(t, postErrors.ErrInvalidUserContext)
	assert.NotNil(t, postErrors.ErrDatabaseOperation)
	assert.NotNil(t, postErrors.ErrSystemError)
	assert.NotNil(t, postErrors.ErrServiceUnavailable)
}

func TestErrorCodes(t *testing.T) {
	// Test that all error codes are defined correctly
	assert.Equal(t, "POST_NOT_FOUND", postErrors.CodePostNotFound)
	assert.Equal(t, "UNAUTHORIZED", postErrors.CodeUnauthorized)
	assert.Equal(t, "INVALID_DATA", postErrors.CodeInvalidData)
	assert.Equal(t, "VALIDATION_FAILED", postErrors.CodeValidationFailed)
	assert.Equal(t, "DATABASE_ERROR", postErrors.CodeDatabaseError)
	assert.Equal(t, "INTERNAL_ERROR", postErrors.CodeInternalError)
	assert.Equal(t, "PERMISSION_DENIED", postErrors.CodePermissionDenied)
	assert.Equal(t, "ACCESS_FORBIDDEN", postErrors.CodeAccessForbidden)
}

// Test error message consistency
func TestErrorMessageConsistency(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Post not found",
			err:      postErrors.ErrPostNotFound,
			expected: "post not found",
		},
		{
			name:     "Unauthorized access",
			err:      postErrors.ErrPostUnauthorized,
			expected: "unauthorized access to post",
		},
		{
			name:     "Invalid post data",
			err:      postErrors.ErrInvalidPostData,
			expected: "invalid post data",
		},
		{
			name:     "Permission denied",
			err:      postErrors.ErrPermissionDenied,
			expected: "permission denied",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, tc.err.Error(), tc.expected)
		})
	}
}

// Test error response format
func TestErrorResponseFormat(t *testing.T) {
	response := postErrors.ErrorResponse{
		Code:    "TEST_CODE",
		Message: "test message",
		Details: "test details",
	}

	assert.Equal(t, "TEST_CODE", response.Code)
	assert.Equal(t, "test message", response.Message)
	assert.Equal(t, "test details", response.Details)
}

// Test error chaining and context preservation
func TestErrorChaining(t *testing.T) {
	// Create a chain of errors
	originalErr := errors.New("database timeout")
	dbErr := postErrors.WrapDatabaseError(originalErr)
	
	// Test that we can unwrap to find the original error
	assert.True(t, errors.Is(dbErr, originalErr))
	
	// Test error message includes context
	assert.Contains(t, dbErr.Error(), "Database operation failed")
	assert.Contains(t, dbErr.Error(), "database timeout")
}

// Test error type checking
func TestErrorTypeChecking(t *testing.T) {
	testCases := []struct {
		name      string
		err       error
		checkFunc func(error) bool
	}{
		{
			name: "Post not found error",
			err:  postErrors.ErrPostNotFound,
			checkFunc: func(err error) bool {
				return errors.Is(err, postErrors.ErrPostNotFound)
			},
		},
		{
			name: "Unauthorized error",
			err:  postErrors.ErrPostUnauthorized,
			checkFunc: func(err error) bool {
				return errors.Is(err, postErrors.ErrPostUnauthorized)
			},
		},
		{
			name: "Database operation error",
			err:  postErrors.ErrDatabaseOperation,
			checkFunc: func(err error) bool {
				return errors.Is(err, postErrors.ErrDatabaseOperation)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.True(t, tc.checkFunc(tc.err))
		})
	}
}

// Test error creation with various parameters
func TestErrorCreationVariations(t *testing.T) {
	// Test with empty message
	err1 := postErrors.NewPostError("TEST", "", nil)
	assert.Equal(t, "TEST: ", err1.Error())

	// Test with empty code
	err2 := postErrors.NewPostError("", "message", nil)
	assert.Equal(t, ": message", err2.Error())

	// Test with both empty
	err3 := postErrors.NewPostError("", "", nil)
	assert.Equal(t, ": ", err3.Error())
}

// Test error with detailed context
func TestErrorWithDetailedContext(t *testing.T) {
	originalErr := errors.New("field validation failed")
	detailedErr := postErrors.WrapValidationError(originalErr, "The 'body' field must be between 1 and 10000 characters")
	
	assert.Contains(t, detailedErr.Error(), "Validation failed")
	assert.Contains(t, detailedErr.Error(), "field validation failed")
	assert.Equal(t, "The 'body' field must be between 1 and 10000 characters", detailedErr.Details)
}

// Test multiple error wrapping levels
func TestMultipleErrorWrapping(t *testing.T) {
	// Create a chain: original -> database -> system
	originalErr := errors.New("connection refused")
	dbErr := postErrors.WrapDatabaseError(originalErr)
	systemErr := postErrors.WrapSystemError(dbErr)
	
	// Check that we can find the original error through the chain
	assert.True(t, errors.Is(systemErr, originalErr))
	assert.True(t, errors.Is(systemErr, dbErr))
	
	// Check that unwrapping works correctly
	unwrapped := errors.Unwrap(systemErr)
	assert.Equal(t, dbErr, unwrapped)
}

// Test concurrent error creation
func TestConcurrentErrorCreation(t *testing.T) {
	// Test that error creation is thread-safe
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func(id int) {
			err := postErrors.NewPostError("CONCURRENT_TEST", "message", nil)
			assert.NotNil(t, err)
			assert.Contains(t, err.Error(), "CONCURRENT_TEST")
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

// Test error comparison and equality
func TestErrorComparison(t *testing.T) {
	// Same errors should be equal
	err1 := postErrors.ErrPostNotFound
	err2 := postErrors.ErrPostNotFound
	assert.True(t, errors.Is(err1, err2))

	// Different errors should not be equal
	err3 := postErrors.ErrPostUnauthorized
	assert.False(t, errors.Is(err1, err3))

	// Wrapped errors should maintain identity
	wrappedErr := postErrors.WrapDatabaseError(postErrors.ErrPostNotFound)
	assert.True(t, errors.Is(wrappedErr, postErrors.ErrPostNotFound))
}

// Test error edge cases
func TestErrorEdgeCases(t *testing.T) {
	// Test nil cause
	err := postErrors.NewPostError("TEST", "message", nil)
	assert.Nil(t, errors.Unwrap(err))

	// Test self-referencing (should not cause infinite loop)
	selfErr := postErrors.NewPostError("SELF", "self reference", nil)
	nonSelfErr := postErrors.NewPostError("NON_SELF", "different", selfErr)
	assert.Equal(t, selfErr, errors.Unwrap(nonSelfErr))
}

// Test large error messages
func TestLargeErrorMessages(t *testing.T) {
	largeMessage := string(make([]byte, 10000)) // 10KB message
	err := postErrors.NewPostError("LARGE", largeMessage, nil)
	
	assert.Contains(t, err.Error(), "LARGE")
	assert.Contains(t, err.Error(), largeMessage)
}

// Test error with special characters
func TestErrorWithSpecialCharacters(t *testing.T) {
	specialMessage := "Error with special chars: 💀☠️🔥💯 and unicode: 你好世界"
	err := postErrors.NewPostError("SPECIAL", specialMessage, nil)
	
	assert.Contains(t, err.Error(), "SPECIAL")
	assert.Contains(t, err.Error(), specialMessage)
}
