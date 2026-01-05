package moderation

import (
	"context"
	"time"

	"github.com/qolzam/telar/apps/ai-engine/internal/analyzer"
)

// LLMAnalyzerAdapter wraps analyzer.Service to implement ContentModerator interface
type LLMAnalyzerAdapter struct {
	service *analyzer.Service
	modelName string
}

// NewLLMAnalyzerAdapter creates a new adapter that wraps analyzer.Service
func NewLLMAnalyzerAdapter(service *analyzer.Service, modelName string) *LLMAnalyzerAdapter {
	return &LLMAnalyzerAdapter{
		service:   service,
		modelName: modelName,
	}
}

func (a *LLMAnalyzerAdapter) Name() string {
	return "L3-Semantic-Model(" + a.modelName + ")"
}

// Moderate implements ContentModerator interface by calling the underlying analyzer service
func (a *LLMAnalyzerAdapter) Moderate(ctx context.Context, text string) (*ModerationResult, error) {
	start := time.Now()

	// Call the existing analyzer service
	analysisResult, err := a.service.AnalyzeContent(ctx, text)
	if err != nil {
		return nil, err
	}

	// Map AnalysisResult to ModerationResult
	// Ensure scores map exists and include confidence for backward compatibility
	scores := analysisResult.Scores
	if scores == nil {
		scores = make(map[string]float64)
	}
	// Store confidence in scores map for easy extraction in handler
	if analysisResult.Confidence > 0 {
		scores["confidence"] = analysisResult.Confidence
	}

	result := &ModerationResult{
		IsFlagged:       analysisResult.IsFlagged,
		FlagReason:      analysisResult.FlagReason,
		Scores:          scores,
		SuggestedAction: analysisResult.SuggestedAction,
		AnalysisTimeMs:  time.Since(start).Milliseconds(),
		ModelUsed:       a.Name(),
	}

	return result, nil
}

