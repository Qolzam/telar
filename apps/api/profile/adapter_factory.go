package profile

import (
	"github.com/qolzam/telar/apps/api/profile/internal/adapters"
	"github.com/qolzam/telar/apps/api/profile/services"
)

// NewDirectCallAdapter creates a direct call adapter for serverless deployment mode
func NewDirectCallAdapter(service *services.Service) services.ProfileServiceClient {
	return adapters.NewDirectCallCreator(service)
}

// NewGrpcAdapter creates a gRPC client adapter for microservices deployment mode
func NewGrpcAdapter(targetAddress string) (services.ProfileServiceClient, error) {
	return adapters.NewGrpcCreator(targetAddress)
}

