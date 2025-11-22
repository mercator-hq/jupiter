package export

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

func TestJSONExporter_Export_EmptyRecords(t *testing.T) {
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	if buf.String() != "[]" {
		t.Errorf("Export() = %q, want %q", buf.String(), "[]")
	}
}

func TestJSONExporter_Export_SingleRecord(t *testing.T) {
	record := createTestRecord("test-id-1")
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Verify it's valid JSON
	var decoded evidence.EvidenceRecord
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Verify key fields
	if decoded.ID != "test-id-1" {
		t.Errorf("Decoded ID = %v, want %v", decoded.ID, "test-id-1")
	}
	if decoded.Model != "gpt-4" {
		t.Errorf("Decoded Model = %v, want %v", decoded.Model, "gpt-4")
	}
}

func TestJSONExporter_Export_MultipleRecords(t *testing.T) {
	records := []*evidence.EvidenceRecord{
		createTestRecord("test-id-1"),
		createTestRecord("test-id-2"),
		createTestRecord("test-id-3"),
	}

	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), records, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Verify it's valid JSON array
	var decoded []*evidence.EvidenceRecord
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if len(decoded) != 3 {
		t.Errorf("Decoded length = %d, want 3", len(decoded))
	}

	// Verify IDs match
	for i, record := range records {
		if decoded[i].ID != record.ID {
			t.Errorf("Decoded[%d].ID = %v, want %v", i, decoded[i].ID, record.ID)
		}
	}
}

func TestJSONExporter_Export_PrettyPrint(t *testing.T) {
	record := createTestRecord("test-id-1")
	exporter := NewJSONExporter(true) // Pretty-print enabled
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	output := buf.String()

	// Pretty-printed JSON should contain newlines and indentation
	if !containsString(output, "\n") {
		t.Error("Pretty-printed JSON missing newlines")
	}
	if !containsString(output, "  ") {
		t.Error("Pretty-printed JSON missing indentation")
	}

	// Should still be valid JSON
	var decoded evidence.EvidenceRecord
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Failed to decode pretty-printed JSON: %v", err)
	}
}

func TestJSONExporter_Export_NoPrettyPrint(t *testing.T) {
	record := createTestRecord("test-id-1")
	exporter := NewJSONExporter(false) // No pretty-print
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	output := buf.String()

	// Compact JSON should not have unnecessary whitespace
	// (Note: single newline at end is OK)
	lines := 0
	for _, c := range output {
		if c == '\n' {
			lines++
		}
	}
	if lines > 1 {
		t.Errorf("Compact JSON has %d newlines, expected 0-1", lines)
	}
}

func TestJSONExporter_Export_ComplexFields(t *testing.T) {
	// Test record with complex nested fields
	record := createTestRecord("test-id-1")
	record.RequestHeaders = map[string]string{
		"User-Agent":   "Mozilla/5.0",
		"Content-Type": "application/json",
	}
	record.MatchedRules = []evidence.MatchedRuleRecord{
		{
			PolicyID:       "policy-1",
			RuleID:         "rule-1",
			Action:         "allow",
			Reason:         "test reason",
			EvaluationTime: 5 * time.Millisecond,
		},
		{
			PolicyID:       "policy-2",
			RuleID:         "rule-2",
			Action:         "transform",
			Reason:         "another reason",
			EvaluationTime: 3 * time.Millisecond,
		},
	}
	record.ToolsUsed = []string{"function1", "function2", "function3"}
	record.PIITypes = []string{"email", "phone", "ssn"}

	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Verify complex fields are preserved
	var decoded evidence.EvidenceRecord
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	if len(decoded.RequestHeaders) != 2 {
		t.Errorf("Decoded RequestHeaders length = %d, want 2", len(decoded.RequestHeaders))
	}
	if len(decoded.MatchedRules) != 2 {
		t.Errorf("Decoded MatchedRules length = %d, want 2", len(decoded.MatchedRules))
	}
	if len(decoded.ToolsUsed) != 3 {
		t.Errorf("Decoded ToolsUsed length = %d, want 3", len(decoded.ToolsUsed))
	}
	if len(decoded.PIITypes) != 3 {
		t.Errorf("Decoded PIITypes length = %d, want 3", len(decoded.PIITypes))
	}
}

