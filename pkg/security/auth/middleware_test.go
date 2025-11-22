package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewAPIKeyMiddleware(t *testing.T) {
	validator := NewAPIKeyValidator([]*APIKeyInfo{})
	sources := []APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
	}

	middleware := NewAPIKeyMiddleware(validator, sources)

	if middleware == nil {
		t.Fatal("NewAPIKeyMiddleware returned nil")
	}
	if middleware.validator != validator {
		t.Error("Validator not set correctly")
	}
	if len(middleware.sources) != 1 {
		t.Errorf("Expected 1 source, got %d", len(middleware.sources))
	}
}

func TestAPIKeyMiddleware_Handle(t *testing.T) {
	tests := []struct {
		name           string
		keys           []*APIKeyInfo
		sources        []APIKeySource
		setupRequest   func(*http.Request)
		expectedStatus int
		checkContext   bool
	}{
		{
			name: "valid bearer token",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-valid-key-123",
					UserID:    "user-123",
					TeamID:    "team-eng",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer sk-valid-key-123")
			},
			expectedStatus: http.StatusOK,
			checkContext:   true,
		},
		{
			name: "valid custom header",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-custom-key-456",
					UserID:    "user-456",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			sources: []APIKeySource{
				{Type: "header", Name: "X-API-Key", Scheme: ""},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "sk-custom-key-456")
			},
			expectedStatus: http.StatusOK,
			checkContext:   true,
		},
		{
			name: "valid query parameter",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-query-key-789",
					UserID:    "user-789",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			sources: []APIKeySource{
				{Type: "query", Name: "api_key"},
			},
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("api_key", "sk-query-key-789")
				r.URL.RawQuery = q.Encode()
			},
			expectedStatus: http.StatusOK,
			checkContext:   true,
		},
		{
			name: "missing API key",
			keys: []*APIKeyInfo{},
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				// Don't set any header
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name: "invalid API key",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-valid-key",
					UserID:    "user-123",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer sk-invalid-key")
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name: "disabled API key",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-disabled-key",
					UserID:    "user-disabled",
					Enabled:   false,
					CreatedAt: time.Now(),
				},
			},
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer sk-disabled-key")
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
		{
			name: "multiple sources - first fails, second succeeds",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-fallback-key",
					UserID:    "user-fallback",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
				{Type: "header", Name: "X-API-Key", Scheme: ""},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "sk-fallback-key")
			},
			expectedStatus: http.StatusOK,
			checkContext:   true,
		},
		{
			name: "wrong bearer scheme format",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-valid-key",
					UserID:    "user-123",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "sk-valid-key") // Missing "Bearer " prefix
			},
			expectedStatus: http.StatusUnauthorized,
			checkContext:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewAPIKeyValidator(tt.keys)
			middleware := NewAPIKeyMiddleware(validator, tt.sources)

			// Create test handler
			var contextChecked bool
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tt.checkContext {
					info, ok := GetAPIKeyInfo(r.Context())
					if !ok {
						t.Error("Expected API key info in context, got none")
					}
					if info == nil {
						t.Error("Expected non-nil API key info")
					}
					contextChecked = true
				}
				w.WriteHeader(http.StatusOK)
			})

			// Create request
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)

			// Create response recorder
			rr := httptest.NewRecorder()

			// Execute middleware
			middleware.Handle(handler).ServeHTTP(rr, req)

			// Check status code
			if rr.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			// Verify context was checked if expected
			if tt.checkContext && !contextChecked {
				t.Error("Context was not checked in handler")
			}
		})
	}
}

