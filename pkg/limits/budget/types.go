package budget

import "time"

// Config contains budget limits for different time windows.
type Config struct {
	// Hourly is the budget limit for a rolling 60-minute window (USD).
	Hourly float64

	// Daily is the budget limit for a rolling 24-hour window (USD).
	Daily float64

	// Monthly is the budget limit for a rolling 30-day window (USD).
	Monthly float64

	// AlertThreshold is the percentage (0.0-1.0) at which to trigger alerts.
	// For example, 0.8 means alert when 80% of budget is used.
	AlertThreshold float64
}

// Status contains the current budget status for a time window.
type Status struct {
	// Allowed indicates if spending is within the budget.
	Allowed bool

	// Reason explains why spending was rejected (if Allowed=false).
	Reason string

	// Limit is the configured budget limit in USD.
	Limit float64

	// Used is the amount spent in USD within the window.
	Used float64

	// Remaining is the budget remaining in USD.
	Remaining float64

	// Percentage is the percentage of budget used (0.0-1.0).
	Percentage float64

	// Reset is when the window resets (rolling window, so this is approximate).
	Reset time.Time

	// Window is the time window duration.
	Window time.Duration

	// AlertTriggered indicates if the alert threshold was reached.
	AlertTriggered bool
}
