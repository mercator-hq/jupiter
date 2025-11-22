// Package health provides health check endpoints for Mercator Jupiter.
//
// # Overview
//
// The health package implements liveness and readiness probes for Kubernetes
// and other orchestration systems, along with version information endpoints.
// It provides a framework for checking the health of various system components.
//
// # Endpoints
//
// The package provides three main endpoints:
//
//   - /health: Liveness probe - indicates if the process is running
//   - /ready: Readiness probe - indicates if the system can serve traffic
//   - /version: Build information - version, commit, build time
//
// # Usage
//
//	// Create health checker
//	cfg := &config.HealthConfig{
//	    Enabled:       true,
//	    LivenessPath:  "/health",
//	    ReadinessPath: "/ready",
//	}
//	checker := health.New(cfg)
//
//	// Register component checks
//	checker.RegisterCheck("config", func(ctx context.Context) error {
//	    if cfg == nil {
//	        return errors.New("config not loaded")
//	    }
//	    return nil
//	})
//
//	// Add HTTP handlers
//	http.HandleFunc("/health", checker.LivenessHandler())
//	http.HandleFunc("/ready", checker.ReadinessHandler())
//	http.HandleFunc("/version", checker.VersionHandler("1.0.0", "abc123", "2025-11-20"))
//
// # Liveness vs Readiness
//
// **Liveness Probe** (/health):
//   - Indicates if the process is alive and running
//   - Returns 200 OK if process is alive
//   - Returns 503 Service Unavailable if critical failure
//   - Used by Kubernetes to restart pods
//   - Fast check (<10ms)
//
// **Readiness Probe** (/ready):
//   - Indicates if the system can serve traffic
//   - Checks all registered component health checks
//   - Returns 200 OK if all components are healthy
//   - Returns 503 Service Unavailable if any component is unhealthy
//   - Used by Kubernetes to route traffic
//   - May take longer (up to 1s for all checks)
//
// # Component Health Checks
//
// Components can register health check functions:
//
//	checker.RegisterCheck("providers", func(ctx context.Context) error {
//	    if numHealthyProviders == 0 {
//	        return errors.New("no healthy providers available")
//	    }
//	    return nil
//	})
//
// Common component checks:
//   - config: Configuration loaded and valid
//   - providers: At least one provider is healthy
//   - policy: Policy engine initialized
//   - storage: Storage backend accessible (if enabled)
//
// # Performance
//
// Health checks are designed to be lightweight:
//   - Liveness: <10ms
//   - Readiness: <100ms (all component checks)
//   - Version: <1ms
//
// # Example Response
//
// Liveness response (/health):
//
//	{
//	    "status": "ok",
//	    "timestamp": "2025-11-20T10:30:00Z"
//	}
//
// Readiness response (/ready):
//
//	{
//	    "status": "ready",
//	    "checks": {
//	        "config": {"status": "ok"},
//	        "providers": {"status": "ok"},
//	        "policy": {"status": "ok"},
//	        "storage": {"status": "disabled"}
//	    },
//	    "timestamp": "2025-11-20T10:30:00Z"
//	}
//
// Degraded response (/ready):
//
//	{
//	    "status": "degraded",
//	    "checks": {
//	        "config": {"status": "ok"},
//	        "providers": {"status": "unhealthy", "message": "no healthy providers"},
//	        "policy": {"status": "ok"}
//	    },
//	    "timestamp": "2025-11-20T10:30:00Z"
//	}
//
// Version response (/version):
//
//	{
//	    "version": "1.0.0",
//	    "commit": "abc123def456",
//	    "build_time": "2025-11-20T00:00:00Z",
//	    "go_version": "go1.21.5"
//	}
package health
