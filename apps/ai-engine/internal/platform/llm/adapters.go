package llm

import (
	"context"

	"github.com/tmc/langchaingo/llms"
)

// OllamaLangChainAdapter adapts our OllamaClient to implement the LangChainGo llms.Model interface
type OllamaLangChainAdapter struct {
	client *OllamaClient
}

// Ensure OllamaLangChainAdapter implements llms.Model
var _ llms.Model = (*OllamaLangChainAdapter)(nil)

// NewOllamaLangChainAdapter creates a new adapter for OllamaClient
func NewOllamaLangChainAdapter(client *OllamaClient) *OllamaLangChainAdapter {
	return &OllamaLangChainAdapter{client: client}
}

// Call implements the deprecated Call method for backwards compatibility
func (a *OllamaLangChainAdapter) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return a.client.GenerateCompletion(ctx, prompt)
}

// GenerateContent implements the main LangChainGo interface
func (a *OllamaLangChainAdapter) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// For simplicity, we'll convert the messages to a single prompt
	var prompt string
	for _, msg := range messages {
		for _, part := range msg.Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				prompt += textPart.Text + "\n"
			}
		}
	}

	response, err := a.client.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response,
			},
		},
	}, nil
}

// GroqLangChainAdapter adapts our GroqClient to implement the LangChainGo llms.Model interface
type GroqLangChainAdapter struct {
	client *GroqClient
}

// Ensure GroqLangChainAdapter implements llms.Model
var _ llms.Model = (*GroqLangChainAdapter)(nil)

// NewGroqLangChainAdapter creates a new adapter for GroqClient
func NewGroqLangChainAdapter(client *GroqClient) *GroqLangChainAdapter {
	return &GroqLangChainAdapter{client: client}
}

// Call implements the deprecated Call method for backwards compatibility
func (a *GroqLangChainAdapter) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return a.client.GenerateCompletion(ctx, prompt)
}

// GenerateContent implements the main LangChainGo interface
func (a *GroqLangChainAdapter) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	// For simplicity, we'll convert the messages to a single prompt
	var prompt string
	for _, msg := range messages {
		for _, part := range msg.Parts {
			if textPart, ok := part.(llms.TextContent); ok {
				prompt += textPart.Text + "\n"
			}
		}
	}

	response, err := a.client.GenerateCompletion(ctx, prompt)
	if err != nil {
		return nil, err
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{
				Content: response,
			},
		},
	}, nil
}
