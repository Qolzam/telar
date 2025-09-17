package weaviate

import (
	"context"
	"fmt"
	"net/url"

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
func (c *Client) StoreDocument(ctx context.Context, doc *Document, embedding []float32) error {
	// extract source from metadata, provide a default if not present
	source, ok := doc.Metadata["source"]
	if !ok {
		source = "unknown"
	}
	
	properties := map[string]interface{}{
		"text":   doc.Text,
		"source": source,
	}

	_, err := c.client.Data().Creator().
		WithClassName("Document").
		// withID is optional, Weaviate can generate one
		WithProperties(properties).
		WithVector(embedding).
		Do(ctx)

	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	return nil
}

// SearchSimilar finds documents similar to the query embedding using vector similarity search
func (c *Client) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
	className := "Document"
	if limit <= 0 {
		limit = 5
	}

	// define the fields we want to retrieve
	fields := []graphql.Field{
		graphql.Field{Name: "text"},
		graphql.Field{Name: "source"}, // source is stored directly, not in metadata
		graphql.Field{Name: "_additional", Fields: []graphql.Field{
			{Name: "id"},
			{Name: "certainty"}, // certainty is Weaviate's score (0 to 1)
		}},
	}

	// build the nearVector operator
	nearVector := c.client.GraphQL().NearVectorArgBuilder().
		WithVector(embedding)

	// execute the query
	response, err := c.client.GraphQL().Get().
		WithClassName(className).
		WithFields(fields...).
		WithNearVector(nearVector).
		WithLimit(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to perform vector search: %w", err)
	}

	var searchResults []*SearchResult
	if getResult, ok := response.Data["Get"].(map[string]interface{}); ok {
		if documents, ok := getResult[className].([]interface{}); ok {
			for _, docRaw := range documents {
				docMap := docRaw.(map[string]interface{})

				text := docMap["text"].(string)
				
				// extract source 
				source := "unknown"
				if sourceVal, ok := docMap["source"].(string); ok {
					source = sourceVal
				}

				var id string
				var certainty float32
				if additional, ok := docMap["_additional"].(map[string]interface{}); ok {
					id = additional["id"].(string)
					certainty = float32(additional["certainty"].(float64))
				}

				searchResults = append(searchResults, &SearchResult{
					Document: &Document{
						ID:   id,
						Text: text,
						Metadata: map[string]string{
							"source": source,
						},
					},
					Score: certainty,
				})
			}
		}
	}

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
	className := "Document"

	// check if the class already exists
	exists, err := c.client.Schema().ClassExistenceChecker().WithClassName(className).Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to check class existence: %w", err)
	}
	if exists {
		// class already exists, no need to create it
		return nil
	}

	// define the class object
	classObj := &models.Class{
		Class:       className,
		Description: "A document containing text and metadata for the AI Engine",
		Vectorizer:  "none", // VERY IMPORTANT: We provide our own vectors
		Properties: []*models.Property{
			{
				Name:        "text",
				DataType:    []string{"text"},
				Description: "The main content of the document",
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
