package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type GroqClient struct {
	apiKey              string
	httpClient          *http.Client
	generationModel     string
	classificationModel string
}

var _ CompletionClient = (*GroqClient)(nil)

// GroqConfig contains Groq client configuration
type GroqConfig struct {
	APIKey              string
	GenerationModel     string
	ClassificationModel string
	Timeout             time.Duration
}

// NewGroqClient creates a new client for interacting with the Groq API.
func NewGroqClient(config GroqConfig) (*GroqClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Groq API key is required")
	}
	if config.GenerationModel == "" {
		config.GenerationModel = "llama-3.1-8b-instant"
	}
	if config.ClassificationModel == "" {
		config.ClassificationModel = config.GenerationModel // Default to same as generation if not specified
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &GroqClient{
		apiKey:              config.APIKey,
		httpClient:          &http.Client{Timeout: config.Timeout},
		generationModel:     config.GenerationModel,
		classificationModel: config.ClassificationModel,
	}, nil
}

// Groq API request/response structures (OpenAI compatible)
type groqCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqCompletionResponse struct {
	Choices []struct {
		Message message `json:"message"`
	} `json:"choices"`
	Usage *Usage `json:"usage,omitempty"`
}

// GenerateCompletion sends a prompt to the Groq API and gets a completion.
func (c *GroqClient) GenerateCompletion(ctx context.Context, modelType ModelType, prompt string) (string, error) {
	apiURL := "https://api.groq.com/openai/v1/chat/completions"

	var modelToUse string
	switch modelType {
	case ModelTypeGeneration:
		modelToUse = c.generationModel
	case ModelTypeClassification:
		modelToUse = c.classificationModel
	default:
		return "", fmt.Errorf("unknown model type: %s", modelType)
	}

	reqBody := groqCompletionRequest{
		Model: modelToUse,
		Messages: []message{
			{Role: "user", Content: prompt},
		},
		Stream: false,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal groq request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create groq request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request to groq: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		bodyStr := string(bodyBytes)
		// Try to parse error response for better error message
		var errorResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			} `json:"error"`
		}
		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil && errorResp.Error.Message != "" {
			return "", fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, errorResp.Error.Message)
		}
		return "", fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, bodyStr)
	}

	var groqResp groqCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&groqResp); err != nil {
		return "", fmt.Errorf("failed to decode groq response: %w", err)
	}

	if len(groqResp.Choices) == 0 {
		return "", fmt.Errorf("received no choices from groq")
	}

	return groqResp.Choices[0].Message.Content, nil
}

// Health checks Groq service availability
func (c *GroqClient) Health(ctx context.Context) error {
	// Test with a simple completion request using generation model
	_, err := c.GenerateCompletion(ctx, ModelTypeGeneration, "test")
	if err != nil {
		return fmt.Errorf("groq service health check failed: %w", err)
	}
	return nil
}
