package ast

import "fmt"

// Location represents the source location of an AST node in the original policy file.
// It enables precise error reporting with file, line, and column information.
type Location struct {
	File   string // Path to the policy file
	Line   int    // Line number (1-based)
	Column int    // Column number (1-based)
}

// String returns a human-readable representation of the location.
// Format: "file:line:column"
func (l Location) String() string {
	if l.File == "" {
		return "<unknown>"
	}
	return fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)
}

// IsValid returns true if the location has valid file and line information.
func (l Location) IsValid() bool {
	return l.File != "" && l.Line > 0
}
