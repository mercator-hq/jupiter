package cli

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
)

// OutputFormat represents the output format for command results.
type OutputFormat string

const (
	// FormatText is plain text output (default).
	FormatText OutputFormat = "text"
	// FormatJSON is JSON output.
	FormatJSON OutputFormat = "json"
	// FormatCSV is CSV output.
	FormatCSV OutputFormat = "csv"
	// FormatJUnit is JUnit XML output (for test results).
	FormatJUnit OutputFormat = "junit"
)

// Formatter formats command output.
type Formatter interface {
	Format(data interface{}) ([]byte, error)
	FormatTo(w io.Writer, data interface{}) error
}

// TextFormatter formats output as plain text.
type TextFormatter struct{}

// Format converts data to text format.
func (f *TextFormatter) Format(data interface{}) ([]byte, error) {
	return []byte(fmt.Sprintf("%v\n", data)), nil
}

// FormatTo writes data to writer in text format.
func (f *TextFormatter) FormatTo(w io.Writer, data interface{}) error {
	_, err := fmt.Fprintf(w, "%v\n", data)
	return err
}

// JSONFormatter formats output as JSON.
type JSONFormatter struct {
	Indent bool
}

// Format converts data to JSON format.
func (f *JSONFormatter) Format(data interface{}) ([]byte, error) {
	if f.Indent {
		return json.MarshalIndent(data, "", "  ")
	}
	return json.Marshal(data)
}

// FormatTo writes data to writer in JSON format.
func (f *JSONFormatter) FormatTo(w io.Writer, data interface{}) error {
	encoder := json.NewEncoder(w)
	if f.Indent {
		encoder.SetIndent("", "  ")
	}
	return encoder.Encode(data)
}

// CSVFormatter formats output as CSV.
type CSVFormatter struct {
	Headers []string
}

// Format converts data to CSV format.
func (f *CSVFormatter) Format(data interface{}) ([]byte, error) {
	return nil, fmt.Errorf("CSV formatting not yet implemented")
}

// FormatTo writes data to writer in CSV format.
func (f *CSVFormatter) FormatTo(w io.Writer, data interface{}) error {
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write headers
	if len(f.Headers) > 0 {
		if err := csvWriter.Write(f.Headers); err != nil {
			return err
		}
	}

	// TODO: Implement data row writing based on data type
	return fmt.Errorf("CSV formatting not yet implemented")
}

// NewFormatter creates a new formatter for the specified format.
func NewFormatter(format OutputFormat) Formatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{Indent: true}
	case FormatCSV:
		return &CSVFormatter{}
	default:
		return &TextFormatter{}
	}
}
