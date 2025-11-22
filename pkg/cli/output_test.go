package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
)

func TestTextFormatter(t *testing.T) {
	formatter := &TextFormatter{}
	data := "test message"

	output, err := formatter.Format(data)
	if err != nil {
		t.Fatalf("Format() error = %v", err)
	}

	expected := "test message\n"
	if string(output) != expected {
		t.Errorf("Format() = %q, want %q", string(output), expected)
	}
}

func TestTextFormatterWriter(t *testing.T) {
	formatter := &TextFormatter{}
	data := "test message"
	buf := &bytes.Buffer{}

	err := formatter.FormatTo(buf, data)
	if err != nil {
		t.Fatalf("FormatTo() error = %v", err)
	}

	expected := "test message\n"
	if buf.String() != expected {
		t.Errorf("FormatTo() = %q, want %q", buf.String(), expected)
	}
}

func TestJSONFormatter(t *testing.T) {
	tests := []struct {
		name   string
		data   interface{}
		indent bool
	}{
		{
			name:   "simple string",
			data:   "test",
			indent: false,
		},
		{
			name: "map with indent",
			data: map[string]string{
				"key": "value",
			},
			indent: true,
		},
		{
			name: "struct",
			data: struct {
				Name  string `json:"name"`
				Value int    `json:"value"`
			}{
				Name:  "test",
				Value: 42,
			},
			indent: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := &JSONFormatter{Indent: tt.indent}
			output, err := formatter.Format(tt.data)
			if err != nil {
				t.Fatalf("Format() error = %v", err)
			}

			// Verify it's valid JSON by unmarshaling
			var result interface{}
			if err := json.Unmarshal(output, &result); err != nil {
				t.Errorf("Format() produced invalid JSON: %v", err)
			}
		})
	}
}

func TestJSONFormatterWriter(t *testing.T) {
	formatter := &JSONFormatter{Indent: true}
	data := map[string]string{"test": "value"}
	buf := &bytes.Buffer{}

	err := formatter.FormatTo(buf, data)
	if err != nil {
		t.Fatalf("FormatTo() error = %v", err)
	}

	// Verify valid JSON
	var result map[string]string
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Errorf("FormatTo() produced invalid JSON: %v", err)
	}

	if result["test"] != "value" {
		t.Errorf("FormatTo() = %v, want %v", result, data)
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name   string
		format OutputFormat
		want   string
	}{
		{
			name:   "text formatter",
			format: FormatText,
			want:   "*cli.TextFormatter",
		},
		{
			name:   "json formatter",
			format: FormatJSON,
			want:   "*cli.JSONFormatter",
		},
		{
			name:   "csv formatter",
			format: FormatCSV,
			want:   "*cli.CSVFormatter",
		},
		{
			name:   "default to text",
			format: "unknown",
			want:   "*cli.TextFormatter",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatter := NewFormatter(tt.format)
			got := fmt.Sprintf("%T", formatter)
			if got != tt.want {
				t.Errorf("NewFormatter(%q) type = %v, want %v", tt.format, got, tt.want)
			}
		})
	}
}

func TestCSVFormatter(t *testing.T) {
	formatter := &CSVFormatter{
		Headers: []string{"name", "value"},
	}

	// CSV formatting is not yet implemented
	_, err := formatter.Format(nil)
	if err == nil {
		t.Error("Format() expected error for unimplemented CSV, got nil")
	}
}
