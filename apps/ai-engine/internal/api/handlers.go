package api

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/qolzam/telar/apps/ai-engine/internal/analyzer"
	"github.com/qolzam/telar/apps/ai-engine/internal/config"
	"github.com/qolzam/telar/apps/ai-engine/internal/generator"
	"github.com/qolzam/telar/apps/ai-engine/internal/knowledge"
	"github.com/qolzam/telar/apps/ai-engine/internal/moderation"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
	"github.com/qolzam/telar/apps/ai-engine/internal/types"
)

// Handler contains HTTP handlers for AI Engine endpoints
type Handler struct {
	knowledgeService *knowledge.Service
	generatorService *generator.Service
	modPipeline      *moderation.Pipeline
	config           *config.Config
}

// NewHandler creates a new handler instance
func NewHandler(knowledgeService *knowledge.Service, generatorService *generator.Service, modPipeline *moderation.Pipeline, config *config.Config) *Handler {
	return &Handler{
		knowledgeService: knowledgeService,
		generatorService: generatorService,
		modPipeline:      modPipeline,
		config:           config,
	}
}

// handleOllamaErrorResponse handles Ollama-specific errors and returns appropriate HTTP responses
// This centralizes error handling logic and uses structured error types instead of string parsing
func handleOllamaErrorResponse(c *fiber.Ctx, err error, defaultMessage string) error {
	ollamaErr, isOllamaErr := llm.IsOllamaError(err)
	if !isOllamaErr {
		// Not an Ollama error, return generic error
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   defaultMessage,
			"details": err.Error(),
		})
	}

	// Map Ollama error codes to HTTP status codes and user-friendly messages
	switch ollamaErr.Code {
	case llm.ErrCodeModelNotFound:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Model not available",
			"details": ollamaErr.Details,
			"code":    ollamaErr.Code,
		})
	case llm.ErrCodeContextLengthExceeded:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Input text exceeds model context window",
			"details": ollamaErr.Details,
			"code":    ollamaErr.Code,
		})
	case llm.ErrCodeConnectionRefused, llm.ErrCodeHostNotFound, llm.ErrCodeServiceUnavailable:
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "AI service temporarily unavailable",
			"details": ollamaErr.Details,
			"code":    "OLLAMA_UNAVAILABLE",
		})
	case llm.ErrCodeTimeout:
		statusCode := fiber.StatusRequestTimeout
		if ollamaErr.Model != "" {
			// For model-specific timeouts (model loading), use ServiceUnavailable
			statusCode = fiber.StatusServiceUnavailable
		}
		return c.Status(statusCode).JSON(fiber.Map{
			"error":   "AI service timeout",
			"details": ollamaErr.Details,
			"code":    ollamaErr.Code,
		})
	default:
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"error":   "AI service temporarily unavailable",
			"details": ollamaErr.Details,
			"code":    "OLLAMA_UNAVAILABLE",
		})
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
	Status                     string `json:"status"`
	KnowledgeEmbeddingProvider string `json:"knowledge_embedding_provider"`
	GeneratorProvider          string `json:"generator_provider"`
	ModerationFallbackProvider string `json:"moderation_fallback_provider"`
	ModerationONNXEnabled      bool   `json:"moderation_onnx_enabled"`
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
		return handleOllamaErrorResponse(c, err, "Failed to store document")
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
	endToEndStart := time.Now()

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

	log.Printf("[RAG] End-to-end query started question_len=%d", len(req.Question))

	result, err := h.knowledgeService.QueryKnowledge(c.Context(), queryReq)
	endToEndDuration := time.Since(endToEndStart)

	if err != nil {
		log.Printf("[RAG] End-to-end query failed question_len=%d total_duration_ms=%d error=%v",
			len(req.Question), endToEndDuration.Milliseconds(), err)
		return handleOllamaErrorResponse(c, err, "Failed to process query")
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

	log.Printf("[RAG] End-to-end query completed question_len=%d answer_len=%d sources_count=%d total_duration_ms=%d",
		len(req.Question), len(result.Answer), len(sources), endToEndDuration.Milliseconds())

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
				"error":       "server is currently processing too many requests",
				"details":     "Please try again in a moment. The server is limiting concurrent requests to prevent overload.",
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

	config := fiber.Map{
		"knowledge": fiber.Map{
			"embedding_provider": llmConfig.KnowledgeEmbeddingProvider,
			"embedding_model":    llmConfig.KnowledgeEmbeddingModel,
		},
		"generator": fiber.Map{
			"provider": llmConfig.GeneratorProvider,
			"model":    llmConfig.GeneratorModel,
		},
		"moderation": fiber.Map{
			"fallback_provider":        llmConfig.ModerationFallbackProvider,
			"fallback_model":           llmConfig.ModerationFallbackModel,
			"onnx_toxicity_model_path": llmConfig.ModerationONNXToxicityModelPath,
			"onnx_spam_model_path":     llmConfig.ModerationONNXSpamModelPath,
			"onnx_enabled":             llmConfig.ModerationONNXToxicityModelPath != "" || llmConfig.ModerationONNXSpamModelPath != "",
		},
		"max_concurrent": llmConfig.MaxConcurrent,
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
	return c.JSON(StatusResponse{
		Status:                     "healthy",
		KnowledgeEmbeddingProvider: h.config.LLM.KnowledgeEmbeddingProvider,
		GeneratorProvider:          h.config.LLM.GeneratorProvider,
		ModerationFallbackProvider: h.config.LLM.ModerationFallbackProvider,
		ModerationONNXEnabled:      h.config.LLM.ModerationONNXToxicityModelPath != "" || h.config.LLM.ModerationONNXSpamModelPath != "",
	})
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
	// Extract tenant/app context (available from APIKeyAuth middleware)
	tenantID := c.Locals(types.CtxTenantID)
	appID := c.Locals(types.CtxAppID)

	// Log tenant/app for debugging
	if tenantID != nil && appID != nil {
		log.Printf("[MODERATION] Request from tenant_id=%v, app_id=%v", tenantID, appID)
	}

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

	// Perform the analysis through the tiered moderation pipeline
	result, err := h.modPipeline.Execute(c.Context(), req.Content)
	if err != nil {
		log.Printf("Content analysis failed: %v", err)
		return handleOllamaErrorResponse(c, err, "Failed to analyze content")
	}

	// Convert ModerationResult to AnalysisResult for backward compatibility
	analysisResult := &analyzer.AnalysisResult{
		IsFlagged:       result.IsFlagged,
		FlagReason:      result.FlagReason,
		Scores:          result.Scores,
		SuggestedAction: result.SuggestedAction,
		ModelUsed:       result.ModelUsed, // Include which model/layer made the decision
		Timestamp:       "",               // Will be set below
	}

	// Extract confidence from scores if available
	// ONNX models set "confidence" explicitly, LLM models set it in the scores map
	if confidence, ok := result.Scores["confidence"]; ok {
		analysisResult.Confidence = confidence
	} else if len(result.Scores) > 0 {
		// Fallback: For flagged content, use the max violation score as confidence
		// For safe content, this should not happen (LLM should set confidence)
		maxScore := 0.0
		for key, score := range result.Scores {
			// Skip non-score keys
			if key != "confidence" && score > maxScore {
				maxScore = score
			}
		}
		analysisResult.Confidence = maxScore
	}

	// Set timestamp if not already set
	if analysisResult.Timestamp == "" {
		analysisResult.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	return c.JSON(analysisResult)
}
