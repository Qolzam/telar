package knowledge

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/weaviate"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

// Service provides knowledge management and retrieval functionality
type Service struct {
	embedClient  llm.EmbeddingClient
	compClient   llms.Model
	vectorClient *weaviate.Client
	embeddingModel string
}

// Config holds knowledge service configuration
type Config struct {
	EmbeddingModel string
}

// QueryRequest represents a knowledge query request
type QueryRequest struct {
	Query   string            `json:"query"`
	Context map[string]string `json:"context,omitempty"`
}

// QueryResponse represents a knowledge query response
type QueryResponse struct {
	Answer         string                   `json:"answer"`
	Sources        []*weaviate.SearchResult `json:"sources"`
	ContextUsed    string                   `json:"context_used"`
	RelevanceScore float32                  `json:"relevance_score"`
}

// DocumentRequest represents a document storage request
type DocumentRequest struct {
	ID       string            `json:"id"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// NewService creates a new knowledge service instance
func NewService(embedClient llm.EmbeddingClient, compClient llms.Model, vectorClient *weaviate.Client, config Config) *Service {
	return &Service{
		embedClient:    embedClient,
		compClient:     compClient,
		vectorClient:   vectorClient,
		embeddingModel: config.EmbeddingModel,
	}
}

// StoreDocument ingests a document by generating embeddings and storing in vector DB
func (s *Service) StoreDocument(ctx context.Context, req *DocumentRequest) error {
	log.Printf("Storing document: %s", req.ID)

	embedding, err := s.embedClient.GenerateEmbeddings(ctx, req.Text)
	if err != nil {
		return fmt.Errorf("failed to generate embeddings: %w", err)
	}

	doc := &weaviate.Document{
		ID:       req.ID,
		Text:     req.Text,
		Metadata: req.Metadata,
	}

	if err := s.vectorClient.StoreDocument(ctx, doc, embedding); err != nil {
		return fmt.Errorf("failed to store document in vector DB: %w", err)
	}

	log.Printf("Successfully stored document: %s", req.ID)
	return nil
}

// QueryKnowledge performs RAG: retrieves relevant documents and generates contextual answers
func (s *Service) QueryKnowledge(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	log.Printf("Processing knowledge query: %s", req.Query)

	queryEmbedding, err := s.embedClient.GenerateEmbeddings(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embeddings: %w", err)
	}

	// Retrieve similar documents
	searchResults, err := s.vectorClient.SearchSimilar(ctx, queryEmbedding, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar documents: %w", err)
	}

	prompt := prompts.NewPromptTemplate(
		"You are an expert Q&A system. Use the following pieces of context to answer the question at the end. If you don't know the answer, just say that you don't know, don't try to make up an answer.\n\nContext:\n{{.context}}\n\nQuestion: {{.question}}\n\nHelpful Answer:",
		[]string{"context", "question"},
	)

	var contextBuilder strings.Builder
	var avgRelevance float32
	
	for i, result := range searchResults {
		contextBuilder.WriteString(fmt.Sprintf("Source %d: %s\n", i+1, result.Document.Text))
		avgRelevance += result.Score
	}

	if len(searchResults) > 0 {
		avgRelevance = avgRelevance / float32(len(searchResults))
	}

	formattedPrompt, err := prompt.Format(map[string]any{
		"context":  contextBuilder.String(),
		"question": req.Query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to format prompt template: %w", err)
	}

	completion, err := s.generateCompletion(ctx, formattedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate completion: %w", err)
	}

	response := &QueryResponse{
		Answer:         completion,
		Sources:        searchResults,
		ContextUsed:    contextBuilder.String(),
		RelevanceScore: avgRelevance,
	}

	log.Printf("Successfully generated answer for query: %s", req.Query)
	return response, nil
}

func (s *Service) generateCompletion(ctx context.Context, prompt string) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, s.compClient, prompt)
}

// HealthCheck verifies connectivity to all external dependencies
func (s *Service) HealthCheck(ctx context.Context) error {
	if err := s.vectorClient.Health(ctx); err != nil {
		return fmt.Errorf("vector database health check failed: %w", err)
	}

	if err := s.embedClient.Health(ctx); err != nil {
		return fmt.Errorf("embedding client health check failed: %w", err)
	}

	_, err := llms.GenerateFromSinglePrompt(ctx, s.compClient, "test")
	if err != nil {
		return fmt.Errorf("completion client health check failed: %w", err)
	}

	return nil
}
