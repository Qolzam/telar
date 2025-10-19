package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrProfileNotFound          = errors.New("profile not found")
	ErrProfileUnauthorized      = errors.New("unauthorized access to profile")
	ErrInvalidProfileData       = errors.New("invalid profile data")
	ErrProfileAlreadyExists     = errors.New("profile already exists")
	ErrProfileOwnershipRequired = errors.New("profile ownership required")
	ErrInvalidUserContext       = errors.New("invalid user context")
	ErrInvalidRequest           = errors.New("invalid request")
	ErrInvalidUUID              = errors.New("invalid UUID format")
	ErrMissingUserContext       = errors.New("missing user context")
	ErrValidationFailed         = errors.New("validation failed")
	ErrPermissionDenied         = errors.New("permission denied")
	ErrAccessForbidden          = errors.New("access forbidden")
	ErrInvalidRequestBody       = errors.New("invalid request body")
	ErrMissingRequiredField     = errors.New("missing required field")
	ErrInvalidFieldValue        = errors.New("invalid field value")
	ErrDatabaseOperation        = errors.New("database operation failed")
	ErrSystemError              = errors.New("system error occurred")
	ErrServiceUnavailable       = errors.New("service temporarily unavailable")
)

type ProfileError struct {
	Code    string
	Message string
	Details string
	Cause   error
}

func (e *ProfileError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *ProfileError) Unwrap() error {
	return e.Cause
}

func NewProfileError(code, message string, cause error) *ProfileError {
	return &ProfileError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

const (
	CodeProfileNotFound  = "PROFILE_NOT_FOUND"
	CodeUnauthorized     = "UNAUTHORIZED"
	CodeInvalidData      = "INVALID_DATA"
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeDatabaseError    = "DATABASE_ERROR"
	CodeInternalError    = "INTERNAL_ERROR"

	CodeInvalidRequest       = "INVALID_REQUEST"
	CodeInvalidUUID          = "INVALID_UUID"
	CodeMissingUserContext   = "MISSING_USER_CONTEXT"
	CodePermissionDenied     = "PERMISSION_DENIED"
	CodeAccessForbidden      = "ACCESS_FORBIDDEN"
	CodeInvalidRequestBody   = "INVALID_REQUEST_BODY"
	CodeMissingRequiredField = "MISSING_REQUIRED_FIELD"
	CodeInvalidFieldValue    = "INVALID_FIELD_VALUE"

	CodeDatabaseOperation  = "DATABASE_OPERATION_FAILED"
	CodeSystemError        = "SYSTEM_ERROR"
	CodeServiceUnavailable = "SERVICE_UNAVAILABLE"
)

type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

func HandleServiceError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrProfileNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Code:    CodeProfileNotFound,
			Message: "Profile not found",
			Details: err.Error(),
		})
	case errors.Is(err, ErrProfileUnauthorized):
		return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
			Code:    CodeUnauthorized,
			Message: "Unauthorized access",
			Details: err.Error(),
		})
	case errors.Is(err, ErrProfileAlreadyExists):
		return c.Status(http.StatusConflict).JSON(ErrorResponse{
			Code:    "DUPLICATE_KEY",
			Message: "Profile already exists",
			Details: err.Error(),
		})
	case errors.Is(err, ErrProfileOwnershipRequired):
		return c.Status(http.StatusForbidden).JSON(ErrorResponse{
			Code:    CodePermissionDenied,
			Message: "Profile ownership required",
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
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:    CodeInternalError,
			Message: "An unexpected error occurred",
			Details: err.Error(),
		})
	}
}

func HandleValidationError(c *fiber.Ctx, message string, details ...string) error {
	response := ErrorResponse{
		Code:    CodeValidationFailed,
		Message: message,
		Details: message,
	}

	if len(details) > 0 {
		response.Details = details[0]
	}

	return c.Status(http.StatusBadRequest).JSON(response)
}

func HandleUserContextError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeMissingUserContext,
		Message: message,
		Details: message,
	})
}

func HandleInvalidRequestError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidRequest,
		Message: message,
		Details: message,
	})
}

func HandleUUIDError(c *fiber.Ctx, fieldName string) error {
	message := fmt.Sprintf("Invalid %s format", fieldName)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidUUID,
		Message: message,
		Details: message,
	})
}

func HandleMissingFieldError(c *fiber.Ctx, fieldName string) error {
	message := fmt.Sprintf("Missing required field: %s", fieldName)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeMissingRequiredField,
		Message: message,
		Details: message,
	})
}

func HandleInvalidFieldError(c *fiber.Ctx, fieldName string, reason string) error {
	message := fmt.Sprintf("Invalid %s: %s", fieldName, reason)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeInvalidFieldValue,
		Message: message,
		Details: message,
	})
}

func HandlePermissionError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusForbidden).JSON(ErrorResponse{
		Code:    CodePermissionDenied,
		Message: message,
		Details: message,
	})
}

func HandleUnauthorizedError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusUnauthorized).JSON(ErrorResponse{
		Code:    CodeUnauthorized,
		Message: message,
		Details: message,
	})
}

func HandleNotFoundError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusNotFound).JSON(ErrorResponse{
		Code:    CodeProfileNotFound,
		Message: message,
		Details: message,
	})
}

func HandleInternalServerError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
		Code:    CodeInternalError,
		Message: message,
		Details: message,
	})
}

func WrapDatabaseError(err error) *ProfileError {
	return NewProfileError(CodeDatabaseError, "Database operation failed", err)
}

func WrapValidationError(err error, details string) *ProfileError {
	return &ProfileError{
		Code:    CodeValidationFailed,
		Message: "Validation failed",
		Details: details,
		Cause:   err,
	}
}

func WrapSystemError(err error) *ProfileError {
	return NewProfileError(CodeSystemError, "System error occurred", err)
}

func WrapRequestError(err error, details string) *ProfileError {
	return NewProfileError(CodeInvalidRequest, "Invalid request", err)
}

func WrapProfileNotFoundError(err error) *ProfileError {
	return NewProfileError(CodeProfileNotFound, "Profile not found", err)
}

func WrapUnauthorizedError(err error) *ProfileError {
	return NewProfileError(CodeUnauthorized, "Unauthorized access", err)
}

