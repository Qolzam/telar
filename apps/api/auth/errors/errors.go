package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// Error codes for auth service
const (
	CodeValidationFailed     = "VALIDATION_FAILED"
	CodeMissingUserContext   = "MISSING_USER_CONTEXT"
	CodeInvalidRequest       = "INVALID_REQUEST"
	CodeInvalidUUID          = "INVALID_UUID"
	CodeMissingRequiredField = "MISSING_REQUIRED_FIELD"
	CodeInvalidFieldValue    = "INVALID_FIELD_VALUE"
	CodePermissionDenied     = "PERMISSION_DENIED"
	CodeDatabaseError        = "DATABASE_ERROR"
	CodeSystemError          = "SYSTEM_ERROR"
	CodeAuthenticationFailed = "AUTHENTICATION_FAILED"
	CodeUserNotFound         = "USER_NOT_FOUND"
	CodeInvalidCredentials   = "INVALID_CREDENTIALS"
	CodeTokenExpired         = "TOKEN_EXPIRED"
	CodeTokenInvalid         = "TOKEN_INVALID"
	CodeUserAlreadyExists    = "USER_ALREADY_EXISTS"
	CodeVerificationFailed   = "VERIFICATION_FAILED"
	CodeRateLimitExceeded    = "RATE_LIMIT_EXCEEDED"
)

// Auth service specific errors
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrUserAlreadyExists    = errors.New("user already exists")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrTokenExpired         = errors.New("token expired")
	ErrTokenInvalid         = errors.New("invalid token")
	ErrVerificationFailed   = errors.New("verification failed")
	ErrDatabaseError        = errors.New("database operation failed")
	ErrSystemError          = errors.New("system error occurred")
	ErrAuthenticationFailed = errors.New("authentication failed")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded")
)

// ErrorResponse represents the standardized error response format
// This model now perfectly matches the common.yaml schema
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"` // Use interface{} for flexibility
}

// AuthError represents an auth service error
type AuthError struct {
	Code    string
	Message string
	Details string
	Cause   error
}

// Error implements the error interface
func (e *AuthError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause error
func (e *AuthError) Unwrap() error {
	return e.Cause
}

// NewAuthError creates a new auth error
func NewAuthError(code, message string, cause error) *AuthError {
	return &AuthError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// HandleValidationError handles validation errors with 400 Bad Request
func HandleValidationError(c *fiber.Ctx, message string, details ...string) error {
	response := ErrorResponse{
		Code:    CodeValidationFailed,
		Message: message,
	}

	if len(details) > 0 {
		response.Details = details[0]
	}

	return c.Status(http.StatusBadRequest).JSON(response)
}

// HandleUserContextError handles user context errors with 400 Bad Request
func HandleUserContextError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeMissingUserContext,
		Message: message,
	})
}

// HandleInvalidRequestError handles invalid request errors with 400 Bad Request
func HandleInvalidRequestError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidRequest,
		Message: message,
	})
}

// HandleServiceError handles service errors and returns appropriate HTTP responses
func HandleServiceError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	// Check for specific error types
	switch {
	case errors.Is(err, ErrUserNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Code:    CodeUserNotFound,
			Message: "User not found",
		})
	case errors.Is(err, ErrInvalidCredentials):
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:    CodeInvalidCredentials,
			Message: "Invalid credentials",
		})
	case errors.Is(err, ErrUserAlreadyExists):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Code:    CodeUserAlreadyExists,
			Message: "User already exists",
		})
	case errors.Is(err, ErrPermissionDenied):
		return c.Status(http.StatusForbidden).JSON(ErrorResponse{
			Code:    CodePermissionDenied,
			Message: "Permission denied",
		})
	case errors.Is(err, ErrTokenExpired):
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:    CodeTokenExpired,
			Message: "Token expired",
		})
	case errors.Is(err, ErrTokenInvalid):
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:    CodeTokenInvalid,
			Message: "Invalid token",
		})
	case errors.Is(err, ErrVerificationFailed):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:    CodeVerificationFailed,
			Message: "Verification failed",
		})
	case errors.Is(err, ErrDatabaseError):
		return c.Status(http.StatusServiceUnavailable).JSON(ErrorResponse{
			Code:    CodeDatabaseError,
			Message: "Database operation failed",
		})
	case errors.Is(err, ErrSystemError):
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:    CodeSystemError,
			Message: "System error occurred",
		})
	default:
		// Generic internal server error
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:    CodeSystemError,
			Message: "An unexpected error occurred",
		})
	}
}

