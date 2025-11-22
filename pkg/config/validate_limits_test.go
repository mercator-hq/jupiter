package config

import (
	"strings"
	"testing"
	"time"
)

// TestValidateLimits_ValidConfig tests validation of valid limits configuration.
func TestValidateLimits_ValidConfig(t *testing.T) {
	cfg := &LimitsConfig{
		Budgets: BudgetsConfig{
			Enabled:        true,
			AlertThreshold: 0.8,
			ByAPIKey: map[string]BudgetLimits{
				"test-key": {
					Hourly:  10.00,
					Daily:   200.00,
					Monthly: 5000.00,
				},
			},
		},
		RateLimits: RateLimitsConfig{
			Enabled: true,
			ByAPIKey: map[string]RateLimits{
				"test-key": {
					RequestsPerSecond: 10,
					RequestsPerMinute: 500,
					TokensPerMinute:   100000,
					MaxConcurrent:     20,
				},
			},
		},
		Enforcement: EnforcementConfig{
			Action:       "block",
			QueueDepth:   100,
			QueueTimeout: 30 * time.Second,
		},
		Storage: LimitsStorageConfig{
			Backend: "memory",
			Memory: LimitsMemoryConfig{
				MaxEntries:      10000,
				CleanupInterval: 1 * time.Minute,
			},
		},
	}

	errs := validateLimits(cfg)
	if len(errs) > 0 {
		t.Errorf("Expected no errors for valid config, got: %v", errs)
	}
}

// TestValidateLimits_AlertThreshold tests alert threshold validation.
func TestValidateLimits_AlertThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		wantErr   bool
	}{
		{"valid 0.0", 0.0, false},
		{"valid 0.5", 0.5, false},
		{"valid 1.0", 1.0, false},
		{"invalid negative", -0.1, true},
		{"invalid > 1.0", 1.5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &LimitsConfig{
				Budgets: BudgetsConfig{
					Enabled:        true,
					AlertThreshold: tt.threshold,
				},
				Storage: LimitsStorageConfig{Backend: "memory"},
			}

			errs := validateLimits(cfg)
			hasErr := false
			for _, err := range errs {
				if strings.Contains(err.Field, "alert_threshold") {
					hasErr = true
					break
				}
			}

			if hasErr != tt.wantErr {
				t.Errorf("Expected error: %v, got errors: %v", tt.wantErr, errs)
			}
		})
	}
}

// TestValidateLimits_BudgetValues tests budget value validation.
func TestValidateLimits_BudgetValues(t *testing.T) {
	tests := []struct {
		name    string
		limits  BudgetLimits
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid budgets",
			limits:  BudgetLimits{Hourly: 10, Daily: 100, Monthly: 1000},
			wantErr: false,
		},
		{
			name:    "negative hourly",
			limits:  BudgetLimits{Hourly: -1, Daily: 100, Monthly: 1000},
			wantErr: true,
			errMsg:  "hourly budget must be non-negative",
		},
		{
			name:    "negative daily",
			limits:  BudgetLimits{Hourly: 10, Daily: -1, Monthly: 1000},
			wantErr: true,
			errMsg:  "daily budget must be non-negative",
		},
		{
			name:    "negative monthly",
			limits:  BudgetLimits{Hourly: 10, Daily: 100, Monthly: -1},
			wantErr: true,
			errMsg:  "monthly budget must be non-negative",
		},
		{
			name:    "hourly exceeds daily",
			limits:  BudgetLimits{Hourly: 200, Daily: 100, Monthly: 1000},
			wantErr: true,
			errMsg:  "hourly budget cannot exceed daily budget",
		},
		{
			name:    "daily exceeds monthly",
			limits:  BudgetLimits{Hourly: 10, Daily: 2000, Monthly: 1000},
			wantErr: true,
			errMsg:  "daily budget cannot exceed monthly budget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &LimitsConfig{
				Budgets: BudgetsConfig{
					Enabled:        true,
					AlertThreshold: 0.8,
					ByAPIKey: map[string]BudgetLimits{
						"test-key": tt.limits,
					},
				},
				Storage: LimitsStorageConfig{Backend: "memory"},
			}

			errs := validateLimits(cfg)
			hasErr := len(errs) > 0

			if hasErr != tt.wantErr {
				t.Errorf("Expected error: %v, got errors: %v", tt.wantErr, errs)
			}

			if tt.wantErr && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Message, tt.errMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing %q, got: %v", tt.errMsg, errs)
				}
			}
		})
	}
}

