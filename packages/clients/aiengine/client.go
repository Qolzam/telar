package aiengine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Error types for structured error handling
var (
	ErrServiceUnavailable = errors.New("ai engine service unavailable")
	ErrUnauthorized       = errors.New("ai engine authentication failed")
	ErrTimeout            = errors.New("ai engine request timeout")
	ErrInvalidResponse    = errors.New("ai engine invalid response")
)

// ClientError represents a structured error from the AI Engine client
type ClientError struct {
	Type        error
	StatusCode  int
	Message     string
	Retryable   bool
	OriginalErr error
}

func (e *ClientError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("%s: %s", e.Type.Error(), e.Message)
	}
	return e.Type.Error()
}

func (e *ClientError) Unwrap() error {
	return e.OriginalErr
}

// IsRetryable returns true if the error indicates a transient failure that should be retried
func (e *ClientError) IsRetryable() bool {
	return e.Retryable
}

// Client provides a typed client for calling the AI Engine analyzer service
type Client interface {
	AnalyzeContent(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error)
}

// AnalysisRequest represents a content analysis request
type AnalysisRequest struct {
	Content     string `json:"content"`
	CommunityID string `json:"community_id,omitempty"`
	ContentID   string `json:"content_id,omitempty"`
}

// AnalysisResult represents the structured result of content analysis
type AnalysisResult struct {
	IsFlagged       bool               `json:"is_flagged"`
	FlagReason      string             `json:"flag_reason,omitempty"`
	Scores          map[string]float64 `json:"scores"`
	Confidence      float64            `json:"confidence"`
	Timestamp       string             `json:"timestamp"`
	SuggestedAction string             `json:"suggested_action"`
}

// httpClient implements the Client interface using HTTP
type httpClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewClient creates a new AI Engine client instance
// Reads the AI Engine URL from AI_ENGINE_URL environment variable
// Reads the API key from AI_ENGINE_INTERNAL_API_KEY environment variable
// Defaults to http://localhost:9066 if not set
func NewClient() Client {
	baseURL := os.Getenv("AI_ENGINE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9066"
	}

	apiKey := os.Getenv("AI_ENGINE_INTERNAL_API_KEY")

	return &httpClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewClientWithConfig creates a new AI Engine client with custom configuration
func NewClientWithConfig(baseURL string, apiKey string, timeout time.Duration) Client {
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &httpClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// AnalyzeContent calls the AI Engine to analyze content for moderation
func (c *httpClient) AnalyzeContent(ctx context.Context, req AnalysisRequest) (*AnalysisResult, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create the HTTP request
	url := fmt.Sprintf("%s/api/v1/analyze/content", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Use Authorization: Bearer header
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	}

	// Execute the request
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		// Check if it's a timeout error
		if ctx.Err() == context.DeadlineExceeded {
			return nil, &ClientError{
				Type:        ErrTimeout,
				Message:     "request timeout",
				Retryable:   true,
				OriginalErr: err,
			}
		}
		return nil, &ClientError{
			Type:        ErrServiceUnavailable,
			Message:     "failed to execute request",
			Retryable:   true,
			OriginalErr: err,
		}
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, &ClientError{
			Type:        ErrInvalidResponse,
			Message:     "failed to read response body",
			Retryable:   false,
			OriginalErr: err,
		}
	}

	// Check for non-200 status codes with structured error handling
	if resp.StatusCode != http.StatusOK {
		clientErr := &ClientError{
			Type:        ErrServiceUnavailable,
			StatusCode:  resp.StatusCode,
			Message:     string(body),
			OriginalErr: fmt.Errorf("analyzer service returned status %d", resp.StatusCode),
		}

		// Classify error types
		switch resp.StatusCode {
		case http.StatusUnauthorized, http.StatusForbidden:
			clientErr.Type = ErrUnauthorized
			clientErr.Retryable = false // Don't retry auth failures
		case http.StatusServiceUnavailable, http.StatusGatewayTimeout, http.StatusTooManyRequests:
			clientErr.Retryable = true // Retry transient errors
		default:
			clientErr.Retryable = false // Don't retry other errors
		}

		return nil, clientErr
	}

	// Parse the response
	var result AnalysisResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, &ClientError{
			Type:        ErrInvalidResponse,
			Message:     "failed to unmarshal response",
			Retryable:   false,
			OriginalErr: err,
		}
	}

	return &result, nil
}