// HandleUUIDError handles UUID parsing errors with 400 Bad Request
func HandleUUIDError(c *fiber.Ctx, fieldName string) error {
	message := fmt.Sprintf("Invalid %s format", fieldName)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidUUID,
		Message: message,
	})
}

// HandleMissingFieldError handles missing required field errors with 400 Bad Request
func HandleMissingFieldError(c *fiber.Ctx, fieldName string) error {
	message := fmt.Sprintf("Missing required field: %s", fieldName)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeMissingRequiredField,
		Message: message,
	})
}

// HandleInvalidFieldError handles invalid field value errors with 400 Bad Request
func HandleInvalidFieldError(c *fiber.Ctx, fieldName string, reason string) error {
	message := fmt.Sprintf("Invalid %s: %s", fieldName, reason)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidFieldValue,
		Message: message,
	})
}

// HandlePermissionError handles permission errors with 403 Forbidden
func HandlePermissionError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusForbidden).JSON(ErrorResponse{
		Code:    CodePermissionDenied,
		Message: message,
	})
}

// HandleAuthenticationError handles authentication errors with 401 Unauthorized
func HandleAuthenticationError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
		Code:    CodeAuthenticationFailed,
		Message: message,
	})
}

// HandleUserNotFoundError handles user not found errors with 400 Bad Request
// This matches the original code behavior where "User not found!" returns 400
func HandleUserNotFoundError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeUserNotFound,
		Message: message,
	})
}

// HandleTokenError handles token-related errors with 401 Unauthorized
func HandleTokenError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
		Code:    CodeTokenInvalid,
		Message: message,
	})
}

// HandleTokenValidationError handles token validation errors with 400 Bad Request
// This is used for invalid tokens during validation, not authentication failures
func HandleTokenValidationError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeTokenInvalid,
		Message: message,
	})
}

// HandleUserExistsError handles user already exists errors with 409 Conflict
func HandleUserExistsError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusConflict).JSON(ErrorResponse{
		Code:    CodeUserAlreadyExists,
		Message: message,
	})
}

// HandleVerificationError handles verification errors with 400 Bad Request
func HandleVerificationError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeVerificationFailed,
		Message: message,
	})
}

// HandleRateLimitError handles rate limit errors with 429 Too Many Requests
func HandleRateLimitError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusTooManyRequests).JSON(ErrorResponse{
		Code:    CodeRateLimitExceeded,
		Message: message,
	})
}

// HandleDatabaseError handles database errors with 500 Internal Server Error
func HandleDatabaseError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
		Code:    CodeDatabaseError,
		Message: message,
	})
}

// HandleSystemError handles system errors with 500 Internal Server Error
func HandleSystemError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
		Code:    CodeSystemError,
		Message: message,
	})
}

// WrapDatabaseError wraps database errors
func WrapDatabaseError(err error) *AuthError {
	return NewAuthError(CodeDatabaseError, "Database operation failed", err)
}

// WrapValidationError wraps validation errors
func WrapValidationError(err error, details string) *AuthError {
	return &AuthError{
		Code:    CodeValidationFailed,
		Message: "Validation failed",
		Details: details,
		Cause:   err,
	}
}

// WrapSystemError wraps system errors
func WrapSystemError(err error) *AuthError {
	return NewAuthError(CodeSystemError, "System error occurred", err)
}

// WrapRequestError wraps request errors
func WrapRequestError(err error, details string) *AuthError {
	return NewAuthError(CodeInvalidRequest, "Invalid request", err)
}

// WrapAuthenticationError wraps authentication errors
func WrapAuthenticationError(err error) *AuthError {
	return NewAuthError(CodeAuthenticationFailed, "Authentication failed", err)
}

// WrapUserNotFoundError wraps user not found errors
func WrapUserNotFoundError(err error) *AuthError {
	return NewAuthError(CodeUserNotFound, "User not found", err)
}

// WrapTokenError wraps token errors
func WrapTokenError(err error) *AuthError {
	return NewAuthError(CodeTokenInvalid, "Token validation failed", err)
}

// WrapVerificationError wraps verification errors
func WrapVerificationError(err error) *AuthError {
	return NewAuthError(CodeVerificationFailed, "Verification failed", err)
}
