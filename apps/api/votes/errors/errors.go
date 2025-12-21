// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gofiber/fiber/v2"
)

// Vote service specific errors
var (
	ErrVoteNotFound          = errors.New("vote not found")
	ErrPostNotFound          = errors.New("post not found")
	ErrInvalidVoteData       = errors.New("invalid vote data")
	ErrInvalidVoteType       = errors.New("invalid vote type")
	ErrInvalidUserContext    = errors.New("invalid user context")
	ErrInvalidRequest        = errors.New("invalid request")
	ErrInvalidUUID           = errors.New("invalid UUID format")
	ErrMissingUserContext    = errors.New("missing user context")
	ErrValidationFailed      = errors.New("validation failed")
	ErrDatabaseOperation     = errors.New("database operation failed")
)

// Error codes
const (
	CodeVoteNotFound     = "VOTE_NOT_FOUND"
	CodePostNotFound     = "POST_NOT_FOUND"
	CodeInvalidVoteData  = "INVALID_VOTE_DATA"
	CodeInvalidVoteType  = "INVALID_VOTE_TYPE"
	CodeInvalidRequest   = "INVALID_REQUEST"
	CodeInvalidUUID      = "INVALID_UUID"
	CodeMissingUserContext = "MISSING_USER_CONTEXT"
	CodeValidationFailed = "VALIDATION_FAILED"
	CodeDatabaseError    = "DATABASE_ERROR"
)

// ErrorResponse represents the standardized error response format
type ErrorResponse struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Details interface{} `json:"details,omitempty"`
}

// HandleServiceError handles service errors and returns appropriate HTTP responses
func HandleServiceError(c *fiber.Ctx, err error) error {
	if err == nil {
		return nil
	}

	switch {
	case errors.Is(err, ErrPostNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Code:    CodePostNotFound,
			Message: "Post not found",
			Details: err.Error(),
		})
	case errors.Is(err, ErrVoteNotFound):
		return c.Status(http.StatusNotFound).JSON(ErrorResponse{
			Code:    CodeVoteNotFound,
			Message: "Vote not found",
			Details: err.Error(),
		})
	case errors.Is(err, ErrInvalidVoteType):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:    CodeInvalidVoteType,
			Message: "Invalid vote type",
			Details: err.Error(),
		})
	case errors.Is(err, ErrInvalidVoteData):
		return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
			Code:    CodeInvalidVoteData,
			Message: "Invalid vote data",
			Details: err.Error(),
		})
	case errors.Is(err, ErrDatabaseOperation):
		return c.Status(http.StatusServiceUnavailable).JSON(ErrorResponse{
			Code:    CodeDatabaseError,
			Message: "Database operation failed",
			Details: err.Error(),
		})
	default:
		return c.Status(http.StatusInternalServerError).JSON(ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "An unexpected error occurred",
			Details: err.Error(),
		})
	}
}

// HandleValidationError handles validation errors with 400 Bad Request
func HandleValidationError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeValidationFailed,
		Message: message,
		Details: message,
	})
}

// HandleUserContextError handles user context errors with 400 Bad Request
func HandleUserContextError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(ErrorResponse{
		Code:    CodeMissingUserContext,
		Message: message,
		Details: message,
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