func TestAPIKeyMiddleware_extractAPIKey(t *testing.T) {
	tests := []struct {
		name          string
		sources       []APIKeySource
		setupRequest  func(*http.Request)
		expectedKey   string
		expectedError bool
	}{
		{
			name: "extract from bearer token",
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer sk-test-key")
			},
			expectedKey:   "sk-test-key",
			expectedError: false,
		},
		{
			name: "extract from custom header",
			sources: []APIKeySource{
				{Type: "header", Name: "X-API-Key", Scheme: ""},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("X-API-Key", "sk-custom-key")
			},
			expectedKey:   "sk-custom-key",
			expectedError: false,
		},
		{
			name: "extract from query parameter",
			sources: []APIKeySource{
				{Type: "query", Name: "api_key"},
			},
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("api_key", "sk-query-key")
				r.URL.RawQuery = q.Encode()
			},
			expectedKey:   "sk-query-key",
			expectedError: false,
		},
		{
			name: "no key found",
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				// Don't set any header
			},
			expectedKey:   "",
			expectedError: true,
		},
		{
			name: "bearer token without scheme",
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "sk-test-key") // Missing "Bearer " prefix
			},
			expectedKey:   "",
			expectedError: true,
		},
		{
			name: "try multiple sources - first succeeds",
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
				{Type: "query", Name: "api_key"},
			},
			setupRequest: func(r *http.Request) {
				r.Header.Set("Authorization", "Bearer sk-header-key")
				q := r.URL.Query()
				q.Add("api_key", "sk-query-key")
				r.URL.RawQuery = q.Encode()
			},
			expectedKey:   "sk-header-key", // First source wins
			expectedError: false,
		},
		{
			name: "try multiple sources - second succeeds",
			sources: []APIKeySource{
				{Type: "header", Name: "Authorization", Scheme: "Bearer"},
				{Type: "query", Name: "api_key"},
			},
			setupRequest: func(r *http.Request) {
				q := r.URL.Query()
				q.Add("api_key", "sk-query-key")
				r.URL.RawQuery = q.Encode()
			},
			expectedKey:   "sk-query-key",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := &APIKeyMiddleware{
				sources: tt.sources,
			}

			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupRequest(req)

			key, err := middleware.extractAPIKey(req)

			if tt.expectedError {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if key != tt.expectedKey {
					t.Errorf("Expected key %s, got %s", tt.expectedKey, key)
				}
			}
		})
	}
}

func TestGetAPIKeyInfo(t *testing.T) {
	tests := []struct {
		name          string
		setupContext  func(*http.Request)
		expectedFound bool
		expectedUser  string
	}{
		{
			name: "key info present in context",
			setupContext: func(r *http.Request) {
				info := &APIKeyInfo{
					Key:       "sk-test-key",
					UserID:    "user-123",
					TeamID:    "team-eng",
					Enabled:   true,
					CreatedAt: time.Now(),
				}
				*r = *r.WithContext(r.Context())
				// We need to simulate what the middleware does
				validator := NewAPIKeyValidator([]*APIKeyInfo{info})
				middleware := NewAPIKeyMiddleware(validator, []APIKeySource{
					{Type: "header", Name: "Authorization", Scheme: "Bearer"},
				})
				r.Header.Set("Authorization", "Bearer sk-test-key")

				// Create a test handler to capture the context
				handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
					*r = *req // Copy the request with the updated context
				})

				rr := httptest.NewRecorder()
				middleware.Handle(handler).ServeHTTP(rr, r)
			},
			expectedFound: true,
			expectedUser:  "user-123",
		},
		{
			name: "no key info in context",
			setupContext: func(r *http.Request) {
				// Don't add anything to context
			},
			expectedFound: false,
			expectedUser:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			tt.setupContext(req)

			info, ok := GetAPIKeyInfo(req.Context())

			if ok != tt.expectedFound {
				t.Errorf("Expected found=%v, got %v", tt.expectedFound, ok)
			}

			if tt.expectedFound {
				if info == nil {
					t.Error("Expected non-nil info when found=true")
				} else if info.UserID != tt.expectedUser {
					t.Errorf("Expected user %s, got %s", tt.expectedUser, info.UserID)
				}
			} else {
				if info != nil {
					t.Error("Expected nil info when found=false")
				}
			}
		})
	}
}
