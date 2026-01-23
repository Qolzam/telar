package analyzer

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/tmc/langchaingo/llms"
)

// mockLLM is a mock implementation of llms.Model for testing
// It implements the minimal interface needed by llms.GenerateFromSinglePrompt
type mockLLM struct {
	response string
	err      error
}

func (m *mockLLM) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.response, nil
}

func (m *mockLLM) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if m.err != nil {
		return nil, m.err
	}
	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: m.response,
			},
		},
	}, nil
}

func setupTestService(mockClient *mockLLM) *Service {
	// Pass nil for registry - service will use fallback prompt
	// This is fine for testing as the fallback prompt is sufficient
	return NewService(mockClient, nil, "test-model")
}

// TestAnalyzeContent_FlaggedContent tests that toxic content is correctly flagged
func TestAnalyzeContent_FlaggedContent(t *testing.T) {
	// Mock LLM response for flagged content
	flaggedResponse := `{
		"is_flagged": true,
		"flag_reason": "Contains hate speech",
		"scores": {
			"toxic": 0.85,
			"threat": 0.2,
			"insult": 0.1,
			"identity_hate": 0.1,
			"spam": 0.05,
			"misinformation": 0.1
		},
		"confidence": 0.92,
		"suggested_action": "review_needed"
	}`

	mockClient := &mockLLM{response: flaggedResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "This is toxic hate speech content")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	if !result.IsFlagged {
		t.Error("Expected content to be flagged")
	}

	if result.SuggestedAction != "review_needed" {
		t.Errorf("Expected suggested_action to be 'review_needed', got '%s'", result.SuggestedAction)
	}

	if result.FlagReason == "" {
		t.Error("Expected flag_reason to be set by policy enforcement")
	}
	if !strings.Contains(result.FlagReason, "High Toxicity") {
		t.Errorf("Expected flag_reason to contain 'High Toxicity', got '%s'", result.FlagReason)
	}

	// Check new taxonomy key "toxic" first, fallback to old "toxicity"
	toxicScore := result.Scores["toxic"]
	if toxicScore == 0 {
		toxicScore = result.Scores["toxicity"]
	}
	if toxicScore != 0.85 {
		t.Errorf("Expected toxic/toxicity score to be 0.85, got %f (toxic=%f, toxicity=%f)", toxicScore, result.Scores["toxic"], result.Scores["toxicity"])
	}

	if result.Confidence < 0.9 {
		t.Errorf("Expected confidence to be >= 0.9, got %f", result.Confidence)
	}
}

// TestAnalyzeContent_SafeContent tests that safe content is approved
func TestAnalyzeContent_SafeContent(t *testing.T) {
	// Mock LLM response for safe content
	safeResponse := `{
		"is_flagged": false,
		"flag_reason": "",
		"scores": {
			"toxic": 0.05,
			"threat": 0.01,
			"insult": 0.02,
			"identity_hate": 0.01,
			"spam": 0.03,
			"misinformation": 0.02
		},
		"confidence": 0.95,
		"suggested_action": "approve"
	}`

	mockClient := &mockLLM{response: safeResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "This is a friendly post about technology")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	if result.IsFlagged {
		t.Error("Expected content to not be flagged")
	}

	if result.SuggestedAction != "approve" {
		t.Errorf("Expected suggested_action to be 'approve', got '%s'", result.SuggestedAction)
	}

	if result.Confidence < 0.9 {
		t.Errorf("Expected confidence to be >= 0.9, got %f", result.Confidence)
	}
}