// TestValidateLimits_RateLimitValues tests rate limit value validation.
func TestValidateLimits_RateLimitValues(t *testing.T) {
	tests := []struct {
		name    string
		limits  RateLimits
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid rate limits",
			limits: RateLimits{
				RequestsPerSecond: 10,
				RequestsPerMinute: 500,
				TokensPerMinute:   100000,
				MaxConcurrent:     20,
			},
			wantErr: false,
		},
		{
			name:    "negative requests per second",
			limits:  RateLimits{RequestsPerSecond: -1},
			wantErr: true,
			errMsg:  "requests per second must be non-negative",
		},
		{
			name:    "negative max concurrent",
			limits:  RateLimits{MaxConcurrent: -1},
			wantErr: true,
			errMsg:  "max concurrent must be non-negative",
		},
		{
			name:    "excessive requests per second",
			limits:  RateLimits{RequestsPerSecond: 200000},
			wantErr: true,
			errMsg:  "requests per second exceeds reasonable limit",
		},
		{
			name:    "excessive tokens per minute",
			limits:  RateLimits{TokensPerMinute: 20000000},
			wantErr: true,
			errMsg:  "tokens per minute exceeds reasonable limit",
		},
		{
			name:    "excessive max concurrent",
			limits:  RateLimits{MaxConcurrent: 20000},
			wantErr: true,
			errMsg:  "max concurrent exceeds reasonable limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &LimitsConfig{
				RateLimits: RateLimitsConfig{
					Enabled: true,
					ByAPIKey: map[string]RateLimits{
						"test-key": tt.limits,
					},
				},
				Storage: LimitsStorageConfig{Backend: "memory"},
			}

			errs := validateLimits(cfg)
			hasErr := len(errs) > 0

			if hasErr != tt.wantErr {
				t.Errorf("Expected error: %v, got errors: %v", tt.wantErr, errs)
			}

			if tt.wantErr && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Message, tt.errMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing %q, got: %v", tt.errMsg, errs)
				}
			}
		})
	}
}

// TestValidateLimits_EnforcementAction tests enforcement action validation.
func TestValidateLimits_EnforcementAction(t *testing.T) {
	tests := []struct {
		name    string
		action  string
		wantErr bool
	}{
		{"valid block", "block", false},
		{"valid queue", "queue", false},
		{"valid downgrade", "downgrade", false},
		{"valid alert", "alert", false},
		{"empty (uses default)", "", false},
		{"invalid action", "invalid", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &LimitsConfig{
				Enforcement: EnforcementConfig{
					Action: tt.action,
				},
				Storage: LimitsStorageConfig{Backend: "memory"},
			}

			errs := validateLimits(cfg)
			hasErr := false
			for _, err := range errs {
				if strings.Contains(err.Field, "enforcement.action") {
					hasErr = true
					break
				}
			}

			if hasErr != tt.wantErr {
				t.Errorf("Expected error: %v, got errors: %v", tt.wantErr, errs)
			}
		})
	}
}

