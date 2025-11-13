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

// Service handles content analysis and moderation tasks
type Service struct {
	compClient     llms.Model
	requestTimeout time.Duration
}

// AnalysisRequest represents a content analysis request
type AnalysisRequest struct {
	Content string            `json:"content"`
	Context map[string]string `json:"context,omitempty"`
}

// AnalysisResult represents the structured result of content analysis
type AnalysisResult struct {
	IsFlagged  bool               `json:"is_flagged"`
	FlagReason string             `json:"flag_reason,omitempty"`
	Scores     map[string]float64 `json:"scores"`
	Confidence float64            `json:"confidence"`
	Timestamp  string             `json:"timestamp"`
}

// NewService creates a new analyzer service instance
func NewService(compClient llms.Model) *Service {
	return &Service{
		compClient:     compClient,
		requestTimeout: 30 * time.Second,
	}
}

// AnalyzeContent performs AI-based content moderation analysis
func (s *Service) AnalyzeContent(ctx context.Context, content string) (*AnalysisResult, error) {
	log.Printf("Analyzing content for moderation (length: %d chars)", len(content))

	// Create a timeout context for this analysis
	analysisCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	// Construct the moderation prompt
	prompt := prompts.NewPromptTemplate(
		`You are an expert content moderation AI. Analyze the following content for potential policy violations.

Content to analyze:
"""
{{.content}}
"""

Evaluate the content across these dimensions:
1. Toxicity (hate speech, harassment, threats)
2. Sexual content (explicit or inappropriate sexual material)
3. Violence (graphic violence, gore, or violent threats)
4. Spam (repetitive, promotional, or low-quality content)
5. Misinformation (false or misleading information)

You MUST respond with ONLY a valid JSON object in this exact format, with no additional text:
{
  "is_flagged": true or false,
  "flag_reason": "brief explanation if flagged, empty string if not",
  "scores": {
    "toxicity": 0.0-1.0,
    "sexual": 0.0-1.0,
    "violence": 0.0-1.0,
    "spam": 0.0-1.0,
    "misinformation": 0.0-1.0
  },
  "confidence": 0.0-1.0
}

Flag the content (is_flagged: true) if ANY score exceeds 0.7 or if you have high confidence it violates community guidelines.`,
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
	
	// Clean the response - some LLMs may add markdown code blocks
	cleanedResponse := strings.TrimSpace(response)
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```json")
	cleanedResponse = strings.TrimPrefix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSuffix(cleanedResponse, "```")
	cleanedResponse = strings.TrimSpace(cleanedResponse)

	if err := json.Unmarshal([]byte(cleanedResponse), &result); err != nil {
		log.Printf("Failed to parse LLM response as JSON. Raw response: %s", response)
		return nil, fmt.Errorf("failed to parse analysis result: %w. Raw response: %s", err, response)
	}

	// Add timestamp
	result.Timestamp = time.Now().UTC().Format(time.RFC3339)

	// Log the analysis result
	if result.IsFlagged {
		log.Printf("[CONTENT_FLAGGED] Reason: %s, Confidence: %.2f, Scores: %+v", 
			result.FlagReason, result.Confidence, result.Scores)
	} else {
		log.Printf("[CONTENT_APPROVED] Confidence: %.2f", result.Confidence)
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