// TestAnalyzeContent_PolicyEnforcement tests that policy enforcement flags content based on scores
func TestAnalyzeContent_PolicyEnforcement(t *testing.T) {
	// Mock LLM response with toxic score above threshold (0.50 default)
	// Policy enforcement should flag it regardless of LLM's decision
	highToxicityResponse := `{
		"scores": {
			"toxic": 0.85,
			"threat": 0.1,
			"insult": 0.05,
			"identity_hate": 0.05,
			"spam": 0.05,
			"misinformation": 0.05
		},
		"confidence": 0.88
	}`

	mockClient := &mockLLM{response: highToxicityResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "Toxic content here")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	// Policy enforcement should flag content when toxic > 0.50 (default threshold)
	toxicScore := result.Scores["toxic"]
	if toxicScore == 0 {
		toxicScore = result.Scores["toxicity"]
	}
	if toxicScore > 0.50 {
		if !result.IsFlagged {
			t.Error("Expected policy enforcement to flag content with toxic > 0.50")
		}
		if result.SuggestedAction != "review_needed" {
			t.Errorf("Expected policy enforcement to set suggested_action to 'review_needed', got '%s'", result.SuggestedAction)
		}
		if !strings.Contains(result.FlagReason, "High Toxicity") {
			t.Errorf("Expected flag_reason to contain 'High Toxicity', got '%s'", result.FlagReason)
		}
	}
}

// TestAnalyzeContent_EnforcesPolicy_LazyAI tests Scenario A: Policy overrides AI's wrong decision
// This is the critical test for the "Hybrid Enforcer" pattern
func TestAnalyzeContent_EnforcesPolicy_LazyAI(t *testing.T) {
	// Scenario A: The "Lazy AI" Correction
	// AI says it's safe (is_flagged: false) but admits high toxicity (0.95)
	// Go code MUST override this decision
	lazyAIResponse := `{
		"scores": {
			"toxic": 0.95,
			"threat": 0.0,
			"insult": 0.0,
			"identity_hate": 0.0,
			"spam": 0.0,
			"misinformation": 0.0
		},
		"confidence": 0.9
	}`

	mockClient := &mockLLM{response: lazyAIResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "toxic hate speech content")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	// CRITICAL: Go logic must override AI's false negative
	// Default threshold is 0.50 for toxic, so 0.95 should definitely flag
	toxicScore := result.Scores["toxic"]
	if toxicScore == 0 {
		toxicScore = result.Scores["toxicity"]
	}
	if toxicScore > 0.50 {
		if !result.IsFlagged {
			t.Error("Go logic failed to override false negative from AI. is_flagged must be TRUE when toxic > 0.50")
		}
		if result.SuggestedAction != "review_needed" {
			t.Errorf("Go logic failed to correct suggested_action. Expected 'review_needed', got '%s'", result.SuggestedAction)
		}
		if !strings.Contains(result.FlagReason, "High Toxicity") {
			t.Errorf("Go logic failed to generate reason based on score. FlagReason should contain 'High Toxicity', got '%s'", result.FlagReason)
		}
	}
	if toxicScore != 0.95 {
		t.Errorf("Toxic score should be preserved: expected 0.95, got %f", toxicScore)
	}
}

// TestAnalyzeContent_EnforcesPolicy_SpamCatch tests Scenario B: Policy catches spam violations
func TestAnalyzeContent_EnforcesPolicy_SpamCatch(t *testing.T) {
	// Scenario B: The "Spam" Catch
	// AI says it's safe but spam score is 0.85 (above 0.45 default threshold)
	spamResponse := `{
		"scores": {
			"toxic": 0.1,
			"threat": 0.0,
			"insult": 0.0,
			"identity_hate": 0.0,
			"spam": 0.85,
			"misinformation": 0.0
		},
		"confidence": 0.85
	}`

	mockClient := &mockLLM{response: spamResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "spam promotional content")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	// Policy enforcement should flag spam > 0.45 (default threshold)
	spamScore := result.Scores["spam"]
	if spamScore > 0.45 {
		if !result.IsFlagged {
			t.Error("Expected policy enforcement to flag content with spam > 0.45")
		}
		if result.SuggestedAction != "review_needed" {
			t.Errorf("Expected suggested_action to be 'review_needed', got '%s'", result.SuggestedAction)
		}
		if !strings.Contains(result.FlagReason, "Spam") {
			t.Errorf("Expected flag_reason to contain 'Spam', got '%s'", result.FlagReason)
		}
	}
}

