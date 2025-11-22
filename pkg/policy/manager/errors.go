package manager

import (
	"fmt"
	"strings"
)

// LoadError represents an error that occurred during policy loading.
// This includes file system errors like "file not found", "permission denied",
// or errors related to file size limits or encoding validation.
type LoadError struct {
	// FilePath is the path to the file that failed to load
	FilePath string

	// Message describes the error
	Message string

	// Cause is the underlying error that caused this load error
	Cause error
}

// Error implements the error interface.
func (e *LoadError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("failed to load policy file %q: %s: %v", e.FilePath, e.Message, e.Cause)
	}
	return fmt.Sprintf("failed to load policy file %q: %s", e.FilePath, e.Message)
}

// Unwrap implements the errors.Unwrap interface for error chain support.
func (e *LoadError) Unwrap() error {
	return e.Cause
}

// ParseError represents an error that occurred during YAML parsing.
// It includes line and column information for precise error reporting.
type ParseError struct {
	// FilePath is the path to the file that failed to parse
	FilePath string

	// Line is the line number where the error occurred (1-indexed)
	Line int

	// Column is the column number where the error occurred (1-indexed)
	Column int

	// Message describes the parsing error
	Message string

	// Cause is the underlying parser error
	Cause error
}

// Error implements the error interface.
func (e *ParseError) Error() string {
	if e.Line > 0 && e.Column > 0 {
		return fmt.Sprintf("parse error in %q at line %d, column %d: %s", e.FilePath, e.Line, e.Column, e.Message)
	}
	if e.Line > 0 {
		return fmt.Sprintf("parse error in %q at line %d: %s", e.FilePath, e.Line, e.Message)
	}
	return fmt.Sprintf("parse error in %q: %s", e.FilePath, e.Message)
}

// Unwrap implements the errors.Unwrap interface for error chain support.
func (e *ParseError) Unwrap() error {
	return e.Cause
}

// ValidationError represents an error that occurred during policy validation.
// This includes semantic errors, rule conflicts, and invalid policy constructs.
type ValidationError struct {
	// PolicyID is the ID of the policy that failed validation
	PolicyID string

	// RuleID is the ID of the rule that failed validation (if applicable)
	RuleID string

	// FieldPath is the path to the field that failed validation (e.g., "rules[0].conditions")
	FieldPath string

	// Message describes the validation error
	Message string

	// Cause is the underlying validation error
	Cause error
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	parts := []string{"validation error"}

	if e.PolicyID != "" {
		parts = append(parts, fmt.Sprintf("in policy %q", e.PolicyID))
	}

	if e.RuleID != "" {
		parts = append(parts, fmt.Sprintf("in rule %q", e.RuleID))
	}

	if e.FieldPath != "" {
		parts = append(parts, fmt.Sprintf("at %s", e.FieldPath))
	}

	parts = append(parts, e.Message)

	return strings.Join(parts, " ")
}

// Unwrap implements the errors.Unwrap interface for error chain support.
func (e *ValidationError) Unwrap() error {
	return e.Cause
}

// IncludeError represents an error that occurred during include resolution.
// This includes missing include files, circular dependencies, and depth limit violations.
type IncludeError struct {
	// FilePath is the path to the file containing the include
	FilePath string

	// IncludePath is the path that was being included
	IncludePath string

	// Cycle contains the list of files in a circular dependency (if applicable)
	Cycle []string

	// Message describes the include error
	Message string

	// Cause is the underlying error
	Cause error
}

// Error implements the error interface.
func (e *IncludeError) Error() string {
	if len(e.Cycle) > 0 {
		cycle := strings.Join(e.Cycle, " -> ")
		return fmt.Sprintf("circular include detected in %q: %s", e.FilePath, cycle)
	}

	if e.IncludePath != "" {
		return fmt.Sprintf("include error in %q: failed to include %q: %s", e.FilePath, e.IncludePath, e.Message)
	}

	return fmt.Sprintf("include error in %q: %s", e.FilePath, e.Message)
}

// Unwrap implements the errors.Unwrap interface for error chain support.
func (e *IncludeError) Unwrap() error {
	return e.Cause
}

// RegistryError represents an error that occurred during registry operations.
// This includes policy registration failures and duplicate policy IDs.
type RegistryError struct {
	// PolicyID is the ID of the policy involved in the error
	PolicyID string

	// Operation is the operation that failed (e.g., "register", "unregister")
	Operation string

	// Message describes the registry error
	Message string

	// Cause is the underlying error
	Cause error
}

// Error implements the error interface.
func (e *RegistryError) Error() string {
	if e.PolicyID != "" {
		return fmt.Sprintf("registry error for policy %q during %s: %s", e.PolicyID, e.Operation, e.Message)
	}
	return fmt.Sprintf("registry error during %s: %s", e.Operation, e.Message)
}

// Unwrap implements the errors.Unwrap interface for error chain support.
func (e *RegistryError) Unwrap() error {
	return e.Cause
}

// ErrorList contains multiple errors that occurred during policy operations.
// This is used when loading multiple policies where some may succeed and others fail.
type ErrorList struct {
	Errors []error
}

// Error implements the error interface.
func (e *ErrorList) Error() string {
	if len(e.Errors) == 0 {
		return "no errors"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%d errors occurred:\n", len(e.Errors)))
	for i, err := range e.Errors {
		sb.WriteString(fmt.Sprintf("  %d. %v\n", i+1, err))
	}
	return sb.String()
}

// Add adds an error to the list.
func (e *ErrorList) Add(err error) {
	if err != nil {
		e.Errors = append(e.Errors, err)
	}
}

// HasErrors returns true if the list contains any errors.
func (e *ErrorList) HasErrors() bool {
	return len(e.Errors) > 0
}

// ToError returns nil if there are no errors, the single error if there is one,
// or the ErrorList itself if there are multiple errors.
func (e *ErrorList) ToError() error {
	if len(e.Errors) == 0 {
		return nil
	}
	if len(e.Errors) == 1 {
		return e.Errors[0]
	}
	return e
}
