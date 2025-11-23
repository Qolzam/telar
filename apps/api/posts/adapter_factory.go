package posts

import (
	"github.com/qolzam/telar/apps/api/posts/internal/adapters"
	"github.com/qolzam/telar/apps/api/posts/services"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
)

// NewDirectCallStatsUpdater creates a direct call adapter for serverless deployment mode
func NewDirectCallStatsUpdater(service services.PostService) sharedInterfaces.PostStatsUpdater {
	return adapters.NewDirectCallStatsUpdater(service)
}

// NewGrpcStatsUpdater creates a gRPC client adapter for microservices deployment mode
func NewGrpcStatsUpdater(targetAddress string) (sharedInterfaces.PostStatsUpdater, error) {
	return adapters.NewGrpcStatsUpdater(targetAddress)
}