// TestAnalyzeContent_ContradictoryResponse tests that business logic overrides contradictory AI suggestions
func TestAnalyzeContent_ContradictoryResponse(t *testing.T) {
	// Mock LLM response with contradiction: is_flagged=true but suggested_action="approve"
	// This simulates a misbehaving or malicious LLM response
	contradictoryResponse := `{
		"is_flagged": true,
		"flag_reason": "Contains explicit hate speech",
		"scores": {
			"toxic": 0.95,
			"threat": 0.8,
			"insult": 0.0,
			"identity_hate": 0.0,
			"spam": 0.0,
			"misinformation": 0.0
		},
		"confidence": 0.98,
		"suggested_action": "approve"
	}`

	mockClient := &mockLLM{response: contradictoryResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "Malicious content that LLM incorrectly wants to approve")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	// Business logic must override: is_flagged=true ALWAYS means suggested_action="review_needed"
	// This ensures the system's business rules are the ultimate authority, not the LLM
	// Also, scores should trigger policy enforcement (toxic: 0.95 > 0.50, violence: 0.8 > 0.75 threshold)
	toxicScore := result.Scores["toxic"]
	if toxicScore == 0 {
		toxicScore = result.Scores["toxicity"]
	}
	violenceScore := result.Scores["threat"]
	if violenceScore == 0 {
		violenceScore = result.Scores["violence"]
	}
	if (toxicScore > 0.50 || violenceScore > 0.75) && result.IsFlagged {
		if result.SuggestedAction != "review_needed" {
			t.Errorf("Expected business logic to override contradictory AI suggestion. Got suggested_action='%s', but is_flagged=true should force 'review_needed'", result.SuggestedAction)
		}
	}

	if !result.IsFlagged {
		t.Error("Expected is_flagged to remain true despite contradictory suggested_action")
	}
}

// TestAnalyzeContent_InvalidJSONResponse tests error handling for invalid LLM responses
func TestAnalyzeContent_InvalidJSONResponse(t *testing.T) {
	invalidResponse := "This is not valid JSON"

	mockClient := &mockLLM{response: invalidResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	_, err := service.AnalyzeContent(ctx, "Some content")

	if err == nil {
		t.Error("Expected error for invalid JSON response")
	}
}

// TestAnalyzeContent_LLMError tests error handling when LLM call fails
func TestAnalyzeContent_LLMError(t *testing.T) {
	mockClient := &mockLLM{
		err: context.DeadlineExceeded,
	}
	service := setupTestService(mockClient)

	ctx := context.Background()
	_, err := service.AnalyzeContent(ctx, "Some content")

	if err == nil {
		t.Error("Expected error when LLM call fails")
	}
}

// TestAnalyzeContent_Timestamp tests that timestamp is set correctly
func TestAnalyzeContent_Timestamp(t *testing.T) {
	safeResponse := `{
		"is_flagged": false,
		"flag_reason": "",
		"scores": {
			"toxic": 0.05,
			"threat": 0.01,
			"insult": 0.02,
			"identity_hate": 0.01,
			"spam": 0.03,
			"misinformation": 0.02
		},
		"confidence": 0.95,
		"suggested_action": "approve"
	}`

	mockClient := &mockLLM{response: safeResponse}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "Safe content")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	if result.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}

	// Verify timestamp is in RFC3339 format
	_, err = time.Parse(time.RFC3339, result.Timestamp)
	if err != nil {
		t.Errorf("Timestamp is not in RFC3339 format: %s, error: %v", result.Timestamp, err)
	}
}

