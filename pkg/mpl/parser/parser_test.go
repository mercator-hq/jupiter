package parser

import (
	"testing"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

func TestParser_Parse_Simple(t *testing.T) {
	parser := NewParser()
	policy, err := parser.Parse("../../../internal/mpl/testdata/valid/simple.yaml")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Validate basic metadata
	if policy.MPLVersion != "1.0" {
		t.Errorf("MPLVersion = %q, want %q", policy.MPLVersion, "1.0")
	}
	if policy.Name != "simple-policy" {
		t.Errorf("Name = %q, want %q", policy.Name, "simple-policy")
	}
	if policy.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", policy.Version, "1.0.0")
	}

	// Validate rules
	if len(policy.Rules) != 1 {
		t.Fatalf("len(Rules) = %d, want 1", len(policy.Rules))
	}

	rule := policy.Rules[0]
	if rule.Name != "deny-high-risk" {
		t.Errorf("Rule.Name = %q, want %q", rule.Name, "deny-high-risk")
	}

	// Validate conditions
	if rule.Conditions == nil {
		t.Fatal("Rule has no conditions")
	}
	if rule.Conditions.Type != ast.ConditionTypeSimple {
		t.Errorf("Condition type = %q, want %q", rule.Conditions.Type, ast.ConditionTypeSimple)
	}
	if rule.Conditions.Field != "processing.risk_score" {
		t.Errorf("Condition field = %q, want %q", rule.Conditions.Field, "processing.risk_score")
	}
	if rule.Conditions.Operator != ast.OperatorGreaterThan {
		t.Errorf("Condition operator = %q, want %q", rule.Conditions.Operator, ast.OperatorGreaterThan)
	}

	// Validate actions
	if len(rule.Actions) != 1 {
		t.Fatalf("len(Actions) = %d, want 1", len(rule.Actions))
	}
	action := rule.Actions[0]
	if action.Type != ast.ActionTypeDeny {
		t.Errorf("Action type = %q, want %q", action.Type, ast.ActionTypeDeny)
	}

	// Validate action parameters
	if !action.HasParameter("message") {
		t.Error("Action missing 'message' parameter")
	}
	if msg := action.GetStringParameter("message"); msg != "Risk score too high" {
		t.Errorf("Action message = %q, want %q", msg, "Risk score too high")
	}
}

func TestParser_Parse_WithVariables(t *testing.T) {
	parser := NewParser()
	policy, err := parser.Parse("../../../internal/mpl/testdata/valid/with-variables.yaml")
	if err != nil {
		t.Fatalf("Parse() failed: %v", err)
	}

	// Validate variables
	if len(policy.Variables) != 3 {
		t.Errorf("len(Variables) = %d, want 3", len(policy.Variables))
	}

	// Check max_tokens variable
	if !policy.HasVariable("max_tokens") {
		t.Error("Missing variable 'max_tokens'")
	}
	maxTokens := policy.GetVariable("max_tokens")
	if maxTokens == nil {
		t.Fatal("max_tokens variable is nil")
	}
	if maxTokens.Type != ast.ValueTypeNumber {
		t.Errorf("max_tokens type = %q, want %q", maxTokens.Type, ast.ValueTypeNumber)
	}

	// Check allowed_models variable
	if !policy.HasVariable("allowed_models") {
		t.Error("Missing variable 'allowed_models'")
	}
	allowedModels := policy.GetVariable("allowed_models")
	if allowedModels == nil {
		t.Fatal("allowed_models variable is nil")
	}
	if allowedModels.Type != ast.ValueTypeArray {
		t.Errorf("allowed_models type = %q, want %q", allowedModels.Type, ast.ValueTypeArray)
	}

	// Validate rules use variables
	if len(policy.Rules) != 2 {
		t.Fatalf("len(Rules) = %d, want 2", len(policy.Rules))
	}

	// Check first rule uses variable
	rule1 := policy.Rules[0]
	if rule1.Conditions.Value.Type != ast.ValueTypeVariable {
		t.Error("Rule condition should use variable reference")
	}
	if rule1.Conditions.Value.VariableName != "max_tokens" {
		t.Errorf("Variable name = %q, want %q", rule1.Conditions.Value.VariableName, "max_tokens")
	}
}

func TestParser_ParseBytes(t *testing.T) {
	yaml := []byte(`
mpl_version: "1.0"
name: "test-policy"
version: "1.0.0"

rules:
  - name: "test-rule"
    conditions:
      - field: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: "allow"
`)

	parser := NewParser()
	policy, err := parser.ParseBytes(yaml, "memory://test")
	if err != nil {
		t.Fatalf("ParseBytes() failed: %v", err)
	}

	if policy.Name != "test-policy" {
		t.Errorf("Name = %q, want %q", policy.Name, "test-policy")
	}
	if len(policy.Rules) != 1 {
		t.Errorf("len(Rules) = %d, want 1", len(policy.Rules))
	}
}

