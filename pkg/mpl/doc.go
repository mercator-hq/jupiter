// Package mpl provides parsing and validation for the Mercator Policy Language (MPL).
//
// MPL is a declarative YAML-based policy language for LLM governance. It enables
// security teams, compliance officers, and platform engineers to define rules
// that control LLM request/response behavior without writing code.
//
// # Architecture
//
// The package is organized into subpackages:
//
// - ast: Abstract Syntax Tree definitions for parsed policies
// - parser: YAML parsing and AST construction
// - validator: Policy validation (structural, semantic, action)
// - errors: Rich error types with location and suggestions
//
// # Basic Usage
//
// Parse and validate a policy:
//
//	import (
//	    "mercator-hq/jupiter/pkg/mpl/parser"
//	    "mercator-hq/jupiter/pkg/mpl/validator"
//	)
//
//	// Parse policy file
//	p := parser.NewParser()
//	policy, err := p.Parse("policies/example.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Validate policy
//	v := validator.NewValidator()
//	if err := v.Validate(policy); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Use policy
//	fmt.Println("Policy:", policy.Name)
//	fmt.Println("Rules:", len(policy.Rules))
//
// # Policy Structure
//
// An MPL policy consists of:
//
//	mpl_version: "1.0"
//	name: "my-policy"
//	version: "1.0.0"
//	description: "Policy description"
//
//	variables:
//	  max_tokens: 4000
//	  allowed_models: ["gpt-4", "claude-3-sonnet"]
//
//	rules:
//	  - name: "deny-high-risk"
//	    conditions:
//	      - field: "processing.risk_score"
//	        operator: ">"
//	        value: 7
//	    actions:
//	      - type: "deny"
//	        message: "Risk too high"
//
// # Validation
//
// The validator performs three types of checks:
//
// 1. Structural: Schema compliance, required fields, naming conventions
// 2. Semantic: Field references, type compatibility, variable usage
// 3. Action: Action parameters, types, conflicts
//
// # Error Handling
//
// Parsing and validation return rich errors with location and suggestions:
//
//	if err := validator.Validate(policy); err != nil {
//	    if errList, ok := err.(*errors.ErrorList); ok {
//	        for _, e := range errList.Errors {
//	            fmt.Println(e.Error())
//	        }
//	    }
//	}
//
// Error format:
//
//	[semantic] Undefined variable 'max_tokens'
//	  --> policies/example.yaml:15:20
//	  |
//	  15 |         value: "{{ variables.max_tokens }}"
//	     |                    ^^^^^^^^^^^^^^^^^^^^^^^
//	  |
//	  = suggestion: Define 'max_tokens' in the variables section
//
// # Policy Composition
//
// Load multiple policy files:
//
//	paths := []string{
//	    "policies/base.yaml",
//	    "policies/additional.yaml",
//	}
//	policy, err := parser.ParseMulti(paths)
//
// Or load from directory:
//
//	composer := parser.NewComposer(parser.NewParser())
//	policy, err := composer.ComposeFromDirectory("policies/*.yaml")
//
// # Performance
//
// The parser is optimized for production use:
// - Parse <100ms for typical policies (<1000 lines)
// - Parse <1s for large policies (10K lines)
// - Memory efficient (<10MB for large policies)
// - Thread-safe (concurrent parsing supported)
//
// # Example Policy
//
//	mpl_version: "1.0"
//	name: "cost-control"
//	version: "1.0.0"
//	description: "Enforce token limits and model allowlists"
//
//	variables:
//	  max_tokens: 4000
//	  allowed_models:
//	    - "gpt-4"
//	    - "gpt-3.5-turbo"
//	    - "claude-3-sonnet"
//
//	rules:
//	  - name: "enforce-token-limit"
//	    description: "Block requests exceeding token limit"
//	    conditions:
//	      - field: "processing.token_estimate.total_tokens"
//	        operator: ">"
//	        value: "{{ variables.max_tokens }}"
//	    actions:
//	      - type: "log"
//	        level: "warn"
//	        message: "Token limit exceeded"
//	      - type: "deny"
//	        message: "Request exceeds maximum {{ variables.max_tokens }} tokens"
//	        code: "token_limit_exceeded"
//
//	  - name: "model-allowlist"
//	    description: "Only allow approved models"
//	    conditions:
//	      - field: "request.model"
//	        operator: "not_in"
//	        value: "{{ variables.allowed_models }}"
//	    actions:
//	      - type: "deny"
//	        message: "Model not in allowlist"
//	        code: "model_not_allowed"
//
// For complete documentation, see:
// - docs/mpl/SPECIFICATION.md - Complete MPL language specification
// - docs/mpl/SYNTAX.md - Quick syntax reference
// - docs/mpl/BEST_PRACTICES.md - Authoring guidelines
// - docs/mpl/examples/ - 21 example policies
package mpl