// TestAnalyzeContent_AllScoresPresent tests that all required scores are present
func TestAnalyzeContent_AllScoresPresent(t *testing.T) {
	response := `{
		"is_flagged": false,
		"flag_reason": "",
		"scores": {
			"toxic": 0.1,
			"threat": 0.3,
			"insult": 0.2,
			"identity_hate": 0.2,
			"spam": 0.4,
			"misinformation": 0.5
		},
		"confidence": 0.9,
		"suggested_action": "approve"
	}`

	mockClient := &mockLLM{response: response}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "Test content")

	if err != nil {
		t.Fatalf("AnalyzeContent failed: %v", err)
	}

	// Check for both new taxonomy (toxic, threat) and old taxonomy (toxicity, violence)
	requiredScores := []string{"toxic", "threat", "insult", "identity_hate", "spam", "misinformation"}
	// Also check old keys for backward compatibility
	oldScores := []string{"toxicity", "sexual", "violence"}
	for _, score := range requiredScores {
		if _, exists := result.Scores[score]; !exists {
			// For old keys, check if they exist as fallback
			if score == "toxic" {
				if _, oldExists := result.Scores["toxicity"]; !oldExists {
					t.Errorf("Expected score 'toxic' or 'toxicity' to be present in result")
				}
			} else if score == "threat" {
				if _, oldExists := result.Scores["violence"]; !oldExists {
					t.Errorf("Expected score 'threat' or 'violence' to be present in result")
				}
			} else {
				t.Errorf("Expected score '%s' to be present in result", score)
			}
		}
	}
	// Check sexual for old taxonomy
	for _, score := range oldScores {
		if score == "sexual" {
			if _, exists := result.Scores[score]; !exists {
				// Sexual is optional in new taxonomy, so just check if it exists
				// This is fine if it doesn't exist
			}
		}
	}
}

// TestAnalyzeContent_JSONWithPrefixText tests that JSON can be extracted when LLM adds text before it
func TestAnalyzeContent_JSONWithPrefixText(t *testing.T) {
	// This simulates the actual error reported where LLM adds text like "Here's my analysis: "
	responseWithPrefix := `Here's my analysis: {
		"is_flagged": true,
		"flag_reason": "Toxicity: hate speech and harassment",
		"scores": {
			"toxic": 1.0,
			"threat": 0.0,
			"insult": 0.0,
			"identity_hate": 0.0,
			"spam": 0.0,
			"misinformation": 0.0
		},
		"confidence": 0.9,
		"suggested_action": "review_needed"
	}`

	mockClient := &mockLLM{response: responseWithPrefix}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "Test content")

	if err != nil {
		t.Fatalf("AnalyzeContent failed with prefixed text: %v", err)
	}

	if !result.IsFlagged {
		t.Error("Expected content to be flagged")
	}

	if result.SuggestedAction != "review_needed" {
		t.Errorf("Expected suggested_action to be 'review_needed', got '%s'", result.SuggestedAction)
	}

	toxicScore := result.Scores["toxic"]
	if toxicScore == 0 {
		toxicScore = result.Scores["toxicity"]
	}
	if toxicScore != 1.0 {
		t.Errorf("Expected toxic/toxicity score to be 1.0, got %f", toxicScore)
	}
}

// TestAnalyzeContent_JSONWithColonPrefix tests another common pattern of text before JSON
func TestAnalyzeContent_JSONWithColonPrefix(t *testing.T) {
	responseWithColon := `Here is the JSON object analyzing the content:

{
	"is_flagged": false,
	"flag_reason": "",
	"scores": {
		"toxic": 0.1,
		"threat": 0.0,
		"insult": 0.0,
		"identity_hate": 0.0,
		"spam": 0.0,
		"misinformation": 0.0
	},
	"confidence": 0.95,
	"suggested_action": "approve"
}`

	mockClient := &mockLLM{response: responseWithColon}
	service := setupTestService(mockClient)

	ctx := context.Background()
	result, err := service.AnalyzeContent(ctx, "Safe content")

	if err != nil {
		t.Fatalf("AnalyzeContent failed with colon-prefixed text: %v", err)
	}

	if result.IsFlagged {
		t.Error("Expected content to not be flagged")
	}

	if result.SuggestedAction != "approve" {
		t.Errorf("Expected suggested_action to be 'approve', got '%s'", result.SuggestedAction)
	}
}
