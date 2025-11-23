package errors_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	commentErrors "github.com/qolzam/telar/apps/api/comments/errors"
)

// Test CommentError functionality
func TestCommentError_Error(t *testing.T) {
	// Test CommentError without cause
	err := commentErrors.NewCommentError("TEST_CODE", "Test message", nil)
	assert.Equal(t, "TEST_CODE: Test message", err.Error())

	// Test CommentError with cause
	cause := errors.New("database connection failed")
	errWithCause := commentErrors.NewCommentError("DB_ERROR", "Database error", cause)
	assert.Contains(t, errWithCause.Error(), "DB_ERROR: Database error")
	assert.Contains(t, errWithCause.Error(), "database connection failed")
}

func TestCommentError_Unwrap(t *testing.T) {
	cause := errors.New("original error")
	err := commentErrors.NewCommentError("TEST_CODE", "Test message", cause)

	unwrapped := errors.Unwrap(err)
	assert.Equal(t, cause, unwrapped)
}

// Test error wrapping functions
func TestWrapDatabaseError(t *testing.T) {
	originalErr := errors.New("connection timeout")
	wrappedErr := commentErrors.WrapDatabaseError(originalErr)

	assert.Equal(t, commentErrors.CodeDatabaseError, wrappedErr.Code)
	assert.Equal(t, "Database operation failed", wrappedErr.Message)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

func TestWrapValidationError(t *testing.T) {
	originalErr := errors.New("field required")
	details := "Text field is required"
	wrappedErr := commentErrors.WrapValidationError(originalErr, details)

	assert.Equal(t, commentErrors.CodeValidationFailed, wrappedErr.Code)
	assert.Equal(t, "Validation failed", wrappedErr.Message)
	assert.Equal(t, details, wrappedErr.Details)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

func TestWrapSystemError(t *testing.T) {
	originalErr := errors.New("memory allocation failed")
	wrappedErr := commentErrors.WrapSystemError(originalErr)

	assert.Equal(t, commentErrors.CodeSystemError, wrappedErr.Code)
	assert.Equal(t, "System error occurred", wrappedErr.Message)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

func TestWrapRequestError(t *testing.T) {
	originalErr := errors.New("malformed JSON")
	details := "Invalid JSON syntax"
	wrappedErr := commentErrors.WrapRequestError(originalErr, details)

	assert.Equal(t, commentErrors.CodeInvalidRequest, wrappedErr.Code)
	assert.Equal(t, "Invalid request", wrappedErr.Message)
	assert.Equal(t, originalErr, wrappedErr.Cause)
}

// Test error constants and codes
func TestErrorConstants(t *testing.T) {
	// Test that all error constants are defined
	assert.NotNil(t, commentErrors.ErrCommentNotFound)
	assert.NotNil(t, commentErrors.ErrCommentUnauthorized)
	assert.NotNil(t, commentErrors.ErrInvalidCommentData)
	assert.NotNil(t, commentErrors.ErrCommentAlreadyExists)
	assert.NotNil(t, commentErrors.ErrCommentOwnershipRequired)
	assert.NotNil(t, commentErrors.ErrInvalidUserContext)
	assert.NotNil(t, commentErrors.ErrDatabaseOperation)
	assert.NotNil(t, commentErrors.ErrSystemError)
	assert.NotNil(t, commentErrors.ErrServiceUnavailable)

	// Test request and validation errors
	assert.NotNil(t, commentErrors.ErrInvalidRequest)
	assert.NotNil(t, commentErrors.ErrInvalidUUID)
	assert.NotNil(t, commentErrors.ErrMissingUserContext)
	assert.NotNil(t, commentErrors.ErrValidationFailed)
	assert.NotNil(t, commentErrors.ErrPermissionDenied)
	assert.NotNil(t, commentErrors.ErrAccessForbidden)
	assert.NotNil(t, commentErrors.ErrInvalidRequestBody)
	assert.NotNil(t, commentErrors.ErrMissingRequiredField)
	assert.NotNil(t, commentErrors.ErrInvalidFieldValue)
}

func TestErrorCodes(t *testing.T) {
	// Test that all error codes are defined correctly
	assert.Equal(t, "COMMENT_NOT_FOUND", commentErrors.CodeCommentNotFound)
	assert.Equal(t, "UNAUTHORIZED", commentErrors.CodeUnauthorized)
	assert.Equal(t, "INVALID_DATA", commentErrors.CodeInvalidData)
	assert.Equal(t, "VALIDATION_FAILED", commentErrors.CodeValidationFailed)
	assert.Equal(t, "DATABASE_ERROR", commentErrors.CodeDatabaseError)
	assert.Equal(t, "INTERNAL_ERROR", commentErrors.CodeInternalError)
	assert.Equal(t, "PERMISSION_DENIED", commentErrors.CodePermissionDenied)
	assert.Equal(t, "ACCESS_FORBIDDEN", commentErrors.CodeAccessForbidden)

	// Test request and validation codes
	assert.Equal(t, "INVALID_REQUEST", commentErrors.CodeInvalidRequest)
	assert.Equal(t, "INVALID_UUID", commentErrors.CodeInvalidUUID)
	assert.Equal(t, "MISSING_USER_CONTEXT", commentErrors.CodeMissingUserContext)
	assert.Equal(t, "INVALID_REQUEST_BODY", commentErrors.CodeInvalidRequestBody)
	assert.Equal(t, "MISSING_REQUIRED_FIELD", commentErrors.CodeMissingRequiredField)
	assert.Equal(t, "INVALID_FIELD_VALUE", commentErrors.CodeInvalidFieldValue)

	// Test database and system codes
	assert.Equal(t, "DATABASE_OPERATION_FAILED", commentErrors.CodeDatabaseOperation)
	assert.Equal(t, "SYSTEM_ERROR", commentErrors.CodeSystemError)
	assert.Equal(t, "SERVICE_UNAVAILABLE", commentErrors.CodeServiceUnavailable)
}

// Test error message consistency
func TestErrorMessageConsistency(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "Comment not found",
			err:      commentErrors.ErrCommentNotFound,
			expected: "comment not found",
		},
		{
			name:     "Unauthorized access",
			err:      commentErrors.ErrCommentUnauthorized,
			expected: "unauthorized access to comment",
		},
		{
			name:     "Invalid comment data",
			err:      commentErrors.ErrInvalidCommentData,
			expected: "invalid comment data",
		},
		{
			name:     "Permission denied",
			err:      commentErrors.ErrPermissionDenied,
			expected: "permission denied",
		},
		{
			name:     "Comment ownership required",
			err:      commentErrors.ErrCommentOwnershipRequired,
			expected: "comment ownership required",
		},
		{
			name:     "Invalid user context",
			err:      commentErrors.ErrInvalidUserContext,
			expected: "invalid user context",
		},
		{
			name:     "Missing user context",
			err:      commentErrors.ErrMissingUserContext,
			expected: "missing user context",
		},
		{
			name:     "Validation failed",
			err:      commentErrors.ErrValidationFailed,
			expected: "validation failed",
		},
		{
			name:     "Invalid request",
			err:      commentErrors.ErrInvalidRequest,
			expected: "invalid request",
		},
		{
			name:     "Invalid UUID",
			err:      commentErrors.ErrInvalidUUID,
			expected: "invalid UUID format",
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
	response := commentErrors.ErrorResponse{
		Message: "test message",
		Code:    "TEST_CODE",
		Details: "test details",
	}

	assert.Equal(t, "test message", response.Message)
	assert.Equal(t, "TEST_CODE", response.Code)
	assert.Equal(t, "test details", response.Details)
}

// Test error chaining and context preservation
func TestErrorChaining(t *testing.T) {
	// Create a chain of errors
	originalErr := errors.New("database timeout")
	dbErr := commentErrors.WrapDatabaseError(originalErr)

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
			name: "Comment not found error",
			err:  commentErrors.ErrCommentNotFound,
			checkFunc: func(err error) bool {
				return errors.Is(err, commentErrors.ErrCommentNotFound)
			},
		},
		{
			name: "Unauthorized error",
			err:  commentErrors.ErrCommentUnauthorized,
			checkFunc: func(err error) bool {
				return errors.Is(err, commentErrors.ErrCommentUnauthorized)
			},
		},
		{
			name: "Database operation error",
			err:  commentErrors.ErrDatabaseOperation,
			checkFunc: func(err error) bool {
				return errors.Is(err, commentErrors.ErrDatabaseOperation)
			},
		},
		{
			name: "Validation failed error",
			err:  commentErrors.ErrValidationFailed,
			checkFunc: func(err error) bool {
				return errors.Is(err, commentErrors.ErrValidationFailed)
			},
		},
		{
			name: "Permission denied error",
			err:  commentErrors.ErrPermissionDenied,
			checkFunc: func(err error) bool {
				return errors.Is(err, commentErrors.ErrPermissionDenied)
			},
		},
		{
			name: "Comment ownership required error",
			err:  commentErrors.ErrCommentOwnershipRequired,
			checkFunc: func(err error) bool {
				return errors.Is(err, commentErrors.ErrCommentOwnershipRequired)
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
	err1 := commentErrors.NewCommentError("TEST", "", nil)
	assert.Equal(t, "TEST: ", err1.Error())

	// Test with empty code
	err2 := commentErrors.NewCommentError("", "message", nil)
	assert.Equal(t, ": message", err2.Error())

	// Test with both empty
	err3 := commentErrors.NewCommentError("", "", nil)
	assert.Equal(t, ": ", err3.Error())
}

// Test error with detailed context
func TestErrorWithDetailedContext(t *testing.T) {
	originalErr := errors.New("field validation failed")
	detailedErr := commentErrors.WrapValidationError(originalErr, "The 'text' field must be between 1 and 1000 characters")

	assert.Contains(t, detailedErr.Error(), "Validation failed")
	assert.Contains(t, detailedErr.Error(), "field validation failed")
	assert.Equal(t, "The 'text' field must be between 1 and 1000 characters", detailedErr.Details)
}

// Test multiple error wrapping levels
func TestMultipleErrorWrapping(t *testing.T) {
	// Create a chain: original -> database -> system
	originalErr := errors.New("connection refused")
	dbErr := commentErrors.WrapDatabaseError(originalErr)
	systemErr := commentErrors.WrapSystemError(dbErr)

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
			err := commentErrors.NewCommentError("CONCURRENT_TEST", "message", nil)
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
	err1 := commentErrors.ErrCommentNotFound
	err2 := commentErrors.ErrCommentNotFound
	assert.True(t, errors.Is(err1, err2))

	// Different errors should not be equal
	err3 := commentErrors.ErrCommentUnauthorized
	assert.False(t, errors.Is(err1, err3))

	// Wrapped errors should maintain identity
	wrappedErr := commentErrors.WrapDatabaseError(commentErrors.ErrCommentNotFound)
	assert.True(t, errors.Is(wrappedErr, commentErrors.ErrCommentNotFound))
}

// Test error edge cases
func TestErrorEdgeCases(t *testing.T) {
	// Test nil cause
	err := commentErrors.NewCommentError("TEST", "message", nil)
	assert.Nil(t, errors.Unwrap(err))

	// Test self-referencing (should not cause infinite loop)
	selfErr := commentErrors.NewCommentError("SELF", "self reference", nil)
	nonSelfErr := commentErrors.NewCommentError("NON_SELF", "different", selfErr)
	assert.Equal(t, selfErr, errors.Unwrap(nonSelfErr))
}

// Test large error messages
func TestLargeErrorMessages(t *testing.T) {
	largeMessage := string(make([]byte, 10000)) // 10KB message
	err := commentErrors.NewCommentError("LARGE", largeMessage, nil)

	assert.Contains(t, err.Error(), "LARGE")
	assert.Contains(t, err.Error(), largeMessage)
}

// Test error with special characters
func TestErrorWithSpecialCharacters(t *testing.T) {
	specialMessage := "Error with special chars: 💬📝🔥💯 and unicode: 你好世界"
	err := commentErrors.NewCommentError("SPECIAL", specialMessage, nil)

	assert.Contains(t, err.Error(), "SPECIAL")
	assert.Contains(t, err.Error(), specialMessage)
}

// Test comment-specific error scenarios
func TestCommentSpecificErrors(t *testing.T) {
	testCases := []struct {
		name     string
		err      error
		code     string
		contains string
	}{
		{
			name:     "Comment already exists",
			err:      commentErrors.ErrCommentAlreadyExists,
			code:     "DUPLICATE_KEY", // Based on HandleServiceError function
			contains: "comment already exists",
		},
		{
			name:     "Invalid request body",
			err:      commentErrors.ErrInvalidRequestBody,
			code:     commentErrors.CodeInvalidRequestBody,
			contains: "invalid request body",
		},
		{
			name:     "Missing required field",
			err:      commentErrors.ErrMissingRequiredField,
			code:     commentErrors.CodeMissingRequiredField,
			contains: "missing required field",
		},
		{
			name:     "Invalid field value",
			err:      commentErrors.ErrInvalidFieldValue,
			code:     commentErrors.CodeInvalidFieldValue,
			contains: "invalid field value",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Contains(t, tc.err.Error(), tc.contains)
		})
	}
}

// Test validation error details
func TestValidationErrorDetails(t *testing.T) {
	originalErr := errors.New("text length validation failed")
	details := "Comment text must be between 1 and 1000 characters"

	wrappedErr := commentErrors.WrapValidationError(originalErr, details)

	// Test error structure
	assert.Equal(t, commentErrors.CodeValidationFailed, wrappedErr.Code)
	assert.Equal(t, "Validation failed", wrappedErr.Message)
	assert.Equal(t, details, wrappedErr.Details)
	assert.Equal(t, originalErr, wrappedErr.Cause)

	// Test error message format
	assert.Contains(t, wrappedErr.Error(), "Validation failed")
	assert.Contains(t, wrappedErr.Error(), "text length validation failed")
}

// Test authorization error scenarios
func TestAuthorizationErrors(t *testing.T) {
	testCases := []struct {
		name           string
		err            error
		expectedCode   string
		expectedStatus string
	}{
		{
			name:           "Unauthorized access",
			err:            commentErrors.ErrCommentUnauthorized,
			expectedCode:   commentErrors.CodeUnauthorized,
			expectedStatus: "401",
		},
		{
			name:           "Permission denied",
			err:            commentErrors.ErrPermissionDenied,
			expectedCode:   commentErrors.CodePermissionDenied,
			expectedStatus: "403",
		},
		{
			name:           "Access forbidden",
			err:            commentErrors.ErrAccessForbidden,
			expectedCode:   commentErrors.CodeAccessForbidden,
			expectedStatus: "403",
		},
		{
			name:           "Comment ownership required",
			err:            commentErrors.ErrCommentOwnershipRequired,
			expectedCode:   commentErrors.CodePermissionDenied,
			expectedStatus: "403",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.NotNil(t, tc.err)
			// Check that error message contains some expected content
			errorMsg := tc.err.Error()
			assert.NotEmpty(t, errorMsg)
		})
	}
}

// Test database error scenarios
func TestDatabaseErrorScenarios(t *testing.T) {
	// Test various database error scenarios
	dbErrors := []error{
		errors.New("connection pool exhausted"),
		errors.New("query timeout"),
		errors.New("deadlock detected"),
		errors.New("constraint violation"),
		errors.New("table not found"),
	}

	for _, dbErr := range dbErrors {
		wrappedErr := commentErrors.WrapDatabaseError(dbErr)

		assert.Equal(t, commentErrors.CodeDatabaseError, wrappedErr.Code)
		assert.Equal(t, "Database operation failed", wrappedErr.Message)
		assert.Equal(t, dbErr, wrappedErr.Cause)
		assert.True(t, errors.Is(wrappedErr, dbErr))
	}
}

// Test system error scenarios
func TestSystemErrorScenarios(t *testing.T) {
	// Test various system error scenarios
	systemErrors := []error{
		errors.New("out of memory"),
		errors.New("disk full"),
		errors.New("network unreachable"),
		errors.New("service unavailable"),
		errors.New("timeout exceeded"),
	}

	for _, sysErr := range systemErrors {
		wrappedErr := commentErrors.WrapSystemError(sysErr)

		assert.Equal(t, commentErrors.CodeSystemError, wrappedErr.Code)
		assert.Equal(t, "System error occurred", wrappedErr.Message)
		assert.Equal(t, sysErr, wrappedErr.Cause)
		assert.True(t, errors.Is(wrappedErr, sysErr))
	}
}

// Test request error scenarios
func TestRequestErrorScenarios(t *testing.T) {
	// Test various request error scenarios
	requestErrors := map[string]error{
		"malformed JSON":       errors.New("invalid character '}' looking for beginning of value"),
		"missing content type": errors.New("Content-Type header is missing"),
		"unsupported method":   errors.New("method not allowed"),
		"invalid encoding":     errors.New("unsupported content encoding"),
	}

	for desc, reqErr := range requestErrors {
		wrappedErr := commentErrors.WrapRequestError(reqErr, desc)

		assert.Equal(t, commentErrors.CodeInvalidRequest, wrappedErr.Code)
		assert.Equal(t, "Invalid request", wrappedErr.Message)
		assert.Equal(t, reqErr, wrappedErr.Cause)
		assert.True(t, errors.Is(wrappedErr, reqErr))
	}
}
