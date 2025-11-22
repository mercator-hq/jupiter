package routing

import (
	"context"
	"fmt"

	"mercator-hq/jupiter/pkg/providers"
)

// MockProvider is a mock implementation of the Provider interface for testing.
type MockProvider struct {
	name     string
	provType string
	healthy  bool
	config   providers.ProviderConfig
	health   providers.ProviderHealth
}

// NewMockProvider creates a new mock provider with the given name.
func NewMockProvider(name string) *MockProvider {
	return &MockProvider{
		name:     name,
		provType: "mock",
		healthy:  true,
		config:   providers.ProviderConfig{},
		health: providers.ProviderHealth{
			IsHealthy: true,
		},
	}
}

// SetHealthy sets the health status of the mock provider.
func (m *MockProvider) SetHealthy(healthy bool) {
	m.healthy = healthy
	m.health.IsHealthy = healthy
}

// SendCompletion is not implemented for mock provider.
func (m *MockProvider) SendCompletion(ctx context.Context, req *providers.CompletionRequest) (*providers.CompletionResponse, error) {
	if !m.healthy {
		return nil, fmt.Errorf("provider %s is unhealthy", m.name)
	}
	return &providers.CompletionResponse{
		Content: "mock response",
	}, nil
}

// StreamCompletion is not implemented for mock provider.
func (m *MockProvider) StreamCompletion(ctx context.Context, req *providers.CompletionRequest) (<-chan *providers.StreamChunk, error) {
	return nil, fmt.Errorf("streaming not implemented for mock provider")
}

// HealthCheck performs a health check.
func (m *MockProvider) HealthCheck(ctx context.Context) error {
	if !m.healthy {
		return fmt.Errorf("provider %s is unhealthy", m.name)
	}
	return nil
}

// GetName returns the provider name.
func (m *MockProvider) GetName() string {
	return m.name
}

// GetType returns the provider type.
func (m *MockProvider) GetType() string {
	return m.provType
}

// GetConfig returns the provider configuration.
func (m *MockProvider) GetConfig() providers.ProviderConfig {
	return m.config
}

// IsHealthy returns the current health status.
func (m *MockProvider) IsHealthy() bool {
	return m.healthy
}

// GetHealth returns detailed health information.
func (m *MockProvider) GetHealth() providers.ProviderHealth {
	return m.health
}

// Close closes the provider.
func (m *MockProvider) Close() error {
	return nil
}
