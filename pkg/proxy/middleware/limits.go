package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"mercator-hq/jupiter/pkg/limits"
	"mercator-hq/jupiter/pkg/limits/budget"
	"mercator-hq/jupiter/pkg/limits/enforcement"
	"mercator-hq/jupiter/pkg/limits/ratelimit"
	"mercator-hq/jupiter/pkg/limits/storage"
	"mercator-hq/jupiter/pkg/proxy"
)

// LimitsMiddleware checks rate limits and budgets before forwarding requests.
//
// This middleware:
//   - Extracts identifier (API key, user, team) from request
//   - Checks rate limits and budget limits
//   - Sets rate limit headers (X-RateLimit-*, X-Budget-*)
//   - Blocks or downgrades requests when limits exceeded
//   - Records usage after request completes
//
// Example:
//
//	manager := limits.NewManager(limits.Config{
//	    RateLimits: rateLimitConfigs,
//	    Budgets:    budgetConfigs,
//	})
//	handler := LimitsMiddleware(manager)(next)
func LimitsMiddleware(manager *limits.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Extract identifier (use API key for now, could be user/team)
			identifier := extractIdentifier(r)
			if identifier == "" {
				// No identifier, skip limits check
				next.ServeHTTP(w, r)
				return
			}

			// Get enriched request from context (set by earlier middleware)
			enriched, ok := ctx.Value("enriched_request").(*enrichedRequestContext)
			if !ok || enriched == nil {
				// No enriched data, estimate defaults
				enriched = &enrichedRequestContext{
					estimatedTokens: 1000, // Default estimate
					estimatedCost:   0.01,  // Default cost
					model:           "gpt-4", // Default model
				}
			}

			// Check rate limits and budgets
			result, err := manager.CheckLimits(
				ctx,
				identifier,
				enriched.estimatedTokens,
				enriched.estimatedCost,
				enriched.model,
			)
			if err != nil {
				http.Error(w, "Internal error checking limits", http.StatusInternalServerError)
				return
			}

			// Set rate limit headers
			setLimitHeaders(w, result)

			// Handle limit violations
			if !result.Allowed {
				handleLimitViolation(w, result)
				return
			}

			// Handle downgrade action
			if result.Action == limits.ActionDowngrade && result.DowngradeTo != "" {
				// Store downgrade info in context for routing layer
				ctx = context.WithValue(ctx, "downgraded_model", result.DowngradeTo)
				r = r.WithContext(ctx)
			}

			// Acquire concurrent slot if configured
			if manager.AcquireConcurrent(identifier) {
				defer manager.ReleaseConcurrent(identifier)

				// Forward request
				next.ServeHTTP(w, r)

				// Record usage after request completes
				// (This would ideally happen in response middleware with actual usage data)
			} else {
				// Concurrent limit exceeded
				w.Header().Set("X-RateLimit-Limit", "concurrent")
				http.Error(w, "Too many concurrent requests", http.StatusTooManyRequests)
				return
			}
		})
	}
}

// extractIdentifier extracts the identifier from the request.
// Priority: API key > User ID > Team ID
func extractIdentifier(r *http.Request) string {
	// Try API key first
	apiKey := proxy.ExtractAPIKey(r)
	if apiKey != "" {
		return apiKey
	}

	// Try user ID
	userID := proxy.ExtractUserID(r)
	if userID != "" {
		return userID
	}

	// Try team ID (if implemented)
	// teamID := extractTeamID(r)
	// if teamID != "" {
	//     return teamID
	// }

	return ""
}

// setLimitHeaders sets rate limit and budget headers on the response.
func setLimitHeaders(w http.ResponseWriter, result *limits.LimitCheckResult) {
	// Set rate limit headers
	if result.RateLimit != nil {
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.RateLimit.Limit))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.RateLimit.Remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.RateLimit.Reset.Unix()))
	}

	// Set budget headers
	if result.Budget != nil {
		w.Header().Set("X-Budget-Limit", fmt.Sprintf("%.2f", result.Budget.Limit))
		w.Header().Set("X-Budget-Used", fmt.Sprintf("%.2f", result.Budget.Used))
		w.Header().Set("X-Budget-Remaining", fmt.Sprintf("%.2f", result.Budget.Remaining))
		w.Header().Set("X-Budget-Reset", fmt.Sprintf("%d", result.Budget.Reset.Unix()))
	}

	// Set retry-after header if applicable
	if result.RetryAfter > 0 {
		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(result.RetryAfter.Seconds())))
	}
}