func TestParser_Parse_InvalidYAML(t *testing.T) {
	yaml := []byte(`
invalid: yaml: syntax:
  - this is not valid
`)

	parser := NewParser()
	_, err := parser.ParseBytes(yaml, "memory://invalid")
	if err == nil {
		t.Error("ParseBytes() should fail on invalid YAML")
	}
}

func TestParser_Parse_MissingFile(t *testing.T) {
	parser := NewParser()
	_, err := parser.Parse("nonexistent.yaml")
	if err == nil {
		t.Error("Parse() should fail on missing file")
	}
}

func TestParser_WithMaxFileSize(t *testing.T) {
	parser := NewParser().WithMaxFileSize(100) // Very small limit

	largeYAML := make([]byte, 200)
	for i := range largeYAML {
		largeYAML[i] = 'a'
	}

	_, err := parser.ParseBytes(largeYAML, "memory://large")
	if err == nil {
		t.Error("ParseBytes() should fail when file exceeds size limit")
	}
}

func TestParser_ParseMulti(t *testing.T) {
	parser := NewParser()
	paths := []string{
		"../../../internal/mpl/testdata/valid/simple.yaml",
		"../../../internal/mpl/testdata/valid/with-variables.yaml",
	}

	policy, err := parser.ParseMulti(paths)
	if err != nil {
		t.Fatalf("ParseMulti() failed: %v", err)
	}

	// Should have combined rules from both files
	if len(policy.Rules) != 3 { // 1 from simple + 2 from with-variables
		t.Errorf("len(Rules) = %d, want 3", len(policy.Rules))
	}

	// Should have variables from second file
	if len(policy.Variables) != 3 {
		t.Errorf("len(Variables) = %d, want 3", len(policy.Variables))
	}

	// Metadata should come from first file
	if policy.Name != "simple-policy" {
		t.Errorf("Name = %q, want %q (from first file)", policy.Name, "simple-policy")
	}
}

func TestBuilder_buildConditionArray(t *testing.T) {
	builder := newBuilder("test.yaml")

	// Test multiple conditions (implicit AND)
	conditions := []interface{}{
		map[string]interface{}{
			"field":    "request.model",
			"operator": "==",
			"value":    "gpt-4",
		},
		map[string]interface{}{
			"field":    "processing.risk_score",
			"operator": "<",
			"value":    5,
		},
	}

	cond, err := builder.buildConditionArray(conditions)
	if err != nil {
		t.Fatalf("buildConditionArray() failed: %v", err)
	}

	// Should create implicit AND
	if cond.Type != ast.ConditionTypeAll {
		t.Errorf("Condition type = %q, want %q", cond.Type, ast.ConditionTypeAll)
	}
	if len(cond.Children) != 2 {
		t.Errorf("len(Children) = %d, want 2", len(cond.Children))
	}
}

func TestBuilder_buildLogicalCondition(t *testing.T) {
	builder := newBuilder("test.yaml")

	children := []interface{}{
		map[string]interface{}{
			"field":    "request.model",
			"operator": "==",
			"value":    "gpt-4",
		},
	}

	cond, err := builder.buildLogicalCondition(ast.ConditionTypeAny, children)
	if err != nil {
		t.Fatalf("buildLogicalCondition() failed: %v", err)
	}

	if cond.Type != ast.ConditionTypeAny {
		t.Errorf("Condition type = %q, want %q", cond.Type, ast.ConditionTypeAny)
	}
	if len(cond.Children) != 1 {
		t.Errorf("len(Children) = %d, want 1", len(cond.Children))
	}
}

func TestBuilder_isVariableReference(t *testing.T) {
	builder := newBuilder("test.yaml")

	tests := []struct {
		input string
		want  bool
	}{
		{"{{ variables.max_tokens }}", true},
		{"{{ variables.name }}", true},
		{"normal string", false},
		{"{{incomplete", false},
		{"incomplete}}", false},
		{"{{ }}", true}, // Edge case - empty but valid syntax
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := builder.isVariableReference(tt.input)
			if got != tt.want {
				t.Errorf("isVariableReference(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestBuilder_extractVariableName(t *testing.T) {
	builder := newBuilder("test.yaml")

	tests := []struct {
		input string
		want  string
	}{
		{"{{ variables.max_tokens }}", "max_tokens"},
		{"{{variables.name}}", "name"},
		{"{{ variables.foo_bar }}", "foo_bar"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := builder.extractVariableName(tt.input)
			if got != tt.want {
				t.Errorf("extractVariableName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
