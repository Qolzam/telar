package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// Ollama error codes
const (
	ErrCodeConnectionRefused      = "OLLAMA_CONNECTION_REFUSED"
	ErrCodeHostNotFound           = "OLLAMA_HOST_NOT_FOUND"
	ErrCodeTimeout                = "OLLAMA_TIMEOUT"
	ErrCodeServiceUnavailable     = "OLLAMA_SERVICE_UNAVAILABLE"
	ErrCodeModelNotFound          = "MODEL_NOT_FOUND"
	ErrCodeContextLengthExceeded  = "CONTEXT_LENGTH_EXCEEDED"
)

// OllamaError represents an Ollama-specific error with structured information
type OllamaError struct {
	Code    string
	Message string
	Details string
	BaseURL string
	Model   string
	Cause   error
}

func (e *OllamaError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (baseURL: %s, model: %s, caused by: %v)", e.Code, e.Message, e.BaseURL, e.Model, e.Cause)
	}
	return fmt.Sprintf("%s: %s (baseURL: %s, model: %s)", e.Code, e.Message, e.BaseURL, e.Model)
}

func (e *OllamaError) Unwrap() error {
	return e.Cause
}

// IsOllamaError checks if an error is an OllamaError and returns it
func IsOllamaError(err error) (*OllamaError, bool) {
	var ollamaErr *OllamaError
	if errors.As(err, &ollamaErr) {
		return ollamaErr, true
	}
	return nil, false
}

// OllamaClient implements the LLM Client interface for Ollama API
type OllamaClient struct {
	baseURL             string
	httpClient          *http.Client
	embeddingModel      string
	generationModel     string
	classificationModel string
	maxTokens           int
}

// Ensure OllamaClient satisfies both interfaces
var _ EmbeddingClient = (*OllamaClient)(nil)
var _ CompletionClient = (*OllamaClient)(nil)

// Helper function for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// handleOllamaError converts common Ollama HTTP client errors into structured OllamaError types
func handleOllamaError(baseURL string, model string, err error) error {
	if err == nil {
		return nil
	}

	underlyingErr := err.Error()

	if strings.Contains(underlyingErr, "connection refused") {
		return &OllamaError{
			Code:    ErrCodeConnectionRefused,
			Message: "Ollama connection refused",
			Details: "Service may not be running or not accessible from this network. If running in Docker, ensure Ollama container is running and on the same network. If running locally, ensure Ollama is started.",
			BaseURL: baseURL,
			Model:   model,
			Cause:   err,
		}
	}
	if strings.Contains(underlyingErr, "no such host") || strings.Contains(underlyingErr, "name resolution") {
		return &OllamaError{
			Code:    ErrCodeHostNotFound,
			Message: "Ollama host not found",
			Details: "Check OLLAMA_BASE_URL configuration and network/DNS settings.",
			BaseURL: baseURL,
			Model:   model,
			Cause:   err,
		}
	}
	if strings.Contains(underlyingErr, "timeout") || strings.Contains(underlyingErr, "deadline exceeded") {
		details := "The model may be downloading (can take several minutes on first use). Check if models are pulled: docker exec telar-ollama ollama list"
		if model == "" {
			details = "Service may be overloaded. Please try again later."
		}
		return &OllamaError{
			Code:    ErrCodeTimeout,
			Message: "Ollama request timeout",
			Details: details,
			BaseURL: baseURL,
			Model:   model,
			Cause:   err,
		}
	}
	return &OllamaError{
		Code:    ErrCodeServiceUnavailable,
		Message: "Ollama service is not available",
		Details: "Ollama LLM service is not running. Please ensure Ollama is started and accessible.",
		BaseURL: baseURL,
		Model:   model,
		Cause:   err,
	}
}

// OllamaConfig contains Ollama client configuration
type OllamaConfig struct {
	BaseURL             string
	EmbeddingModel      string
	GenerationModel     string
	ClassificationModel string
	Timeout             time.Duration
	MaxTokens           int // Maximum tokens to generate (num_predict), 0 = unlimited
}

// NewOllamaClient creates a new Ollama client with sensible defaults
func NewOllamaClient(config OllamaConfig) *OllamaClient {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "nomic-embed-text"
	}
	if config.GenerationModel == "" {
		config.GenerationModel = "llama3:8b"
	}
	if config.ClassificationModel == "" {
		config.ClassificationModel = config.GenerationModel // Default to same as generation if not specified
	}
	if config.Timeout == 0 {
		// Increased timeout to 300 seconds to handle model loading which can take 60-90 seconds
		// Ollama models need to be loaded into memory on first use, which can exceed the default 60s timeout
		config.Timeout = 300 * time.Second
	}

	// Create HTTP client with connection pooling and keep-alive for better reliability
	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     90 * time.Second,
		DisableKeepAlives:   false, // Enable keep-alive for better connection reuse
	}

	httpClient := &http.Client{
		Timeout:   config.Timeout,
		Transport: transport,
	}

	return &OllamaClient{
		baseURL:             config.BaseURL,
		httpClient:          httpClient,
		embeddingModel:      config.EmbeddingModel,
		generationModel:     config.GenerationModel,
		classificationModel: config.ClassificationModel,
		maxTokens:           config.MaxTokens,
	}
}

