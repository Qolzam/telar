package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

// extractJSONFromText extracts a JSON object from text that may have explanatory text before it.
// It finds the first '{' character and extracts the complete JSON object by matching braces.
func extractJSONFromText(text string) string {
	text = strings.TrimSpace(text)

	// If it already starts with '{', return as-is
	if strings.HasPrefix(text, "{") {
		// Verify it's valid JSON by trying to find the matching closing brace
		return extractCompleteJSONObject(text)
	}

	// Find the first '{' character
	startIdx := strings.Index(text, "{")
	if startIdx == -1 {
		// No JSON found, return original text
		return text
	}

	// Extract from the first '{' to the end
	jsonCandidate := text[startIdx:]
	return extractCompleteJSONObject(jsonCandidate)
}

// extractCompleteJSONObject finds the complete JSON object starting from the first '{'
// by counting braces to find the matching closing '}'
func extractCompleteJSONObject(text string) string {
	if len(text) == 0 || text[0] != '{' {
		return text
	}

	braceCount := 0
	inString := false
	escapeNext := false

	for i, char := range text {
		if escapeNext {
			escapeNext = false
			continue
		}

		if char == '\\' && inString {
			escapeNext = true
			continue
		}

		if char == '"' {
			inString = !inString
			continue
		}

		if !inString {
			if char == '{' {
				braceCount++
			} else if char == '}' {
				braceCount--
				if braceCount == 0 {
					// Found the matching closing brace
					return text[:i+1]
				}
			}
		}
	}

	return text
}

// Service handles content analysis and moderation tasks
type Service struct {
	compClient              llms.Model
	requestTimeout          time.Duration
	toxicityThreshold       float64
	spamThreshold           float64
	sexualThreshold         float64
	violenceThreshold       float64
	misinformationThreshold float64
}

// AnalysisRequest represents a content analysis request
type AnalysisRequest struct {
	Content     string            `json:"content"`
	Context     map[string]string `json:"context,omitempty"`
	CommunityID string            `json:"community_id,omitempty"`
	ContentID   string            `json:"content_id,omitempty"`
}

// AnalysisResult represents the structured result of content analysis
type AnalysisResult struct {
	IsFlagged       bool               `json:"is_flagged"`
	FlagReason      string             `json:"flag_reason,omitempty"`
	Scores          map[string]float64 `json:"scores"`
	Confidence      float64            `json:"confidence"`
	Timestamp       string             `json:"timestamp"`
	SuggestedAction string             `json:"suggested_action"` // "approve" or "review_needed"
}

// NewService creates a new analyzer service instance
func NewService(compClient llms.Model, thresholds ...ModerationThresholds) *Service {
	// Default thresholds (calibrated for qwen2.5:1.5b)
	toxicity := 0.50
	spam := 0.45
	sexual := 0.75
	violence := 0.75
	misinformation := 0.70

	// Override with provided thresholds if any
	if len(thresholds) > 0 {
		t := thresholds[0]
		if t.Toxicity > 0 {
			toxicity = t.Toxicity
		}
		if t.Spam > 0 {
			spam = t.Spam
		}
		if t.Sexual > 0 {
			sexual = t.Sexual
		}
		if t.Violence > 0 {
			violence = t.Violence
		}
		if t.Misinformation > 0 {
			misinformation = t.Misinformation
		}
	}

	return &Service{
		compClient:              compClient,
		requestTimeout:          30 * time.Second,
		toxicityThreshold:       toxicity,
		spamThreshold:           spam,
		sexualThreshold:         sexual,
		violenceThreshold:       violence,
		misinformationThreshold: misinformation,
	}
}

// ModerationThresholds holds configurable moderation thresholds
type ModerationThresholds struct {
	Toxicity       float64
	Spam           float64
	Sexual         float64
	Violence       float64
	Misinformation float64
}

