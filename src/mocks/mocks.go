package mocks

import (
	"context"

	"github.com/stretchr/testify/mock"
	"www.github.com/Wanderer0074348/HybridLM/src/models"
)

// MockLLMClient implements models.LLMInferencer
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Infer(ctx context.Context, req *models.InferenceRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

// MockSLMEngine implements models.SLMInferencer
type MockSLMEngine struct {
	mock.Mock
}

func (m *MockSLMEngine) Infer(ctx context.Context, req *models.InferenceRequest) (string, error) {
	args := m.Called(ctx, req)
	return args.String(0), args.Error(1)
}

func (m *MockSLMEngine) Close() error {
	args := m.Called()
	return args.Error(0)
}

// MockCache implements models.CacheStore
type MockCache struct {
	mock.Mock
}

func (m *MockCache) Get(ctx context.Context, key string) (*models.InferenceResponse, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.InferenceResponse), args.Error(1)
}

func (m *MockCache) Set(ctx context.Context, key string, response *models.InferenceResponse) error {
	args := m.Called(ctx, key, response)
	return args.Error(0)
}

func (m *MockCache) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCache) Close() error {
	args := m.Called()
	return args.Error(0)
}
