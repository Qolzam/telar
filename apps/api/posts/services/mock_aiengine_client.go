package services

import (
	"context"

	"github.com/qolzam/telar/packages/clients/aiengine"
	"github.com/stretchr/testify/mock"
)

// MockAIEngineClient is a mock implementation of aiengine.Client for testing
type MockAIEngineClient struct {
	mock.Mock
}

// AnalyzeContent mocks the AnalyzeContent method
func (m *MockAIEngineClient) AnalyzeContent(ctx context.Context, req aiengine.AnalysisRequest) (*aiengine.AnalysisResult, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*aiengine.AnalysisResult), args.Error(1)
}


