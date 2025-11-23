package common

import (
	"strings"
	"testing"

	uuid "github.com/gofrs/uuid"
)

func TestStringRand_LengthAndCharset(t *testing.T) {
	t.Parallel()

	result := StringRand(32)
	if len(result) != 32 {
		t.Fatalf("expected random string of length 32, got %d", len(result))
	}

	if strings.ContainsAny(result, " \t\n") {
		t.Fatalf("expected random string to avoid whitespace, got %q", result)
	}
}

func TestGenerateCommentPreview(t *testing.T) {
	t.Parallel()

	longText := "This is a comment that should be truncated gracefully without breaking words."
	preview := GenerateCommentPreview(longText, 40)

	if !strings.HasSuffix(preview, "…") {
		t.Fatalf("expected preview to end with ellipsis, got %q", preview)
	}

	if len(preview) == 0 || len(preview) > 41 {
		t.Fatalf("preview length should be within limit, got %d", len(preview))
	}

	shortText := "Short comment"
	if got := GenerateCommentPreview(shortText, 40); got != shortText {
		t.Fatalf("expected short text to remain untouched, got %q", got)
	}
}

func TestBuildCommentCacheKey(t *testing.T) {
	t.Parallel()

	postID := uuid.Must(uuid.NewV4())
	ownerID := uuid.Must(uuid.NewV4())

	keyWithOwner := BuildCommentCacheKey(postID, &ownerID, 2, 25)
	if !strings.Contains(keyWithOwner, ownerID.String()) {
		t.Fatalf("expected key to contain owner id, got %q", keyWithOwner)
	}

	keyWithoutOwner := BuildCommentCacheKey(postID, nil, 1, 10)
	if strings.Contains(keyWithoutOwner, "owner") {
		t.Fatalf("expected key without owner section, got %q", keyWithoutOwner)
	}

	if keyWithOwner == keyWithoutOwner {
		t.Fatal("expected different cache keys for different filters")
	}
}
