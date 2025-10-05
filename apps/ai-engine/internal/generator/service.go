package generator

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/prompts"
)

// Service handles generative AI tasks that don't require RAG.
type Service struct {
	compClient     llms.Model
	semaphore      chan struct{} 
	maxConcurrent  int
	requestTimeout time.Duration
}

// Request represents a queued generation request
type Request struct {
	Topic  string
	Style  string
	Result chan Result
}

// Result represents the result of a generation request
type Result struct {
	Starters []string
	Error    error
}

// Queue manages pending requests
type Queue struct {
	requests chan Request
	mu       sync.RWMutex
	closed   bool
}

// NewService creates a new generator service with concurrent request limiting.
func NewService(compClient llms.Model, maxConcurrent int) *Service {
	if maxConcurrent <= 0 {
		maxConcurrent = 2 
	}
	return &Service{
		compClient:     compClient,
		semaphore:      make(chan struct{}, maxConcurrent),
		maxConcurrent:  maxConcurrent,
		requestTimeout: 60 * time.Second, 
	}
}

// NewQueue creates a new request queue.
func NewQueue(bufferSize int) *Queue {
	return &Queue{
		requests: make(chan Request, bufferSize),
	}
}

// Enqueue adds a request to the queue.
func (q *Queue) Enqueue(req Request) error {
	q.mu.RLock()
	defer q.mu.RUnlock()
	
	if q.closed {
		return fmt.Errorf("queue is closed")
	}
	
	select {
	case q.requests <- req:
		return nil
	default:
		return fmt.Errorf("queue is full, please try again later")
	}
}

// Close closes the queue.
func (q *Queue) Close() {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	if !q.closed {
		close(q.requests)
		q.closed = true
	}
}

// GenerateConversationStarters creates engaging prompts for a community with concurrent request limiting.
func (s *Service) GenerateConversationStarters(ctx context.Context, topic, style string) ([]string, error) {
	// Try to acquire semaphore with timeout
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }() 
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("server is currently processing too many requests, please try again in a moment")
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	genCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	return s.generateConversationStartersInternal(genCtx, topic, style)
}

// generateConversationStartersInternal performs the actual generation work.
func (s *Service) generateConversationStartersInternal(ctx context.Context, topic, style string) ([]string, error) {
	// A robust, instruction-following prompt template.
	prompt := prompts.NewPromptTemplate(
		`You are an expert community manager. Your task is to generate three high-quality, open-ended discussion prompts for a community of '{{.topic}}'.
		The desired tone is '{{.style}}'.
		You must follow these rules:
		1. Generate exactly three distinct prompts.
		2. The prompts must be engaging and encourage conversation.
		3. Your response MUST be ONLY a raw JSON array of strings, with no other text, comments, or explanations.
		Example response: ["What is your favorite new feature in Go 1.23?", "How do you handle work-life balance as a developer?", "If you could refactor any public Go repository, which one would it be and why?"]`,
		[]string{"topic", "style"},
	)

	// Use LangChainGo to format the prompt, just like in the KnowledgeService.
	formattedPrompt, err := prompt.Format(map[string]any{
		"topic": topic,
		"style": style,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to format prompt: %w", err)
	}

	// Call your existing, provider-agnostic completion client.
	response, err := llms.GenerateFromSinglePrompt(ctx, s.compClient, formattedPrompt)
	if err != nil {
		return nil, fmt.Errorf("llm client failed to generate starters: %w", err)
	}

	// Parse the clean JSON response from the LLM.
	var starters []string
	if err := json.Unmarshal([]byte(response), &starters); err != nil {
		// This is a critical fallback for when the LLM doesn't follow instructions perfectly.
		return nil, fmt.Errorf("failed to parse LLM JSON response: %w. Raw response: %s", err, response)
	}

	return starters, nil
}

// GetConcurrentStatus returns the current status of concurrent request handling.
func (s *Service) GetConcurrentStatus() map[string]interface{} {
	activeRequests := len(s.semaphore)
	availableSlots := s.maxConcurrent - activeRequests
	
	return map[string]interface{}{
		"max_concurrent":    s.maxConcurrent,
		"active_requests":   activeRequests,
		"available_slots":   availableSlots,
		"request_timeout":   s.requestTimeout.String(),
		"can_accept_request": availableSlots > 0,
	}
}
