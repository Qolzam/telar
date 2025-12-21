// Copyright (c) 2024 Telar Social
//
// This software is released under the MIT License.
// https://opensource.org/licenses/MIT

package errors

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
)

// HandleInvalidRequestError handles invalid request errors
func HandleInvalidRequestError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(fiber.Map{
		"error":   "INVALID_REQUEST",
		"message": message,
	})
}

// HandleValidationError handles validation errors
func HandleValidationError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusBadRequest).JSON(fiber.Map{
		"error":   "VALIDATION_ERROR",
		"message": message,
	})
}

// HandleServiceError handles service layer errors
func HandleServiceError(c *fiber.Ctx, err error) error {
	errMsg := err.Error()
	
	// Check for specific error types
	if errMsg == "file not found" {
		return c.Status(http.StatusNotFound).JSON(fiber.Map{
			"error":   "FILE_NOT_FOUND",
			"message": errMsg,
		})
	}

	if strings.Contains(errMsg, "file too large") {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":   "FILE_TOO_LARGE",
			"message": errMsg,
		})
	}

	// Quota errors
	if errMsg == "daily upload limit reached" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error":   "QUOTA_EXCEEDED",
			"message": errMsg,
		})
	}

	if errMsg == "system storage busy, try again later" {
		return c.Status(http.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "GLOBAL_LIMIT_REACHED",
			"message": errMsg,
		})
	}

	if errMsg == "quota exceeded" {
		return c.Status(http.StatusForbidden).JSON(fiber.Map{
			"error":   "QUOTA_EXCEEDED",
			"message": errMsg,
		})
	}

	if strings.Contains(errMsg, "invalid MIME type") {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{
			"error":   "INVALID_MIME_TYPE",
			"message": errMsg,
		})
	}

	return c.Status(http.StatusInternalServerError).JSON(fiber.Map{
		"error":   "INTERNAL_ERROR",
		"message": errMsg,
	})
}

// HandleUserContextError handles user context errors
func HandleUserContextError(c *fiber.Ctx, message string) error {
	return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
		"error":   "UNAUTHORIZED",
		"message": message,
	})
}


