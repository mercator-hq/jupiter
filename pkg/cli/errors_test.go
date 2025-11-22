package cli

import (
	"errors"
	"testing"
)

func TestConfigError(t *testing.T) {
	err := &ConfigError{
		Field:   "proxy.listen_address",
		Message: "missing required field",
	}

	expected := "config error in proxy.listen_address: missing required field"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestNewConfigError(t *testing.T) {
	err := NewConfigError("field", "message")
	if err.Field != "field" {
		t.Errorf("Field = %q, want %q", err.Field, "field")
	}
	if err.Message != "message" {
		t.Errorf("Message = %q, want %q", err.Message, "message")
	}
}

func TestCommandError(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &CommandError{
		Command: "run",
		Err:     underlyingErr,
	}

	expected := "command run failed: underlying error"
	if err.Error() != expected {
		t.Errorf("Error() = %q, want %q", err.Error(), expected)
	}
}

func TestCommandErrorUnwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	err := &CommandError{
		Command: "run",
		Err:     underlyingErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlyingErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlyingErr)
	}

	// Test with errors.Is
	if !errors.Is(err, underlyingErr) {
		t.Error("errors.Is() should work with CommandError.Unwrap()")
	}
}

func TestNewCommandError(t *testing.T) {
	underlyingErr := errors.New("test")
	err := NewCommandError("command", underlyingErr)

	if err.Command != "command" {
		t.Errorf("Command = %q, want %q", err.Command, "command")
	}
	if err.Err != underlyingErr {
		t.Errorf("Err = %v, want %v", err.Err, underlyingErr)
	}
}
