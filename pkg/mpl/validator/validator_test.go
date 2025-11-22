package validator

import (
	"testing"

	"mercator-hq/jupiter/pkg/mpl/ast"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
)

func TestStructuralValidator_ValidateMetadata(t *testing.T) {
	tests := []struct {
		name    string
		policy  *ast.Policy
		wantErr bool
		errType mplErrors.ErrorType
	}{
		{
			name: "valid metadata",
			policy: &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test-policy",
				Version:    "1.0.0",
				Rules: []*ast.Rule{{
					Name:       "rule1",
					Conditions: &ast.ConditionNode{Type: ast.ConditionTypeSimple, Field: "request.model", Operator: ast.OperatorEqual, Value: &ast.ValueNode{Type: ast.ValueTypeString, Value: "gpt-4"}},
					Actions:    []*ast.Action{{Type: ast.ActionTypeAllow}},
				}},
			},
			wantErr: false,
		},
		{
			name: "missing mpl_version",
			policy: &ast.Policy{
				Name:    "test-policy",
				Version: "1.0.0",
				Rules:   []*ast.Rule{{Name: "rule1", Actions: []*ast.Action{{Type: ast.ActionTypeAllow}}}},
			},
			wantErr: true,
			errType: mplErrors.ErrorTypeStructural,
		},
		{
			name: "missing name",
			policy: &ast.Policy{
				MPLVersion: "1.0",
				Version:    "1.0.0",
				Rules:      []*ast.Rule{{Name: "rule1", Actions: []*ast.Action{{Type: ast.ActionTypeAllow}}}},
			},
			wantErr: true,
			errType: mplErrors.ErrorTypeStructural,
		},
		{
			name: "invalid version format",
			policy: &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test-policy",
				Version:    "invalid",
				Rules:      []*ast.Rule{{Name: "rule1", Actions: []*ast.Action{{Type: ast.ActionTypeAllow}}}},
			},
			wantErr: true,
			errType: mplErrors.ErrorTypeStructural,
		},
		{
			name: "no rules",
			policy: &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test-policy",
				Version:    "1.0.0",
				Rules:      []*ast.Rule{},
			},
			wantErr: true,
			errType: mplErrors.ErrorTypeStructural,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewStructuralValidator()
			err := validator.Validate(tt.policy)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				errList, ok := err.(*mplErrors.ErrorList)
				if !ok {
					t.Fatalf("Expected ErrorList, got %T", err)
				}
				if !errList.HasErrorType(tt.errType) {
					t.Errorf("Expected error type %v, got errors: %v", tt.errType, errList.Errors)
				}
			}
		})
	}
}

