package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// Comment service specific errors
var (
	ErrCommentNotFound          = errors.New("comment not found")
	ErrCommentUnauthorized      = errors.New("unauthorized access to comment")
	ErrInvalidCommentData       = errors.New("invalid comment data")
	ErrCommentAlreadyExists     = errors.New("comment already exists")
	ErrCommentOwnershipRequired = errors.New("comment ownership required")
	ErrInvalidUserContext       = errors.New("invalid user context")
	ErrUserNotFound             = errors.New("user does not exist")
	ErrPostNotFound             = errors.New("post does not exist")

	// Request and validation errors
	ErrInvalidRequest       = errors.New("invalid request")
	ErrInvalidUUID          = errors.New("invalid UUID format")
	ErrMissingUserContext   = errors.New("missing user context")
	ErrValidationFailed     = errors.New("validation failed")
	ErrPermissionDenied     = errors.New("permission denied")
	ErrAccessForbidden      = errors.New("access forbidden")
	ErrInvalidRequestBody   = errors.New("invalid request body")
	ErrMissingRequiredField = errors.New("missing required field")
	ErrInvalidFieldValue    = errors.New("invalid field value")

	// Database and system errors
	ErrDatabaseOperation  = errors.New("database operation failed")
	ErrSystemError        = errors.New("system error occurred")
	ErrServiceUnavailable = errors.New("service temporarily unavailable")
)

// CommentError represents a comment service error with additional context
type CommentError struct {
	Code    string
	Message string
	Details string
	Cause   error
}

func (e *CommentError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *CommentError) Unwrap() error {
	return e.Cause
}

// NewCommentError creates a new CommentError
func NewCommentError(code, message string, cause error) *CommentError {
	return &CommentError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error codes
const (
	CodeCommentNotFound  = "COMMENT_NOT_FOUND"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeInvalidData      = "INVALID_DATA"
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeDatabaseError    = "DATABASE_ERROR"
	CodeInternalError    = "INTERNAL_ERROR"

	// Request and validation codes
	CodeInvalidRequest       = "INVALID_REQUEST"
	CodeInvalidUUID          = "INVALID_UUID"
	CodeMissingUserContext   = "MISSING_USER_CONTEXT"
	CodePermissionDenied     = "PERMISSION_DENIED"
	CodeAccessForbidden      = "ACCESS_FORBIDDEN"
	CodeInvalidRequestBody   = "INVALID_REQUEST_BODY"
	CodeMissingRequiredField = "MISSING_REQUIRED_FIELD"
	CodeInvalidFieldValue    = "INVALID_FIELD_VALUE"

	// Database and system codes
	CodeDatabaseOperation  = "DATABASE_OPERATION_FAILED"
	CodeSystemError        = "SYSTEM_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

// ErrorResponse represents the standardized error response format
// This model now perfectly matches the common.yaml schema
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"` // Use interface{} for flexibility
}

// HandleServiceError handles service errors and returns appropriate HTTP responses
func HandleServiceError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	// Check for specific error types
	switch {
	case errors.Is(err, ErrCommentNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Code:    CodeCommentNotFound,
			Message: "Comment not found",
			Details: err.Error(),
		})
	case errors.Is(err, ErrCommentUnauthorized):
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:    CodeUnauthorized,
			Message: "Unauthorized access",
			Details: err.Error(),
		})
	case errors.Is(err, ErrUserNotFound):
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:    CodeUnauthorized,
			Message: "User not found (stale authentication token)",
			Details: err.Error(),
		})
	case errors.Is(err, ErrPostNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Code:    CodeCommentNotFound,
			Message: "Post not found",
			Details: err.Error(),
		})
	case errors.Is(err, ErrCommentAlreadyExists):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Code:    "DUPLICATE_KEY",
			Message: "Comment already exists",
			Details: err.Error(),
		})
	case errors.Is(err, ErrCommentOwnershipRequired):
		return c.Status(http.StatusForbidden).JSON(ErrorResponse{
			Code:    CodePermissionDenied,
			Message: "Comment ownership required",
			Details: err.Error(),
		})
	case errors.Is(err, ErrPermissionDenied):
		return c.Status(http.StatusForbidden).JSON(ErrorResponse{
			Code:    CodePermissionDenied,
			Message: "Permission denied",
			Details: err.Error(),
		})
	case errors.Is(err, ErrAccessForbidden):
		return c.Status(http.StatusForbidden).JSON(ErrorResponse{
			Code:    CodeAccessForbidden,
			Message: "Access forbidden",
			Details: err.Error(),
		})
	case errors.Is(err, ErrDatabaseOperation):
		return c.Status(http.StatusServiceUnavailable).JSON(ErrorResponse{
			Code:    CodeDatabaseOperation,
			Message: "Database operation failed",
			Details: err.Error(),
		})
	case errors.Is(err, ErrServiceUnavailable):
		return c.Status(http.StatusServiceUnavailable).JSON(ErrorResponse{
			Code:    CodeServiceUnavailable,
			Message: "Service temporarily unavailable",
			Details: err.Error(),
		})
	default:
		// Generic internal server error
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:    CodeInternalError,
			Message: "An unexpected error occurred",
			Details: err.Error(),
		})
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

// HandleUserContextError returns an error for invalid user context
func HandleUserContextError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
		Code:    "UNAUTHORIZED",
		Message: message,
	})
}

// HandleForbiddenError returns an error for forbidden access
func HandleForbiddenError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusForbidden).JSON(ErrorResponse{
		Code:    "FORBIDDEN",
		Message: message,
	})
}

// HandleInvalidRequestError handles invalid request errors with 400 Bad Request
func HandleInvalidRequestError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidRequest,
		Message: message,
		Details: message,
	})
}

// HandleUUIDError handles UUID parsing errors with 400 Bad Request
func HandleUUIDError(c *fiber.Ctx, fieldName string) error {
	message := fmt.Sprintf("Invalid %s format", fieldName)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidUUID,
		Message: message,
		Details: message,
	})
}

// HandleMissingFieldError handles missing required field errors with 400 Bad Request
func HandleMissingFieldError(c *fiber.Ctx, fieldName string) error {
	message := fmt.Sprintf("Missing required field: %s", fieldName)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeMissingRequiredField,
		Message: message,
		Details: message,
	})
}

// HandleInvalidFieldError handles invalid field value errors with 400 Bad Request
func HandleInvalidFieldError(c *fiber.Ctx, fieldName string, reason string) error {
	message := fmt.Sprintf("Invalid %s: %s", fieldName, reason)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidFieldValue,
		Message: message,
		Details: message,
	})
}

// HandlePermissionError handles permission errors with 403 Forbidden
func HandlePermissionError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusForbidden).JSON(ErrorResponse{
		Code:    CodePermissionDenied,
		Message: message,
		Details: message,
	})
}

// WrapDatabaseError wraps database errors
func WrapDatabaseError(err error) *CommentError {
	return NewCommentError(CodeDatabaseError, "Database operation failed", err)
}

// WrapValidationError wraps validation errors
func WrapValidationError(err error, details string) *CommentError {
	return &CommentError{
		Code:    CodeValidationFailed,
		Message: "Validation failed",
		Details: details,
		Cause:   err,
	}
}

// WrapSystemError wraps system errors
func WrapSystemError(err error) *CommentError {
	return NewCommentError(CodeSystemError, "System error occurred", err)
}

// WrapRequestError wraps request errors
func WrapRequestError(err error, details string) *CommentError {
	return NewCommentError(CodeInvalidRequest, "Invalid request", err)
}
