package weaviate

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// Client wraps the Weaviate Go client with AI Engine specific functionality
type Client struct {
	client *weaviate.Client
}

// Config contains Weaviate connection settings
type Config struct {
	URL    string
	APIKey string
}

type Document struct {
	ID       string            `json:"id"`
	Text     string            `json:"text"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type SearchResult struct {
	Document *Document `json:"document"`
	Score    float32   `json:"score"`
}

// NewClient creates a new Weaviate client instance
func NewClient(config Config) (*Client, error) {
	parsedURL, err := url.Parse(config.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid Weaviate URL: %w", err)
	}

	cfg := weaviate.Config{
		Host:   parsedURL.Host,
		Scheme: parsedURL.Scheme,
	}

	if config.APIKey != "" {
		cfg.AuthConfig = auth.ApiKey{Value: config.APIKey}
	}

	client, err := weaviate.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create Weaviate client: %w", err)
	}

	return &Client{client: client}, nil
}

// StoreDocument saves a document with its vector embedding to Weaviate
// This is kept for backward compatibility but is deprecated in favor of StoreChunk
func (c *Client) StoreDocument(ctx context.Context, doc *Document, embedding []float32) error {
	return c.StoreChunk(ctx, doc.ID, 0, doc.Text, doc.Metadata, embedding)
}

// StoreChunk saves a document chunk with its vector embedding to Weaviate
func (c *Client) StoreChunk(ctx context.Context, sourceID string, chunkIndex int, content string, metadata map[string]string, embedding []float32) error {
	// extract source from metadata, provide a default if not present
	source, ok := metadata["source"]
	if !ok {
		source = "unknown"
	}

	properties := map[string]interface{}{
		"content":     content,
		"source_id":   sourceID,
		"chunk_index": chunkIndex,
		"source":      source,
	}

	_, err := c.client.Data().Creator().
		WithClassName("DocumentChunk").
		WithProperties(properties).
		WithVector(embedding).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to store chunk: %w", err)
	}

	return nil
}

// SearchSimilar finds document chunks similar to the query embedding using vector similarity search
func (c *Client) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
	start := time.Now()
	className := "DocumentChunk"
	if limit <= 0 {
		limit = 5
	}

	// define the fields we want to retrieve
	fields := []graphql.Field{
		{Name: "content"},
		{Name: "source_id"},
		{Name: "chunk_index"},
		{Name: "source"},
		{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "certainty"}, // certainty is Weaviate's score (0 to 1)
		}},
	}

	// build the nearVector operator
	nearVector := c.client.GraphQL().NearVectorArgBuilder().
		WithVector(embedding)

	// execute the query
	queryStart := time.Now()
	response, err := c.client.GraphQL().Get().
		WithClassName(className).
		WithFields(fields...).
		WithNearVector(nearVector).
		WithLimit(limit).
		Do(ctx)
	queryDuration := time.Since(queryStart)
	if err != nil {
		log.Printf("[TIMING] vector_search failed embedding_dim=%d limit=%d query_duration_ms=%d total_duration_ms=%d error=%v",
			len(embedding), limit, queryDuration.Milliseconds(), time.Since(start).Milliseconds(), err)
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	parseStart := time.Now()
	var searchResults []*SearchResult
	if getResult, ok := response.Data["Get"].(map[string]interface{}); ok {
		if chunks, ok := getResult[className].([]interface{}); ok {
			for _, chunkRaw := range chunks {
				chunkMap := chunkRaw.(map[string]interface{})

				content := chunkMap["content"].(string)

				// extract source_id, chunk_index, and source
				sourceID := ""
				if sourceIDVal, ok := chunkMap["source_id"].(string); ok {
					sourceID = sourceIDVal
				}

				chunkIndex := 0
				if chunkIndexVal, ok := chunkMap["chunk_index"].(float64); ok {
					chunkIndex = int(chunkIndexVal)
				}

				source := "unknown"
				if sourceVal, ok := chunkMap["source"].(string); ok {
					source = sourceVal
				}

				var id string
				var certainty float32
				if additional, ok := chunkMap["_additional"].(map[string]interface{}); ok {
					id = additional["id"].(string)
					certainty = float32(additional["certainty"].(float64))
				}

				searchResults = append(searchResults, &SearchResult{
					Document: &Document{
						ID:   sourceID, // Use source_id as the document ID for consistency
						Text: content,  // Store chunk content as text
						Metadata: map[string]string{
							"source":      source,
							"chunk_index": fmt.Sprintf("%d", chunkIndex),
							"chunk_id":    id,
						},
					},
					Score: certainty,
				})
			}
		}
	}
	parseDuration := time.Since(parseStart)
	totalDuration := time.Since(start)

	var avgScore float32
	if len(searchResults) > 0 {
		sum := float32(0)
		for _, result := range searchResults {
			sum += result.Score
		}
		avgScore = sum / float32(len(searchResults))
	}

	log.Printf("[TIMING] vector_search embedding_dim=%d limit=%d results_count=%d query_duration_ms=%d parse_duration_ms=%d total_duration_ms=%d avg_score=%.4f",
		len(embedding), limit, len(searchResults), queryDuration.Milliseconds(), parseDuration.Milliseconds(), totalDuration.Milliseconds(), avgScore)

	return searchResults, nil
}

// Health verifies Weaviate service connectivity and readiness
func (c *Client) Health(ctx context.Context) error {
	ready, err := c.client.Misc().ReadyChecker().Do(ctx)
	if err != nil {
		return fmt.Errorf("weaviate health check failed: %w", err)
	}

	if !ready {
		return fmt.Errorf("weaviate is not ready")
	}

	return nil
}

// EnsureSchema creates the required Weaviate schema for AI Engine
func (c *Client) EnsureSchema(ctx context.Context) error {
	className := "DocumentChunk"

	// check if the class already exists
	exists, err := c.client.Schema().ClassExistenceChecker().WithClassName(className).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to check class existence: %w", err)
	}
	if exists {
		// class already exists, no need to create it
		return nil
	}

	// define the class object for DocumentChunk
	classObj := &models.Class{
		Class:       className,
		Description: "A chunk of a document containing text and metadata for the AI Engine RAG system",
		Vectorizer:  "none", // VERY IMPORTANT: We provide our own vectors
		Properties: []*models.Property{
			{
				Name:        "content",
				DataType:    []string{"text"},
				Description: "The text content of the chunk",
			},
			{
				Name:        "source_id",
				DataType:    []string{"text"},
				Description: "The ID of the original document this chunk belongs to",
			},
			{
				Name:        "chunk_index",
				DataType:    []string{"int"},
				Description: "The index of this chunk within the original document (0-based)",
			},
			{
				Name:        "source",
				DataType:    []string{"text"},
				Description: "The source of the document (e.g., URL, filename)",
			},
		},
	}

	// create the class
	err = c.client.Schema().ClassCreator().WithClass(classObj).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}
