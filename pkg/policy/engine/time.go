package engine

import (
	"time"
)

// BusinessHoursConfig defines business hours for time-based policy conditions.
type BusinessHoursConfig struct {
	// Timezone for business hours (e.g., "America/New_York", "UTC")
	Timezone string

	// DaysOfWeek defines which days are business days (1 = Monday, 7 = Sunday)
	// Empty slice means all days
	DaysOfWeek []int

	// StartHour is the start of business hours (0-23)
	StartHour int

	// EndHour is the end of business hours (0-23)
	EndHour int
}

// DefaultBusinessHoursConfig returns default business hours (Mon-Fri, 9am-5pm UTC).
func DefaultBusinessHoursConfig() *BusinessHoursConfig {
	return &BusinessHoursConfig{
		Timezone:   "UTC",
		DaysOfWeek: []int{1, 2, 3, 4, 5}, // Monday-Friday
		StartHour:  9,
		EndHour:    17,
	}
}

// IsBusinessHours checks if the given time falls within business hours.
func (c *BusinessHoursConfig) IsBusinessHours(t time.Time) bool {
	// Load timezone
	loc, err := time.LoadLocation(c.Timezone)
	if err != nil {
		// Fall back to UTC if timezone load fails
		loc = time.UTC
	}

	// Convert time to configured timezone
	localTime := t.In(loc)

	// Check day of week (if configured)
	if len(c.DaysOfWeek) > 0 {
		dayOfWeek := int(localTime.Weekday())
		if dayOfWeek == 0 {
			dayOfWeek = 7 // Convert Sunday from 0 to 7
		}

		isBusinessDay := false
		for _, day := range c.DaysOfWeek {
			if dayOfWeek == day {
				isBusinessDay = true
				break
			}
		}

		if !isBusinessDay {
			return false
		}
	}

	// Check hour range
	hour := localTime.Hour()
	return hour >= c.StartHour && hour < c.EndHour
}

// TimeWindowConfig defines a time window for policy conditions.
type TimeWindowConfig struct {
	// Start is the start time of the window
	Start time.Time

	// End is the end time of the window
	End time.Time

	// Recurring indicates if the window repeats (e.g., daily, weekly)
	Recurring bool

	// RecurrenceType defines how the window recurs ("daily", "weekly", "monthly")
	RecurrenceType string
}

// IsInWindow checks if the given time falls within the time window.
func (c *TimeWindowConfig) IsInWindow(t time.Time) bool {
	if !c.Recurring {
		// Simple time range check
		return !t.Before(c.Start) && !t.After(c.End)
	}

	// Handle recurring windows
	switch c.RecurrenceType {
	case "daily":
		return c.isInDailyWindow(t)
	case "weekly":
		return c.isInWeeklyWindow(t)
	case "monthly":
		return c.isInMonthlyWindow(t)
	default:
		return !t.Before(c.Start) && !t.After(c.End)
	}
}

// isInDailyWindow checks if time matches the daily recurring pattern.
func (c *TimeWindowConfig) isInDailyWindow(t time.Time) bool {
	// Extract hour and minute from start/end
	startHour, startMin, _ := c.Start.Clock()
	endHour, endMin, _ := c.End.Clock()

	tHour, tMin, _ := t.Clock()

	// Convert to minutes for easier comparison
	startMinutes := startHour*60 + startMin
	endMinutes := endHour*60 + endMin
	tMinutes := tHour*60 + tMin

	return tMinutes >= startMinutes && tMinutes < endMinutes
}

// isInWeeklyWindow checks if time matches the weekly recurring pattern.
func (c *TimeWindowConfig) isInWeeklyWindow(t time.Time) bool {
	// Check if day of week matches
	if t.Weekday() != c.Start.Weekday() {
		return false
	}

	// Check time within the day
	return c.isInDailyWindow(t)
}

// isInMonthlyWindow checks if time matches the monthly recurring pattern.
func (c *TimeWindowConfig) isInMonthlyWindow(t time.Time) bool {
	// Check if day of month matches
	if t.Day() != c.Start.Day() {
		return false
	}

	// Check time within the day
	return c.isInDailyWindow(t)
}
