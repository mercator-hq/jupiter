// Package errors provides rich error types for MPL parsing and validation.
//
// The error types include source location, context, and suggestions to help
// users quickly identify and fix policy issues.
//
// # Error Types
//
// ErrorTypeSyntax: YAML syntax errors (malformed YAML)
//
// ErrorTypeStructural: Schema violations (missing required fields, invalid types)
//
// ErrorTypeSemantic: Semantic errors (undefined variables, type mismatches)
//
// ErrorTypeValidation: Action/condition validation errors
//
// ErrorTypeIO: File I/O errors
//
// # Basic Usage
//
// Create an error with location:
//
//	err := &errors.Error{
//	    Type:     errors.ErrorTypeSemantic,
//	    Message:  "Undefined variable 'max_tokens'",
//	    Location: varLocation,
//	}
//
// Add context from source file:
//
//	err = errors.AddContextToError(err)
//	fmt.Println(err.Error())
//
// Accumulate multiple errors:
//
//	errList := errors.NewErrorList()
//	errList.AddError(errors.ErrorTypeStructural, "Missing 'mpl_version'", location)
//	errList.AddError(errors.ErrorTypeSemantic, "Undefined variable", varLocation)
//
//	if errList.HasErrors() {
//	    return errList.ToError()
//	}
//
// # Error Format
//
// Errors are formatted with location, context, and suggestions:
//
//	[semantic] Undefined variable 'max_tokens'
//	  --> policies/example.yaml:15:20
//	  |
//	  15 |         value: "{{ variables.max_tokens }}"
//	     |                    ^^^^^^^^^^^^^^^^^^^^^^^
//	  |
//	  = suggestion: Define 'max_tokens' in the variables section
//
// # Context Extraction
//
// The package automatically extracts surrounding lines from the source file
// to show the error in context. This helps users quickly locate and fix issues.
//
// # Suggestions
//
// The suggestion generator uses Levenshtein distance to suggest similar names
// when users make typos in field names or action types:
//
//	suggestion := errors.SuggestFieldName("request.max_token",
//	    []string{"request.max_tokens", "request.model", "request.temperature"})
//	// Returns: "Did you mean 'request.max_tokens'?"
package errors
