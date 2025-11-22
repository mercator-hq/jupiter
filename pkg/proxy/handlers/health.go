package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// HealthHandler handles health check requests for liveness probes.
type HealthHandler struct{}

// NewHealthHandler creates a new health check handler.
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// ServeHTTP implements http.Handler for liveness checks.
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// ReadyHandler handles readiness check requests.
type ReadyHandler struct {
	ProviderManager ProviderManager
}

// NewReadyHandler creates a new readiness check handler.
func NewReadyHandler(pm ProviderManager) *ReadyHandler {
	return &ReadyHandler{ProviderManager: pm}
}

// ServeHTTP implements http.Handler for readiness checks.
func (h *ReadyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get healthy providers
	healthyProviders := h.ProviderManager.GetHealthyProviders()
	healthyCount := len(healthyProviders)

	// Service is ready if at least one provider is healthy
	isReady := healthyCount > 0

	status := "ready"
	statusCode := http.StatusOK
	if !isReady {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := map[string]interface{}{
		"status": status,
		"providers": map[string]interface{}{
			"healthy": healthyCount,
		},
		"timestamp": time.Now().Unix(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// ProviderHealthHandler provides detailed health information.
type ProviderHealthHandler struct {
	ProviderManager ProviderManager
}

// NewProviderHealthHandler creates a new provider health handler.
func NewProviderHealthHandler(pm ProviderManager) *ProviderHealthHandler {
	return &ProviderHealthHandler{ProviderManager: pm}
}

// ServeHTTP implements http.Handler for detailed provider health.
func (h *ProviderHealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	healthyProviders := h.ProviderManager.GetHealthyProviders()

	providersHealth := make(map[string]interface{})
	for name, provider := range healthyProviders {
		health := provider.GetHealth()

		var lastError interface{}
		if health.LastError != nil {
			lastError = health.LastError.Error()
		}

		providersHealth[name] = map[string]interface{}{
			"healthy":    health.IsHealthy,
			"last_check": health.LastCheck.Unix(),
			"last_error": lastError,
		}
	}

	response := map[string]interface{}{
		"providers": providersHealth,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// PerformHealthCheck performs an on-demand health check.
func (h *ProviderHealthHandler) PerformHealthCheck(ctx context.Context) map[string]error {
	healthyProviders := h.ProviderManager.GetHealthyProviders()

	results := make(map[string]error)
	for name, provider := range healthyProviders {
		err := provider.HealthCheck(ctx)
		results[name] = err
	}

	return results
}
