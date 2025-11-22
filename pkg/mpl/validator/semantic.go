package validator

import (
	"fmt"

	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

// SemanticValidator validates semantic correctness of policies.
// It checks field references, type compatibility, variable usage, and circular references.
type SemanticValidator struct {
	policy *ast.Policy
	errors *mplErrors.ErrorList
}

// NewSemanticValidator creates a new semantic validator.
func NewSemanticValidator() *SemanticValidator {
	return &SemanticValidator{
		errors: mplErrors.NewErrorList(),
	}
}

// Validate performs semantic validation on a policy.
func (v *SemanticValidator) Validate(policy *ast.Policy) error {
	v.policy = policy
	v.errors = mplErrors.NewErrorList()

	// Validate variable circular references
	v.validateVariableReferences()

	// Validate conditions in all rules
	for _, rule := range policy.Rules {
		if rule.Conditions != nil {
			v.validateCondition(rule.Conditions, rule.Name)
		}
	}

	return v.errors.ToError()
}

// validateVariableReferences checks for circular variable references.
func (v *SemanticValidator) validateVariableReferences() {
	visited := make(map[string]bool)
	inProgress := make(map[string]bool)

	for name := range v.policy.Variables {
		if !visited[name] {
			v.checkVariableCycle(name, visited, inProgress, []string{})
		}
	}
}

// checkVariableCycle performs DFS to detect circular variable references.
func (v *SemanticValidator) checkVariableCycle(varName string, visited, inProgress map[string]bool, path []string) {
	visited[varName] = true
	inProgress[varName] = true
	path = append(path, varName)

	variable, ok := v.policy.Variables[varName]
	if !ok {
		return
	}

	// Check if variable value references other variables
	refs := v.extractVariableReferences(variable.Value)
	for _, ref := range refs {
		if inProgress[ref] {
			// Circular reference detected
			cycle := append(path, ref)
			v.errors.AddErrorWithSuggestion(
				mplErrors.ErrorTypeSemantic,
				fmt.Sprintf("Circular variable reference: %v", cycle),
				variable.Location,
				"Remove the circular dependency between variables",
			)
			continue
		}

		if !visited[ref] {
			v.checkVariableCycle(ref, visited, inProgress, path)
		}
	}

	inProgress[varName] = false
}

// extractVariableReferences extracts variable names referenced in a value.
func (v *SemanticValidator) extractVariableReferences(value *ast.ValueNode) []string {
	if value == nil {
		return nil
	}

	if value.Type == ast.ValueTypeVariable {
		return []string{value.VariableName}
	}

	// For arrays and objects, we'd need to recursively extract
	// Simplified for now - handle in future if needed
	return nil
}

// validateCondition validates a condition node.
func (v *SemanticValidator) validateCondition(cond *ast.ConditionNode, ruleName string) {
	switch cond.Type {
	case ast.ConditionTypeSimple:
		v.validateSimpleCondition(cond, ruleName)

	case ast.ConditionTypeAll, ast.ConditionTypeAny, ast.ConditionTypeNot:
		// Validate children recursively
		for _, child := range cond.Children {
			v.validateCondition(child, ruleName)
		}

	case ast.ConditionTypeFunction:
		v.validateFunctionCondition(cond, ruleName)
	}
}

// validateSimpleCondition validates a simple comparison condition.
func (v *SemanticValidator) validateSimpleCondition(cond *ast.ConditionNode, ruleName string) {
	// Validate field reference exists in data model
	fieldInfo, ok := LookupField(cond.Field)
	if !ok {
		allFields := GetAllFieldPaths()
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeSemantic,
			fmt.Sprintf("Rule %q references undefined field %q", ruleName, cond.Field),
			cond.Location,
			mplErrors.SuggestFieldName(cond.Field, allFields),
		)
		return // Skip further validation if field doesn't exist
	}

	// Validate operator is valid for field type
	if !v.isValidOperatorForType(cond.Operator, fieldInfo.Type) {
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeSemantic,
			fmt.Sprintf("Rule %q uses invalid operator %q for field type %q", ruleName, cond.Operator, fieldInfo.Type),
			cond.Location,
			mplErrors.SuggestOperator(string(fieldInfo.Type)),
		)
	}

	// Validate value type matches field type (or is a variable)
	if cond.Value != nil {
		if cond.Value.Type == ast.ValueTypeVariable {
			// Validate variable exists
			if !v.policy.HasVariable(cond.Value.VariableName) {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeSemantic,
					fmt.Sprintf("Rule %q references undefined variable %q", ruleName, cond.Value.VariableName),
					cond.Location,
					fmt.Sprintf("Define '%s' in the variables section", cond.Value.VariableName),
				)
			}
		} else {
			// Validate value type matches field type
			if !v.isCompatibleType(cond.Value.Type, fieldInfo.Type, cond.Operator) {
				v.errors.AddError(
					mplErrors.ErrorTypeSemantic,
					fmt.Sprintf("Rule %q compares field %q (type %q) with incompatible value type %q",
						ruleName, cond.Field, fieldInfo.Type, cond.Value.Type),
					cond.Location,
				)
			}
		}
	}
}

