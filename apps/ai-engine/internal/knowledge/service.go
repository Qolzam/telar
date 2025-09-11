package knowledge

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/weaviate"
)

// Service provides knowledge management and retrieval functionality
type Service struct {
	llmClient      llm.Client
	vectorClient   *weaviate.Client
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
func NewService(llmClient llm.Client, vectorClient *weaviate.Client, config Config) *Service {
	return &Service{
		llmClient:      llmClient,
		vectorClient:   vectorClient,
		embeddingModel: config.EmbeddingModel,
	}
}

// StoreDocument ingests a document by generating embeddings and storing in vector DB
func (s *Service) StoreDocument(ctx context.Context, req *DocumentRequest) error {
	log.Printf("Storing document: %s", req.ID)

	embedding, err := s.llmClient.GenerateEmbeddings(ctx, req.Text)
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

	// Generate query embeddings
	queryEmbedding, err := s.llmClient.GenerateEmbeddings(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embeddings: %w", err)
	}

	// Retrieve similar documents
	searchResults, err := s.vectorClient.SearchSimilar(ctx, queryEmbedding, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar documents: %w", err)
	}

	// Build context from retrieved documents
	var contextParts []string
	var avgRelevance float32

	for i, result := range searchResults {
		contextParts = append(contextParts, fmt.Sprintf("Source %d: %s", i+1, result.Document.Text))
		avgRelevance += result.Score
	}

	if len(searchResults) > 0 {
		avgRelevance = avgRelevance / float32(len(searchResults))
	}

	context := strings.Join(contextParts, "\n\n")

	// Generate answer using LLM with retrieved context
	prompt := s.buildRAGPrompt(req.Query, context, req.Context)

	completion, err := s.llmClient.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate completion: %w", err)
	}

	response := &QueryResponse{
		Answer:         completion,
		Sources:        searchResults,
		ContextUsed:    context,
		RelevanceScore: avgRelevance,
	}

	log.Printf("Successfully generated answer for query: %s", req.Query)
	return response, nil
}

// HealthCheck verifies connectivity to all external dependencies
func (s *Service) HealthCheck(ctx context.Context) error {
	if err := s.vectorClient.Health(ctx); err != nil {
		return fmt.Errorf("vector database health check failed: %w", err)
	}

	if err := s.llmClient.Health(ctx); err != nil {
		return fmt.Errorf("LLM health check failed: %w", err)
	}

	return nil
}

// buildRAGPrompt constructs an optimized prompt for retrieval-augmented generation
func (s *Service) buildRAGPrompt(query, retrievedContext string, userContext map[string]string) string {
	var prompt strings.Builder

	prompt.WriteString("You are an AI assistant that answers questions based on provided context. ")
	prompt.WriteString("Use the following retrieved information to answer the user's question accurately. ")
	prompt.WriteString("If the retrieved context doesn't contain relevant information, say so clearly.\n\n")

	if retrievedContext != "" {
		prompt.WriteString("Retrieved Context:\n")
		prompt.WriteString(retrievedContext)
		prompt.WriteString("\n\n")
	}

	if len(userContext) > 0 {
		prompt.WriteString("Additional Context:\n")
		for key, value := range userContext {
			prompt.WriteString(fmt.Sprintf("%s: %s\n", key, value))
		}
		prompt.WriteString("\n")
	}

	prompt.WriteString("Question: ")
	prompt.WriteString(query)
	prompt.WriteString("\n\nAnswer:")

	return prompt.String()
}
