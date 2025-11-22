package cli

import "fmt"

// ConfigError represents an error in configuration.
type ConfigError struct {
	Field   string
	Message string
}

func (e *ConfigError) Error() string {
	return fmt.Sprintf("config error in %s: %s", e.Field, e.Message)
}

// CommandError represents an error from a command execution.
type CommandError struct {
	Command string
	Err     error
}

func (e *CommandError) Error() string {
	return fmt.Sprintf("command %s failed: %v", e.Command, e.Err)
}

func (e *CommandError) Unwrap() error {
	return e.Err
}

// NewConfigError creates a new ConfigError.
func NewConfigError(field, message string) *ConfigError {
	return &ConfigError{
		Field:   field,
		Message: message,
	}
}

// NewCommandError creates a new CommandError.
func NewCommandError(command string, err error) *CommandError {
	return &CommandError{
		Command: command,
		Err:     err,
	}
}
