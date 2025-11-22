// Package validator provides validation for MPL policies.
//
// The validator performs three types of validation:
//
// 1. Structural Validation: Checks schema compliance, required fields, naming conventions
//
// 2. Semantic Validation: Validates field references, type compatibility, variable usage
//
// 3. Action Validation: Validates action parameters, types, and detects conflicts
//
// # Basic Usage
//
// Validate a parsed policy:
//
//	policy, err := parser.Parse("policy.yaml")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	validator := validator.NewValidator()
//	if err := validator.Validate(policy); err != nil {
//	    if errList, ok := err.(*errors.ErrorList); ok {
//	        for _, e := range errList.Errors {
//	            fmt.Println(e.Error())
//	        }
//	    }
//	    log.Fatal(err)
//	}
//
// Run specific validation passes:
//
//	validator := validator.NewValidator()
//
//	// Only structural validation
//	if err := validator.ValidateStructural(policy); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Only semantic validation
//	if err := validator.ValidateSemantic(policy); err != nil {
//	    log.Fatal(err)
//	}
//
//	// Only action validation
//	if err := validator.ValidateActions(policy); err != nil {
//	    log.Fatal(err)
//	}
//
// # Validation Passes
//
// Structural Validation checks:
// - Required fields (mpl_version, name, version, rules)
// - Field types (strings, numbers, booleans, arrays)
// - Naming conventions (kebab-case names, semver versions)
// - Rule uniqueness (no duplicate rule names)
// - Condition structure (max nesting depth, required fields)
// - Action structure (recognized action types)
//
// Semantic Validation checks:
// - Field references (field exists in data model)
// - Operator compatibility (operator valid for field type)
// - Type compatibility (value type matches field type)
// - Variable references (variable defined before use)
// - Circular variable references
// - Function signatures (correct argument count and types)
//
// Action Validation checks:
// - Required parameters (each action type has different requirements)
// - Parameter types (string, number, boolean, array)
// - Parameter values (enums, ranges)
// - Conflicting actions (allow + deny in same rule)
//
// # Data Model
//
// The validator validates field references against the MPL data model:
//
//	request.*         - LLM request fields (model, temperature, max_tokens, etc.)
//	response.*        - LLM response fields (content, usage, finish_reason)
//	processing.*      - Processing metadata (risk_score, token_estimate, content_analysis)
//	context.*         - Request context (environment, time, user_attributes)
//
// Lookup a field:
//
//	field, ok := validator.LookupField("request.model")
//	if !ok {
//	    log.Fatal("Field not found")
//	}
//	fmt.Println("Field type:", field.Type)
//
// Get all valid field paths (for suggestions):
//
//	allFields := validator.GetAllFieldPaths()
//	fmt.Println("Valid fields:", allFields)
//
// # Error Handling
//
// The validator returns rich errors with location and suggestions:
//
//	if err := validator.Validate(policy); err != nil {
//	    if errList, ok := err.(*errors.ErrorList); ok {
//	        fmt.Printf("Found %d errors:\n", errList.Count())
//	        for _, e := range errList.Errors {
//	            // Each error has: Type, Message, Location, Context, Suggestion
//	            fmt.Printf("[%s] %s at %s\n", e.Type, e.Message, e.Location)
//	            if e.Suggestion != "" {
//	                fmt.Printf("  Suggestion: %s\n", e.Suggestion)
//	            }
//	        }
//	    }
//	}
//
// # Validation Order
//
// Validations run in sequence:
// 1. Structural validation (fail fast on schema errors)
// 2. Semantic validation (only if structural passed)
// 3. Action validation (only if structural passed)
//
// This prevents cascading errors and provides clearer error messages.
package validator
