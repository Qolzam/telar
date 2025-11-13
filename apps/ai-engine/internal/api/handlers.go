package api

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/qolzam/telar/apps/ai-engine/internal/analyzer"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
	"github.com/qolzam/telar/apps/ai-engine/internal/generator"
	"github.com/qolzam/telar/apps/ai-engine/internal/knowledge"
)

// Handler contains HTTP handlers for AI Engine endpoints
type Handler struct {
	knowledgeService *knowledge.Service
	generatorService *generator.Service
	analyzerService  *analyzer.Service
	config           *config.Config
}

// NewHandler creates a new handler instance
func NewHandler(knowledgeService *knowledge.Service, generatorService *generator.Service, analyzerService *analyzer.Service, config *config.Config) *Handler {
	return &Handler{
		knowledgeService: knowledgeService,
		generatorService: generatorService,
		analyzerService:  analyzerService,
		config:           config,
	}
}

type IngestRequest struct {
	Text     string            `json:"text" binding:"required"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type IngestResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	ID      string `json:"id,omitempty"`
}

type QueryRequest struct {
	Question string            `json:"question" binding:"required"`
	Limit    int               `json:"limit,omitempty"`
	Context  map[string]string `json:"context,omitempty"`
}

type QueryResponse struct {
	Answer  string        `json:"answer"`
	Sources []SourceChunk `json:"sources,omitempty"`
}

type SourceChunk struct {
	ID       string            `json:"id"`
	Text     string            `json:"text"`
	Score    float32           `json:"score"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

type StatusResponse struct {
	Status             string `json:"status"`
	EmbeddingProvider  string `json:"embedding_provider"`
	CompletionProvider string `json:"completion_provider"`
}

// GenerateRequest represents a request to generate conversation starters
type GenerateRequest struct {
	Topic string `json:"topic" binding:"required"`
	Style string `json:"style,omitempty"` 
	Count int    `json:"count,omitempty"`
}

// GenerateResponse represents the generated conversation starters
type GenerateResponse struct {
	Topic    string           `json:"topic"`
	Style    string           `json:"style"`
	Starters []string         `json:"starters"`
	Metadata GenerateMetadata `json:"metadata"`
}

// GenerateMetadata provides additional information about the generation
type GenerateMetadata struct {
	GeneratedAt      string `json:"generated_at"`
	Model            string `json:"model"`
	ResponseTimeMs   int64  `json:"response_time_ms"`
	PromptTokens     int    `json:"prompt_tokens,omitempty"`
	CompletionTokens int    `json:"completion_tokens,omitempty"`
}

// Ingest processes document ingestion requests
func (h *Handler) Ingest(c *fiber.Ctx) error {
	var req IngestRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
	}

	docID := uuid.New().String()

	docReq := &knowledge.DocumentRequest{
		ID:       docID,
		Text:     req.Text,
		Metadata: req.Metadata,
	}

	if err := h.knowledgeService.StoreDocument(c.Context(), docReq); err != nil {
		log.Printf("Failed to store document: %v", err)

		if strings.Contains(err.Error(), "ollama service is not available") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "AI service temporarily unavailable",
				"details": "Ollama LLM service is not running. Please ensure Ollama is started and accessible.",
				"code":    "OLLAMA_UNAVAILABLE",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to store document",
			"details": err.Error(),
		})
	}

	response := IngestResponse{
		Status:  "success",
		Message: "Document ingested successfully",
		ID:      docID,
	}

	return c.JSON(response)
}

// Query processes knowledge query requests using RAG
func (h *Handler) Query(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
	}

	queryReq := &knowledge.QueryRequest{
		Query:   req.Question,
		Context: req.Context,
	}

	result, err := h.knowledgeService.QueryKnowledge(c.Context(), queryReq)
	if err != nil {
		log.Printf("Failed to query knowledge: %v", err)

		if strings.Contains(err.Error(), "ollama service is not available") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "AI service temporarily unavailable",
				"details": "Ollama LLM service is not running. Please ensure Ollama is started and accessible.",
				"code":    "OLLAMA_UNAVAILABLE",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to process query",
			"details": err.Error(),
		})
	}

	var sources []SourceChunk
	for _, source := range result.Sources {
		sources = append(sources, SourceChunk{
			ID:       source.Document.ID,
			Text:     source.Document.Text,
			Score:    source.Score,
			Metadata: source.Document.Metadata,
		})
	}

	response := QueryResponse{
		Answer:  result.Answer,
		Sources: sources,
	}

	return c.JSON(response)
}