// Ollama API request/response types
type ollamaEmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type ollamaEmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

// GenerateEmbeddings creates vector embeddings for text
func (c *OllamaClient) GenerateEmbeddings(ctx context.Context, text string) ([]float32, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/embeddings", c.baseURL)

	reqBody := ollamaEmbeddingRequest{
		Model:  c.embeddingModel,
		Prompt: text,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	httpStart := time.Now()
	resp, err := c.httpClient.Do(req)
	httpDuration := time.Since(httpStart)
	if err != nil {
		log.Printf("[TIMING] embedding_generation failed model=%s text_len=%d http_duration_ms=%d total_duration_ms=%d error=%v",
			c.embeddingModel, len(text), httpDuration.Milliseconds(), time.Since(start).Milliseconds(), err)
		return nil, handleOllamaError(c.baseURL, c.embeddingModel, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Check if it's a context length exceeded error
		if strings.Contains(strings.ToLower(bodyStr), "context length") {
			return nil, &OllamaError{
				Code:    ErrCodeContextLengthExceeded,
				Message: "Context length exceeded",
				Details: fmt.Sprintf("Input text exceeds the model's context window. Model: %s, Text length: %d characters. Consider using chunking.", c.embeddingModel, len(text)),
				BaseURL: c.baseURL,
				Model:   c.embeddingModel,
				Cause:   fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, bodyStr),
			}
		}
		
		// Check if it's a model not found error
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			var errorResp map[string]interface{}
			if json.Unmarshal(body, &errorResp) == nil {
				if errorMsg, ok := errorResp["error"].(string); ok && (strings.Contains(strings.ToLower(errorMsg), "not found") || strings.Contains(strings.ToLower(errorMsg), "try pulling")) {
					return nil, &OllamaError{
						Code:    ErrCodeModelNotFound,
						Message: "Model not found",
						Details: fmt.Sprintf("Model '%s' not found - please pull it first using: ollama pull %s", c.embeddingModel, c.embeddingModel),
						BaseURL: c.baseURL,
						Model:   c.embeddingModel,
						Cause:   fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, bodyStr),
					}
				}
			}
		}
		return nil, &OllamaError{
			Code:    ErrCodeServiceUnavailable,
			Message: "Ollama API request failed",
			Details: fmt.Sprintf("API returned status %d: %s", resp.StatusCode, bodyStr),
			BaseURL: c.baseURL,
			Model:   c.embeddingModel,
			Cause:   fmt.Errorf("ollama API request failed with status %d: %s", resp.StatusCode, bodyStr),
		}
	}

	decodeStart := time.Now()
	var embResp ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		log.Printf("[TIMING] embedding_generation failed model=%s text_len=%d http_duration_ms=%d decode_duration_ms=%d total_duration_ms=%d error=decode_failed",
			c.embeddingModel, len(text), httpDuration.Milliseconds(), time.Since(decodeStart).Milliseconds(), time.Since(start).Milliseconds())
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}
	decodeDuration := time.Since(decodeStart)
	totalDuration := time.Since(start)

	log.Printf("[TIMING] embedding_generation model=%s text_len=%d embedding_dim=%d http_duration_ms=%d decode_duration_ms=%d total_duration_ms=%d",
		c.embeddingModel, len(text), len(embResp.Embedding), httpDuration.Milliseconds(), decodeDuration.Milliseconds(), totalDuration.Milliseconds())

	return embResp.Embedding, nil
}

