package moderation

import (
	"context"
	"errors"
	"log"
)

type Pipeline struct {
	layers []ContentModerator
	cache  *InMemoryCache // Explicit reference for Set() operations
}

func NewPipeline(cache *InMemoryCache, layers ...ContentModerator) *Pipeline {
	// Cache is prepended to layers for execution order
	allLayers := append([]ContentModerator{cache}, layers...)
	return &Pipeline{
		layers: allLayers,
		cache:  cache,
	}
}

func (p *Pipeline) Execute(ctx context.Context, text string) (*ModerationResult, error) {
	for i, layer := range p.layers {
		result, err := layer.Moderate(ctx, text)
		if err != nil {
			log.Printf("⚠️ Layer %s failed: %v", layer.Name(), err)
			continue
		}

		if result != nil {
			// Cache results from expensive layers (L3/L4), not from cache itself
			if i > 0 && result.ModelUsed != p.cache.Name() {
				p.cache.Set(text, result)
			}
			return result, nil
		}
	}
	return nil, errors.New("moderation pipeline exhausted with no result")
}



