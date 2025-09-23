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
    className := "Document"
    if limit <= 0 {
        limit = 5
    }

    // Define the fields we want to retrieve
    fields := []graphql.Field{
        graphql.Field{Name: "text"},
        graphql.Field{Name: "metadata", Fields: []graphql.Field{
            {Name: "source"},
        }},
        graphql.Field{Name: "_additional", Fields: []graphql.Field{
            {Name: "id"},
            {Name: "certainty"}, // Certainty is Weaviate's score (0 to 1)
        }},
    }

    // Build the nearVector operator
    nearVector := c.client.GraphQL().NearVectorArgBuilder().
        WithVector(embedding)

    // Execute the query
    response, err := c.client.GraphQL().Get().
        WithClassName(className).
        WithFields(fields...).
        WithNearVector(nearVector).
        WithLimit(limit).
        Do(ctx)
    if err != nil {
        return nil, fmt.Errorf("failed to perform vector search: %w", err)
    }

    // Parse the response
    var searchResults []*SearchResult
    if getResult, ok := response.Data["Get"].(map[string]interface{}); ok {
        if documents, ok := getResult[className].([]interface{}); ok {
            for _, docRaw := range documents {
                docMap := docRaw.(map[string]interface{})

                text := docMap["text"].(string)
                
                // Extract source from nested metadata structure
                source := "unknown"
                if metadata, ok := docMap["metadata"].(map[string]interface{}); ok {
                    if sourceVal, ok := metadata["source"].(string); ok {
                        source = sourceVal
                    }
                }

                var id string
                var certainty float32
                if additional, ok := docMap["_additional"].(map[string]interface{}); ok {
                    id = additional["id"].(string)
                    // Certainty is float64 in the response, cast it
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

