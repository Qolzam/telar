package api

import (
	"github.com/gofiber/fiber/v2"
)

type Handler struct {
	// TODO: Add service dependencies
}

func NewHandler() *Handler {
	return &Handler{}
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
	Question string `json:"question" binding:"required"`
	Limit    int    `json:"limit,omitempty"`
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

func (h *Handler) Ingest(c *fiber.Ctx) error {
	var req IngestRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
	}

	// TODO: Implement RAG ingestion
	response := IngestResponse{
		Status:  "success",
		Message: "Document ingested successfully",
		ID:      "mock-document-id-" + req.Text[:min(10, len(req.Text))],
	}

	return c.JSON(response)
}

func (h *Handler) Query(c *fiber.Ctx) error {
	var req QueryRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   "Invalid request payload",
			"details": err.Error(),
		})
	}

	// TODO: Implement RAG query
	response := QueryResponse{
		Answer: "This is a mock response to the question: " + req.Question,
		Sources: []SourceChunk{
			{
				ID:    "mock-chunk-1",
				Text:  "This is a mock source chunk that would contain relevant information.",
				Score: 0.95,
				Metadata: map[string]string{
					"source": "mock-document",
				},
			},
		},
	}

	return c.JSON(response)
}

func (h *Handler) Health(c *fiber.Ctx) error {
	response := HealthResponse{
		Status: "healthy",
		Services: map[string]string{
			"api":      "healthy",
			"llm":      "not_implemented",
			"weaviate": "not_implemented",
		},
	}

	return c.JSON(response)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
