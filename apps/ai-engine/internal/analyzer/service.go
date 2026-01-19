package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/qolzam/telar/apps/ai-engine/internal/prompt"
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
	registry                *prompt.Registry
	modelName               string
	variantOverride         string
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
	SuggestedAction string             `json:"suggested_action"`     // "approve" or "review_needed"
	ModelUsed       string             `json:"model_used,omitempty"` // e.g. "L3-ONNX-toxicity", "L3-ONNX-spam", "L3-Semantic-Model(qwen2.5:1.5b)"
}

// NewService creates a new analyzer service instance
// registry: The prompt registry for dynamic prompt selection
// modelName: The name of the LLM model (e.g., "qwen2.5:1.5b", "llama3:8b") used for prompt matching
func NewService(compClient llms.Model, registry *prompt.Registry, modelName string, thresholds ...ModerationThresholds) *Service {
	toxicity := 0.50
	spam := 0.45
	sexual := 0.75
	violence := 0.75
	misinformation := 0.70

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
		registry:                registry,
		modelName:               modelName,
		variantOverride:         "", // Default: auto-select based on model
		requestTimeout:          30 * time.Second,
		toxicityThreshold:       toxicity,
		spamThreshold:           spam,
		sexualThreshold:         sexual,
		violenceThreshold:       violence,
		misinformationThreshold: misinformation,
	}
}

// SetVariantOverride sets the variant override for A/B testing
func (s *Service) SetVariantOverride(variantID string) {
	s.variantOverride = variantID
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
	analysisCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	// Dynamic Prompt Selection: Get the appropriate template based on model
	var templateStr string
	var err error
	var promptSource string

	if s.registry != nil {
		// Use variant override if set (for A/B testing)
		templateStr, err = s.registry.GetTemplateWithOverride("moderation", s.modelName, s.variantOverride)
		if err != nil {
			log.Printf("Warning: Failed to get prompt template from registry: %v. Using fallback.", err)
			templateStr = s.getFallbackPrompt()
			promptSource = "fallback (registry error)"
		} else {
			if s.variantOverride != "" {
				promptSource = fmt.Sprintf("registry (override: %s)", s.variantOverride)
			} else {
				promptSource = "registry"
			}
		}
	} else {
		// Registry not initialized, use fallback
		templateStr = s.getFallbackPrompt()
		promptSource = "fallback (no registry)"
	}

	log.Printf("[PROMPT] Using prompt from: %s (model: %s)", promptSource, s.modelName)

	prompt := prompts.NewPromptTemplate(
		templateStr,
		[]string{"content"},
	)

	formattedPrompt, err := prompt.Format(map[string]any{
		"content": content,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to format analysis prompt: %w", err)
	}

	response, err := llms.GenerateFromSinglePrompt(analysisCtx, s.compClient, formattedPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm analysis failed: %w", err)
	}

	var result AnalysisResult

	cleanedResponse := strings.TrimSpace(response)
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```json")
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSpace(cleanedResponse)
	cleanedResponse = extractJSONFromText(cleanedResponse)

	var llmResponse struct {
		Scores          map[string]float64 `json:"scores"`
		Confidence      float64            `json:"confidence"`
		IsFlagged       bool               `json:"is_flagged"`
		FlagReason      string             `json:"flag_reason"`
		SuggestedAction string             `json:"suggested_action"`
	}

	if err := json.Unmarshal([]byte(cleanedResponse), &llmResponse); err != nil {
		return nil, fmt.Errorf("failed to parse analysis result: %w. Raw response: %s", err, response)
	}

	result.Scores = llmResponse.Scores
	result.Confidence = llmResponse.Confidence
	if result.Scores == nil {
		result.Scores = make(map[string]float64)
	}

	violationFound := false
	violationReasons := []string{}

	// Check new taxonomy keys (aligned with ONNX)
	toxicScore := result.Scores["toxic"]
	if toxicScore > s.toxicityThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("High Toxicity (%.2f)", toxicScore))
	}

	threatScore := result.Scores["threat"]
	if threatScore > s.violenceThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Threat/Violence (%.2f)", threatScore))
	}

	insultScore := result.Scores["insult"]
	if insultScore > s.toxicityThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Personal Insult (%.2f)", insultScore))
	}

	identityHateScore := result.Scores["identity_hate"]
	if identityHateScore > s.violenceThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Hate Speech (%.2f)", identityHateScore))
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

	// Backward compatibility: Check old keys if new keys not present
	if toxicScore == 0 && result.Scores["toxicity"] > 0 {
		if result.Scores["toxicity"] > s.toxicityThreshold {
			violationFound = true
			violationReasons = append(violationReasons, fmt.Sprintf("High Toxicity (%.2f)", result.Scores["toxicity"]))
		}
	}
	if threatScore == 0 && result.Scores["violence"] > 0 {
		if result.Scores["violence"] > s.violenceThreshold {
			violationFound = true
			violationReasons = append(violationReasons, fmt.Sprintf("Violence (%.2f)", result.Scores["violence"]))
		}
	}
	if result.Scores["sexual"] > s.sexualThreshold {
		violationFound = true
		violationReasons = append(violationReasons, fmt.Sprintf("Sexual Content (%.2f)", result.Scores["sexual"]))
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

	if result.IsFlagged {
		log.Printf("[CONTENT_FLAGGED] Reason: %s, Confidence: %.2f, Scores: %+v, Action: %s",
			result.FlagReason, result.Confidence, result.Scores, result.SuggestedAction)
	} else {
		log.Printf("[CONTENT_APPROVED] Confidence: %.2f, Action: %s", result.Confidence, result.SuggestedAction)
	}

	return &result, nil
}

// getFallbackPrompt returns a basic prompt template when registry is unavailable
func (s *Service) getFallbackPrompt() string {
	return `You are a Content Safety AI for a SOFTWARE DEVELOPER COMMUNITY. Analyze this text: """{{.content}}""" and return a JSON object with scores (0.0-1.0) for: toxic, threat, insult, identity_hate, spam, misinformation, and confidence.`
}

// truncateString truncates a string to the specified length
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// HealthCheck verifies the analyzer service is operational
func (s *Service) HealthCheck(ctx context.Context) error {
	if s.compClient == nil {
		return fmt.Errorf("completion client is not initialized")
	}

	testCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	testPrompt := "Respond with only the word 'OK'"
	_, err := llms.GenerateFromSinglePrompt(testCtx, s.compClient, testPrompt)
	if err != nil {
		return fmt.Errorf("analyzer health check failed: %w", err)
	}

	return nil
}
