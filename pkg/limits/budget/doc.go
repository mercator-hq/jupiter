// Package budget provides budget tracking for LLM request costs.
//
// # Overview
//
// The budget package implements rolling window budget tracking to prevent
// cost overruns. It supports multiple time windows (hourly, daily, monthly)
// and tracks spending across different dimensions (API key, user, team).
//
// # Rolling Windows
//
// Unlike fixed time windows, rolling windows provide smooth budget enforcement:
//
//   - Hourly: Last 60 minutes (not current hour)
//   - Daily: Last 24 hours (not current day)
//   - Monthly: Last 30 days (not current month)
//
// This prevents "reset spikes" where users can double-spend at window boundaries.
//
// # Usage
//
//	tracker := budget.NewTracker(budget.Config{
//	    Hourly:  10.00,  // $10/hour
//	    Daily:   200.00, // $200/day
//	    Monthly: 5000.00, // $5000/month
//	    AlertThreshold: 0.8, // Alert at 80%
//	})
//
//	// Add spending
//	tracker.Add(2.50) // $2.50 spent
//
//	// Check budget
//	status := tracker.Check()
//	if !status.Allowed {
//	    // Budget exceeded
//	}
//
// # Alert Thresholds
//
// Budget tracker can trigger alerts when spending reaches a percentage of
// the limit (e.g., 80%). This provides early warning before hitting limits.
//
// # Thread Safety
//
// All budget operations are thread-safe using sync.RWMutex for concurrent access.
package budget
