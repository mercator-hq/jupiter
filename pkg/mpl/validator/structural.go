package validator

import (
	"fmt"
	"regexp"

	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

var (
	// semverPattern validates semantic version strings (e.g., "1.0.0", "2.1.3")
	semverPattern = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?(\+[a-zA-Z0-9.-]+)?$`)

	// kebabCasePattern validates kebab-case names (e.g., "my-policy-name")
	kebabCasePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

	// supportedMPLVersions defines which MPL versions this parser supports
	supportedMPLVersions = map[string]bool{
		"1.0": true,
	}
)

// StructuralValidator validates the structural integrity of a policy.
// It checks required fields, field types, naming conventions, and schema compliance.
type StructuralValidator struct {
	errors *mplErrors.ErrorList
}

// NewStructuralValidator creates a new structural validator.
func NewStructuralValidator() *StructuralValidator {
	return &StructuralValidator{
		errors: mplErrors.NewErrorList(),
	}
}

// Validate performs structural validation on a policy.
// It returns an ErrorList containing all structural errors found.
func (v *StructuralValidator) Validate(policy *ast.Policy) error {
	v.errors = mplErrors.NewErrorList()

	// Validate metadata
	v.validateMetadata(policy)

	// Validate variables
	v.validateVariables(policy)

	// Validate rules
	v.validateRules(policy)

	return v.errors.ToError()
}

// validateMetadata validates policy metadata fields.
func (v *StructuralValidator) validateMetadata(policy *ast.Policy) {
	// MPL version is required
	if policy.MPLVersion == "" {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			"Missing required field 'mpl_version'",
			policy.Location,
			mplErrors.SuggestMissingField("mpl_version", `"1.0"`),
		)
	} else if !supportedMPLVersions[policy.MPLVersion] {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			fmt.Sprintf("Unsupported MPL version %q", policy.MPLVersion),
			policy.Location,
			"Supported versions: 1.0",
		)
	}

	// Name is required
	if policy.Name == "" {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			"Missing required field 'name'",
			policy.Location,
			mplErrors.SuggestMissingField("name", `"my-policy"`),
		)
	} else if !kebabCasePattern.MatchString(policy.Name) {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			fmt.Sprintf("Policy name %q must be kebab-case (lowercase with hyphens)", policy.Name),
			policy.Location,
			"Example: 'my-policy-name'",
		)
	}

	// Version is required
	if policy.Version == "" {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			"Missing required field 'version'",
			policy.Location,
			mplErrors.SuggestMissingField("version", `"1.0.0"`),
		)
	} else if !semverPattern.MatchString(policy.Version) {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			fmt.Sprintf("Policy version %q must follow semantic versioning", policy.Version),
			policy.Location,
			"Example: '1.0.0' or '2.1.3-beta.1'",
		)
	}

	// Rules are required
	if len(policy.Rules) == 0 {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			"Policy must have at least one rule",
			policy.Location,
			"Add a 'rules' section with at least one rule",
		)
	}
}

// validateVariables validates variable definitions.
func (v *StructuralValidator) validateVariables(policy *ast.Policy) {
	for name, variable := range policy.Variables {
		// Validate variable name (should be valid identifier)
		if !isValidIdentifier(name) {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Invalid variable name %q (must be alphanumeric with underscores)", name),
				variable.Location,
			)
		}

		// Validate variable value is not null
		if variable.Value.Type == ast.ValueTypeNull {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Variable %q cannot have null value", name),
				variable.Location,
			)
		}
	}
}

// validateRules validates all rules in the policy.
func (v *StructuralValidator) validateRules(policy *ast.Policy) {
	ruleNames := make(map[string]bool)

	for i, rule := range policy.Rules {
		// Rule name is required
		if rule.Name == "" {
			v.errors.AddErrorWithSuggestion(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule at index %d missing required field 'name'", i),
				rule.Location,
				"Add a unique name for this rule",
			)
			continue
		}

		// Rule names must be unique
		if ruleNames[rule.Name] {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Duplicate rule name %q", rule.Name),
				rule.Location,
			)
		}
		ruleNames[rule.Name] = true

		// Rule must have conditions or always-execute flag
		if rule.Conditions == nil {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has no conditions", rule.Name),
				rule.Location,
			)
		}

		// Rule must have at least one action
		if len(rule.Actions) == 0 {
			v.errors.AddErrorWithSuggestion(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has no actions", rule.Name),
				rule.Location,
				"Add at least one action (allow, deny, log, etc.)",
			)
		}

		// Validate conditions structure
		if rule.Conditions != nil {
			v.validateConditionStructure(rule.Conditions, rule.Name, 0)
		}

		// Validate actions structure
		for j, action := range rule.Actions {
			v.validateActionStructure(action, rule.Name, j)
		}
	}
}

// validateConditionStructure validates the structure of a condition node.
func (v *StructuralValidator) validateConditionStructure(cond *ast.ConditionNode, ruleName string, depth int) {
	const maxDepth = 10

	// Check maximum nesting depth
	if depth > maxDepth {
		v.errors.AddError(
			mplErrors.ErrorTypeStructural,
			fmt.Sprintf("Rule %q exceeds maximum condition nesting depth of %d", ruleName, maxDepth),
			cond.Location,
		)
		return
	}

	switch cond.Type {
	case ast.ConditionTypeSimple:
		// Simple condition must have field, operator, and value
		if cond.Field == "" {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has condition with missing 'field'", ruleName),
				cond.Location,
			)
		}
		if cond.Operator == "" {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has condition with missing 'operator'", ruleName),
				cond.Location,
			)
		}
		if cond.Value == nil {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has condition with missing 'value'", ruleName),
				cond.Location,
			)
		}

	case ast.ConditionTypeAll, ast.ConditionTypeAny:
		// Logical operators must have children
		if len(cond.Children) == 0 {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has %q condition with no children", ruleName, cond.Type),
				cond.Location,
			)
		}

		// Validate children recursively
		for _, child := range cond.Children {
			v.validateConditionStructure(child, ruleName, depth+1)
		}

	case ast.ConditionTypeNot:
		// NOT must have exactly one child
		if len(cond.Children) != 1 {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has 'not' condition with %d children (must be exactly 1)", ruleName, len(cond.Children)),
				cond.Location,
			)
		}

		// Validate child recursively
		for _, child := range cond.Children {
			v.validateConditionStructure(child, ruleName, depth+1)
		}

	case ast.ConditionTypeFunction:
		// Function condition must have function name
		if cond.Function == "" {
			v.errors.AddError(
				mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Rule %q has function condition with missing 'function'", ruleName),
				cond.Location,
			)
		}
	}
}

// validateActionStructure validates the structure of an action.
func (v *StructuralValidator) validateActionStructure(action *ast.Action, ruleName string, index int) {
	// Action type is required
	if action.Type == "" {
		v.errors.AddError(
			mplErrors.ErrorTypeStructural,
			fmt.Sprintf("Rule %q action at index %d missing 'type'", ruleName, index),
			action.Location,
		)
	}

	// Validate action type is recognized
	validActionTypes := map[ast.ActionType]bool{
		ast.ActionTypeAllow:     true,
		ast.ActionTypeDeny:      true,
		ast.ActionTypeLog:       true,
		ast.ActionTypeRedact:    true,
		ast.ActionTypeModify:    true,
		ast.ActionTypeRoute:     true,
		ast.ActionTypeAlert:     true,
		ast.ActionTypeRateLimit: true,
		ast.ActionTypeBudget:    true,
	}

	if !validActionTypes[action.Type] {
		validTypes := []string{
			string(ast.ActionTypeAllow),
			string(ast.ActionTypeDeny),
			string(ast.ActionTypeLog),
			string(ast.ActionTypeRedact),
			string(ast.ActionTypeModify),
			string(ast.ActionTypeRoute),
			string(ast.ActionTypeAlert),
			string(ast.ActionTypeRateLimit),
			string(ast.ActionTypeBudget),
		}
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeStructural,
			fmt.Sprintf("Rule %q has unknown action type %q", ruleName, action.Type),
			action.Location,
			mplErrors.SuggestActionType(string(action.Type), validTypes),
		)
	}
}

// isValidIdentifier checks if a string is a valid identifier (variable name).
func isValidIdentifier(s string) bool {
	if s == "" {
		return false
	}

	// Must start with letter or underscore
	if !(s[0] >= 'a' && s[0] <= 'z') && !(s[0] >= 'A' && s[0] <= 'Z') && s[0] != '_' {
		return false
	}

	// Rest must be alphanumeric or underscore
	for i := 1; i < len(s); i++ {
		c := s[i]
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '_' {
			return false
		}
	}

	return true
}
