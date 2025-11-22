package strategies

import (
	"sync"
	"testing"

	"mercator-hq/jupiter/internal/routing"
	"mercator-hq/jupiter/pkg/providers"
	pkgrouting "mercator-hq/jupiter/pkg/routing"
)

func TestNewRoundRobinStrategy(t *testing.T) {
	tests := []struct {
		name    string
		weights map[string]int
	}{
		{
			name:    "with weights",
			weights: map[string]int{"openai": 2, "anthropic": 1},
		},
		{
			name:    "with nil weights",
			weights: nil,
		},
		{
			name:    "with empty weights",
			weights: map[string]int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewRoundRobinStrategy(tt.weights)
			if strategy == nil {
				t.Fatal("NewRoundRobinStrategy() returned nil")
			}
			if strategy.weights == nil {
				t.Error("strategy.weights should not be nil")
			}
		})
	}
}

func TestRoundRobinStrategy_SelectProvider(t *testing.T) {
	tests := []struct {
		name      string
		providers []providers.Provider
		weights   map[string]int
		wantErr   bool
	}{
		{
			name: "single provider",
			providers: []providers.Provider{
				routing.NewMockProvider("openai"),
			},
			weights: nil,
			wantErr: false,
		},
		{
			name: "multiple providers no weights",
			providers: []providers.Provider{
				routing.NewMockProvider("openai"),
				routing.NewMockProvider("anthropic"),
				routing.NewMockProvider("ollama"),
			},
			weights: nil,
			wantErr: false,
		},
		{
			name: "multiple providers with weights",
			providers: []providers.Provider{
				routing.NewMockProvider("openai"),
				routing.NewMockProvider("anthropic"),
			},
			weights: map[string]int{
				"openai":    2,
				"anthropic": 1,
			},
			wantErr: false,
		},
		{
			name:      "no providers",
			providers: []providers.Provider{},
			weights:   nil,
			wantErr:   true,
		},
		{
			name:      "nil providers",
			providers: nil,
			weights:   nil,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := NewRoundRobinStrategy(tt.weights)
			req := &pkgrouting.RoutingRequest{
				RequestID: "test-req",
				Model:     "gpt-4",
			}

			provider, err := strategy.SelectProvider(req, tt.providers)

			if (err != nil) != tt.wantErr {
				t.Errorf("SelectProvider() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && provider == nil {
				t.Error("SelectProvider() returned nil provider without error")
			}
		})
	}
}

func TestRoundRobinStrategy_EvenDistribution(t *testing.T) {
	providers := []providers.Provider{
		routing.NewMockProvider("openai"),
		routing.NewMockProvider("anthropic"),
		routing.NewMockProvider("ollama"),
	}

	strategy := NewRoundRobinStrategy(nil)
	req := &pkgrouting.RoutingRequest{
		RequestID: "test-req",
		Model:     "gpt-4",
	}

	// Track selection counts
	counts := make(map[string]int)
	iterations := 300 // 100 per provider

	for i := 0; i < iterations; i++ {
		provider, err := strategy.SelectProvider(req, providers)
		if err != nil {
			t.Fatalf("SelectProvider() error = %v", err)
		}
		counts[provider.GetName()]++
	}

	// Each provider should get exactly 100 requests
	expectedCount := iterations / len(providers)
	for _, p := range providers {
		name := p.GetName()
		if counts[name] != expectedCount {
			t.Errorf("Provider %s got %d requests, expected %d", name, counts[name], expectedCount)
		}
	}
}

func TestRoundRobinStrategy_WeightedDistribution(t *testing.T) {
	providers := []providers.Provider{
		routing.NewMockProvider("openai"),
		routing.NewMockProvider("anthropic"),
	}

	weights := map[string]int{
		"openai":    2, // 2x more traffic
		"anthropic": 1,
	}

	strategy := NewRoundRobinStrategy(weights)
	req := &pkgrouting.RoutingRequest{
		RequestID: "test-req",
		Model:     "gpt-4",
	}

	// Track selection counts
	counts := make(map[string]int)
	iterations := 300 // OpenAI should get 200, Anthropic should get 100

	for i := 0; i < iterations; i++ {
		provider, err := strategy.SelectProvider(req, providers)
		if err != nil {
			t.Fatalf("SelectProvider() error = %v", err)
		}
		counts[provider.GetName()]++
	}

	// OpenAI should get 2/3 of requests (200)
	// Anthropic should get 1/3 of requests (100)
	expectedOpenAI := 200
	expectedAnthropic := 100

	if counts["openai"] != expectedOpenAI {
		t.Errorf("OpenAI got %d requests, expected %d", counts["openai"], expectedOpenAI)
	}
	if counts["anthropic"] != expectedAnthropic {
		t.Errorf("Anthropic got %d requests, expected %d", counts["anthropic"], expectedAnthropic)
	}
}

func TestRoundRobinStrategy_ZeroWeight(t *testing.T) {
	providers := []providers.Provider{
		routing.NewMockProvider("openai"),
		routing.NewMockProvider("anthropic"),
		routing.NewMockProvider("ollama"),
	}

	weights := map[string]int{
		"openai":    1,
		"anthropic": 1,
		"ollama":    0, // Exclude ollama
	}

	strategy := NewRoundRobinStrategy(weights)
	req := &pkgrouting.RoutingRequest{
		RequestID: "test-req",
		Model:     "gpt-4",
	}

	// Track selection counts
	counts := make(map[string]int)
	iterations := 200

	for i := 0; i < iterations; i++ {
		provider, err := strategy.SelectProvider(req, providers)
		if err != nil {
			t.Fatalf("SelectProvider() error = %v", err)
		}
		counts[provider.GetName()]++
	}

	// Ollama should never be selected
	if counts["ollama"] > 0 {
		t.Errorf("Ollama got %d requests, expected 0 (zero weight)", counts["ollama"])
	}

	// OpenAI and Anthropic should split evenly
	if counts["openai"] != 100 || counts["anthropic"] != 100 {
		t.Errorf("OpenAI got %d, Anthropic got %d, expected 100 each",
			counts["openai"], counts["anthropic"])
	}
}