func TestJSONExporter_Export_SpecialCharacters(t *testing.T) {
	// Test record with special characters that need escaping
	record := createTestRecord("test-id-1")
	record.SystemPrompt = "Line 1\nLine 2\tTabbed\r\nWindows line"
	record.UserPrompt = `JSON special chars: "quotes", \backslash, /forward`
	record.Error = "Error: <script>alert('xss')</script>"

	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Verify special characters are properly escaped
	var decoded evidence.EvidenceRecord
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Failed to decode JSON with special chars: %v", err)
	}

	if decoded.SystemPrompt != record.SystemPrompt {
		t.Errorf("SystemPrompt not preserved: got %q, want %q", decoded.SystemPrompt, record.SystemPrompt)
	}
	if decoded.UserPrompt != record.UserPrompt {
		t.Errorf("UserPrompt not preserved: got %q, want %q", decoded.UserPrompt, record.UserPrompt)
	}
	if decoded.Error != record.Error {
		t.Errorf("Error not preserved: got %q, want %q", decoded.Error, record.Error)
	}
}

func TestJSONExporter_Export_Timestamps(t *testing.T) {
	record := createTestRecord("test-id-1")
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	// Verify timestamps are preserved with correct precision
	var decoded evidence.EvidenceRecord
	err = json.Unmarshal(buf.Bytes(), &decoded)
	if err != nil {
		t.Fatalf("Failed to decode JSON: %v", err)
	}

	// Timestamps should match (allowing for JSON round-trip precision)
	if !decoded.RequestTime.Equal(record.RequestTime) {
		t.Errorf("RequestTime not preserved: got %v, want %v", decoded.RequestTime, record.RequestTime)
	}
}

// BenchmarkJSONExporter_Export benchmarks JSON export
func BenchmarkJSONExporter_Export(b *testing.B) {
	sizes := []int{1, 10, 100, 1000}

	for _, size := range sizes {
		records := make([]*evidence.EvidenceRecord, size)
		for i := 0; i < size; i++ {
			records[i] = createTestRecord("test-id-" + string(rune(i)))
		}

		b.Run("records_"+string(rune(size)), func(b *testing.B) {
			exporter := NewJSONExporter(false)
			ctx := context.Background()

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				_ = exporter.Export(ctx, records, &buf)
			}
		})
	}
}

// BenchmarkJSONExporter_PrettyPrint benchmarks pretty-print overhead
func BenchmarkJSONExporter_PrettyPrint(b *testing.B) {
	record := createTestRecord("test-id-1")
	ctx := context.Background()

	b.Run("compact", func(b *testing.B) {
		exporter := NewJSONExporter(false)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			_ = exporter.Export(ctx, []*evidence.EvidenceRecord{record}, &buf)
		}
	})

	b.Run("pretty", func(b *testing.B) {
		exporter := NewJSONExporter(true)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			_ = exporter.Export(ctx, []*evidence.EvidenceRecord{record}, &buf)
		}
	})
}

// Helper function to create a test evidence record
func createTestRecord(id string) *evidence.EvidenceRecord {
	now := time.Now()
	return &evidence.EvidenceRecord{
		ID:               id,
		RequestID:        "req-" + id,
		RequestTime:      now,
		PolicyEvalTime:   now.Add(1 * time.Millisecond),
		ProviderCallTime: now.Add(2 * time.Millisecond),
		ResponseTime:     now.Add(100 * time.Millisecond),
		RecordedTime:     now.Add(101 * time.Millisecond),
		RequestHash:      "hash123",
		RequestMethod:    "POST",
		RequestPath:      "/v1/chat/completions",
		Model:            "gpt-4",
		Provider:         "openai",
		Messages:         3,
		EstimatedTokens:  150,
		EstimatedCost:    0.0045,
		PolicyDecision:   "allow",
		ResponseHash:     "hash456",
		ResponseStatus:   200,
		PromptTokens:     100,
		CompletionTokens: 50,
		TotalTokens:      150,
		ActualCost:       0.0045,
		ProviderLatency:  98 * time.Millisecond,
		ProviderModel:    "gpt-4-0613",
		UserID:           "user-123",
		APIKey:           "sha256:abcdef",
	}
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && findSubstr(s, substr)
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
