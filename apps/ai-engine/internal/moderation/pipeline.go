package moderation

import (
	"context"
	"log"
)

type Pipeline struct {
	cache   *InMemoryCache
	filters []ContentModerator // L2 Heuristics
	llm     ContentModerator   // L3 Deep Analysis
}

func NewPipeline(cache *InMemoryCache, llm ContentModerator, filters ...ContentModerator) *Pipeline {
	return &Pipeline{
		cache:   cache,
		llm:     llm,
		filters: filters,
	}
}

// Execute runs the tiered moderation strategy
func (p *Pipeline) Execute(ctx context.Context, text string) (*ModerationResult, error) {
	// 1. Check L1 Cache
	if res, err := p.cache.Moderate(ctx, text); err == nil && res != nil {
		log.Printf("Moderation: L1 Cache Hit")
		return res, nil
	}

	// 2. Check L2 Heuristics (Keywords, Regex)
	for _, filter := range p.filters {
		if res, err := filter.Moderate(ctx, text); err == nil && res != nil {
			log.Printf("Moderation: L2 Filter Hit - %s", filter.Name())
			// Optimization: We could cache this "bad" result too if we wanted
			return res, nil
		}
	}

	// 3. Fallback to L3 Deep Analysis (LLM)
	log.Printf("Moderation: L1/L2 Miss. Delegating to L3 AI.")
	res, err := p.llm.Moderate(ctx, text)
	if err != nil {
		return nil, err
	}

	// 4. Update L1 Cache with the expensive result
	p.cache.Set(text, res)

	return res, nil
}