func TestRoundRobinStrategy_AllZeroWeights(t *testing.T) {
	providers := []providers.Provider{
		routing.NewMockProvider("openai"),
		routing.NewMockProvider("anthropic"),
	}

	weights := map[string]int{
		"openai":    0,
		"anthropic": 0,
	}

	strategy := NewRoundRobinStrategy(weights)
	req := &pkgrouting.RoutingRequest{
		RequestID: "test-req",
		Model:     "gpt-4",
	}

	// Should fall back to unweighted distribution
	counts := make(map[string]int)
	iterations := 100

	for i := 0; i < iterations; i++ {
		provider, err := strategy.SelectProvider(req, providers)
		if err != nil {
			t.Fatalf("SelectProvider() error = %v", err)
		}
		counts[provider.GetName()]++
	}

	// Both providers should get some requests (fallback behavior)
	if counts["openai"] == 0 || counts["anthropic"] == 0 {
		t.Errorf("With all zero weights, should fall back to unweighted. Got OpenAI=%d, Anthropic=%d",
			counts["openai"], counts["anthropic"])
	}
}

func TestRoundRobinStrategy_ConcurrentAccess(t *testing.T) {
	providers := []providers.Provider{
		routing.NewMockProvider("openai"),
		routing.NewMockProvider("anthropic"),
		routing.NewMockProvider("ollama"),
	}

	strategy := NewRoundRobinStrategy(nil)

	// Run concurrent requests
	concurrency := 100
	requestsPerGoroutine := 100

	var wg sync.WaitGroup
	counts := make(map[string]*int)
	var mu sync.Mutex

	// Initialize counts
	for _, p := range providers {
		count := 0
		counts[p.GetName()] = &count
	}

	// Launch concurrent goroutines
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < requestsPerGoroutine; j++ {
				req := &pkgrouting.RoutingRequest{
					RequestID: "test-req",
					Model:     "gpt-4",
				}

				provider, err := strategy.SelectProvider(req, providers)
				if err != nil {
					t.Errorf("Goroutine %d: SelectProvider() error = %v", id, err)
					return
				}

				// Safely increment count
				mu.Lock()
				*counts[provider.GetName()]++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Verify total requests
	totalRequests := concurrency * requestsPerGoroutine
	sum := 0
	for _, count := range counts {
		sum += *count
	}

	if sum != totalRequests {
		t.Errorf("Total requests = %d, expected %d", sum, totalRequests)
	}

	// Each provider should get approximately 1/3 of requests
	expectedPerProvider := totalRequests / len(providers)
	tolerance := expectedPerProvider / 10 // 10% tolerance

	for name, count := range counts {
		diff := abs(*count - expectedPerProvider)
		if diff > tolerance {
			t.Logf("Provider %s got %d requests (expected ~%d, tolerance %d)",
				name, *count, expectedPerProvider, tolerance)
		}
	}
}

func TestRoundRobinStrategy_Reset(t *testing.T) {
	strategy := NewRoundRobinStrategy(nil)

	providers := []providers.Provider{
		routing.NewMockProvider("openai"),
		routing.NewMockProvider("anthropic"),
	}

	req := &pkgrouting.RoutingRequest{
		RequestID: "test-req",
		Model:     "gpt-4",
	}

	// Select a few providers
	for i := 0; i < 10; i++ {
		_, err := strategy.SelectProvider(req, providers)
		if err != nil {
			t.Fatalf("SelectProvider() error = %v", err)
		}
	}

	// Counter should be > 0
	if strategy.counter.Load() == 0 {
		t.Error("Counter should be > 0 before reset")
	}

	// Reset
	strategy.Reset()

	// Counter should be 0
	if strategy.counter.Load() != 0 {
		t.Errorf("Counter after reset = %d, expected 0", strategy.counter.Load())
	}
}

func TestRoundRobinStrategy_CounterOverflow(t *testing.T) {
	strategy := NewRoundRobinStrategy(nil)

	// Use multiple providers to trigger round-robin logic
	providers := []providers.Provider{
		routing.NewMockProvider("openai"),
		routing.NewMockProvider("anthropic"),
	}

	req := &pkgrouting.RoutingRequest{
		RequestID: "test-req",
		Model:     "gpt-4",
	}

	// Set counter near overflow threshold
	strategy.counter.Store(1_000_000_001)

	// Next selection should reset counter
	provider, err := strategy.SelectProvider(req, providers)
	if err != nil {
		t.Fatalf("SelectProvider() error = %v", err)
	}
	if provider == nil {
		t.Fatal("SelectProvider() returned nil provider")
	}

	// Counter should be reset to 0 via CompareAndSwap
	if strategy.counter.Load() != 0 {
		t.Errorf("Counter after overflow = %d, expected 0", strategy.counter.Load())
	}
}

func TestRoundRobinStrategy_GetName(t *testing.T) {
	strategy := NewRoundRobinStrategy(nil)
	if strategy.GetName() != "round-robin" {
		t.Errorf("GetName() = %s, expected %s", strategy.GetName(), "round-robin")
	}
}

// Helper function
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
