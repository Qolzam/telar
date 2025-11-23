package comments

import (
	"github.com/qolzam/telar/apps/api/comments/internal/adapters"
	"github.com/qolzam/telar/apps/api/comments/services"
	sharedInterfaces "github.com/qolzam/telar/apps/api/shared/interfaces"
)

// NewDirectCallCounter creates a direct call adapter for serverless deployment mode
func NewDirectCallCounter(service services.CommentService) sharedInterfaces.CommentCounter {
	return adapters.NewDirectCallCounter(service)
}

// NewGrpcCounter creates a gRPC client adapter for microservices deployment mode
func NewGrpcCounter(targetAddress string) (sharedInterfaces.CommentCounter, error) {
	return adapters.NewGrpcCounter(targetAddress)
}

