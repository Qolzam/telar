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

// OllamaClient implements the LLM Client interface for Ollama API
type OllamaClient struct {
	baseURL             string
	httpClient          *http.Client
	embeddingModel      string
	generationModel     string
	classificationModel string
}

// Ensure OllamaClient satisfies both interfaces
var _ EmbeddingClient = (*OllamaClient)(nil)
var _ CompletionClient = (*OllamaClient)(nil)

// OllamaConfig contains Ollama client configuration
type OllamaConfig struct {
	BaseURL             string
	EmbeddingModel      string
	CompletionModel     string // Legacy: defaults generation model
	GenerationModel     string
	ClassificationModel string
	Timeout             time.Duration
}

// NewOllamaClient creates a new Ollama client with sensible defaults
func NewOllamaClient(config OllamaConfig) *OllamaClient {
	if config.BaseURL == "" {
		config.BaseURL = "http://localhost:11434"
	}
	if config.EmbeddingModel == "" {
		config.EmbeddingModel = "nomic-embed-text"
	}
	// Backward compatibility: if CompletionModel is set but GenerationModel is not, use CompletionModel
	if config.GenerationModel == "" && config.CompletionModel != "" {
		config.GenerationModel = config.CompletionModel
	}
	if config.GenerationModel == "" {
		config.GenerationModel = "llama3:8b"
	}
	if config.ClassificationModel == "" {
		config.ClassificationModel = config.GenerationModel // Default to same as generation if not specified
	}
	if config.Timeout == 0 {
		config.Timeout = 60 * time.Second
	}

	return &OllamaClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		embeddingModel:      config.EmbeddingModel,
		generationModel:     config.GenerationModel,
		classificationModel: config.ClassificationModel,
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

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama service is not available - please ensure ollama is running at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var embResp ollamaEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("failed to decode ollama response: %w", err)
	}

	return embResp.Embedding, nil
}

type ollamaGenerateRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
	Format string `json:"format,omitempty"` // "json" to enforce JSON output
}

type ollamaGenerateResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

// GenerateCompletion generates text completions from prompts
func (c *OllamaClient) GenerateCompletion(ctx context.Context, modelType ModelType, prompt string) (string, error) {
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

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ollama service is not available - please ensure ollama is running at %s: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ollama API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var genResp ollamaGenerateResponse
	if err := json.NewDecoder(resp.Body).Decode(&genResp); err != nil {
		return "", fmt.Errorf("failed to decode ollama response: %w", err)
	}

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
		return fmt.Errorf("ollama service is not available at %s - please ensure ollama is running: %w", c.baseURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama service not healthy at %s, status: %d", c.baseURL, resp.StatusCode)
	}

	return nil
}
