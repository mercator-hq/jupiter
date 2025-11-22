package errors

import (
	"fmt"
	"strings"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// ErrorType categorizes the type of error encountered during parsing or validation.
type ErrorType string

const (
	ErrorTypeSyntax     ErrorType = "syntax"     // YAML syntax error
	ErrorTypeStructural ErrorType = "structural" // Schema violation (missing/invalid fields)
	ErrorTypeSemantic   ErrorType = "semantic"   // Undefined reference, type mismatch
	ErrorTypeValidation ErrorType = "validation" // Action/condition validation error
	ErrorTypeIO         ErrorType = "io"         // File I/O error
)

// Error represents a rich error with location, context, and suggestions.
// It provides detailed information for debugging policy issues.
type Error struct {
	Type       ErrorType    // Category of error
	Message    string       // Error message
	Location   ast.Location // Source location (file, line, column)
	Context    string       // Surrounding lines of code
	Suggestion string       // Suggested fix (optional)
}

// Error implements the error interface.
// It returns a formatted error message with location and context.
func (e *Error) Error() string {
	var sb strings.Builder

	// Error type and message
	sb.WriteString(fmt.Sprintf("[%s] %s\n", e.Type, e.Message))

	// Location
	if e.Location.IsValid() {
		sb.WriteString(fmt.Sprintf("  --> %s\n", e.Location.String()))
	}

	// Context (surrounding code)
	if e.Context != "" {
		sb.WriteString("  |\n")
		sb.WriteString(e.Context)
		sb.WriteString("  |\n")
	}

	// Suggestion
	if e.Suggestion != "" {
		sb.WriteString(fmt.Sprintf("  = suggestion: %s\n", e.Suggestion))
	}

	return sb.String()
}

// ErrorList represents a collection of errors encountered during parsing/validation.
// It allows accumulating multiple errors instead of failing on the first error.
type ErrorList struct {
	Errors []*Error
}

// NewErrorList creates a new empty error list.
func NewErrorList() *ErrorList {
	return &ErrorList{
		Errors: make([]*Error, 0),
	}
}

// Add appends an error to the list.
func (el *ErrorList) Add(err *Error) {
	el.Errors = append(el.Errors, err)
}

// AddError creates and adds a new error with the given parameters.
func (el *ErrorList) AddError(errType ErrorType, message string, location ast.Location) {
	el.Add(&Error{
		Type:     errType,
		Message:  message,
		Location: location,
	})
}

// AddErrorWithSuggestion creates and adds a new error with a suggestion.
func (el *ErrorList) AddErrorWithSuggestion(errType ErrorType, message string, location ast.Location, suggestion string) {
	el.Add(&Error{
		Type:       errType,
		Message:    message,
		Location:   location,
		Suggestion: suggestion,
	})
}

// HasErrors returns true if the error list contains any errors.
func (el *ErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

// Count returns the number of errors in the list.
func (el *ErrorList) Count() int {
	return len(el.Errors)
}

// Error implements the error interface.
// It returns all errors formatted as a single string.
func (el *ErrorList) Error() string {
	if !el.HasErrors() {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d error(s):\n\n", el.Count()))

	for i, err := range el.Errors {
		sb.WriteString(fmt.Sprintf("Error %d:\n", i+1))
		sb.WriteString(err.Error())
		sb.WriteString("\n")
	}

	return sb.String()
}

// ToError returns nil if the error list is empty, otherwise returns the error list itself.
func (el *ErrorList) ToError() error {
	if !el.HasErrors() {
		return nil
	}
	return el
}

// ByType returns all errors of the given type.
func (el *ErrorList) ByType(errType ErrorType) []*Error {
	var result []*Error
	for _, err := range el.Errors {
		if err.Type == errType {
			result = append(result, err)
		}
	}
	return result
}

// HasErrorType returns true if the error list contains at least one error of the given type.
func (el *ErrorList) HasErrorType(errType ErrorType) bool {
	for _, err := range el.Errors {
		if err.Type == errType {
			return true
		}
	}
	return false
}
