package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)


type GroqClient struct {
	apiKey          string
	httpClient      *http.Client
	completionModel string
}

var _ CompletionClient = (*GroqClient)(nil)

// GroqConfig contains Groq client configuration
type GroqConfig struct {
	APIKey          string
	CompletionModel string
	Timeout         time.Duration
}

// NewGroqClient creates a new client for interacting with the Groq API.
func NewGroqClient(config GroqConfig) (*GroqClient, error) {
	if config.APIKey == "" {
		return nil, fmt.Errorf("Groq API key is required")
	}
	if config.CompletionModel == "" {
		config.CompletionModel = "llama3-8b-8192"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	return &GroqClient{
		apiKey:          config.APIKey,
		httpClient:      &http.Client{Timeout: config.Timeout},
		completionModel: config.CompletionModel,
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
func (c *GroqClient) GenerateCompletion(ctx context.Context, prompt string) (string, error) {
	apiURL := "https://api.groq.com/openai/v1/chat/completions"

	reqBody := groqCompletionRequest{
		Model: c.completionModel,
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
		body, _ := json.Marshal(resp.Body)
		return "", fmt.Errorf("groq API returned status %d: %s", resp.StatusCode, string(body))
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
	// Test with a simple completion request
	_, err := c.GenerateCompletion(ctx, "test")
	if err != nil {
		return fmt.Errorf("groq service health check failed: %w", err)
	}
	return nil
}

