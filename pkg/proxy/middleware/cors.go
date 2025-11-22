package middleware

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig contains configuration for CORS middleware.
type CORSConfig struct {
	// Enabled controls whether CORS is enabled.
	Enabled bool

	// AllowedOrigins is a list of allowed origins for CORS.
	// Use ["*"] to allow all origins.
	AllowedOrigins []string

	// AllowedMethods is a list of allowed HTTP methods.
	AllowedMethods []string

	// AllowedHeaders is a list of allowed HTTP headers.
	AllowedHeaders []string

	// ExposedHeaders is a list of headers exposed to clients.
	ExposedHeaders []string

	// MaxAge is the maximum age (in seconds) for preflight cache.
	MaxAge int

	// AllowCredentials controls whether credentials are allowed.
	AllowCredentials bool
}

// DefaultCORSConfig returns a default CORS configuration.
func DefaultCORSConfig() *CORSConfig {
	return &CORSConfig{
		Enabled:        true,
		AllowedOrigins: []string{"*"},
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"Authorization", "Content-Type", "X-Request-ID", "X-User-ID"},
		ExposedHeaders: []string{"X-Request-ID"},
		MaxAge:         3600, // 1 hour
	}
}

// CORSMiddleware adds Cross-Origin Resource Sharing (CORS) headers to responses.
// It handles preflight OPTIONS requests and adds appropriate CORS headers for
// all requests.
//
// Configuration:
//
//	config := &CORSConfig{
//	    Enabled: true,
//	    AllowedOrigins: []string{"https://example.com"},
//	    AllowedMethods: []string{"GET", "POST", "OPTIONS"},
//	    AllowedHeaders: []string{"Authorization", "Content-Type"},
//	    MaxAge: 3600,
//	}
//	handler = CORSMiddleware(config)(handler)
//
// Example usage:
//
//	handler = CORSMiddleware(DefaultCORSConfig())(handler)
func CORSMiddleware(config *CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip CORS if disabled
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Get origin from request
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if origin != "" && isOriginAllowed(origin, config.AllowedOrigins) {
				// Set Access-Control-Allow-Origin
				w.Header().Set("Access-Control-Allow-Origin", origin)

				// Set Access-Control-Allow-Credentials if enabled
				if config.AllowCredentials {
					w.Header().Set("Access-Control-Allow-Credentials", "true")
				}

				// Set Access-Control-Expose-Headers
				if len(config.ExposedHeaders) > 0 {
					w.Header().Set("Access-Control-Expose-Headers", strings.Join(config.ExposedHeaders, ", "))
				}
			} else if contains(config.AllowedOrigins, "*") {
				// Allow all origins
				w.Header().Set("Access-Control-Allow-Origin", "*")
			}

			// Handle preflight OPTIONS request
			if r.Method == http.MethodOptions {
				// Set Access-Control-Allow-Methods
				if len(config.AllowedMethods) > 0 {
					w.Header().Set("Access-Control-Allow-Methods", strings.Join(config.AllowedMethods, ", "))
				}

				// Set Access-Control-Allow-Headers
				if len(config.AllowedHeaders) > 0 {
					w.Header().Set("Access-Control-Allow-Headers", strings.Join(config.AllowedHeaders, ", "))
				}

				// Set Access-Control-Max-Age
				if config.MaxAge > 0 {
					w.Header().Set("Access-Control-Max-Age", strconv.Itoa(config.MaxAge))
				}

				// Respond with 204 No Content for preflight
				w.WriteHeader(http.StatusNoContent)
				return
			}

			// Call next handler
			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if an origin is in the allowed list.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	for _, allowed := range allowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