// AnalyzeContent performs AI-based content moderation analysis
func (s *Service) AnalyzeContent(ctx context.Context, content string) (*AnalysisResult, error) {
	log.Printf("Analyzing content for moderation (length: %d chars)", len(content))

	// Create a timeout context for this analysis
	analysisCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	// Construct the moderation prompt (simplified for SLM - only scores, no decisions)
	prompt := prompts.NewPromptTemplate(
		`Analyze this text and return ONLY a JSON object with scores. No explanations, no decisions, just scores.

Text: "{{.content}}"

Return JSON:
{
  "scores": {
    "toxicity": 0.0-1.0,
    "sexual": 0.0-1.0,
    "violence": 0.0-1.0,
    "spam": 0.0-1.0,
    "misinformation": 0.0-1.0
  },
  "confidence": 0.0-1.0
}`,
		[]string{"content"},
	)

	// Format the prompt with the content
	formattedPrompt, err := prompt.Format(map[string]any{
		"content": content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to format analysis prompt: %w", err)
	}

	// Call the LLM for analysis
	response, err := llms.GenerateFromSinglePrompt(analysisCtx, s.compClient, formattedPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm analysis failed: %w", err)
	}

	// Parse the JSON response
	var result AnalysisResult

	// Clean the response - some LLMs may add markdown code blocks or explanatory text
	cleanedResponse := strings.TrimSpace(response)
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```json")
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	// Extract JSON object from response if there's text before it
	// LLMs sometimes add explanatory text like "Here's my analysis: { ... }"
	cleanedResponse = extractJSONFromText(cleanedResponse)

	// Parse LLM response - simplified format (only scores), but backward-compatible with old format
	var llmResponse struct {
		Scores          map[string]float64 `json:"scores"`
		Confidence      float64            `json:"confidence"`
		IsFlagged       bool               `json:"is_flagged"`       // Ignored - we use policy
		FlagReason      string             `json:"flag_reason"`      // Ignored - we generate from scores
		SuggestedAction string             `json:"suggested_action"` // Ignored - we use policy
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &llmResponse); err != nil {
		log.Printf("Failed to parse LLM response as JSON. Raw response: %s", response)
		return nil, fmt.Errorf("failed to parse analysis result: %w. Raw response: %s", err, response)
	}

	// Initialize result with LLM scores and confidence
	result.Scores = llmResponse.Scores
	result.Confidence = llmResponse.Confidence
	if result.Scores == nil {
		result.Scores = make(map[string]float64)
	}

	violationFound := false
	violationReasons := []string{}

	toxicityScore := result.Scores["toxicity"]
	if toxicityScore > s.toxicityThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("High Toxicity (%.2f)", toxicityScore))
	}

	sexualScore := result.Scores["sexual"]
	if sexualScore > s.sexualThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Sexual Content (%.2f)", sexualScore))
	}

	violenceScore := result.Scores["violence"]
	if violenceScore > s.violenceThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Violence (%.2f)", violenceScore))
	}

	spamScore := result.Scores["spam"]
	if spamScore > s.spamThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Spam (%.2f)", spamScore))
	}

	misinformationScore := result.Scores["misinformation"]
	if misinformationScore > s.misinformationThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Misinformation (%.2f)", misinformationScore))
	}

	if violationFound {
		result.IsFlagged = true
		result.SuggestedAction = "review_needed"
		result.FlagReason = strings.Join(violationReasons, ", ")
	} else {
		result.IsFlagged = false
		result.SuggestedAction = "approve"
		result.FlagReason = ""
	}

	result.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Log the analysis result
	if result.IsFlagged {
		log.Printf("[CONTENT_FLAGGED] Reason: %s, Confidence: %.2f, Scores: %+v, Action: %s",
			result.FlagReason, result.Confidence, result.Scores, result.SuggestedAction)
	} else {
		log.Printf("[CONTENT_APPROVED] Confidence: %.2f, Action: %s", result.Confidence, result.SuggestedAction)
	}

	return &result, nil
}

// HealthCheck verifies the analyzer service is operational
func (s *Service) HealthCheck(ctx context.Context) error {
	if s.compClient == nil {
		return fmt.Errorf("completion client is not initialized")
	}

	// Perform a simple test analysis
	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	testPrompt := "Respond with only the word 'OK'"
	_, err := llms.GenerateFromSinglePrompt(testCtx, s.compClient, testPrompt)
	if err != nil {
		return fmt.Errorf("analyzer health check failed: %w", err)
	}

	return nil
}
