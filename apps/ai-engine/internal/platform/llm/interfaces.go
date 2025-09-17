package llm

import "context"

// CompletionClient is responsible for generating text responses from a prompt.
type CompletionClient interface {
	GenerateCompletion(ctx context.Context, prompt string) (string, error)
	Health(ctx context.Context) error 
}

// EmbeddingClient is responsible for turning text into vector embeddings.
type EmbeddingClient interface {
	GenerateEmbeddings(ctx context.Context, text string) ([]float32, error)
	Health(ctx context.Context) error 
} 