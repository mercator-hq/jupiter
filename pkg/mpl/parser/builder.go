package parser

import (
	"fmt"
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

// builder constructs AST nodes from intermediate YAML structures.
// It handles type conversion, validation, and preserves source locations.
type builder struct {
	sourcePath string
	errors     *mplErrors.ErrorList
}

// newBuilder creates a new AST builder for the given source file.
func newBuilder(sourcePath string) *builder {
	return &builder{
		sourcePath: sourcePath,
		errors:     mplErrors.NewErrorList(),
	}
}

// buildPolicy transforms a yamlPolicy into an ast.Policy.
func (b *builder) buildPolicy(yp *yamlPolicy) (*ast.Policy, error) {
	policy := &ast.Policy{
		MPLVersion:  yp.MPLVersion,
		Name:        yp.Name,
		Version:     yp.Version,
		Description: yp.Description,
		Author:      yp.Author,
		Tags:        yp.Tags,
		Includes:    yp.Includes,
		SourceFile:  b.sourcePath,
		Variables:   make(map[string]*ast.Variable),
		Rules:       make([]*ast.Rule, 0, len(yp.Rules)),
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1,
			Column: 1,
		},
	}

	// Parse timestamps
	if yp.Created != "" {
		if t, err := time.Parse(time.RFC3339, yp.Created); err == nil {
			policy.Created = t
		}
	}
	if yp.Updated != "" {
		if t, err := time.Parse(time.RFC3339, yp.Updated); err == nil {
			policy.Updated = t
		}
	}

	// Build variables
	for name, value := range yp.Variables {
		variable, err := b.buildVariable(name, value)
		if err != nil {
			b.errors.AddError(mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Invalid variable %q: %v", name, err),
				policy.Location)
			continue
		}
		policy.Variables[name] = variable
	}

	// Build rules
	for i, yr := range yp.Rules {
		rule, err := b.buildRule(&yr, i)
		if err != nil {
			b.errors.AddError(mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Invalid rule at index %d: %v", i, err),
				policy.Location)
			continue
		}
		policy.Rules = append(policy.Rules, rule)
	}

	// Build tests
	policy.Tests = make([]*ast.PolicyTest, 0, len(yp.Tests))
	for i, yt := range yp.Tests {
		test, err := b.buildTest(&yt, i)
		if err != nil {
			b.errors.AddError(mplErrors.ErrorTypeStructural,
				fmt.Sprintf("Invalid test at index %d: %v", i, err),
				policy.Location)
			continue
		}
		policy.Tests = append(policy.Tests, test)
	}

	if b.errors.HasErrors() {
		return nil, b.errors
	}

	return policy, nil
}

// buildVariable transforms a variable value into an ast.Variable.
func (b *builder) buildVariable(name string, value interface{}) (*ast.Variable, error) {
	valueNode, err := b.buildValue(value)
	if err != nil {
		return nil, err
	}

	return &ast.Variable{
		Name:  name,
		Value: valueNode,
		Type:  valueNode.Type,
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1, // TODO: Extract from YAML node
			Column: 1,
		},
	}, nil
}

// buildRule transforms a yamlRule into an ast.Rule.
func (b *builder) buildRule(yr *yamlRule, index int) (*ast.Rule, error) {
	rule := &ast.Rule{
		Name:        yr.Name,
		Description: yr.Description,
		Enabled:     true, // Default to true
		Priority:    yr.Priority,
		Actions:     make([]*ast.Action, 0, len(yr.Actions)),
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1, // TODO: Extract from YAML node
			Column: 1,
		},
	}

	// Handle enabled flag (default is true)
	if yr.Enabled != nil {
		rule.Enabled = *yr.Enabled
	}

	// Build conditions
	if yr.Conditions != nil {
		cond, err := b.buildConditions(yr.Conditions)
		if err != nil {
			return nil, fmt.Errorf("invalid conditions: %w", err)
		}
		rule.Conditions = cond
	}

	// Build actions
	for i, ya := range yr.Actions {
		action, err := b.buildAction(ya)
		if err != nil {
			return nil, fmt.Errorf("invalid action at index %d: %w", i, err)
		}
		rule.Actions = append(rule.Actions, action)
	}

	return rule, nil
}

// buildConditions transforms condition YAML into an ast.ConditionNode.
// Conditions can be:
// - Single condition (map with field, operator, value)
// - Array of conditions (implicit AND)
// - Logical operator (all, any, not with array of children)
func (b *builder) buildConditions(cond interface{}) (*ast.ConditionNode, error) {
	switch v := cond.(type) {
	case map[string]interface{}:
		return b.buildConditionMap(v)
	case []interface{}:
		return b.buildConditionArray(v)
	default:
		return nil, fmt.Errorf("invalid condition type: %T", cond)
	}
}

