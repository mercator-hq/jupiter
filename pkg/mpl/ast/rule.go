package ast

// Rule represents a single policy rule in the AST.
// A rule consists of conditions (when to apply) and actions (what to do).
// Rules are evaluated sequentially, and the first matching rule determines the outcome.
type Rule struct {
	Name        string         // Unique rule name within policy
	Description string         // Human-readable description
	Enabled     bool           // Whether rule is active (default: true)
	Conditions  *ConditionNode // Root condition node (can be logical operator)
	Actions     []*Action      // Actions to execute when conditions match
	Priority    int            // Explicit priority (lower = higher priority)
	Location    Location       // Source location
}

// IsEnabled returns true if the rule is enabled.
// Rules are enabled by default unless explicitly disabled.
func (r *Rule) IsEnabled() bool {
	return r.Enabled
}

// HasConditions returns true if the rule has conditions defined.
func (r *Rule) HasConditions() bool {
	return r.Conditions != nil
}

// HasActions returns true if the rule has actions defined.
func (r *Rule) HasActions() bool {
	return len(r.Actions) > 0
}

// GetActionsByType returns all actions of the given type in this rule.
func (r *Rule) GetActionsByType(actionType ActionType) []*Action {
	var result []*Action
	for _, action := range r.Actions {
		if action.Type == actionType {
			result = append(result, action)
		}
	}
	return result
}

// HasActionType returns true if the rule has at least one action of the given type.
func (r *Rule) HasActionType(actionType ActionType) bool {
	for _, action := range r.Actions {
		if action.Type == actionType {
			return true
		}
	}
	return false
}
