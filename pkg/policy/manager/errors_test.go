package manager

import (
	"errors"
	"strings"
	"testing"
)

func TestLoadError(t *testing.T) {
	path := "/test/path.yaml"
	cause := errors.New("file not found")

	err := &LoadError{
		FilePath: path,
		Message:  "failed to load",
		Cause:    cause,
	}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, path) {
		t.Errorf("Error() = %q, want to contain path %q", errMsg, path)
	}
	if !strings.Contains(errMsg, "failed to load") {
		t.Errorf("Error() = %q, want to contain message", errMsg)
	}

	// Test Unwrap() method
	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}

	// Test errors.Is()
	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true")
	}
}

func TestParseError(t *testing.T) {
	path := "/test/policy.yaml"
	line := 42
	cause := errors.New("invalid YAML")

	err := &ParseError{
		FilePath: path,
		Line:     line,
		Message:  "syntax error",
		Cause:    cause,
	}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, path) {
		t.Errorf("Error() = %q, want to contain path %q", errMsg, path)
	}
	if !strings.Contains(errMsg, "42") {
		t.Errorf("Error() = %q, want to contain line number", errMsg)
	}
	if !strings.Contains(errMsg, "syntax error") {
		t.Errorf("Error() = %q, want to contain message", errMsg)
	}

	// Test Unwrap() method
	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestValidationError(t *testing.T) {
	policyID := "test-policy"
	cause := errors.New("missing required field")

	err := &ValidationError{
		PolicyID: policyID,
		Message:  "validation failed",
		Cause:    cause,
	}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, policyID) {
		t.Errorf("Error() = %q, want to contain policy ID %q", errMsg, policyID)
	}
	if !strings.Contains(errMsg, "validation failed") {
		t.Errorf("Error() = %q, want to contain message", errMsg)
	}

	// Test Unwrap() method
	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestIncludeError(t *testing.T) {
	path := "/test/base.yaml"
	includePath := "/test/included.yaml"
	cause := errors.New("circular dependency")

	err := &IncludeError{
		FilePath:    path,
		IncludePath: includePath,
		Message:     "include resolution failed",
		Cause:       cause,
	}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, path) {
		t.Errorf("Error() = %q, want to contain path %q", errMsg, path)
	}
	if !strings.Contains(errMsg, includePath) {
		t.Errorf("Error() = %q, want to contain include path %q", errMsg, includePath)
	}
	if !strings.Contains(errMsg, "include resolution failed") {
		t.Errorf("Error() = %q, want to contain message", errMsg)
	}

	// Test Unwrap() method
	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestRegistryError(t *testing.T) {
	policyID := "test-policy"
	cause := errors.New("policy not found")

	err := &RegistryError{
		PolicyID: policyID,
		Message:  "registry operation failed",
		Cause:    cause,
	}

	// Test Error() method
	errMsg := err.Error()
	if !strings.Contains(errMsg, policyID) {
		t.Errorf("Error() = %q, want to contain policy ID %q", errMsg, policyID)
	}
	if !strings.Contains(errMsg, "registry operation failed") {
		t.Errorf("Error() = %q, want to contain message", errMsg)
	}

	// Test Unwrap() method
	if unwrapped := errors.Unwrap(err); unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestErrorList_Add(t *testing.T) {
	errList := &ErrorList{}

	// Initially empty
	if errList.HasErrors() {
		t.Error("HasErrors() = true for new ErrorList, want false")
	}
	if len(errList.Errors) != 0 {
		t.Errorf("len(Errors) = %d for new ErrorList, want 0", len(errList.Errors))
	}

	// Add first error
	err1 := &LoadError{FilePath: "/test/1.yaml", Message: "error 1"}
	errList.Add(err1)

	if !errList.HasErrors() {
		t.Error("HasErrors() = false after adding error, want true")
	}
	if len(errList.Errors) != 1 {
		t.Errorf("len(Errors) = %d after adding one error, want 1", len(errList.Errors))
	}

	// Add second error
	err2 := &ParseError{FilePath: "/test/2.yaml", Message: "error 2"}
	errList.Add(err2)

	if len(errList.Errors) != 2 {
		t.Errorf("len(Errors) = %d after adding two errors, want 2", len(errList.Errors))
	}

	// Verify order is preserved
	if errList.Errors[0] != err1 {
		t.Error("First error not preserved in order")
	}
	if errList.Errors[1] != err2 {
		t.Error("Second error not preserved in order")
	}
}

func TestErrorList_ToError(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() *ErrorList
		wantNil   bool
		wantCount int
	}{
		{
			name: "empty list returns nil",
			setup: func() *ErrorList {
				return &ErrorList{}
			},
			wantNil:   true,
			wantCount: 0,
		},
		{
			name: "single error",
			setup: func() *ErrorList {
				errList := &ErrorList{}
				errList.Add(&LoadError{FilePath: "/test/1.yaml", Message: "error 1"})
				return errList
			},
			wantNil:   false,
			wantCount: 1,
		},
		{
			name: "multiple errors",
			setup: func() *ErrorList {
				errList := &ErrorList{}
				errList.Add(&LoadError{FilePath: "/test/1.yaml", Message: "error 1"})
				errList.Add(&ParseError{FilePath: "/test/2.yaml", Message: "error 2"})
				errList.Add(&ValidationError{PolicyID: "policy-3", Message: "error 3"})
				return errList
			},
			wantNil:   false,
			wantCount: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errList := tt.setup()
			err := errList.ToError()

			if tt.wantNil {
				if err != nil {
					t.Errorf("ToError() = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatal("ToError() = nil, want error")
			}

			// Verify error message
			errMsg := err.Error()

			// For multiple errors, should contain "occurred"
			// For single error, returns the error directly
			if len(errList.Errors) > 1 {
				if !strings.Contains(errMsg, "occurred") {
					t.Errorf("Error message = %q, want to contain 'occurred'", errMsg)
				}
			}

			// Verify all individual errors are mentioned
			for i, e := range errList.Errors {
				if !strings.Contains(errMsg, e.Error()) {
					t.Errorf("Error message missing error %d: %v", i+1, e)
				}
			}
		})
	}
}

func TestErrorList_Error(t *testing.T) {
	errList := &ErrorList{}
	errList.Add(&LoadError{FilePath: "/test/1.yaml", Message: "error 1"})
	errList.Add(&ParseError{FilePath: "/test/2.yaml", Message: "error 2"})

	// Test Error() method
	errMsg := errList.Error()

	if !strings.Contains(errMsg, "2 errors") {
		t.Errorf("Error() = %q, want to contain '2 errors'", errMsg)
	}
	if !strings.Contains(errMsg, "error 1") {
		t.Errorf("Error() = %q, want to contain 'error 1'", errMsg)
	}
	if !strings.Contains(errMsg, "error 2") {
		t.Errorf("Error() = %q, want to contain 'error 2'", errMsg)
	}
}

// Note: ErrorList is intentionally NOT thread-safe. It's designed to be used
// within a single goroutine to accumulate errors during sequential operations.
// If you need thread-safe error accumulation, use a mutex-protected ErrorList
// or channel-based error collection.

func TestErrorList_AddNil(t *testing.T) {
	errList := &ErrorList{}

	// Adding nil should be safe (no-op)
	errList.Add(nil)

	if errList.HasErrors() {
		t.Error("HasErrors() = true after adding nil, want false")
	}
	if len(errList.Errors) != 0 {
		t.Errorf("len(Errors) = %d after adding nil, want 0", len(errList.Errors))
	}
}

func TestErrorTypes_Chaining(t *testing.T) {
	// Test error wrapping chain
	baseErr := errors.New("base error")
	loadErr := &LoadError{
		FilePath: "/test/policy.yaml",
		Message:  "load failed",
		Cause:    baseErr,
	}

	// Test errors.Is() works through the chain
	if !errors.Is(loadErr, baseErr) {
		t.Error("errors.Is() does not work through LoadError wrapper")
	}

	// Test errors.As() works
	var le *LoadError
	if !errors.As(loadErr, &le) {
		t.Error("errors.As() failed to extract LoadError")
	}
	if le.FilePath != "/test/policy.yaml" {
		t.Errorf("LoadError.FilePath = %q, want %q", le.FilePath, "/test/policy.yaml")
	}
}

func TestErrorList_FirstError(t *testing.T) {
	errList := &ErrorList{}
	err1 := &LoadError{FilePath: "/test/1.yaml", Message: "first error"}
	err2 := &ParseError{FilePath: "/test/2.yaml", Message: "second error"}

	errList.Add(err1)
	errList.Add(err2)

	// Verify first error is accessible
	if len(errList.Errors) < 1 {
		t.Fatal("Errors field is empty")
	}
	if errList.Errors[0] != err1 {
		t.Error("First error in list is not the first added error")
	}
}