// buildConditionMap builds a condition from a map.
func (b *builder) buildConditionMap(m map[string]interface{}) (*ast.ConditionNode, error) {
	// Check for logical operators (all, any, not)
	if children, ok := m["all"]; ok {
		return b.buildLogicalCondition(ast.ConditionTypeAll, children)
	}
	if children, ok := m["any"]; ok {
		return b.buildLogicalCondition(ast.ConditionTypeAny, children)
	}
	if children, ok := m["not"]; ok {
		return b.buildLogicalCondition(ast.ConditionTypeNot, children)
	}

	// Check for function call
	if fn, ok := m["function"]; ok {
		return b.buildFunctionCondition(fn, m)
	}

	// Simple condition
	return b.buildSimpleCondition(m)
}

// buildSimpleCondition builds a simple comparison condition.
func (b *builder) buildSimpleCondition(m map[string]interface{}) (*ast.ConditionNode, error) {
	field, ok := m["field"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'field'")
	}

	operatorStr, ok := m["operator"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'operator'")
	}

	value := m["value"]
	valueNode, err := b.buildValue(value)
	if err != nil {
		return nil, fmt.Errorf("invalid value: %w", err)
	}

	return &ast.ConditionNode{
		Type:     ast.ConditionTypeSimple,
		Field:    field,
		Operator: ast.Operator(operatorStr),
		Value:    valueNode,
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1,
			Column: 1,
		},
	}, nil
}

// buildLogicalCondition builds a logical operator condition (all/any/not).
func (b *builder) buildLogicalCondition(condType ast.ConditionType, children interface{}) (*ast.ConditionNode, error) {
	childArray, ok := children.([]interface{})
	if !ok {
		return nil, fmt.Errorf("logical operator must have array of children")
	}

	childNodes := make([]*ast.ConditionNode, 0, len(childArray))
	for i, child := range childArray {
		childNode, err := b.buildConditions(child)
		if err != nil {
			return nil, fmt.Errorf("invalid child condition at index %d: %w", i, err)
		}
		childNodes = append(childNodes, childNode)
	}

	return &ast.ConditionNode{
		Type:     condType,
		Children: childNodes,
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1,
			Column: 1,
		},
	}, nil
}

// buildFunctionCondition builds a function call condition.
func (b *builder) buildFunctionCondition(fn interface{}, m map[string]interface{}) (*ast.ConditionNode, error) {
	fnName, ok := fn.(string)
	if !ok {
		return nil, fmt.Errorf("function name must be a string")
	}

	// Extract arguments if present
	var args []*ast.ValueNode
	if argsRaw, ok := m["args"]; ok {
		argsArray, ok := argsRaw.([]interface{})
		if !ok {
			return nil, fmt.Errorf("function args must be an array")
		}

		args = make([]*ast.ValueNode, 0, len(argsArray))
		for i, arg := range argsArray {
			argNode, err := b.buildValue(arg)
			if err != nil {
				return nil, fmt.Errorf("invalid argument at index %d: %w", i, err)
			}
			args = append(args, argNode)
		}
	}

	return &ast.ConditionNode{
		Type:     ast.ConditionTypeFunction,
		Function: fnName,
		Args:     args,
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1,
			Column: 1,
		},
	}, nil
}

// buildConditionArray builds an implicit AND of conditions from an array.
func (b *builder) buildConditionArray(arr []interface{}) (*ast.ConditionNode, error) {
	if len(arr) == 0 {
		return nil, fmt.Errorf("empty condition array")
	}

	// Single condition - unwrap
	if len(arr) == 1 {
		return b.buildConditions(arr[0])
	}

	// Multiple conditions - implicit AND
	children := make([]*ast.ConditionNode, 0, len(arr))
	for i, cond := range arr {
		childNode, err := b.buildConditions(cond)
		if err != nil {
			return nil, fmt.Errorf("invalid condition at index %d: %w", i, err)
		}
		children = append(children, childNode)
	}

	return &ast.ConditionNode{
		Type:     ast.ConditionTypeAll,
		Children: children,
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1,
			Column: 1,
		},
	}, nil
}

