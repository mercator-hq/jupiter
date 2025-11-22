package query

import (
	"fmt"

	"mercator-hq/jupiter/pkg/evidence"
)

const (
	// DefaultLimit is the default number of records to return if not specified.
	DefaultLimit = 100

	// MaxLimit is the maximum number of records that can be returned in a single query.
	MaxLimit = 10000
)

// ValidSortFields contains the fields that can be used for sorting.
var ValidSortFields = map[string]bool{
	"request_time":      true,
	"recorded_time":     true,
	"response_time":     true,
	"actual_cost":       true,
	"total_tokens":      true,
	"provider_latency":  true,
}

// ValidSortOrders contains the valid sort orders.
var ValidSortOrders = map[string]bool{
	"asc":  true,
	"desc": true,
}

// Validate validates a query and returns an error if any parameters are invalid.
func Validate(q *evidence.Query) error {
	// Validate limit
	if q.Limit < 0 {
		return evidence.NewQueryError(q, fmt.Errorf("limit must be >= 0, got %d", q.Limit))
	}
	if q.Limit > MaxLimit {
		return evidence.NewQueryError(q, fmt.Errorf("limit must be <= %d, got %d", MaxLimit, q.Limit))
	}

	// Validate offset
	if q.Offset < 0 {
		return evidence.NewQueryError(q, fmt.Errorf("offset must be >= 0, got %d", q.Offset))
	}

	// Validate sort field
	if q.SortBy != "" && !ValidSortFields[q.SortBy] {
		return evidence.NewQueryError(q, fmt.Errorf("invalid sort field: %s", q.SortBy))
	}

	// Validate sort order
	if q.SortOrder != "" && !ValidSortOrders[q.SortOrder] {
		return evidence.NewQueryError(q, fmt.Errorf("invalid sort order: %s (must be 'asc' or 'desc')", q.SortOrder))
	}

	// Validate time range
	if q.StartTime != nil && q.EndTime != nil {
		if q.StartTime.After(*q.EndTime) {
			return evidence.NewQueryError(q, fmt.Errorf("start_time must be before end_time"))
		}
	}

	// Validate cost thresholds
	if q.MinCost != nil && q.MaxCost != nil {
		if *q.MinCost > *q.MaxCost {
			return evidence.NewQueryError(q, fmt.Errorf("min_cost must be <= max_cost"))
		}
	}

	// Validate token thresholds
	if q.MinTokens != nil && q.MaxTokens != nil {
		if *q.MinTokens > *q.MaxTokens {
			return evidence.NewQueryError(q, fmt.Errorf("min_tokens must be <= max_tokens"))
		}
	}

	// Validate status
	if q.Status != "" {
		validStatuses := map[string]bool{
			"success": true,
			"error":   true,
			"blocked": true,
		}
		if !validStatuses[q.Status] {
			return evidence.NewQueryError(q, fmt.Errorf("invalid status: %s (must be 'success', 'error', or 'blocked')", q.Status))
		}
	}

	return nil
}

// ApplyDefaults applies default values to a query.
func ApplyDefaults(q *evidence.Query) {
	// Apply default limit
	if q.Limit == 0 {
		q.Limit = DefaultLimit
	}

	// Apply default sort field
	if q.SortBy == "" {
		q.SortBy = "request_time"
	}

	// Apply default sort order
	if q.SortOrder == "" {
		q.SortOrder = "desc"
	}
}
