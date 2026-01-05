package moderation

import (
	"context"
)

// ModerationResult represents the outcome of an analysis from ANY source (Cache, Regex, or LLM).
type ModerationResult struct {
	IsFlagged       bool               `json:"is_flagged"`
	FlagReason      string             `json:"flag_reason"`      // e.g. "toxicity", "bad_keywords"
	Scores          map[string]float64 `json:"scores,omitempty"` // AI scores, or simple 1.0/0.0 for regex
	SuggestedAction string             `json:"suggested_action"` // "approve", "review_needed"
	AnalysisTimeMs  int64              `json:"analysis_time_ms"`
	ModelUsed       string             `json:"model_used"`       // e.g. "cache-hit", "keyword-filter", "qwen2.5"
	RequestID       string             `json:"request_id,omitempty"`
}

// ContentModerator is the interface that all layers (L1, L2, L3) must implement.
type ContentModerator interface {
	// Moderate returns a result if a decision is made.
	// If the layer cannot decide (e.g., cache miss, no keyword found), it returns nil, nil.
	Moderate(ctx context.Context, text string) (*ModerationResult, error)
	Name() string
}





