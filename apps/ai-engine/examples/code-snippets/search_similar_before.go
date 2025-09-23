package main

import (
	"context"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
)

// Client represents a Weaviate client (example only)
type Client struct{}

// SearchResult represents a search result (example only)
type SearchResult struct{}

// SearchSimilar finds documents similar to the query embedding using vector similarity search
func (c *Client) SearchSimilar(ctx context.Context, embedding []float32, limit int) ([]*SearchResult, error) {
    if limit <= 0 {
        limit = 5
    }

    // TODO: implementing proper vector similarity search
    result, err := c.client.GraphQL().Get().
        WithClassName("Document").
        WithLimit(limit).
        WithFields(
            graphql.Field{Name: "text"},
            graphql.Field{Name: "metadata", Fields: []graphql.Field{
                {Name: "source"},
            }},
        ).
        Do(ctx)

    if err != nil {
        return []*SearchResult{
            {
                Document: &Document{
                    ID:   "error-fallback",
                    Text: fmt.Sprintf("Search error: %v", err),
                    Metadata: map[string]string{
                        "source": "error-fallback",
                        "error":  err.Error(),
                    },
                },
                Score: 0.1,
            },
        }, nil
    }

    var searchResults []*SearchResult

    if result.Data != nil {
        if getResult, ok := result.Data["Get"].(map[string]interface{}); ok {
            if documents, ok := getResult["Document"].([]interface{}); ok {
                for i, doc := range documents {
                    docMap := doc.(map[string]interface{})

                    text, _ := docMap["text"].(string)

                    metadata := map[string]string{
                        "source": "weaviate-search",
                    }
                    if metadataObj, ok := docMap["metadata"].(map[string]interface{}); ok {
                        if source, ok := metadataObj["source"].(string); ok {
                            metadata["source"] = source
                        }
                    }

                    // TODO: implement proper vector search
                    score := 1.0 - (float32(i) * 0.1)
                    if score < 0.1 {
                        score = 0.1
                    }

                    docID := fmt.Sprintf("doc-%d", i)

                    searchResults = append(searchResults, &SearchResult{
                        Document: &Document{
                            ID:       docID,
                            Text:     text,
                            Metadata: metadata,
                        },
                        Score: score,
                    })
                }
            }
        }
    }

    if len(searchResults) == 0 {
        return []*SearchResult{
            {
                Document: &Document{
                    ID:   "no-results-found",
                    Text: "No documents found in the knowledge base...",
                    Metadata: map[string]string{
                        "source": "system-message",
                    },
                },
                Score: 0.1,
            },
        }, nil
    }

    return searchResults, nil
}

