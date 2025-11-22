// Package ast provides Abstract Syntax Tree (AST) definitions for the Mercator Policy Language (MPL).
//
// The AST represents the parsed structure of an MPL policy, enabling validation,
// transformation, and evaluation. All AST nodes preserve source location information
// for precise error reporting.
//
// # Core Types
//
// Policy: Root AST node containing metadata, variables, and rules
//
// Rule: Individual policy rule with conditions and actions
//
// ConditionNode: Condition expression (simple, logical, or function)
//
// Action: Policy action (allow, deny, log, redact, modify, route, alert, rate_limit, budget)
//
// ValueNode: Generic value (string, number, boolean, array, object, variable reference, null)
//
// Variable: Variable definition with name and value
//
// Location: Source location (file, line, column)
//
// # Basic Usage
//
// Parse a policy and traverse the AST:
//
//	policy, err := parser.Parse("policy.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Access policy metadata
//	fmt.Println("Policy:", policy.Name, "version:", policy.Version)
//
//	// Iterate over rules
//	for _, rule := range policy.Rules {
//	    fmt.Println("Rule:", rule.Name)
//	    if rule.Conditions != nil {
//	        fmt.Println("  Conditions:", rule.Conditions.Type)
//	    }
//	    for _, action := range rule.Actions {
//	        fmt.Println("  Action:", action.Type)
//	    }
//	}
//
// Use the visitor pattern for AST traversal:
//
//	type MyVisitor struct{}
//
//	func (v *MyVisitor) VisitPolicy(p *ast.Policy) error {
//	    fmt.Println("Visiting policy:", p.Name)
//	    return nil
//	}
//
//	// Implement other visitor methods...
//
//	visitor := &MyVisitor{}
//	if err := ast.Walk(policy, visitor); err != nil {
//	    log.Fatal(err)
//	}
//
// # AST Structure
//
// The AST mirrors the MPL YAML structure:
//
//	Policy
//	├── Metadata (name, version, description, etc.)
//	├── Variables (map[string]*Variable)
//	└── Rules ([]*Rule)
//	    ├── Conditions (*ConditionNode)
//	    │   ├── Simple (field, operator, value)
//	    │   ├── Logical (all/any/not with children)
//	    │   └── Function (function name with arguments)
//	    └── Actions ([]*Action)
//	        └── Parameters (map[string]*ValueNode)
//
// # Source Locations
//
// All AST nodes include a Location field for error reporting:
//
//	if rule.Conditions == nil {
//	    return fmt.Errorf("%s: rule %q has no conditions",
//	        rule.Location, rule.Name)
//	}
//
// # Immutability
//
// AST nodes should be treated as immutable after construction.
// The parser builds the AST once and validators inspect it without modification.
package ast