// GenerateConversationStarters creates engaging prompts for a community.
func (h *Handler) GenerateConversationStarters(c *fiber.Ctx) error {
	var req struct {
		CommunityTopic string `json:"community_topic"`
		Style          string `json:"style"`
	}
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	starters, err := h.generatorService.GenerateConversationStarters(c.Context(), req.CommunityTopic, req.Style)
	if err != nil {
		log.Printf("Generator service error: %v", err)
		
		if strings.Contains(err.Error(), "server is currently processing too many requests") {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "server is currently processing too many requests",
				"details": "Please try again in a moment. The server is limiting concurrent requests to prevent overload.",
				"retry_after": "5 seconds",
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to generate conversation starters", "details": err.Error()})
	}

	return c.Status(fiber.StatusOK).JSON(starters)
}

// GetConcurrentStatus returns the current concurrent request status
func (h *Handler) GetConcurrentStatus(c *fiber.Ctx) error {
	status := h.generatorService.GetConcurrentStatus()
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   status,
	})
}

// GetModelConfig returns the current model configuration
func (h *Handler) GetModelConfig(c *fiber.Ctx) error {
	llmConfig := h.config.LLM
	
	var currentModel string
	var provider string
	
	switch llmConfig.CompletionProvider {
	case "openai":
		provider = "OpenAI"
		currentModel = llmConfig.OpenAIModel
	case "groq":
		provider = "Groq"
		currentModel = llmConfig.GroqModel
	case "ollama":
		provider = "Ollama"
		currentModel = llmConfig.CompletionModel
	default:
		provider = "Unknown"
		currentModel = "Unknown"
	}
	
	config := fiber.Map{
		"provider":           llmConfig.CompletionProvider,
		"provider_display":   provider,
		"model":              currentModel,
		"embedding_provider": llmConfig.EmbeddingProvider,
		"embedding_model":    llmConfig.EmbeddingModel,
		"max_concurrent":     llmConfig.MaxConcurrent,
	}
	
	return c.JSON(fiber.Map{
		"status": "success",
		"data":   config,
	})
}

// Health returns service health status and dependency checks
func (h *Handler) Health(c *fiber.Ctx) error {
	services := map[string]string{
		"api": "healthy",
	}

	if err := h.knowledgeService.HealthCheck(c.Context()); err != nil {
		log.Printf("Knowledge service health check failed: %v", err)
		services["knowledge"] = "unhealthy"
		services["details"] = err.Error()

		return c.Status(fiber.StatusServiceUnavailable).JSON(HealthResponse{
			Status:   "unhealthy",
			Services: services,
		})
	}

	services["knowledge"] = "healthy"
	services["llm"] = "healthy"
	services["weaviate"] = "healthy"

	response := HealthResponse{
		Status:   "healthy",
		Services: services,
	}

	return c.JSON(response)
}

// GetStatus returns the current configuration status
func (h *Handler) GetStatus(c *fiber.Ctx) error {
	embeddingProvider := os.Getenv("EMBEDDING_PROVIDER")
	if embeddingProvider == "" {
		embeddingProvider = "ollama"
	}

	completionProvider := os.Getenv("COMPLETION_PROVIDER")
	if completionProvider == "" {
		completionProvider = "ollama"
	}

	response := StatusResponse{
		Status:             "healthy",
		EmbeddingProvider:  embeddingProvider,
		CompletionProvider: completionProvider,
	}

	return c.JSON(response)
}

// ServeDemo serves the demo UI
func (h *Handler) ServeDemo(c *fiber.Ctx) error {
	indexPath := filepath.Join("./public", "index.html")
	content, err := os.ReadFile(indexPath)
	if err != nil {
		log.Printf("Failed to read index.html: %v", err)
		return c.Status(fiber.StatusInternalServerError).SendString("Demo UI not available")
	}

	c.Set("Content-Type", "text/html")
	return c.Send(content)
}

// AnalyzeContent handles content moderation analysis requests
func (h *Handler) AnalyzeContent(c *fiber.Ctx) error {
	var req analyzer.AnalysisRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
	}

	// Validate that content is provided
	if strings.TrimSpace(req.Content) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Content is required",
			"details": "The 'content' field cannot be empty",
		})
	}

	// Perform the analysis
	result, err := h.analyzerService.AnalyzeContent(c.Context(), req.Content)
	if err != nil {
		log.Printf("Content analysis failed: %v", err)

		// Check for specific error types
		if strings.Contains(err.Error(), "timeout") {
			return c.Status(fiber.StatusRequestTimeout).JSON(fiber.Map{
				"error":   "Analysis request timed out",
				"details": "The content analysis took too long. Please try again.",
			})
		}

		if strings.Contains(err.Error(), "ollama service is not available") {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error":   "AI service temporarily unavailable",
				"details": "Ollama LLM service is not running. Please ensure Ollama is started and accessible.",
				"code":    "OLLAMA_UNAVAILABLE",
			})
		}

		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to analyze content",
			"details": err.Error(),
		})
	}

	return c.JSON(result)
}
