package budget

import (
	"sync"
	"time"
)

// Tracker tracks budget spending across multiple time windows.
//
// The Tracker maintains separate rolling windows for hourly, daily, and
// monthly budgets. All windows are checked together - if any limit is
// exceeded, the request is rejected.
//
// # Alert Thresholds
//
// The tracker can trigger alerts when spending reaches a percentage of
// the configured limit. Alerts are detected during Check() and indicated
// in the returned Status.
type Tracker struct {
	config Config

	// Rolling windows for different time periods
	hourly  *RollingWindow
	daily   *RollingWindow
	monthly *RollingWindow

	// Total spending (all-time, not windowed)
	totalSpent float64

	mu sync.RWMutex
}

// NewTracker creates a new budget tracker with the given configuration.
//
// Only non-zero limits in the config are enforced. Zero values mean no limit.
//
// Example:
//
//	tracker := NewTracker(Config{
//	    Hourly:         10.00,   // $10/hour
//	    Daily:          200.00,  // $200/day
//	    Monthly:        5000.00, // $5000/month
//	    AlertThreshold: 0.8,     // Alert at 80%
//	})
func NewTracker(config Config) *Tracker {
	tracker := &Tracker{
		config:     config,
		totalSpent: 0,
	}

	// Initialize rolling windows only for configured limits
	if config.Hourly > 0 {
		// 1-minute buckets for hourly window
		tracker.hourly = NewRollingWindow(time.Hour, time.Minute)
	}

	if config.Daily > 0 {
		// 1-hour buckets for daily window
		tracker.daily = NewRollingWindow(24*time.Hour, time.Hour)
	}

	if config.Monthly > 0 {
		// 1-day buckets for monthly window (30 days)
		tracker.monthly = NewRollingWindow(30*24*time.Hour, 24*time.Hour)
	}

	return tracker
}

// Add records spending in all configured windows.
//
// The amount is added to hourly, daily, and monthly windows (whichever
// are configured) and to the all-time total.
func (t *Tracker) Add(amount float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Add to rolling windows
	if t.hourly != nil {
		t.hourly.Add(amount)
	}
	if t.daily != nil {
		t.daily.Add(amount)
	}
	if t.monthly != nil {
		t.monthly.Add(amount)
	}

	// Add to total
	t.totalSpent += amount
}

// Check verifies if spending is within all configured budget limits.
//
// Returns Status indicating if spending is allowed and which limit (if any)
// was exceeded. Also indicates if alert threshold was reached.
//
// If multiple limits are exceeded, the most restrictive (shortest window)
// is returned.
func (t *Tracker) Check() *Status {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Check hourly limit first (most restrictive)
	if t.config.Hourly > 0 && t.hourly != nil {
		used := t.hourly.Sum()
		percentage := used / t.config.Hourly

		if used > t.config.Hourly {
			return &Status{
				Allowed:    false,
				Reason:     "hourly budget limit exceeded",
				Limit:      t.config.Hourly,
				Used:       used,
				Remaining:  0,
				Percentage: percentage,
				Reset:      t.calculateReset(t.hourly),
				Window:     time.Hour,
			}
		}

		// Check alert threshold
		if t.config.AlertThreshold > 0 && percentage >= t.config.AlertThreshold {
			return &Status{
				Allowed:        true,
				Limit:          t.config.Hourly,
				Used:           used,
				Remaining:      t.config.Hourly - used,
				Percentage:     percentage,
				Reset:          t.calculateReset(t.hourly),
				Window:         time.Hour,
				AlertTriggered: true,
			}
		}
	}

	// Check daily limit
	if t.config.Daily > 0 && t.daily != nil {
		used := t.daily.Sum()
		percentage := used / t.config.Daily

		if used > t.config.Daily {
			return &Status{
				Allowed:    false,
				Reason:     "daily budget limit exceeded",
				Limit:      t.config.Daily,
				Used:       used,
				Remaining:  0,
				Percentage: percentage,
				Reset:      t.calculateReset(t.daily),
				Window:     24 * time.Hour,
			}
		}

		// Check alert threshold
		if t.config.AlertThreshold > 0 && percentage >= t.config.AlertThreshold {
			return &Status{
				Allowed:        true,
				Limit:          t.config.Daily,
				Used:           used,
				Remaining:      t.config.Daily - used,
				Percentage:     percentage,
				Reset:          t.calculateReset(t.daily),
				Window:         24 * time.Hour,
				AlertTriggered: true,
			}
		}
	}

	// Check monthly limit
	if t.config.Monthly > 0 && t.monthly != nil {
		used := t.monthly.Sum()
		percentage := used / t.config.Monthly

		if used > t.config.Monthly {
			return &Status{
				Allowed:    false,
				Reason:     "monthly budget limit exceeded",
				Limit:      t.config.Monthly,
				Used:       used,
				Remaining:  0,
				Percentage: percentage,
				Reset:      t.calculateReset(t.monthly),
				Window:     30 * 24 * time.Hour,
			}
		}

		// Check alert threshold
		if t.config.AlertThreshold > 0 && percentage >= t.config.AlertThreshold {
			return &Status{
				Allowed:        true,
				Limit:          t.config.Monthly,
				Used:           used,
				Remaining:      t.config.Monthly - used,
				Percentage:     percentage,
				Reset:          t.calculateReset(t.monthly),
				Window:         30 * 24 * time.Hour,
				AlertTriggered: true,
			}
		}
	}

	// All limits passed, no alerts
	return &Status{
		Allowed: true,
	}
}

