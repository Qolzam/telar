package llm

import "context"

// ModelType defines the class of model to be used for a task.
type ModelType string

const (
	// ModelTypeGeneration is used for creative tasks like RAG responses and conversation starters
	ModelTypeGeneration ModelType = "generation"
	// ModelTypeClassification is used for fast classification tasks like content moderation
	ModelTypeClassification ModelType = "classification"
)

// CompletionClient is responsible for generating text responses from a prompt.
// It accepts a ModelType to select the appropriate underlying model.
type CompletionClient interface {
	GenerateCompletion(ctx context.Context, modelType ModelType, prompt string) (string, error)
	Health(ctx context.Context) error
}

// EmbeddingClient is responsible for turning text into vector embeddings.
type EmbeddingClient interface {
	GenerateEmbeddings(ctx context.Context, text string) ([]float32, error)
	Health(ctx context.Context) error
}
