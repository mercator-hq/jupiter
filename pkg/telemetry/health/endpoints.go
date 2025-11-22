package health

import (
	"encoding/json"
	"net/http"
	"runtime"
	"time"
)

// VersionInfo contains build and version information.
type VersionInfo struct {
	// Version is the semantic version (e.g., "1.0.0")
	Version string `json:"version"`

	// Commit is the git commit hash
	Commit string `json:"commit"`

	// BuildTime is when the binary was built
	BuildTime string `json:"build_time"`

	// GoVersion is the Go version used to build
	GoVersion string `json:"go_version"`
}

// LivenessHandler returns an HTTP handler for the liveness probe endpoint.
// It performs a simple check to verify the process is alive.
//
// Example response:
//
//	{
//	    "status": "ok",
//	    "timestamp": "2025-11-20T10:30:00Z"
//	}
func (c *Checker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept GET requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status := c.CheckLiveness(r.Context())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.Method != http.MethodHead {
			_ = json.NewEncoder(w).Encode(status)
		}
	}
}

// ReadinessHandler returns an HTTP handler for the readiness probe endpoint.
// It performs all registered component health checks.
//
// Returns:
//   - 200 OK: System is ready to serve traffic
//   - 503 Service Unavailable: System is not ready (degraded or unhealthy)
//
// Example response (ready):
//
//	{
//	    "status": "ready",
//	    "checks": {
//	        "config": {"status": "ok", "duration_ms": 0.1},
//	        "providers": {"status": "ok", "duration_ms": 5.2}
//	    },
//	    "timestamp": "2025-11-20T10:30:00Z"
//	}
//
// Example response (degraded):
//
//	{
//	    "status": "degraded",
//	    "checks": {
//	        "config": {"status": "ok"},
//	        "providers": {"status": "unhealthy", "message": "no healthy providers"}
//	    },
//	    "timestamp": "2025-11-20T10:30:00Z"
//	}
func (c *Checker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept GET requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		status := c.CheckReadiness(r.Context())

		w.Header().Set("Content-Type", "application/json")

		// Return 503 if not ready
		if status.Status == "degraded" || status.Status == "unhealthy" {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if r.Method != http.MethodHead {
			_ = json.NewEncoder(w).Encode(status)
		}
	}
}

// VersionHandler returns an HTTP handler for the version information endpoint.
// It returns build information including version, commit, and build time.
//
// Example response:
//
//	{
//	    "version": "1.0.0",
//	    "commit": "abc123def456",
//	    "build_time": "2025-11-20T00:00:00Z",
//	    "go_version": "go1.21.5"
//	}
func VersionHandler(version, commit, buildTime string) http.HandlerFunc {
	info := VersionInfo{
		Version:   version,
		Commit:    commit,
		BuildTime: buildTime,
		GoVersion: runtime.Version(),
	}

	return func(w http.ResponseWriter, r *http.Request) {
		// Only accept GET requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if r.Method != http.MethodHead {
			_ = json.NewEncoder(w).Encode(info)
		}
	}
}

// HealthCheckHandlers bundles all health check HTTP handlers.
type HealthCheckHandlers struct {
	// LivenessHandler is the /health endpoint handler
	LivenessHandler http.HandlerFunc

	// ReadinessHandler is the /ready endpoint handler
	ReadinessHandler http.HandlerFunc

	// VersionHandler is the /version endpoint handler
	VersionHandler http.HandlerFunc
}

// CreateHandlers creates HTTP handlers for all health check endpoints.
// This is a convenience function to get all handlers at once.
//
// Usage:
//
//	handlers := checker.CreateHandlers("1.0.0", "abc123", "2025-11-20")
//	http.HandleFunc("/health", handlers.LivenessHandler)
//	http.HandleFunc("/ready", handlers.ReadinessHandler)
//	http.HandleFunc("/version", handlers.VersionHandler)
func (c *Checker) CreateHandlers(version, commit, buildTime string) HealthCheckHandlers {
	return HealthCheckHandlers{
		LivenessHandler:  c.LivenessHandler(),
		ReadinessHandler: c.ReadinessHandler(),
		VersionHandler:   VersionHandler(version, commit, buildTime),
	}
}

// HTTPMiddleware is a middleware that adds health check endpoints to an HTTP mux.
// It registers the standard health check paths:
//   - /health: Liveness probe
//   - /ready: Readiness probe
//   - /version: Version information
//
// Usage:
//
//	mux := http.NewServeMux()
//	checker := health.New(5 * time.Second)
//	health.HTTPMiddleware(mux, checker, "1.0.0", "abc123", "2025-11-20")
func HTTPMiddleware(mux *http.ServeMux, checker *Checker, version, commit, buildTime string) {
	handlers := checker.CreateHandlers(version, commit, buildTime)

	mux.HandleFunc("/health", handlers.LivenessHandler)
	mux.HandleFunc("/ready", handlers.ReadinessHandler)
	mux.HandleFunc("/version", handlers.VersionHandler)
}

// RateLimitedHandler wraps a handler with simple rate limiting.
// It prevents health check endpoint abuse by limiting requests per second.
//
// Usage:
//
//	handler := RateLimitedHandler(checker.LivenessHandler(), 10) // 10 req/s
//	http.HandleFunc("/health", handler)
func RateLimitedHandler(handler http.HandlerFunc, requestsPerSecond int) http.HandlerFunc {
	if requestsPerSecond <= 0 {
		return handler
	}

	limiter := make(chan struct{}, requestsPerSecond)

	// Fill the channel
	for i := 0; i < requestsPerSecond; i++ {
		limiter <- struct{}{}
	}

	// Refill at rate
	ticker := time.NewTicker(time.Second / time.Duration(requestsPerSecond))
	go func() {
		for range ticker.C {
			select {
			case limiter <- struct{}{}:
			default:
			}
		}
	}()

	return func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-limiter:
			handler(w, r)
		default:
			http.Error(w, "Too many requests", http.StatusTooManyRequests)
		}
	}
}