// handleLimitViolation handles a limit violation by returning appropriate error.
func handleLimitViolation(w http.ResponseWriter, result *limits.LimitCheckResult) {
	// Set headers
	w.WriteHeader(http.StatusTooManyRequests)

	// Write error response
	fmt.Fprintf(w, `{"error": {"message": "%s", "type": "rate_limit_exceeded"}}`, result.Reason)
}

// enrichedRequestContext holds enriched request data for limit checking.
type enrichedRequestContext struct {
	estimatedTokens int
	estimatedCost   float64
	model           string
}

// NewLimitsManagerFromConfig creates a limits manager from configuration.
// This is a helper to initialize the manager with config-based limits.
func NewLimitsManagerFromConfig(cfg *limitsConfig) (*limits.Manager, error) {
	// Convert config format to manager format
	rateLimitsMap := make(map[string]ratelimit.Config)
	budgetsMap := make(map[string]budget.Config)

	// Convert rate limits by API key
	for identifier, limits := range cfg.RateLimits.ByAPIKey {
		rateLimitsMap[identifier] = ratelimit.Config{
			RequestsPerSecond: limits.RequestsPerSecond,
			RequestsPerMinute: limits.RequestsPerMinute,
			RequestsPerHour:   limits.RequestsPerHour,
			TokensPerMinute:   limits.TokensPerMinute,
			TokensPerHour:     limits.TokensPerHour,
			MaxConcurrent:     limits.MaxConcurrent,
		}
	}

	// Convert budgets by API key
	for identifier, budgetLimits := range cfg.Budgets.ByAPIKey {
		budgetsMap[identifier] = budget.Config{
			Hourly:         budgetLimits.Hourly,
			Daily:          budgetLimits.Daily,
			Monthly:        budgetLimits.Monthly,
			AlertThreshold: cfg.Budgets.AlertThreshold,
		}
	}

	// Create storage backend
	var storageBackend storage.Backend
	switch cfg.Storage.Backend {
	case "sqlite":
		backend, err := storage.NewSQLiteBackend(cfg.Storage.SQLite.Path)
		if err != nil {
			return nil, fmt.Errorf("failed to create SQLite backend: %w", err)
		}
		storageBackend = backend
	case "memory":
		storageBackend = storage.NewMemoryBackendWithConfig(storage.MemoryBackendConfig{
			MaxEntries:      cfg.Storage.Memory.MaxEntries,
			CleanupInterval: cfg.Storage.Memory.CleanupInterval,
		})
	default:
		storageBackend = storage.NewMemoryBackend()
	}

	// Create manager
	manager := limits.NewManager(limits.Config{
		RateLimits: rateLimitsMap,
		Budgets:    budgetsMap,
		Enforcement: enforcement.Config{
			DefaultAction:   enforcement.Action(cfg.Enforcement.Action),
			QueueDepth:      cfg.Enforcement.QueueDepth,
			QueueTimeout:    cfg.Enforcement.QueueTimeout,
			ModelDowngrades: cfg.Enforcement.ModelDowngrades,
		},
		Storage: storageBackend,
	})

	return manager, nil
}

// limitsConfig mirrors the config package structure for limits.
// This avoids circular dependencies.
type limitsConfig struct {
	Budgets struct {
		Enabled        bool
		AlertThreshold float64
		ByAPIKey       map[string]struct {
			Hourly  float64
			Daily   float64
			Monthly float64
		}
		ByUser map[string]struct {
			Hourly  float64
			Daily   float64
			Monthly float64
		}
		ByTeam map[string]struct {
			Hourly  float64
			Daily   float64
			Monthly float64
		}
	}
	RateLimits struct {
		Enabled  bool
		ByAPIKey map[string]struct {
			RequestsPerSecond int
			RequestsPerMinute int
			RequestsPerHour   int
			TokensPerMinute   int
			TokensPerHour     int
			MaxConcurrent     int
		}
		ByUser map[string]struct {
			RequestsPerSecond int
			RequestsPerMinute int
			RequestsPerHour   int
			TokensPerMinute   int
			TokensPerHour     int
			MaxConcurrent     int
		}
		ByTeam map[string]struct {
			RequestsPerSecond int
			RequestsPerMinute int
			RequestsPerHour   int
			TokensPerMinute   int
			TokensPerHour     int
			MaxConcurrent     int
		}
	}
	Enforcement struct {
		Action          string
		QueueDepth      int
		QueueTimeout    time.Duration
		ModelDowngrades map[string]string
	}
	Storage struct {
		Backend string
		SQLite  struct {
			Path             string
			SnapshotInterval time.Duration
		}
		Memory struct {
			MaxEntries      int
			CleanupInterval time.Duration
		}
	}
}
