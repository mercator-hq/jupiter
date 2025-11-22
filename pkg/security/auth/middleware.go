package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
)

// APIKeySource defines where to extract API keys from
type APIKeySource struct {
	Type   string // header, query
	Name   string // Header name or query param
	Scheme string // "Bearer", etc. (optional)
}

// APIKeyMiddleware is HTTP middleware for API key authentication
type APIKeyMiddleware struct {
	validator *APIKeyValidator
	sources   []APIKeySource
}

// NewAPIKeyMiddleware creates a new API key authentication middleware
func NewAPIKeyMiddleware(validator *APIKeyValidator, sources []APIKeySource) *APIKeyMiddleware {
	return &APIKeyMiddleware{
		validator: validator,
		sources:   sources,
	}
}

// Handle wraps an HTTP handler with API key authentication
func (m *APIKeyMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract API key
		apiKey, err := m.extractAPIKey(r)
		if err != nil {
			slog.Warn("missing API key",
				"error", err,
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
			)
			http.Error(w, "Missing or invalid API key", http.StatusUnauthorized)
			return
		}

		// Validate API key
		keyInfo, err := m.validator.Validate(apiKey)
		if err != nil {
			slog.Warn("invalid API key",
				"error", err,
				"remote_addr", r.RemoteAddr,
				"path", r.URL.Path,
			)
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Log successful authentication
		slog.Debug("API key authenticated",
			"user_id", keyInfo.UserID,
			"team_id", keyInfo.TeamID,
			"path", r.URL.Path,
		)

		// Add key info to request context
		ctx := context.WithValue(r.Context(), apiKeyInfoKey, keyInfo)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// extractAPIKey extracts the API key from the request using configured sources
func (m *APIKeyMiddleware) extractAPIKey(r *http.Request) (string, error) {
	for _, source := range m.sources {
		switch source.Type {
		case "header":
			value := r.Header.Get(source.Name)
			if value != "" {
				// Remove scheme prefix if present
				if source.Scheme != "" {
					prefix := source.Scheme + " "
					if strings.HasPrefix(value, prefix) {
						return strings.TrimPrefix(value, prefix), nil
					}
				} else {
					return value, nil
				}
			}

		case "query":
			value := r.URL.Query().Get(source.Name)
			if value != "" {
				return value, nil
			}
		}
	}

	return "", fmt.Errorf("no API key found")
}

// Context key for API key info
type contextKey string

// #nosec G101 - This is a context key constant, not a credential
const apiKeyInfoKey contextKey = "api_key_info"

// GetAPIKeyInfo retrieves API key info from request context
func GetAPIKeyInfo(ctx context.Context) (*APIKeyInfo, bool) {
	info, ok := ctx.Value(apiKeyInfoKey).(*APIKeyInfo)
	return info, ok
}
