package moderation

import (
	"context"
	"strings"
	"time"
)

type KeywordFilter struct {
	blockedWords []string
}

func NewKeywordFilter() *KeywordFilter {
	// In a real prod app, load this from a config file or DB.
	// For this demo, a hardcoded list proves the architectural point perfectly.
	return &KeywordFilter{
		blockedWords: []string{"scam", "crypto", "100x", "free money", "idiot"},
	}
}

func (f *KeywordFilter) Name() string {
	return "L2-Keyword-Heuristics"
}

func (f *KeywordFilter) Moderate(ctx context.Context, text string) (*ModerationResult, error) {
	start := time.Now()
	lowerText := strings.ToLower(text)

	for _, word := range f.blockedWords {
		if strings.Contains(lowerText, word) {
			return &ModerationResult{
				IsFlagged:       true,
				FlagReason:      "keyword_violation: " + word,
				SuggestedAction: "review_needed",
				Scores:          map[string]float64{"keyword_match": 1.0},
				AnalysisTimeMs:  time.Since(start).Milliseconds(),
				ModelUsed:       f.Name(),
			}, nil
		}
	}

	// Return nil to signal "I didn't find anything, pass to next layer"
	return nil, nil
}