// GetHourlyStatus returns the current hourly budget status.
func (t *Tracker) GetHourlyStatus() *Status {
	if t.config.Hourly == 0 || t.hourly == nil {
		return &Status{Allowed: true}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	used := t.hourly.Sum()
	percentage := used / t.config.Hourly

	return &Status{
		Allowed:        used <= t.config.Hourly,
		Limit:          t.config.Hourly,
		Used:           used,
		Remaining:      max(0, t.config.Hourly-used),
		Percentage:     percentage,
		Reset:          t.calculateReset(t.hourly),
		Window:         time.Hour,
		AlertTriggered: t.config.AlertThreshold > 0 && percentage >= t.config.AlertThreshold,
	}
}

// GetDailyStatus returns the current daily budget status.
func (t *Tracker) GetDailyStatus() *Status {
	if t.config.Daily == 0 || t.daily == nil {
		return &Status{Allowed: true}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	used := t.daily.Sum()
	percentage := used / t.config.Daily

	return &Status{
		Allowed:        used <= t.config.Daily,
		Limit:          t.config.Daily,
		Used:           used,
		Remaining:      max(0, t.config.Daily-used),
		Percentage:     percentage,
		Reset:          t.calculateReset(t.daily),
		Window:         24 * time.Hour,
		AlertTriggered: t.config.AlertThreshold > 0 && percentage >= t.config.AlertThreshold,
	}
}

// GetMonthlyStatus returns the current monthly budget status.
func (t *Tracker) GetMonthlyStatus() *Status {
	if t.config.Monthly == 0 || t.monthly == nil {
		return &Status{Allowed: true}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	used := t.monthly.Sum()
	percentage := used / t.config.Monthly

	return &Status{
		Allowed:        used <= t.config.Monthly,
		Limit:          t.config.Monthly,
		Used:           used,
		Remaining:      max(0, t.config.Monthly-used),
		Percentage:     percentage,
		Reset:          t.calculateReset(t.monthly),
		Window:         30 * 24 * time.Hour,
		AlertTriggered: t.config.AlertThreshold > 0 && percentage >= t.config.AlertThreshold,
	}
}

// GetTotalSpent returns the all-time total spending.
func (t *Tracker) GetTotalSpent() float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.totalSpent
}

// Reset clears all windows and resets total spent to zero.
// This is primarily for testing.
func (t *Tracker) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.hourly != nil {
		t.hourly.Reset()
	}
	if t.daily != nil {
		t.daily.Reset()
	}
	if t.monthly != nil {
		t.monthly.Reset()
	}

	t.totalSpent = 0
}

// calculateReset estimates when the rolling window will reset.
// This returns the time when the oldest bucket will expire.
// Caller must hold read lock.
func (t *Tracker) calculateReset(window *RollingWindow) time.Time {
	oldest := window.OldestTimestamp()
	if oldest.IsZero() {
		// No spending yet, window "resets" continuously
		return time.Now()
	}

	// The window resets when the oldest bucket expires
	return oldest.Add(window.window)
}

// max returns the maximum of two float64 values.
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
