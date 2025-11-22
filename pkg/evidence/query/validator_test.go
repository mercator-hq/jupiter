package query

import (
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

func TestValidate(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	minCost := 0.01
	maxCost := 10.0
	minTokens := 100
	maxTokens := 10000

	tests := []struct {
		name    string
		query   *evidence.Query
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid query with all filters",
			query: &evidence.Query{
				StartTime:      &past,
				EndTime:        &now,
				UserID:         "user-123",
				Provider:       "openai",
				Model:          "gpt-4",
				PolicyDecision: "allow",
				MinCost:        &minCost,
				MaxCost:        &maxCost,
				MinTokens:      &minTokens,
				MaxTokens:      &maxTokens,
				Status:         "success",
				Limit:          100,
				Offset:         0,
				SortBy:         "request_time",
				SortOrder:      "desc",
			},
			wantErr: false,
		},
		{
			name: "valid query with minimal filters",
			query: &evidence.Query{
				Limit: 50,
			},
			wantErr: false,
		},
		{
			name: "negative limit",
			query: &evidence.Query{
				Limit: -1,
			},
			wantErr: true,
			errMsg:  "limit must be >= 0",
		},
		{
			name: "limit exceeds max",
			query: &evidence.Query{
				Limit: MaxLimit + 1,
			},
			wantErr: true,
			errMsg:  "limit must be <=",
		},
		{
			name: "negative offset",
			query: &evidence.Query{
				Offset: -1,
			},
			wantErr: true,
			errMsg:  "offset must be >= 0",
		},
		{
			name: "invalid sort field",
			query: &evidence.Query{
				SortBy: "invalid_field",
			},
			wantErr: true,
			errMsg:  "invalid sort field",
		},
		{
			name: "invalid sort order",
			query: &evidence.Query{
				SortBy:    "request_time",
				SortOrder: "invalid",
			},
			wantErr: true,
			errMsg:  "invalid sort order",
		},
		{
			name: "start time after end time",
			query: &evidence.Query{
				StartTime: &future,
				EndTime:   &past,
			},
			wantErr: true,
			errMsg:  "start_time must be before end_time",
		},
		{
			name: "min cost greater than max cost",
			query: &evidence.Query{
				MinCost: &maxCost,
				MaxCost: &minCost,
			},
			wantErr: true,
			errMsg:  "min_cost must be <= max_cost",
		},
		{
			name: "min tokens greater than max tokens",
			query: &evidence.Query{
				MinTokens: &maxTokens,
				MaxTokens: &minTokens,
			},
			wantErr: true,
			errMsg:  "min_tokens must be <= max_tokens",
		},
		{
			name: "invalid status",
			query: &evidence.Query{
				Status: "invalid_status",
			},
			wantErr: true,
			errMsg:  "invalid status",
		},
		{
			name: "valid status - success",
			query: &evidence.Query{
				Status: "success",
			},
			wantErr: false,
		},
		{
			name: "valid status - error",
			query: &evidence.Query{
				Status: "error",
			},
			wantErr: false,
		},
		{
			name: "valid status - blocked",
			query: &evidence.Query{
				Status: "blocked",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.query)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestValidate_ValidSortFields(t *testing.T) {
	// Test all valid sort fields
	validFields := []string{
		"request_time",
		"recorded_time",
		"response_time",
		"actual_cost",
		"total_tokens",
		"provider_latency",
	}

	for _, field := range validFields {
		t.Run("sort_by_"+field, func(t *testing.T) {
			query := &evidence.Query{
				SortBy: field,
			}
			err := Validate(query)
			if err != nil {
				t.Errorf("Validate() with sort field %q failed: %v", field, err)
			}
		})
	}
}

func TestValidate_ValidSortOrders(t *testing.T) {
	// Test all valid sort orders
	validOrders := []string{"asc", "desc"}

	for _, order := range validOrders {
		t.Run("sort_order_"+order, func(t *testing.T) {
			query := &evidence.Query{
				SortBy:    "request_time",
				SortOrder: order,
			}
			err := Validate(query)
			if err != nil {
				t.Errorf("Validate() with sort order %q failed: %v", order, err)
			}
		})
	}
}

func TestApplyDefaults(t *testing.T) {
	tests := []struct {
		name          string
		query         *evidence.Query
		expectedLimit int
		expectedSort  string
		expectedOrder string
	}{
		{
			name:          "empty query gets all defaults",
			query:         &evidence.Query{},
			expectedLimit: DefaultLimit,
			expectedSort:  "request_time",
			expectedOrder: "desc",
		},
		{
			name: "query with limit keeps it",
			query: &evidence.Query{
				Limit: 50,
			},
			expectedLimit: 50,
			expectedSort:  "request_time",
			expectedOrder: "desc",
		},
		{
			name: "query with sort keeps it",
			query: &evidence.Query{
				SortBy: "actual_cost",
			},
			expectedLimit: DefaultLimit,
			expectedSort:  "actual_cost",
			expectedOrder: "desc",
		},
		{
			name: "query with sort order keeps it",
			query: &evidence.Query{
				SortOrder: "asc",
			},
			expectedLimit: DefaultLimit,
			expectedSort:  "request_time",
			expectedOrder: "asc",
		},
		{
			name: "query with all set keeps all",
			query: &evidence.Query{
				Limit:     25,
				SortBy:    "total_tokens",
				SortOrder: "asc",
			},
			expectedLimit: 25,
			expectedSort:  "total_tokens",
			expectedOrder: "asc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ApplyDefaults(tt.query)

			if tt.query.Limit != tt.expectedLimit {
				t.Errorf("Limit = %d, want %d", tt.query.Limit, tt.expectedLimit)
			}
			if tt.query.SortBy != tt.expectedSort {
				t.Errorf("SortBy = %s, want %s", tt.query.SortBy, tt.expectedSort)
			}
			if tt.query.SortOrder != tt.expectedOrder {
				t.Errorf("SortOrder = %s, want %s", tt.query.SortOrder, tt.expectedOrder)
			}
		})
	}
}

func TestApplyDefaults_Idempotent(t *testing.T) {
	// Applying defaults multiple times should have same effect
	query := &evidence.Query{}

	ApplyDefaults(query)
	firstLimit := query.Limit
	firstSort := query.SortBy
	firstOrder := query.SortOrder

	ApplyDefaults(query)
	ApplyDefaults(query)

	if query.Limit != firstLimit {
		t.Errorf("Limit changed after multiple ApplyDefaults: %d -> %d", firstLimit, query.Limit)
	}
	if query.SortBy != firstSort {
		t.Errorf("SortBy changed after multiple ApplyDefaults: %s -> %s", firstSort, query.SortBy)
	}
	if query.SortOrder != firstOrder {
		t.Errorf("SortOrder changed after multiple ApplyDefaults: %s -> %s", firstOrder, query.SortOrder)
	}
}

func TestConstants(t *testing.T) {
	// Verify constants have expected values
	if DefaultLimit != 100 {
		t.Errorf("DefaultLimit = %d, want 100", DefaultLimit)
	}
	if MaxLimit != 10000 {
		t.Errorf("MaxLimit = %d, want 10000", MaxLimit)
	}
}

func TestValidSortFields(t *testing.T) {
	// Verify all expected sort fields are present
	expectedFields := []string{
		"request_time",
		"recorded_time",
		"response_time",
		"actual_cost",
		"total_tokens",
		"provider_latency",
	}

	for _, field := range expectedFields {
		if !ValidSortFields[field] {
			t.Errorf("ValidSortFields missing expected field: %s", field)
		}
	}

	// Verify count matches (no extra fields)
	if len(ValidSortFields) != len(expectedFields) {
		t.Errorf("ValidSortFields has %d fields, expected %d", len(ValidSortFields), len(expectedFields))
	}
}

func TestValidSortOrders(t *testing.T) {
	// Verify sort orders
	if !ValidSortOrders["asc"] {
		t.Error("ValidSortOrders missing 'asc'")
	}
	if !ValidSortOrders["desc"] {
		t.Error("ValidSortOrders missing 'desc'")
	}
	if len(ValidSortOrders) != 2 {
		t.Errorf("ValidSortOrders has %d orders, expected 2", len(ValidSortOrders))
	}
}

// BenchmarkValidate benchmarks query validation
func BenchmarkValidate(b *testing.B) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	minCost := 0.01
	maxCost := 10.0
	minTokens := 100
	maxTokens := 10000

	query := &evidence.Query{
		StartTime:      &past,
		EndTime:        &now,
		UserID:         "user-123",
		Provider:       "openai",
		Model:          "gpt-4",
		PolicyDecision: "allow",
		MinCost:        &minCost,
		MaxCost:        &maxCost,
		MinTokens:      &minTokens,
		MaxTokens:      &maxTokens,
		Status:         "success",
		Limit:          100,
		Offset:         0,
		SortBy:         "request_time",
		SortOrder:      "desc",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Validate(query)
	}
}

// BenchmarkApplyDefaults benchmarks applying defaults
func BenchmarkApplyDefaults(b *testing.B) {
	for i := 0; i < b.N; i++ {
		query := &evidence.Query{}
		ApplyDefaults(query)
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
