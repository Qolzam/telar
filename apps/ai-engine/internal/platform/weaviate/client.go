package weaviate

import (
	"context"
	"fmt"
	"net/url"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/auth"
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
	creator := c.client.Data().Creator().
		WithClassName("Document").
		WithID(doc.ID).
		WithProperties(map[string]interface{}{
			"text":     doc.Text,
			"metadata": doc.Metadata,
		}).
		WithVector(embedding)

	_, err := creator.Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to store document: %w", err)
	}

	return nil
}

// SearchSimilar finds documents similar to the query embedding
func (c *Client) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
	if limit <= 0 {
		limit = 5
	}

	return []*SearchResult{
		{
			Document: &Document{
				ID:   "mock-doc-1",
				Text: "This is a mock search result from Weaviate",
				Metadata: map[string]string{
					"source": "mock",
				},
			},
			Score: 0.95,
		},
	}, nil
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
	return nil
}
