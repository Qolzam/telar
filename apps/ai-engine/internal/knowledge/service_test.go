package knowledge

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/qolzam/telar/apps/ai-engine/internal/platform/llm"
	"github.com/qolzam/telar/apps/ai-engine/internal/platform/weaviate"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

// VectorClientInterface defines the interface we need for mocking Weaviate operations
type VectorClientInterface interface {
	StoreDocument(ctx context.Context, doc *weaviate.Document, embedding []float32) error
	SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]*weaviate.SearchResult, error)
	Health(ctx context.Context) error
	EnsureSchema(ctx context.Context) error
}

// TestableService is a version of Service that accepts interfaces for testing
type TestableService struct {
	embedClient    llm.EmbeddingClient
	compClient     llms.Model
	vectorClient   VectorClientInterface
	embeddingModel string
	completionFunc func(ctx context.Context, prompt string) (string, error)
}

// NewTestableService creates a testable version of the service
func NewTestableService(embedClient llm.EmbeddingClient, compClient llms.Model, vectorClient VectorClientInterface, config Config) *TestableService {
	return &TestableService{
		embedClient:    embedClient,
		compClient:     compClient,
		vectorClient:   vectorClient,
		embeddingModel: config.EmbeddingModel,
		completionFunc: func(ctx context.Context, prompt string) (string, error) {
			return llms.GenerateFromSinglePrompt(ctx, compClient, prompt)
		},
	}
}

// Copy the exact same methods from Service but using the interface
func (s *TestableService) StoreDocument(ctx context.Context, req *DocumentRequest) error {
	if req.Text == "" {
		return fmt.Errorf("document text cannot be empty")
	}

	embedding, err := s.embedClient.GenerateEmbeddings(ctx, req.Text)
	if err != nil {
		return fmt.Errorf("failed to generate embedding: %w", err)
	}

	doc := &weaviate.Document{
		ID:       req.ID,
		Text:     req.Text,
		Metadata: req.Metadata,
	}

	if err := s.vectorClient.StoreDocument(ctx, doc, embedding); err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	return nil
}

func (s *TestableService) QueryKnowledge(ctx context.Context, req *QueryRequest) (*QueryResponse, error) {
	if req.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	// Generate query embedding
	queryEmbedding, err := s.embedClient.GenerateEmbeddings(ctx, req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query embedding: %w", err)
	}

	// Search for similar documents
	searchResults, err := s.vectorClient.SearchSimilar(ctx, queryEmbedding, 5)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar documents: %w", err)
	}

	if len(searchResults) == 0 {
		return &QueryResponse{
			Answer:         "I don't have enough information to answer your question.",
			Sources:        []*weaviate.SearchResult{},
			ContextUsed:    "",
			RelevanceScore: 0.0,
		}, nil
	}

	// Build context from search results
	var contextParts []string
	var totalScore float32
	for _, result := range searchResults {
		contextParts = append(contextParts, result.Document.Text)
		totalScore += result.Score
	}
	context := strings.Join(contextParts, "\n\n")

	// Create prompt template using LangChainGo
	template := `Context information is below:
---
{{.context}}
---

Based on the context information above, please answer the following question:
{{.question}}

If the context doesn't contain enough information to answer the question, please say so.`

	promptTemplate := prompts.NewPromptTemplate(template, []string{"context", "question"})

	// Format the prompt with actual values
	formattedPrompt, err := promptTemplate.Format(map[string]any{
		"context":  context,
		"question": req.Query,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to format prompt: %w", err)
	}

	// Generate response using LangChainGo
	response, err := s.generateCompletion(ctx, formattedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate response: %w", err)
	}

	avgScore := totalScore / float32(len(searchResults))

	return &QueryResponse{
		Answer:         response,
		Sources:        searchResults,
		ContextUsed:    context,
		RelevanceScore: avgScore,
	}, nil
}

func (s *TestableService) HealthCheck(ctx context.Context) error {
	if err := s.embedClient.Health(ctx); err != nil {
		return fmt.Errorf("embedding service unhealthy: %w", err)
	}

	if err := s.vectorClient.Health(ctx); err != nil {
		return fmt.Errorf("vector database unhealthy: %w", err)
	}

	return nil
}

func (s *TestableService) generateCompletion(ctx context.Context, prompt string) (string, error) {
	return s.completionFunc(ctx, prompt)
}

// MockEmbeddingClient is a mock for the EmbeddingClient interface.
type MockEmbeddingClient struct {
	mock.Mock
}

func (m *MockEmbeddingClient) GenerateEmbeddings(ctx context.Context, text string) ([]float32, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float32), args.Error(1)
}

func (m *MockEmbeddingClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

// MockCompletionClient is a mock for LangChainGo's LLM interface.
type MockCompletionClient struct {
	mock.Mock
}

func (m *MockCompletionClient) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	args := m.Called(ctx, prompt, options)
	return args.String(0), args.Error(1)
}


func (m *MockCompletionClient) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	return nil, nil
}

// MockWeaviateClient is a mock for the Weaviate client.
type MockWeaviateClient struct {
	mock.Mock
}

func (m *MockWeaviateClient) StoreDocument(ctx context.Context, doc *weaviate.Document, embedding []float32) error {
	args := m.Called(ctx, doc, embedding)
	return args.Error(0)
}

