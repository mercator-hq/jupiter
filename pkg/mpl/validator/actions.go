package validator

import (
	"fmt"

	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

// ActionValidator validates action definitions and parameters.
// It checks required parameters, parameter types, and conflicting actions.
type ActionValidator struct {
	errors *mplErrors.ErrorList
}

// NewActionValidator creates a new action validator.
func NewActionValidator() *ActionValidator {
	return &ActionValidator{
		errors: mplErrors.NewErrorList(),
	}
}

// Validate performs action validation on a policy.
func (v *ActionValidator) Validate(policy *ast.Policy) error {
	v.errors = mplErrors.NewErrorList()

	for _, rule := range policy.Rules {
		v.validateRuleActions(rule)
	}

	return v.errors.ToError()
}

// validateRuleActions validates all actions in a rule.
func (v *ActionValidator) validateRuleActions(rule *ast.Rule) {
	// Check for conflicting actions
	v.detectConflictingActions(rule)

	// Validate each action's parameters
	for _, action := range rule.Actions {
		v.validateAction(action, rule.Name)
	}
}

// detectConflictingActions detects conflicting actions in a rule.
func (v *ActionValidator) detectConflictingActions(rule *ast.Rule) {
	hasAllow := rule.HasActionType(ast.ActionTypeAllow)
	hasDeny := rule.HasActionType(ast.ActionTypeDeny)

	// Allow and deny are mutually exclusive
	if hasAllow && hasDeny {
		v.errors.AddError(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q has both 'allow' and 'deny' actions (conflicting)", rule.Name),
			rule.Location,
		)
	}
}

// validateAction validates a single action.
func (v *ActionValidator) validateAction(action *ast.Action, ruleName string) {
	switch action.Type {
	case ast.ActionTypeAllow:
		v.validateAllowAction(action, ruleName)
	case ast.ActionTypeDeny:
		v.validateDenyAction(action, ruleName)
	case ast.ActionTypeLog:
		v.validateLogAction(action, ruleName)
	case ast.ActionTypeRedact:
		v.validateRedactAction(action, ruleName)
	case ast.ActionTypeModify:
		v.validateModifyAction(action, ruleName)
	case ast.ActionTypeRoute:
		v.validateRouteAction(action, ruleName)
	case ast.ActionTypeAlert:
		v.validateAlertAction(action, ruleName)
	case ast.ActionTypeRateLimit:
		v.validateRateLimitAction(action, ruleName)
	case ast.ActionTypeBudget:
		v.validateBudgetAction(action, ruleName)
	}
}

// validateAllowAction validates an 'allow' action.
func (v *ActionValidator) validateAllowAction(action *ast.Action, ruleName string) {
	// Allow action has no required parameters
	// But we can warn about unexpected parameters
	for param := range action.Parameters {
		v.errors.AddError(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'allow' action has unexpected parameter %q", ruleName, param),
			action.Location,
		)
	}
}

// validateDenyAction validates a 'deny' action.
func (v *ActionValidator) validateDenyAction(action *ast.Action, ruleName string) {
	// Required: message
	if !action.HasParameter("message") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'deny' action missing required parameter 'message'", ruleName),
			action.Location,
			"Add 'message: \"Reason for denial\"'",
		)
	} else {
		msg := action.GetParameter("message")
		if msg.Type != ast.ValueTypeString && msg.Type != ast.ValueTypeVariable {
			v.errors.AddError(
				mplErrors.ErrorTypeValidation,
				fmt.Sprintf("Rule %q 'deny' action 'message' must be a string", ruleName),
				action.Location,
			)
		}
	}

	// Optional: code (string)
	if action.HasParameter("code") {
		code := action.GetParameter("code")
		if code.Type != ast.ValueTypeString && code.Type != ast.ValueTypeVariable {
			v.errors.AddError(
				mplErrors.ErrorTypeValidation,
				fmt.Sprintf("Rule %q 'deny' action 'code' must be a string", ruleName),
				action.Location,
			)
		}
	}
}