func TestStructuralValidator_ValidateRules(t *testing.T) {
	tests := []struct {
		name    string
		policy  *ast.Policy
		wantErr bool
	}{
		{
			name: "duplicate rule names",
			policy: &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test-policy",
				Version:    "1.0.0",
				Rules: []*ast.Rule{
					{Name: "rule1", Conditions: &ast.ConditionNode{Type: ast.ConditionTypeSimple}, Actions: []*ast.Action{{Type: ast.ActionTypeAllow}}},
					{Name: "rule1", Conditions: &ast.ConditionNode{Type: ast.ConditionTypeSimple}, Actions: []*ast.Action{{Type: ast.ActionTypeAllow}}},
				},
			},
			wantErr: true,
		},
		{
			name: "rule missing name",
			policy: &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test-policy",
				Version:    "1.0.0",
				Rules: []*ast.Rule{
					{Name: "", Conditions: &ast.ConditionNode{Type: ast.ConditionTypeSimple}, Actions: []*ast.Action{{Type: ast.ActionTypeAllow}}},
				},
			},
			wantErr: true,
		},
		{
			name: "rule missing actions",
			policy: &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test-policy",
				Version:    "1.0.0",
				Rules: []*ast.Rule{
					{Name: "rule1", Conditions: &ast.ConditionNode{Type: ast.ConditionTypeSimple}, Actions: []*ast.Action{}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewStructuralValidator()
			err := validator.Validate(tt.policy)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSemanticValidator_ValidateFieldReferences(t *testing.T) {
	tests := []struct {
		name      string
		fieldName string
		fieldType ast.ValueType
		value     interface{}
		wantErr   bool
	}{
		{"valid field", "request.model", ast.ValueTypeString, "test", false},
		{"valid nested field", "processing.token_estimate.total_tokens", ast.ValueTypeNumber, float64(100), false},
		{"invalid field", "request.invalid_field", ast.ValueTypeString, "test", true},
		{"invalid namespace", "invalid.field", ast.ValueTypeString, "test", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test",
				Version:    "1.0.0",
				Rules: []*ast.Rule{
					{
						Name: "test-rule",
						Conditions: &ast.ConditionNode{
							Type:     ast.ConditionTypeSimple,
							Field:    tt.fieldName,
							Operator: ast.OperatorEqual,
							Value:    &ast.ValueNode{Type: tt.fieldType, Value: tt.value},
						},
						Actions: []*ast.Action{{Type: ast.ActionTypeAllow}},
					},
				},
			}

			validator := NewSemanticValidator()
			err := validator.Validate(policy)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSemanticValidator_ValidateVariableReferences(t *testing.T) {
	tests := []struct {
		name      string
		variables map[string]*ast.Variable
		varName   string
		wantErr   bool
	}{
		{
			name: "valid variable reference",
			variables: map[string]*ast.Variable{
				"max_tokens": {
					Name:  "max_tokens",
					Value: &ast.ValueNode{Type: ast.ValueTypeNumber, Value: float64(4000)},
					Type:  ast.ValueTypeNumber,
				},
			},
			varName: "max_tokens",
			wantErr: false,
		},
		{
			name:      "undefined variable",
			variables: map[string]*ast.Variable{},
			varName:   "undefined_var",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test",
				Version:    "1.0.0",
				Variables:  tt.variables,
				Rules: []*ast.Rule{
					{
						Name: "test-rule",
						Conditions: &ast.ConditionNode{
							Type:     ast.ConditionTypeSimple,
							Field:    "request.model",
							Operator: ast.OperatorEqual,
							Value: &ast.ValueNode{
								Type:         ast.ValueTypeVariable,
								VariableName: tt.varName,
							},
						},
						Actions: []*ast.Action{{Type: ast.ActionTypeAllow}},
					},
				},
			}

			validator := NewSemanticValidator()
			err := validator.Validate(policy)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActionValidator_ValidateDenyAction(t *testing.T) {
	tests := []struct {
		name        string
		action      *ast.Action
		wantErr     bool
		errContains string
	}{
		{
			name: "valid deny action",
			action: &ast.Action{
				Type: ast.ActionTypeDeny,
				Parameters: map[string]*ast.ValueNode{
					"message": {Type: ast.ValueTypeString, Value: "Access denied"},
					"code":    {Type: ast.ValueTypeString, Value: "access_denied"},
				},
			},
			wantErr: false,
		},
		{
			name: "deny missing message",
			action: &ast.Action{
				Type:       ast.ActionTypeDeny,
				Parameters: map[string]*ast.ValueNode{},
			},
			wantErr:     true,
			errContains: "missing required parameter 'message'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			policy := &ast.Policy{
				MPLVersion: "1.0",
				Name:       "test",
				Version:    "1.0.0",
				Rules: []*ast.Rule{
					{
						Name:       "test-rule",
						Conditions: &ast.ConditionNode{Type: ast.ConditionTypeSimple},
						Actions:    []*ast.Action{tt.action},
					},
				},
			}

			validator := NewActionValidator()
			err := validator.Validate(policy)

			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestActionValidator_DetectConflictingActions(t *testing.T) {
	policy := &ast.Policy{
		MPLVersion: "1.0",
		Name:       "test",
		Version:    "1.0.0",
		Rules: []*ast.Rule{
			{
				Name:       "conflicting-rule",
				Conditions: &ast.ConditionNode{Type: ast.ConditionTypeSimple},
				Actions: []*ast.Action{
					{Type: ast.ActionTypeAllow},
					{Type: ast.ActionTypeDeny, Parameters: map[string]*ast.ValueNode{
						"message": {Type: ast.ValueTypeString, Value: "Denied"},
					}},
				},
			},
		},
	}

	validator := NewActionValidator()
	err := validator.Validate(policy)

	if err == nil {
		t.Error("Expected error for conflicting allow/deny actions")
	}
}

func TestLookupField(t *testing.T) {
	tests := []struct {
		path      string
		wantFound bool
		wantType  ast.ValueType
	}{
		{"request.model", true, ast.ValueTypeString},
		{"request.max_tokens", true, ast.ValueTypeNumber},
		{"processing.risk_score", true, ast.ValueTypeNumber},
		{"context.environment", true, ast.ValueTypeString},
		{"invalid.field", false, ""},
		{"request.nonexistent", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			field, found := LookupField(tt.path)

			if found != tt.wantFound {
				t.Errorf("LookupField(%q) found = %v, want %v", tt.path, found, tt.wantFound)
				return
			}

			if found && field.Type != tt.wantType {
				t.Errorf("LookupField(%q) type = %v, want %v", tt.path, field.Type, tt.wantType)
			}
		})
	}
}

func TestGetAllFieldPaths(t *testing.T) {
	paths := GetAllFieldPaths()

	// Should have multiple field paths
	if len(paths) == 0 {
		t.Error("GetAllFieldPaths() returned empty list")
	}

	// Check for some expected fields
	expectedFields := []string{
		"request.model",
		"processing.risk_score",
		"context.environment",
	}

	for _, expected := range expectedFields {
		found := false
		for _, path := range paths {
			if path == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected field %q not found in field paths", expected)
		}
	}
}
