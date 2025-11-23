package common

import (
	"crypto/rand"
	"fmt"
	"strings"
	"time"

	uuid "github.com/gofrs/uuid"
)

const (
	randomCharset       = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	defaultPreviewLimit = 80
)

// stringWithCharset generates a cryptographically secure random string for testing helpers.
func stringWithCharset(length int, charset string) string {
	if length <= 0 {
		return ""
	}

	buffer := make([]byte, length)
	randomBytes := make([]byte, length)

	if _, err := rand.Read(randomBytes); err != nil {
		// Fall back to timestamp-based generator to keep helpers deterministic in failure scenarios.
		seed := time.Now().UnixNano()
		for i := range buffer {
			seed = (seed*1103515245 + 12345) & 0x7fffffff
			buffer[i] = charset[int(seed)%len(charset)]
		}
		return string(buffer)
	}

	for i := range buffer {
		buffer[i] = charset[int(randomBytes[i])%len(charset)]
	}

	return string(buffer)
}

// StringRand exposes a simple helper to generate random test strings.
func StringRand(length int) string {
	return stringWithCharset(length, randomCharset)
}

// GenerateCommentPreview trims comment text for list responses while preserving word boundaries.
func GenerateCommentPreview(text string, limit int) string {
	if limit <= 0 {
		limit = defaultPreviewLimit
	}

	trimmed := strings.TrimSpace(text)
	if len(trimmed) <= limit {
		return trimmed
	}

	preview := trimmed[:limit]
	lastSpace := strings.LastIndex(preview, " ")
	if lastSpace > 0 {
		preview = preview[:lastSpace]
	}

	return strings.TrimSpace(preview) + "…"
}

// BuildCommentCacheKey ensures cache keys remain consistent across handlers and services.
func BuildCommentCacheKey(postID uuid.UUID, ownerID *uuid.UUID, page, limit int) string {
	keyBuilder := strings.Builder{}
	keyBuilder.Grow(64)

	keyBuilder.WriteString("comments:")
	keyBuilder.WriteString(postID.String())
	keyBuilder.WriteString(":page:")
	keyBuilder.WriteString(fmt.Sprintf("%d", page))
	keyBuilder.WriteString(":limit:")
	keyBuilder.WriteString(fmt.Sprintf("%d", limit))

	if ownerID != nil {
		keyBuilder.WriteString(":owner:")
		keyBuilder.WriteString(ownerID.String())
	}

	return keyBuilder.String()
}

