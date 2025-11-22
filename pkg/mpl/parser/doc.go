// Package parser provides YAML parsing and AST construction for MPL policies.
//
// The parser reads MPL policy files (YAML format), validates syntax,
// and constructs Abstract Syntax Trees (AST) that can be validated
// and evaluated by the policy engine.
//
// # Basic Usage
//
// Parse a policy file:
//
//	parser := parser.NewParser()
//	policy, err := parser.Parse("policies/example.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fmt.Println("Loaded policy:", policy.Name)
//	fmt.Println("Rules:", len(policy.Rules))
//
// Parse from memory:
//
//	yamlData := []byte(`
//	mpl_version: "1.0"
//	name: "my-policy"
//	version: "1.0.0"
//	rules:
//	  - name: "deny-high-risk"
//	    conditions:
//	      - field: "processing.risk_score"
//	        operator: ">"
//	        value: 7
//	    actions:
//	      - type: "deny"
//	        message: "Risk too high"
//	`)
//
//	policy, err := parser.ParseBytes(yamlData, "memory://policy")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// Parse multiple files (composition):
//
//	paths := []string{
//	    "policies/base.yaml",
//	    "policies/additional-rules.yaml",
//	}
//	policy, err := parser.ParseMulti(paths)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Configuration
//
// Configure parser limits:
//
//	parser := parser.NewParser().
//	    WithMaxFileSize(5 * 1024 * 1024).  // 5MB limit
//	    WithMaxDepth(15).                   // Max nesting depth
//	    WithStrictMode(true)                // Warnings become errors
//
// # Error Handling
//
// The parser returns rich errors with location and context:
//
//	policy, err := parser.Parse("policy.yaml")
//	if err != nil {
//	    if errList, ok := err.(*errors.ErrorList); ok {
//	        fmt.Printf("Found %d errors:\n", errList.Count())
//	        for _, e := range errList.Errors {
//	            fmt.Println(e.Error())
//	        }
//	    } else {
//	        fmt.Println(err)
//	    }
//	}
//
// # Parsing Stages
//
// The parser operates in two stages:
//
// 1. YAML Parsing: Read YAML and construct intermediate structures
//
// 2. AST Building: Transform intermediate structures to typed AST nodes
//
// This two-stage approach enables:
// - Better error messages (preserve YAML line numbers)
// - Type-safe AST (strongly typed Go structs)
// - Validation during construction (fail fast)
//
// # Performance
//
// The parser is designed for production use:
// - Parse <100ms for typical policies (<1000 lines)
// - Parse <1s for large policies (10K lines)
// - Memory efficient (<10MB for large policies)
// - Thread-safe (concurrent parsing supported)
package parser