func (m *MockWeaviateClient) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]*weaviate.SearchResult, error) {
	args := m.Called(ctx, embedding, limit)
	return args.Get(0).([]*weaviate.SearchResult), args.Error(1)
}

func (m *MockWeaviateClient) Health(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockWeaviateClient) EnsureSchema(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestQueryKnowledge(t *testing.T) {
	// 1. Setup Mocks
	mockEmbedder := new(MockEmbeddingClient)
	mockCompleter := new(MockCompletionClient)
	mockWeaviate := new(MockWeaviateClient)

	// Define expected inputs and outputs
	question := "What is Telar?"
	expectedAnswer := "Telar is a cool project."
	mockEmbedding := []float32{0.1, 0.2, 0.3}
	mockDocs := []*weaviate.SearchResult{
		{
			Document: &weaviate.Document{
				ID:   "doc1",
				Text: "Telar is a social platform built with Go and React.",
				Metadata: map[string]string{
					"source": "docs",
				},
			},
			Score: 0.95,
		},
		{
			Document: &weaviate.Document{
				ID:   "doc2",
				Text: "The Telar platform supports AI-powered features.",
				Metadata: map[string]string{
					"source": "readme",
				},
			},
			Score: 0.85,
		},
	}

	mockEmbedder.On("GenerateEmbeddings", mock.Anything, question).Return(mockEmbedding, nil)
	mockWeaviate.On("SearchSimilar", mock.Anything, mockEmbedding, 5).Return(mockDocs, nil)

	// 2. Instantiate the service with mocks
	service := NewTestableService(mockEmbedder, mockCompleter, mockWeaviate, Config{
		EmbeddingModel: "test-model",
	})
	
	// Override the completion function with a mock
	service.completionFunc = func(ctx context.Context, prompt string) (string, error) {
		return expectedAnswer, nil
	}

	// 3. Call the method to test
	req := &QueryRequest{
		Query: question,
	}
	answer, err := service.QueryKnowledge(context.Background(), req)

	// 4. Assert the results
	assert.NoError(t, err)
	assert.NotNil(t, answer)
	assert.Equal(t, expectedAnswer, answer.Answer)
	assert.Equal(t, mockDocs, answer.Sources)
	assert.Equal(t, float32(0.9), answer.RelevanceScore) // (0.95 + 0.85) / 2

	mockEmbedder.AssertExpectations(t)
	mockWeaviate.AssertExpectations(t)
}

func TestStoreDocument(t *testing.T) {
	// Setup mocks
	mockEmbedder := new(MockEmbeddingClient)
	mockCompleter := new(MockCompletionClient)
	mockWeaviate := new(MockWeaviateClient)

	// Test data
	docReq := &DocumentRequest{
		ID:   "test-doc",
		Text: "This is a test document",
		Metadata: map[string]string{
			"source": "test",
		},
	}
	mockEmbedding := []float32{0.1, 0.2, 0.3, 0.4}

	// Setup expectations
	mockEmbedder.On("GenerateEmbeddings", mock.Anything, docReq.Text).Return(mockEmbedding, nil)
	mockWeaviate.On("StoreDocument", mock.Anything, mock.AnythingOfType("*weaviate.Document"), mockEmbedding).Return(nil)

	// Instantiate service
	service := NewTestableService(mockEmbedder, mockCompleter, mockWeaviate, Config{
		EmbeddingModel: "test-model",
	})

	// Execute test
	err := service.StoreDocument(context.Background(), docReq)

	// Assert results
	assert.NoError(t, err)

	// Verify expectations
	mockEmbedder.AssertExpectations(t)
	mockWeaviate.AssertExpectations(t)
}

func TestHealthCheck(t *testing.T) {
	// Setup mocks
	mockEmbedder := new(MockEmbeddingClient)
	mockCompleter := new(MockCompletionClient)
	mockWeaviate := new(MockWeaviateClient)

	// Setup expectations - all health checks pass
	mockWeaviate.On("Health", mock.Anything).Return(nil)
	mockEmbedder.On("Health", mock.Anything).Return(nil)

	// Instantiate service
	service := NewTestableService(mockEmbedder, mockCompleter, mockWeaviate, Config{
		EmbeddingModel: "test-model",
	})

	// Execute test
	err := service.HealthCheck(context.Background())

	// Assert results
	assert.NoError(t, err)

	// Verify expectations
	mockEmbedder.AssertExpectations(t)
	mockWeaviate.AssertExpectations(t)
}

func TestHealthCheck_WeaviateFailure(t *testing.T) {
	// Setup mocks
	mockEmbedder := new(MockEmbeddingClient)
	mockCompleter := new(MockCompletionClient)
	mockWeaviate := new(MockWeaviateClient)

	// Setup expectations - Weaviate health check fails
	mockWeaviate.On("Health", mock.Anything).Return(assert.AnError)
	mockEmbedder.On("Health", mock.Anything).Return(nil)

	// Instantiate service
	service := NewTestableService(mockEmbedder, mockCompleter, mockWeaviate, Config{
		EmbeddingModel: "test-model",
	})

	// Execute test
	err := service.HealthCheck(context.Background())

	// Assert results
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "vector database unhealthy")

	// Verify expectations
	mockEmbedder.AssertExpectations(t)
	mockWeaviate.AssertExpectations(t)
}
