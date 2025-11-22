package enforcement

import "time"

// Action defines what to do when a limit is exceeded.
type Action string

const (
	// ActionAllow permits the request to proceed.
	ActionAllow Action = "allow"

	// ActionBlock rejects the request with 429 Too Many Requests.
	ActionBlock Action = "block"

	// ActionQueue holds the request until capacity is available.
	ActionQueue Action = "queue"

	// ActionDowngrade routes to a cheaper model.
	ActionDowngrade Action = "downgrade"

	// ActionAlert triggers an alert but allows the request.
	ActionAlert Action = "alert"
)

// Config contains configuration for the enforcer.
type Config struct {
	// DefaultAction is the action to take when no specific action is configured.
	DefaultAction Action

	// QueueDepth is the maximum number of requests to queue (when action=queue).
	QueueDepth int

	// QueueTimeout is how long to wait for queue capacity before giving up.
	QueueTimeout time.Duration

	// ModelDowngrades maps expensive models to cheaper alternatives.
	// Example: "gpt-4" -> "gpt-3.5-turbo"
	ModelDowngrades map[string]string
}

// Result contains the result of an enforcement action.
type Result struct {
	// Allowed indicates if the request should proceed.
	Allowed bool

	// Action is the enforcement action that was taken.
	Action Action

	// Reason explains why the request was blocked (if Allowed=false).
	Reason string

	// DowngradedModel is the cheaper model to use (if action=downgrade).
	DowngradedModel string

	// RetryAfter suggests how long to wait before retrying (if action=block).
	RetryAfter time.Duration

	// AlertMessage contains the alert message (if action=alert).
	AlertMessage string
}