// TestValidateLimits_CircularDowngrade tests circular downgrade detection.
func TestValidateLimits_CircularDowngrade(t *testing.T) {
	tests := []struct {
		name       string
		downgrades map[string]string
		wantErr    bool
	}{
		{
			name: "valid chain",
			downgrades: map[string]string{
				"gpt-4":         "gpt-4-turbo",
				"gpt-4-turbo":   "gpt-3.5-turbo",
				"claude-3-opus": "claude-3-sonnet",
			},
			wantErr: false,
		},
		{
			name: "circular reference",
			downgrades: map[string]string{
				"gpt-4":       "gpt-4-turbo",
				"gpt-4-turbo": "gpt-4",
			},
			wantErr: true,
		},
		{
			name: "self reference",
			downgrades: map[string]string{
				"gpt-4": "gpt-4",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &LimitsConfig{
				Enforcement: EnforcementConfig{
					Action:          "downgrade",
					ModelDowngrades: tt.downgrades,
				},
				Storage: LimitsStorageConfig{Backend: "memory"},
			}

			errs := validateLimits(cfg)
			hasErr := false
			for _, err := range errs {
				if strings.Contains(err.Message, "circular") {
					hasErr = true
					break
				}
			}

			if hasErr != tt.wantErr {
				t.Errorf("Expected error: %v, got errors: %v", tt.wantErr, errs)
			}
		})
	}
}

// TestValidateLimits_StorageBackend tests storage backend validation.
func TestValidateLimits_StorageBackend(t *testing.T) {
	tests := []struct {
		name    string
		storage LimitsStorageConfig
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid memory backend",
			storage: LimitsStorageConfig{Backend: "memory"},
			wantErr: false,
		},
		{
			name: "valid sqlite backend",
			storage: LimitsStorageConfig{
				Backend: "sqlite",
				SQLite:  LimitsSQLiteConfig{Path: "/tmp/limits.db"},
			},
			wantErr: false,
		},
		{
			name:    "empty backend",
			storage: LimitsStorageConfig{},
			wantErr: true,
			errMsg:  "backend is required",
		},
		{
			name:    "invalid backend",
			storage: LimitsStorageConfig{Backend: "redis"},
			wantErr: true,
			errMsg:  "invalid backend",
		},
		{
			name: "sqlite without path",
			storage: LimitsStorageConfig{
				Backend: "sqlite",
				SQLite:  LimitsSQLiteConfig{},
			},
			wantErr: true,
			errMsg:  "SQLite path is required",
		},
		{
			name: "memory with negative max entries",
			storage: LimitsStorageConfig{
				Backend: "memory",
				Memory:  LimitsMemoryConfig{MaxEntries: -1},
			},
			wantErr: true,
			errMsg:  "max entries must be non-negative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &LimitsConfig{
				Storage: tt.storage,
			}

			errs := validateLimits(cfg)
			hasErr := len(errs) > 0

			if hasErr != tt.wantErr {
				t.Errorf("Expected error: %v, got errors: %v", tt.wantErr, errs)
			}

			if tt.wantErr && len(errs) > 0 {
				found := false
				for _, err := range errs {
					if strings.Contains(err.Message, tt.errMsg) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected error message containing %q, got: %v", tt.errMsg, errs)
				}
			}
		})
	}
}

// TestValidateLimits_MultiDimensional tests validation across all dimensions.
func TestValidateLimits_MultiDimensional(t *testing.T) {
	cfg := &LimitsConfig{
		Budgets: BudgetsConfig{
			Enabled:        true,
			AlertThreshold: 0.8,
			ByAPIKey: map[string]BudgetLimits{
				"api-key-1": {Hourly: 10, Daily: 200, Monthly: 5000},
			},
			ByUser: map[string]BudgetLimits{
				"user-1": {Hourly: 5, Daily: 100, Monthly: 2500},
			},
			ByTeam: map[string]BudgetLimits{
				"team-1": {Hourly: 50, Daily: 1000, Monthly: 25000},
			},
		},
		RateLimits: RateLimitsConfig{
			Enabled: true,
			ByAPIKey: map[string]RateLimits{
				"api-key-1": {RequestsPerSecond: 10},
			},
			ByUser: map[string]RateLimits{
				"user-1": {RequestsPerMinute: 500},
			},
			ByTeam: map[string]RateLimits{
				"team-1": {TokensPerMinute: 100000},
			},
		},
		Storage: LimitsStorageConfig{Backend: "memory"},
	}

	errs := validateLimits(cfg)
	if len(errs) > 0 {
		t.Errorf("Expected no errors for multi-dimensional config, got: %v", errs)
	}
}
