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
	embedClient    llm.EmbeddingClient
	compClient     llms.Model
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

// GenerationRequest represents a request for generating conversation starters
type GenerationRequest struct {
	Topic string `json:"topic"`
	Style string `json:"style"`
	Count int    `json:"count"`
}

// GenerationResponse represents the result of conversation starter generation
type GenerationResponse struct {
	Starters         []string `json:"starters"`
	Model            string   `json:"model"`
	PromptTokens     int      `json:"prompt_tokens,omitempty"`
	CompletionTokens int      `json:"completion_tokens,omitempty"`
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

// GenerateConversationStarters creates engaging conversation starters for community topics
func (s *Service) GenerateConversationStarters(ctx context.Context, req *GenerationRequest) (*GenerationResponse, error) {
	log.Printf("Generating conversation starters for topic: %s", req.Topic)

	prompt := prompts.NewPromptTemplate(`Create {{.count}} conversation starters for: "{{.topic}}"

Style: {{.style_guide}}

MANDATORY REQUIREMENT: Create DIFFERENT formats. DO NOT make all questions!

STRICT FORMAT DISTRIBUTION:
- Only 1 question maximum
- Use statements, tips, challenges, scenarios, opinions

REQUIRED FORMATS (copy these exactly):
FORMAT 1 - STATEMENT: "The biggest mistake in [topic] is..."
FORMAT 2 - TIP: "Pro tip: [specific advice]"  
FORMAT 3 - CHALLENGE: "Try this [topic] challenge: [specific task]"
FORMAT 4 - OPINION: "Unpopular opinion: [controversial take]"
FORMAT 5 - SCENARIO: "Imagine you had to [specific scenario]..."
FORMAT 6 - SHARING: "Share your experience with [specific situation]"

EXAMPLES (follow these patterns):
1. The biggest fitness myth is that you need 2 hours at the gym daily
2. Pro tip: Track your protein intake for one week and you'll be amazed at the gaps
3. Try this challenge: Do 10 push-ups every time you check social media today  
4. Unpopular opinion: Rest days are more important than workout days
5. Imagine you could only do bodyweight exercises for the next month - what's your plan?

Create {{.count}} starters using THESE EXACT FORMATS:`,
		[]string{"topic", "count", "style_guide"},
	)

	var styleGuide string
	switch strings.ToLower(req.Style) {
	case "professional":
		styleGuide = "Keep the tone professional and industry-focused. Emphasize expertise, best practices, and career development."
	case "casual":
		styleGuide = "Use a friendly, relaxed tone. Make it feel like chatting with friends. Include personal experiences and fun elements."
	case "educational":
		styleGuide = "Focus on learning and knowledge sharing. Include questions that help people discover new concepts and deepen understanding."
	case "creative":
		styleGuide = "Encourage imaginative thinking and creative solutions. Include 'what if' scenarios and innovative approaches."
	default: // "engaging" or any other style
		styleGuide = "Balance professionalism with approachability. Make it exciting and thought-provoking while remaining inclusive and welcoming."
	}

	formattedPrompt, err := prompt.Format(map[string]any{
		"topic":       req.Topic,
		"count":       req.Count,
		"style_guide": styleGuide,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to format generation prompt: %w", err)
	}

	completion, err := s.generateCompletion(ctx, formattedPrompt)
	if err != nil {
		return nil, fmt.Errorf("failed to generate completion: %w", err)
	}

	starters := s.parseConversationStarters(completion)

	starters = s.diversifyStarterFormats(starters, req.Topic, req.Style)

	if len(starters) == 0 {
		return nil, fmt.Errorf("failed to generate any conversation starters from completion")
	}

	response := &GenerationResponse{
		Starters: starters,
		Model:    "completion-model", 
	}

	log.Printf("Successfully generated %d conversation starters for topic: %s", len(starters), req.Topic)
	return response, nil
}

// parseConversationStarters extracts individual conversation starters from LLM completion
func (s *Service) parseConversationStarters(completion string) []string {
	var starters []string
	lines := strings.Split(completion, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.Contains(line, ".") && len(line) > 3 {
			
			parts := strings.SplitN(line, ".", 2)
			if len(parts) == 2 {
				firstPart := strings.TrimSpace(parts[0])
				if len(firstPart) <= 3 && strings.ContainsAny(firstPart, "0123456789") {
					line = strings.TrimSpace(parts[1])
				}
			}
		}

		line = strings.Trim(line, "\"'-")
		line = strings.TrimSpace(line)

		lowerLine := strings.ToLower(line)
		isMetaText := strings.HasPrefix(lowerLine, "here are") ||
			strings.HasPrefix(lowerLine, "here is") ||
			strings.HasPrefix(lowerLine, "these are") ||
			strings.HasPrefix(lowerLine, "this is") ||
			strings.Contains(lowerLine, "conversation starter") ||
			strings.Contains(lowerLine, "discussion prompt") ||
			strings.Contains(lowerLine, "community:") ||
			strings.Contains(lowerLine, "for the") && strings.Contains(lowerLine, "community") ||
			strings.HasPrefix(lowerLine, "i hope") ||
			strings.HasPrefix(lowerLine, "these questions") ||
			strings.HasPrefix(lowerLine, "these prompts") ||
			strings.Contains(lowerLine, "encourages sharing") ||
			strings.Contains(lowerLine, "accessible to both beginners")

		if len(line) > 20 && !isMetaText {
			starters = append(starters, line)
		}
	}

	return starters
}

// diversifyStarterFormats transforms some questions into other formats for variety
func (s *Service) diversifyStarterFormats(starters []string, topic, style string) []string {
	if len(starters) < 2 {
		return starters
	}

	diversified := make([]string, len(starters))
	copy(diversified, starters)

	for i := 1; i < len(diversified) && i < 4; i += 2 {
		starter := diversified[i]
		lowerStarter := strings.ToLower(starter)

		if strings.HasPrefix(lowerStarter, "what") || strings.HasPrefix(lowerStarter, "how") ||
			strings.HasPrefix(lowerStarter, "why") || strings.HasPrefix(lowerStarter, "when") ||
			strings.HasPrefix(lowerStarter, "where") || strings.HasSuffix(starter, "?") {

			switch i {
			case 1:
				diversified[i] = s.transformToTip(starter, topic)
			case 3:
				diversified[i] = s.transformToStatement(starter, topic)
			}
		}
	}

	return diversified
}

// transformToTip converts a question to a tip format
func (s *Service) transformToTip(question, topic string) string {
	cleaned := strings.TrimSuffix(question, "?")
	cleaned = strings.TrimSpace(cleaned)

	lowerQ := strings.ToLower(cleaned)

	if strings.HasPrefix(lowerQ, "what's the best") {
		return fmt.Sprintf("Pro tip: %s", strings.TrimPrefix(cleaned, "What's the best "))
	}
	if strings.HasPrefix(lowerQ, "how do you") {
		return fmt.Sprintf("Here's how to %s effectively", strings.TrimPrefix(lowerQ, "how do you "))
	}
	if strings.HasPrefix(lowerQ, "what") {
		return fmt.Sprintf("Remember: %s matters more than you think in %s",
			strings.TrimPrefix(cleaned, "What "), topic)
	}

	return fmt.Sprintf("Pro tip for %s: Focus on %s", topic, strings.ToLower(cleaned))
}

// transformToStatement converts a question to an opinion/statement format
func (s *Service) transformToStatement(question, topic string) string {
	cleaned := strings.TrimSuffix(question, "?")
	cleaned = strings.TrimSpace(cleaned)

	lowerQ := strings.ToLower(cleaned)

	if strings.Contains(lowerQ, "best") || strings.Contains(lowerQ, "favorite") {
		return fmt.Sprintf("Unpopular opinion: The most overrated thing in %s is [fill in the blank]", topic)
	}
	if strings.Contains(lowerQ, "challenge") || strings.Contains(lowerQ, "difficult") {
		return fmt.Sprintf("The biggest %s mistake I see everywhere: [share yours]", topic)
	}
	if strings.Contains(lowerQ, "experience") || strings.Contains(lowerQ, "time") {
		return fmt.Sprintf("Share your most surprising %s discovery", topic)
	}

	return fmt.Sprintf("Hot take: %s isn't as important as everyone thinks in %s",
		strings.Split(lowerQ, " ")[0], topic)
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