type ollamaGenerateRequest struct {
	Model   string                 `json:"model"`
	Prompt  string                 `json:"prompt"`
	Stream  bool                   `json:"stream"`
	Format  string                 `json:"format,omitempty"` // "json" to enforce JSON output
	Options map[string]interface{} `json:"options,omitempty"` // Ollama generation options
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// GenerateCompletion generates text completions from prompts
func (c *OllamaClient) GenerateCompletion(ctx context.Context, modelType ModelType, prompt string) (string, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/generate", c.baseURL)

	var modelToUse string
	switch modelType {
	case ModelTypeGeneration:
		modelToUse = c.generationModel
	case ModelTypeClassification:
		modelToUse = c.classificationModel
	default:
		return "", fmt.Errorf("unknown model type: %s", modelType)
	}

	reqBody := ollamaGenerateRequest{
		Model:  modelToUse,
		Prompt: prompt,
		Stream: false,
	}

	// Enforce JSON format for classification tasks (moderation analysis)
	if modelType == ModelTypeClassification {
		reqBody.Format = "json"
	}

	// Add generation options if maxTokens is set (for RAG performance optimization)
	if c.maxTokens > 0 && modelType == ModelTypeGeneration {
		reqBody.Options = map[string]interface{}{
			"num_predict": c.maxTokens,
			"temperature": 0.7,
			"top_p":       0.9,
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Check if context is already cancelled before making request
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("context cancelled before HTTP request: %w", ctx.Err())
	default:
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	httpStart := time.Now()
	resp, err := c.httpClient.Do(req)
	httpDuration := time.Since(httpStart)
	if err != nil {
		log.Printf("[TIMING] llm_generation failed model=%s model_type=%s prompt_len=%d http_duration_ms=%d total_duration_ms=%d error=%v",
			modelToUse, modelType, len(prompt), httpDuration.Milliseconds(), time.Since(start).Milliseconds(), err)
		return "", handleOllamaError(c.baseURL, modelToUse, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := string(body)
		
		// Check if it's a context length exceeded error
		if strings.Contains(strings.ToLower(bodyStr), "context length") {
			return "", &OllamaError{
				Code:    ErrCodeContextLengthExceeded,
				Message: "Context length exceeded",
				Details: fmt.Sprintf("Input prompt exceeds the model's context window. Model: %s, Prompt length: %d characters.", modelToUse, len(prompt)),
				BaseURL: c.baseURL,
				Model:   modelToUse,
				Cause:   fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, bodyStr),
			}
		}
		
		// Check if it's a model not found error
		if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusBadRequest {
			var errorResp map[string]interface{}
			if json.Unmarshal(body, &errorResp) == nil {
				if errorMsg, ok := errorResp["error"].(string); ok && (strings.Contains(strings.ToLower(errorMsg), "not found") || strings.Contains(strings.ToLower(errorMsg), "try pulling")) {
					return "", &OllamaError{
						Code:    ErrCodeModelNotFound,
						Message: "Model not found",
						Details: fmt.Sprintf("Model '%s' not found - please pull it first using: ollama pull %s", modelToUse, modelToUse),
						BaseURL: c.baseURL,
						Model:   modelToUse,
						Cause:   fmt.Errorf("ollama API returned status %d: %s", resp.StatusCode, bodyStr),
					}
				}
			}
		}
		return "", &OllamaError{
			Code:    ErrCodeServiceUnavailable,
			Message: "Ollama API request failed",
			Details: fmt.Sprintf("API returned status %d: %s", resp.StatusCode, bodyStr),
			BaseURL: c.baseURL,
			Model:   modelToUse,
			Cause:   fmt.Errorf("ollama API request failed with status %d: %s", resp.StatusCode, bodyStr),
		}
	}

	readStart := time.Now()
	bodyBytes, err := io.ReadAll(resp.Body)
	readDuration := time.Since(readStart)
	if err != nil {
		log.Printf("[TIMING] llm_generation failed model=%s model_type=%s prompt_len=%d http_duration_ms=%d read_duration_ms=%d total_duration_ms=%d error=read_failed",
			modelToUse, modelType, len(prompt), httpDuration.Milliseconds(), readDuration.Milliseconds(), time.Since(start).Milliseconds())
		return "", fmt.Errorf("failed to read ollama response body: %w", err)
	}

	decodeStart := time.Now()
	var genResp ollamaGenerateResponse
	if err := json.Unmarshal(bodyBytes, &genResp); err != nil {
		log.Printf("[TIMING] llm_generation failed model=%s model_type=%s prompt_len=%d http_duration_ms=%d read_duration_ms=%d decode_duration_ms=%d total_duration_ms=%d error=decode_failed",
			modelToUse, modelType, len(prompt), httpDuration.Milliseconds(), readDuration.Milliseconds(), time.Since(decodeStart).Milliseconds(), time.Since(start).Milliseconds())
		return "", fmt.Errorf("failed to decode ollama response: %w (body: %s)", err, truncateString(string(bodyBytes), 500))
	}
	decodeDuration := time.Since(decodeStart)
	totalDuration := time.Since(start)

	log.Printf("[TIMING] llm_generation model=%s model_type=%s prompt_len=%d response_len=%d http_duration_ms=%d read_duration_ms=%d decode_duration_ms=%d total_duration_ms=%d",
		modelToUse, modelType, len(prompt), len(genResp.Response), httpDuration.Milliseconds(), readDuration.Milliseconds(), decodeDuration.Milliseconds(), totalDuration.Milliseconds())

	return genResp.Response, nil
}

// Health checks Ollama service availability
func (c *OllamaClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/api/tags", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return handleOllamaError(c.baseURL, "", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama service not healthy at %s, status: %d", c.baseURL, resp.StatusCode)
	}

	return nil
}