// validateLogAction validates a 'log' action.
func (v *ActionValidator) validateLogAction(action *ast.Action, ruleName string) {
	// Required: message
	if !action.HasParameter("message") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'log' action missing required parameter 'message'", ruleName),
			action.Location,
			"Add 'message: \"Log message\"'",
		)
	}

	// Optional: level (debug, info, warn, error)
	if action.HasParameter("level") {
		level := action.GetParameter("level")
		if level.Type == ast.ValueTypeString {
			levelStr := level.Value.(string)
			validLevels := map[string]bool{
				"debug": true,
				"info":  true,
				"warn":  true,
				"error": true,
			}
			if !validLevels[levelStr] {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeValidation,
					fmt.Sprintf("Rule %q 'log' action has invalid level %q", ruleName, levelStr),
					action.Location,
					"Valid levels: debug, info, warn, error",
				)
			}
		}
	}
}

// validateRedactAction validates a 'redact' action.
func (v *ActionValidator) validateRedactAction(action *ast.Action, ruleName string) {
	// Required: fields (array of field paths)
	if !action.HasParameter("fields") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'redact' action missing required parameter 'fields'", ruleName),
			action.Location,
			"Add 'fields: [\"field.path\"]'",
		)
	} else {
		fields := action.GetParameter("fields")
		if fields.Type != ast.ValueTypeArray && fields.Type != ast.ValueTypeVariable {
			v.errors.AddError(
				mplErrors.ErrorTypeValidation,
				fmt.Sprintf("Rule %q 'redact' action 'fields' must be an array", ruleName),
				action.Location,
			)
		}
	}

	// Optional: strategy (mask, remove, replace)
	if action.HasParameter("strategy") {
		strategy := action.GetParameter("strategy")
		if strategy.Type == ast.ValueTypeString {
			strategyStr := strategy.Value.(string)
			validStrategies := map[string]bool{
				"mask":    true,
				"remove":  true,
				"replace": true,
			}
			if !validStrategies[strategyStr] {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeValidation,
					fmt.Sprintf("Rule %q 'redact' action has invalid strategy %q", ruleName, strategyStr),
					action.Location,
					"Valid strategies: mask, remove, replace",
				)
			}
		}
	}

	// Optional: replacement (required if strategy is 'replace')
	if action.HasParameter("strategy") {
		strategy := action.GetParameter("strategy")
		if strategy.Type == ast.ValueTypeString && strategy.Value.(string) == "replace" {
			if !action.HasParameter("replacement") {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeValidation,
					fmt.Sprintf("Rule %q 'redact' action with strategy 'replace' missing 'replacement'", ruleName),
					action.Location,
					"Add 'replacement: \"[REDACTED]\"'",
				)
			}
		}
	}
}

// validateModifyAction validates a 'modify' action.
func (v *ActionValidator) validateModifyAction(action *ast.Action, ruleName string) {
	// Required: field (string path)
	if !action.HasParameter("field") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'modify' action missing required parameter 'field'", ruleName),
			action.Location,
			"Add 'field: \"request.field_name\"'",
		)
	}

	// Required: value (new value)
	if !action.HasParameter("value") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'modify' action missing required parameter 'value'", ruleName),
			action.Location,
			"Add 'value: \"new_value\"'",
		)
	}
}

// validateRouteAction validates a 'route' action.
func (v *ActionValidator) validateRouteAction(action *ast.Action, ruleName string) {
	// At least one of 'provider' or 'model' is required
	hasProvider := action.HasParameter("provider")
	hasModel := action.HasParameter("model")

	if !hasProvider && !hasModel {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'route' action must specify 'provider' or 'model'", ruleName),
			action.Location,
			"Add 'provider: \"openai\"' or 'model: \"gpt-4\"'",
		)
	}
}

