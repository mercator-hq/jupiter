package errors

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// ExtractContext reads the policy file and extracts the surrounding lines
// around the given location for error context display.
// It returns a formatted string showing the error location with line numbers.
func ExtractContext(location ast.Location, contextLines int) string {
	if !location.IsValid() {
		return ""
	}

	file, err := os.Open(location.File)
	if err != nil {
		// File not accessible, return empty context
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lines := make([]string, 0)
	lineNum := 0

	// Read all lines
	for scanner.Scan() {
		lineNum++
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return ""
	}

	// Calculate context range
	errorLine := location.Line - 1 // Convert to 0-based index
	startLine := errorLine - contextLines
	endLine := errorLine + contextLines

	if startLine < 0 {
		startLine = 0
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// Build context string
	var sb strings.Builder
	maxLineNumWidth := len(fmt.Sprintf("%d", endLine+1))

	for i := startLine; i <= endLine; i++ {
		lineNumStr := fmt.Sprintf("%*d", maxLineNumWidth, i+1)
		prefix := "  "
		if i == errorLine {
			prefix = "->"
		}

		sb.WriteString(fmt.Sprintf("%s %s | %s\n", prefix, lineNumStr, lines[i]))

		// Add column indicator for error line
		if i == errorLine && location.Column > 0 {
			padding := strings.Repeat(" ", maxLineNumWidth+3+location.Column)
			sb.WriteString(fmt.Sprintf("  %s | %s^\n", strings.Repeat(" ", maxLineNumWidth), padding))
		}
	}

	return sb.String()
}

// WithContext creates a new error with context extracted from the file.
func WithContext(err *Error, contextLines int) *Error {
	if err.Location.IsValid() {
		err.Context = ExtractContext(err.Location, contextLines)
	}
	return err
}

// AddContextToError adds context to an error by reading the source file.
// This is typically called after creating an error to enrich it with source context.
func AddContextToError(err *Error) *Error {
	return WithContext(err, 2) // Show 2 lines before and after by default
}
