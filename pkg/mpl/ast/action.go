package ast

// ActionType represents the type of action in an MPL policy rule.
// Actions define what happens when a rule's conditions are met.
type ActionType string

const (
	ActionTypeAllow     ActionType = "allow"      // Explicitly allow the request
	ActionTypeDeny      ActionType = "deny"       // Block the request
	ActionTypeLog       ActionType = "log"        // Log an event
	ActionTypeRedact    ActionType = "redact"     // Redact sensitive content
	ActionTypeModify    ActionType = "modify"     // Modify request/response fields
	ActionTypeRoute     ActionType = "route"      // Route to specific provider/model
	ActionTypeAlert     ActionType = "alert"      // Trigger external alert
	ActionTypeTag       ActionType = "tag"        // Add metadata tags
	ActionTypeRateLimit ActionType = "rate_limit" // Apply rate limiting
	ActionTypeBudget    ActionType = "budget"     // Enforce budget constraints
)

// LogLevel represents the severity level for log actions.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
)

// RedactStrategy represents the strategy for redacting content.
type RedactStrategy string

const (
	RedactStrategyMask    RedactStrategy = "mask"    // Replace with ***
	RedactStrategyRemove  RedactStrategy = "remove"  // Remove entirely
	RedactStrategyReplace RedactStrategy = "replace" // Replace with specific text
)

// Action represents an action node in the AST.
// Actions are executed when a rule's conditions are satisfied.
type Action struct {
	Type       ActionType            // Type of action
	Parameters map[string]*ValueNode // Action parameters (type-specific)
	Location   Location              // Source location
}

// GetParameter returns the parameter value for the given key, or nil if not found.
func (a *Action) GetParameter(key string) *ValueNode {
	return a.Parameters[key]
}

// HasParameter returns true if the action has a parameter with the given key.
func (a *Action) HasParameter(key string) bool {
	_, ok := a.Parameters[key]
	return ok
}

// GetStringParameter returns the string value of a parameter.
// Returns empty string if parameter doesn't exist or is not a string.
func (a *Action) GetStringParameter(key string) string {
	if val := a.GetParameter(key); val != nil && val.Type == ValueTypeString {
		if str, ok := val.Value.(string); ok {
			return str
		}
	}
	return ""
}

// GetBoolParameter returns the boolean value of a parameter.
// Returns false if parameter doesn't exist or is not a boolean.
func (a *Action) GetBoolParameter(key string) bool {
	if val := a.GetParameter(key); val != nil && val.Type == ValueTypeBoolean {
		if b, ok := val.Value.(bool); ok {
			return b
		}
	}
	return false
}

// GetNumberParameter returns the numeric value of a parameter.
// Returns 0 if parameter doesn't exist or is not a number.
func (a *Action) GetNumberParameter(key string) float64 {
	if val := a.GetParameter(key); val != nil && val.Type == ValueTypeNumber {
		if num, ok := val.Value.(float64); ok {
			return num
		}
	}
	return 0
}
