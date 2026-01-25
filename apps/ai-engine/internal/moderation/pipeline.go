package moderation

import (
	"context"
	"log"
)

type Pipeline struct {
	layers []ContentModerator
	cache  *InMemoryCache // Explicit reference for Set() operations
}

func NewPipeline(cache *InMemoryCache, layers ...ContentModerator) *Pipeline {
	allLayers := append([]ContentModerator{cache}, layers...)
	return &Pipeline{
		layers: allLayers,
		cache:  cache,
	}
}

func (p *Pipeline) Execute(ctx context.Context, text string) (*ModerationResult, error) {
	// Architecture: Only stop early on violations (IsFlagged=true). Don't stop on safe results
	// to allow multi-expert evaluation (e.g., Toxicity=0.01 but Spam=0.99). Only the final
	// layer (L4 LLM) can make the final "Safe" determination.

	if res, err := p.cache.Moderate(ctx, text); err == nil && res != nil {
		return res, nil
	}

	processingLayers := p.layers[1:]
	for i, layer := range processingLayers {
		result, err := layer.Moderate(ctx, text)
		if err != nil {
			log.Printf("⚠️ Layer %s failed: %v", layer.Name(), err)
			continue
		}

		if result != nil {
			if result.IsFlagged {
				p.cache.Set(text, result)
				return result, nil
		}

			if i == len(processingLayers)-1 {
				p.cache.Set(text, result)
				return result, nil
			}
		}
	}

	return &ModerationResult{
		IsFlagged:       false,
		FlagReason:      "safe_fallback",
		SuggestedAction: "approve",
		ModelUsed:       "pipeline-fallback",
	}, nil
}