// validateAlertAction validates an 'alert' action.
func (v *ActionValidator) validateAlertAction(action *ast.Action, ruleName string) {
	// Required: webhook (URL)
	if !action.HasParameter("webhook") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'alert' action missing required parameter 'webhook'", ruleName),
			action.Location,
			"Add 'webhook: \"https://example.com/webhook\"'",
		)
	}

	// Optional: message
	// Optional: severity (low, medium, high, critical)
	if action.HasParameter("severity") {
		severity := action.GetParameter("severity")
		if severity.Type == ast.ValueTypeString {
			severityStr := severity.Value.(string)
			validSeverities := map[string]bool{
				"low":      true,
				"medium":   true,
				"high":     true,
				"critical": true,
			}
			if !validSeverities[severityStr] {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeValidation,
					fmt.Sprintf("Rule %q 'alert' action has invalid severity %q", ruleName, severityStr),
					action.Location,
					"Valid severities: low, medium, high, critical",
				)
			}
		}
	}
}

// validateRateLimitAction validates a 'rate_limit' action.
func (v *ActionValidator) validateRateLimitAction(action *ast.Action, ruleName string) {
	// Required: key (rate limit key)
	if !action.HasParameter("key") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'rate_limit' action missing required parameter 'key'", ruleName),
			action.Location,
			"Add 'key: \"user\"' or 'key: \"ip\"'",
		)
	}

	// Required: limit (number)
	if !action.HasParameter("limit") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'rate_limit' action missing required parameter 'limit'", ruleName),
			action.Location,
			"Add 'limit: 100'",
		)
	} else {
		limit := action.GetParameter("limit")
		if limit.Type != ast.ValueTypeNumber && limit.Type != ast.ValueTypeVariable {
			v.errors.AddError(
				mplErrors.ErrorTypeValidation,
				fmt.Sprintf("Rule %q 'rate_limit' action 'limit' must be a number", ruleName),
				action.Location,
			)
		}
	}

	// Required: window (time window in seconds)
	if !action.HasParameter("window") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'rate_limit' action missing required parameter 'window'", ruleName),
			action.Location,
			"Add 'window: 3600' (time window in seconds)",
		)
	} else {
		window := action.GetParameter("window")
		if window.Type != ast.ValueTypeNumber && window.Type != ast.ValueTypeVariable {
			v.errors.AddError(
				mplErrors.ErrorTypeValidation,
				fmt.Sprintf("Rule %q 'rate_limit' action 'window' must be a number", ruleName),
				action.Location,
			)
		}
	}
}

// validateBudgetAction validates a 'budget' action.
func (v *ActionValidator) validateBudgetAction(action *ast.Action, ruleName string) {
	// Required: type (tokens or cost)
	if !action.HasParameter("type") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'budget' action missing required parameter 'type'", ruleName),
			action.Location,
			"Add 'type: \"tokens\"' or 'type: \"cost\"'",
		)
	} else {
		budgetType := action.GetParameter("type")
		if budgetType.Type == ast.ValueTypeString {
			typeStr := budgetType.Value.(string)
			if typeStr != "tokens" && typeStr != "cost" {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeValidation,
					fmt.Sprintf("Rule %q 'budget' action has invalid type %q", ruleName, typeStr),
					action.Location,
					"Valid types: tokens, cost",
				)
			}
		}
	}

	// Required: limit (number)
	if !action.HasParameter("limit") {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeValidation,
			fmt.Sprintf("Rule %q 'budget' action missing required parameter 'limit'", ruleName),
			action.Location,
			"Add 'limit: 10000'",
		)
	} else {
		limit := action.GetParameter("limit")
		if limit.Type != ast.ValueTypeNumber && limit.Type != ast.ValueTypeVariable {
			v.errors.AddError(
				mplErrors.ErrorTypeValidation,
				fmt.Sprintf("Rule %q 'budget' action 'limit' must be a number", ruleName),
				action.Location,
			)
		}
	}

	// Optional: window (time window - daily, hourly, etc.)
	if action.HasParameter("window") {
		window := action.GetParameter("window")
		if window.Type == ast.ValueTypeString {
			windowStr := window.Value.(string)
			validWindows := map[string]bool{
				"hourly":  true,
				"daily":   true,
				"weekly":  true,
				"monthly": true,
			}
			if !validWindows[windowStr] {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeValidation,
					fmt.Sprintf("Rule %q 'budget' action has invalid window %q", ruleName, windowStr),
					action.Location,
					"Valid windows: hourly, daily, weekly, monthly",
				)
			}
		}
	}
}
