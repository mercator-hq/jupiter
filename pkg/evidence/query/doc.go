// Package query provides query building and validation for evidence records.
//
// # Query Builder
//
// The query builder constructs SQL WHERE clauses from evidence query filters:
//
//   - Time range filtering (start_time, end_time)
//   - User/API key filtering
//   - Provider/model filtering
//   - Policy decision filtering
//   - Cost/token threshold filtering
//   - Status filtering (success, error, blocked)
//   - Pagination (limit, offset)
//   - Sorting (by timestamp, cost, tokens)
//
// # Query Validation
//
// The validator ensures query parameters are valid before execution:
//
//   - Limit > 0 and <= MaxLimit
//   - Offset >= 0
//   - Sort field is valid (timestamp, cost, tokens)
//   - Sort order is valid (asc, desc)
//   - Time range is valid (start <= end)
//   - Cost/token thresholds are valid (min <= max)
//
// # Basic Usage
//
//	// Create query
//	query := &evidence.Query{
//	    StartTime: &startTime,
//	    EndTime: &endTime,
//	    UserID: "user-123",
//	    PolicyDecision: "block",
//	    Limit: 100,
//	    SortBy: "request_time",
//	    SortOrder: "desc",
//	}
//
//	// Validate query
//	if err := query.Validate(query); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Execute query
//	records, err := storage.Query(ctx, query)
//	if err != nil {
//	    log.Fatal(err)
//	}
package query
