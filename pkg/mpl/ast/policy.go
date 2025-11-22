package ast

import "time"

// Policy represents the root AST node for an MPL policy.
// It contains metadata, variables, and rules that define the governance behavior.
type Policy struct {
	// Metadata
	MPLVersion  string    // MPL specification version (e.g., "1.0")
	Name        string    // Policy name (kebab-case)
	Version     string    // Policy version (semver)
	Description string    // Human-readable description
	Author      string    // Policy author
	Created     time.Time // Creation timestamp
	Updated     time.Time // Last update timestamp
	Tags        []string  // Tags for categorization

	// Content
	Variables map[string]*Variable // Variable definitions
	Rules     []*Rule              // Policy rules (evaluated in order)
	Includes  []string             // Paths to included policy files
	Tests     []*PolicyTest        // Policy test cases

	// Source tracking
	SourceFile string   // Path to the policy file
	Location   Location // Source location
}

// Variable represents a variable definition in an MPL policy.
// Variables enable reusable values that can be referenced in conditions and actions.
type Variable struct {
	Name     string     // Variable name
	Value    *ValueNode // Variable value
	Type     ValueType  // Variable type (inferred from value)
	Location Location   // Source location
}

// GetVariable returns the variable with the given name, or nil if not found.
func (p *Policy) GetVariable(name string) *Variable {
	return p.Variables[name]
}

// HasVariable returns true if the policy has a variable with the given name.
func (p *Policy) HasVariable(name string) bool {
	_, ok := p.Variables[name]
	return ok
}

// GetRule returns the rule with the given name, or nil if not found.
func (p *Policy) GetRule(name string) *Rule {
	for _, rule := range p.Rules {
		if rule.Name == name {
			return rule
		}
	}
	return nil
}

// HasRule returns true if the policy has a rule with the given name.
func (p *Policy) HasRule(name string) bool {
	return p.GetRule(name) != nil
}

// EnabledRules returns all enabled rules in the policy.
func (p *Policy) EnabledRules() []*Rule {
	var enabled []*Rule
	for _, rule := range p.Rules {
		if rule.IsEnabled() {
			enabled = append(enabled, rule)
		}
	}
	return enabled
}

// RuleCount returns the total number of rules in the policy.
func (p *Policy) RuleCount() int {
	return len(p.Rules)
}

// EnabledRuleCount returns the number of enabled rules in the policy.
func (p *Policy) EnabledRuleCount() int {
	return len(p.EnabledRules())
}

// PolicyTest represents a test case for validating policy behavior.
// Tests can be defined alongside policies to ensure correct evaluation.
type PolicyTest struct {
	Name        string                 // Test name
	Description string                 // Test description
	Request     map[string]interface{} // Mock request data for testing
	Expected    TestExpectation        // Expected outcome
	Location    Location               // Source location
}

// TestExpectation defines what outcome is expected from a policy test.
type TestExpectation struct {
	Action      string                 // Expected action: "allow", "deny", "transform", etc.
	RuleMatches []string               // Expected rules that should match (by name)
	Transforms  map[string]interface{} // Expected transformations (if action is "transform")
	Error       bool                   // Whether an error is expected
	ErrorMsg    string                 // Expected error message (if error is true)
}