// buildAction transforms an action map into an ast.Action.
func (b *builder) buildAction(m map[string]interface{}) (*ast.Action, error) {
	actionTypeStr, ok := m["type"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid action 'type'")
	}

	action := &ast.Action{
		Type:       ast.ActionType(actionTypeStr),
		Parameters: make(map[string]*ast.ValueNode),
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1,
			Column: 1,
		},
	}

	// Convert all other fields to parameters
	for key, value := range m {
		if key == "type" {
			continue
		}

		valueNode, err := b.buildValue(value)
		if err != nil {
			return nil, fmt.Errorf("invalid parameter %q: %w", key, err)
		}
		action.Parameters[key] = valueNode
	}

	return action, nil
}

// buildValue transforms a Go value into an ast.ValueNode.
func (b *builder) buildValue(value interface{}) (*ast.ValueNode, error) {
	if value == nil {
		return &ast.ValueNode{
			Type:  ast.ValueTypeNull,
			Value: nil,
			Location: ast.Location{
				File:   b.sourcePath,
				Line:   1,
				Column: 1,
			},
		}, nil
	}

	switch v := value.(type) {
	case string:
		// Check for variable reference
		if b.isVariableReference(v) {
			varName := b.extractVariableName(v)
			return &ast.ValueNode{
				Type:         ast.ValueTypeVariable,
				Value:        v,
				VariableName: varName,
				Location: ast.Location{
					File:   b.sourcePath,
					Line:   1,
					Column: 1,
				},
			}, nil
		}

		return &ast.ValueNode{
			Type:  ast.ValueTypeString,
			Value: v,
			Location: ast.Location{
				File:   b.sourcePath,
				Line:   1,
				Column: 1,
			},
		}, nil

	case int, int64, float64:
		// Convert all numbers to float64 for consistency
		var numVal float64
		switch n := v.(type) {
		case int:
			numVal = float64(n)
		case int64:
			numVal = float64(n)
		case float64:
			numVal = n
		}

		return &ast.ValueNode{
			Type:  ast.ValueTypeNumber,
			Value: numVal,
			Location: ast.Location{
				File:   b.sourcePath,
				Line:   1,
				Column: 1,
			},
		}, nil

	case bool:
		return &ast.ValueNode{
			Type:  ast.ValueTypeBoolean,
			Value: v,
			Location: ast.Location{
				File:   b.sourcePath,
				Line:   1,
				Column: 1,
			},
		}, nil

	case []interface{}:
		return &ast.ValueNode{
			Type:  ast.ValueTypeArray,
			Value: v,
			Location: ast.Location{
				File:   b.sourcePath,
				Line:   1,
				Column: 1,
			},
		}, nil

	case map[string]interface{}:
		return &ast.ValueNode{
			Type:  ast.ValueTypeObject,
			Value: v,
			Location: ast.Location{
				File:   b.sourcePath,
				Line:   1,
				Column: 1,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unsupported value type: %T", value)
	}
}

// isVariableReference checks if a string is a variable reference ({{ variables.name }}).
func (b *builder) isVariableReference(s string) bool {
	return len(s) > 4 && s[:2] == "{{" && s[len(s)-2:] == "}}"
}

// extractVariableName extracts the variable name from a reference string.
// Input: "{{ variables.max_tokens }}" -> Output: "max_tokens"
func (b *builder) extractVariableName(s string) string {
	// Remove {{ and }}
	s = s[2 : len(s)-2]
	// Trim spaces
	s = trimSpaces(s)
	// Remove "variables." prefix
	if len(s) > 10 && s[:10] == "variables." {
		return s[10:]
	}
	return s
}

// trimSpaces removes leading and trailing spaces.
func trimSpaces(s string) string {
	start := 0
	end := len(s)

	for start < end && s[start] == ' ' {
		start++
	}
	for end > start && s[end-1] == ' ' {
		end--
	}

	return s[start:end]
}

// buildTest transforms a yamlTest into an ast.PolicyTest.
func (b *builder) buildTest(yt *yamlTest, index int) (*ast.PolicyTest, error) {
	test := &ast.PolicyTest{
		Name:        yt.Name,
		Description: yt.Description,
		Request:     yt.Request,
		Expected: ast.TestExpectation{
			Action:      yt.Expected.Action,
			RuleMatches: yt.Expected.RuleMatches,
			Transforms:  yt.Expected.Transforms,
			Error:       yt.Expected.Error,
			ErrorMsg:    yt.Expected.ErrorMsg,
		},
		Location: ast.Location{
			File:   b.sourcePath,
			Line:   1, // TODO: Extract from YAML node
			Column: 1,
		},
	}

	// Validate required fields
	if test.Name == "" {
		return nil, fmt.Errorf("test name is required")
	}

	if test.Request == nil {
		return nil, fmt.Errorf("test request is required")
	}

	if test.Expected.Action == "" {
		return nil, fmt.Errorf("test expected action is required")
	}

	return test, nil
}