// validateFunctionCondition validates a function call condition.
func (v *SemanticValidator) validateFunctionCondition(cond *ast.ConditionNode, ruleName string) {
	// Define supported functions and their signatures
	supportedFunctions := map[string]FunctionSignature{
		"has_pii": {
			Name:        "has_pii",
			Description: "Detects PII in content",
			MinArgs:     0,
			MaxArgs:     1, // Optional field to check
		},
		"has_injection": {
			Name:        "has_injection",
			Description: "Detects prompt injection attempts",
			MinArgs:     0,
			MaxArgs:     1,
		},
		"has_sensitive": {
			Name:        "has_sensitive",
			Description: "Detects sensitive content",
			MinArgs:     0,
			MaxArgs:     2, // Optional field and severity
		},
		"len": {
			Name:        "len",
			Description: "Returns length of string or array",
			MinArgs:     1,
			MaxArgs:     1,
		},
		"lower": {
			Name:        "lower",
			Description: "Converts string to lowercase",
			MinArgs:     1,
			MaxArgs:     1,
		},
		"upper": {
			Name:        "upper",
			Description: "Converts string to uppercase",
			MinArgs:     1,
			MaxArgs:     1,
		},
		"contains": {
			Name:        "contains",
			Description: "Checks if string contains substring",
			MinArgs:     2,
			MaxArgs:     2,
		},
	}

	sig, ok := supportedFunctions[cond.Function]
	if !ok {
		var funcNames []string
		for name := range supportedFunctions {
			funcNames = append(funcNames, name)
		}
		v.errors.AddErrorWithSuggestion(
			mplErrors.ErrorTypeSemantic,
			fmt.Sprintf("Rule %q uses unknown function %q", ruleName, cond.Function),
			cond.Location,
			mplErrors.SuggestFieldName(cond.Function, funcNames),
		)
		return
	}

	// Validate argument count
	argCount := len(cond.Args)
	if argCount < sig.MinArgs || argCount > sig.MaxArgs {
		v.errors.AddError(
			mplErrors.ErrorTypeSemantic,
			fmt.Sprintf("Rule %q calls function %q with %d arguments (expected %d-%d)",
				ruleName, cond.Function, argCount, sig.MinArgs, sig.MaxArgs),
			cond.Location,
		)
	}

	// Validate arguments
	for i, arg := range cond.Args {
		if arg.Type == ast.ValueTypeVariable {
			if !v.policy.HasVariable(arg.VariableName) {
				v.errors.AddErrorWithSuggestion(
					mplErrors.ErrorTypeSemantic,
					fmt.Sprintf("Rule %q uses undefined variable %q in function argument", ruleName, arg.VariableName),
					cond.Location,
					fmt.Sprintf("Define '%s' in the variables section", arg.VariableName),
				)
			}
		}

		// Additional argument type validation could be added here
		_ = i // Unused for now
	}
}

// FunctionSignature describes a built-in function's signature.
type FunctionSignature struct {
	Name        string
	Description string
	MinArgs     int
	MaxArgs     int
}

// isValidOperatorForType checks if an operator is valid for a field type.
func (v *SemanticValidator) isValidOperatorForType(op ast.Operator, fieldType ast.ValueType) bool {
	switch fieldType {
	case ast.ValueTypeString:
		return op == ast.OperatorEqual ||
			op == ast.OperatorNotEqual ||
			op == ast.OperatorContains ||
			op == ast.OperatorMatches ||
			op == ast.OperatorStartsWith ||
			op == ast.OperatorEndsWith ||
			op == ast.OperatorIn ||
			op == ast.OperatorNotIn

	case ast.ValueTypeNumber:
		return op == ast.OperatorEqual ||
			op == ast.OperatorNotEqual ||
			op == ast.OperatorLessThan ||
			op == ast.OperatorGreaterThan ||
			op == ast.OperatorLessEqual ||
			op == ast.OperatorGreaterEqual ||
			op == ast.OperatorIn ||
			op == ast.OperatorNotIn

	case ast.ValueTypeBoolean:
		return op == ast.OperatorEqual ||
			op == ast.OperatorNotEqual

	case ast.ValueTypeArray:
		return op == ast.OperatorContains ||
			op == ast.OperatorIn ||
			op == ast.OperatorNotIn

	case ast.ValueTypeObject:
		return op == ast.OperatorEqual ||
			op == ast.OperatorNotEqual

	default:
		return true // Allow all operators for unknown types
	}
}

// isCompatibleType checks if a value type is compatible with a field type for comparison.
func (v *SemanticValidator) isCompatibleType(valueType, fieldType ast.ValueType, op ast.Operator) bool {
	// For 'in' and 'not_in' operators, value should be array
	if op == ast.OperatorIn || op == ast.OperatorNotIn {
		return valueType == ast.ValueTypeArray
	}

	// For 'contains' on arrays, value can be any type
	if op == ast.OperatorContains && fieldType == ast.ValueTypeArray {
		return true
	}

	// Otherwise, types should match
	return valueType == fieldType
}
