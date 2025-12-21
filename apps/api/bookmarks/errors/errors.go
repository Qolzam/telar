package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

var (
	ErrInvalidRequest     = errors.New("invalid request")
	ErrInvalidUUID        = errors.New("invalid uuid")
	ErrMissingUserContext = errors.New("missing user context")
	ErrDatabaseOperation  = errors.New("database operation failed")
)

const (
	CodeInvalidRequest = "INVALID_REQUEST"
	CodeInvalidUUID    = "INVALID_UUID"
	CodeMissingUserCtx = "MISSING_USER_CONTEXT"
	CodeDatabaseError  = "DATABASE_ERROR"
	CodeInternalError  = "INTERNAL_ERROR"
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
	case errors.Is(err, ErrInvalidRequest):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{Code: CodeInvalidRequest, Message: err.Error(), Details: err.Error()})
	case errors.Is(err, ErrInvalidUUID):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{Code: CodeInvalidUUID, Message: err.Error(), Details: err.Error()})
	case errors.Is(err, ErrMissingUserContext):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{Code: CodeMissingUserCtx, Message: err.Error(), Details: err.Error()})
	case errors.Is(err, ErrDatabaseOperation):
		return c.Status(http.StatusServiceUnavailable).JSON(ErrorResponse{Code: CodeDatabaseError, Message: err.Error(), Details: err.Error()})
	default:
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{Code: CodeInternalError, Message: "An unexpected error occurred", Details: err.Error()})
	}
}

func HandleValidationError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{Code: CodeInvalidRequest, Message: message, Details: message})
}

func HandleUUIDError(c *fiber.Ctx, fieldName string) error {
	msg := fmt.Sprintf("Invalid %s format", fieldName)
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{Code: CodeInvalidUUID, Message: msg, Details: msg})
}

func HandleUserContextError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{Code: CodeMissingUserCtx, Message: message, Details: message})
}
