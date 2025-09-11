package api

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/qolzam/telar/apps/ai-engine/internal/knowledge"
)

// Handler contains HTTP handlers for AI Engine endpoints
type Handler struct {
	knowledgeService *knowledge.Service
}

// NewHandler creates a new handler instance
func NewHandler(knowledgeService *knowledge.Service) *Handler {
	return &Handler{
		knowledgeService: knowledgeService,
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

// Ingest processes document ingestion requests
func (h *Handler) Ingest(c *fiber.Ctx) error {
	var req IngestRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
	}

	docID := "doc-" + strconv.FormatInt(time.Now().UnixNano(), 36)

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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
