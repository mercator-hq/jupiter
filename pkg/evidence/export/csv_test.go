package export

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

// TestCSVExporter_EmptyRecords tests exporting an empty record set.
func TestCSVExporter_EmptyRecords(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()

	// Should only have header row
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 1 {
		t.Errorf("Expected 1 line (header), got %d", len(lines))
	}

	// Verify header is present
	if !strings.Contains(output, "id,request_id") {
		t.Error("Expected header row with 'id,request_id'")
	}
}

// TestCSVExporter_SingleRecord tests exporting a single record.
func TestCSVExporter_SingleRecord(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	record := &evidence.EvidenceRecord{
		ID:             "test-id-123",
		RequestID:      "req-456",
		RequestTime:    now,
		RecordedTime:   now,
		RequestMethod:  "POST",
		RequestPath:    "/v1/chat/completions",
		Model:          "gpt-4",
		Provider:       "openai",
		Messages:       2,
		SystemPrompt:   "You are a helpful assistant",
		UserPrompt:     "What is the weather?",
		PolicyDecision: "allow",
		TotalTokens:    150,
		ActualCost:     0.015,
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 1 data row
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines (header + data), got %d", len(lines))
	}

	// Verify record data is present
	dataRow := lines[1]
	if !strings.Contains(dataRow, "test-id-123") {
		t.Error("Expected data row to contain record ID")
	}
	if !strings.Contains(dataRow, "req-456") {
		t.Error("Expected data row to contain request ID")
	}
	if !strings.Contains(dataRow, "gpt-4") {
		t.Error("Expected data row to contain model name")
	}
	if !strings.Contains(dataRow, "openai") {
		t.Error("Expected data row to contain provider name")
	}
}

// TestCSVExporter_MultipleRecords tests exporting multiple records.
func TestCSVExporter_MultipleRecords(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	now := time.Now()
	records := []*evidence.EvidenceRecord{
		{
			ID:             "record-1",
			RequestID:      "req-1",
			RequestTime:    now,
			Model:          "gpt-4",
			Provider:       "openai",
			PolicyDecision: "allow",
		},
		{
			ID:             "record-2",
			RequestID:      "req-2",
			RequestTime:    now.Add(1 * time.Second),
			Model:          "claude-3-opus",
			Provider:       "anthropic",
			PolicyDecision: "block",
		},
		{
			ID:             "record-3",
			RequestID:      "req-3",
			RequestTime:    now.Add(2 * time.Second),
			Model:          "gpt-3.5-turbo",
			Provider:       "openai",
			PolicyDecision: "allow",
		},
	}

	err := exporter.Export(context.Background(), records, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have header + 3 data rows
	if len(lines) != 4 {
		t.Errorf("Expected 4 lines (header + 3 data), got %d", len(lines))
	}

	// Verify all record IDs are present
	if !strings.Contains(output, "record-1") {
		t.Error("Expected output to contain record-1")
	}
	if !strings.Contains(output, "record-2") {
		t.Error("Expected output to contain record-2")
	}
	if !strings.Contains(output, "record-3") {
		t.Error("Expected output to contain record-3")
	}
}

// TestCSVExporter_NoHeader tests exporting without header row.
func TestCSVExporter_NoHeader(t *testing.T) {
	exporter := NewCSVExporter(false)
	var buf bytes.Buffer

	record := &evidence.EvidenceRecord{
		ID:        "test-id",
		RequestID: "req-id",
		Model:     "gpt-4",
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	// Should have only 1 data row (no header)
	if len(lines) != 1 {
		t.Errorf("Expected 1 line (data only), got %d", len(lines))
	}

	// Should not contain header keywords
	if strings.Contains(output, "id,request_id") {
		t.Error("Should not contain header row")
	}

	// But should contain data
	if !strings.Contains(output, "test-id") {
		t.Error("Expected data row to contain record ID")
	}
}

// TestCSVExporter_ComplexFields tests CSV export with complex nested fields.
func TestCSVExporter_ComplexFields(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	now := time.Now()
	record := &evidence.EvidenceRecord{
		ID:          "complex-record",
		RequestID:   "req-complex",
		RequestTime: now,
		RequestHeaders: map[string]string{
			"user-agent":    "test-agent",
			"authorization": "Bearer token",
		},
		ToolsUsed: []string{"web_search", "calculator", "code_interpreter"},
		PIITypes:  []string{"email", "phone_number", "ssn"},
		MatchedRules: []evidence.MatchedRuleRecord{
			{
				PolicyID:       "policy-1",
				RuleID:         "rule-1",
				Action:         "allow",
				Reason:         "Within rate limit",
				EvaluationTime: 5 * time.Millisecond,
			},
			{
				PolicyID:       "policy-2",
				RuleID:         "rule-2",
				Action:         "log",
				Reason:         "Contains sensitive data",
				EvaluationTime: 3 * time.Millisecond,
			},
		},
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()

	// Verify JSON-encoded fields are present
	if !strings.Contains(output, "user-agent") {
		t.Error("Expected headers to be JSON-encoded and present")
	}
	if !strings.Contains(output, "web_search") {
		t.Error("Expected tools to be JSON-encoded and present")
	}
	if !strings.Contains(output, "email") {
		t.Error("Expected PII types to be JSON-encoded and present")
	}
	if !strings.Contains(output, "policy-1") {
		t.Error("Expected matched rules to be JSON-encoded and present")
	}

	// Verify JSON arrays are properly formatted
	lines := strings.Split(output, "\n")
	dataRow := lines[1]

	// Check that the data row contains valid JSON structures
	if !strings.Contains(dataRow, "[") || !strings.Contains(dataRow, "]") {
		t.Error("Expected JSON arrays in output")
	}
}

// TestCSVExporter_SpecialCharacters tests CSV escaping for special characters.
func TestCSVExporter_SpecialCharacters(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	now := time.Now()
	record := &evidence.EvidenceRecord{
		ID:             "special-chars",
		RequestID:      "req-special",
		RequestTime:    now,
		SystemPrompt:   "Prompt with \"quotes\" and commas, newlines\nand tabs\there",
		UserPrompt:     "Question with special chars: <>&\"'",
		ResponseContent: "Response with\nnewlines\nand \"quotes\"",
		BlockReason:    "Contains: commas, quotes\", and\nnewlines",
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()

	// The CSV package should properly escape special characters
	// Verify the output contains the special characters (possibly escaped)
	if !strings.Contains(output, "special-chars") {
		t.Error("Expected record ID to be present")
	}

	// Verify we have proper CSV structure (comma-separated)
	lines := strings.Split(output, "\n")
	if len(lines) < 2 {
		t.Error("Expected at least 2 lines (header + data)")
	}
}

// TestCSVExporter_TimestampFormatting tests timestamp formatting in CSV.
func TestCSVExporter_TimestampFormatting(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	// Use specific timestamp for deterministic testing
	timestamp := time.Date(2025, 1, 15, 14, 30, 45, 0, time.UTC)

	record := &evidence.EvidenceRecord{
		ID:               "timestamp-test",
		RequestID:        "req-ts",
		RequestTime:      timestamp,
		PolicyEvalTime:   timestamp.Add(5 * time.Millisecond),
		ProviderCallTime: timestamp.Add(10 * time.Millisecond),
		ResponseTime:     timestamp.Add(100 * time.Millisecond),
		RecordedTime:     timestamp.Add(105 * time.Millisecond),
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()

	// Verify RFC3339 timestamp format
	expectedTime := "2025-01-15T14:30:45Z"
	if !strings.Contains(output, expectedTime) {
		t.Errorf("Expected timestamp in RFC3339 format: %s", expectedTime)
	}
}

// TestCSVExporter_NumericFields tests numeric field formatting.
func TestCSVExporter_NumericFields(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	now := time.Now()
	record := &evidence.EvidenceRecord{
		ID:               "numeric-test",
		RequestID:        "req-num",
		RequestTime:      now,
		Messages:         5,
		EstimatedTokens:  1234,
		EstimatedCost:    0.123456,
		RiskScore:        8,
		ComplexityScore:  6,
		PromptTokens:     1000,
		CompletionTokens: 500,
		TotalTokens:      1500,
		ActualCost:       0.234567,
		ProviderLatency:  250 * time.Millisecond,
		TurnNumber:       3,
		ContextUsage:     0.75,
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()

	// Verify numeric fields are present with correct formatting
	if !strings.Contains(output, "1234") {
		t.Error("Expected estimated tokens to be present")
	}
	if !strings.Contains(output, "0.123456") {
		t.Error("Expected estimated cost with 6 decimal places")
	}
	if !strings.Contains(output, "1500") {
		t.Error("Expected total tokens to be present")
	}
	if !strings.Contains(output, "0.234567") {
		t.Error("Expected actual cost with 6 decimal places")
	}
	if !strings.Contains(output, "250") {
		t.Error("Expected provider latency in milliseconds")
	}
	if !strings.Contains(output, "0.75") {
		t.Error("Expected context usage")
	}
}

// TestCSVExporter_ZeroValues tests handling of zero/empty values.
func TestCSVExporter_ZeroValues(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	record := &evidence.EvidenceRecord{
		ID:        "zero-values",
		RequestID: "req-zero",
		// All other fields left as zero values
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	// Verify the record exports without errors even with zero values
	dataRow := lines[1]
	if !strings.Contains(dataRow, "zero-values") {
		t.Error("Expected record ID in output")
	}
}

// BenchmarkCSVExport_SingleRecord benchmarks exporting a single record.
func BenchmarkCSVExport_SingleRecord(b *testing.B) {
	exporter := NewCSVExporter(true)
	record := createTestCSVRecord("bench-1")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	}
}

// BenchmarkCSVExport_10Records benchmarks exporting 10 records.
func BenchmarkCSVExport_10Records(b *testing.B) {
	exporter := NewCSVExporter(true)
	records := make([]*evidence.EvidenceRecord, 10)
	for i := 0; i < 10; i++ {
		records[i] = createTestCSVRecord("bench-" + string(rune('0'+i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.Export(context.Background(), records, &buf)
	}
}

// BenchmarkCSVExport_100Records benchmarks exporting 100 records.
func BenchmarkCSVExport_100Records(b *testing.B) {
	exporter := NewCSVExporter(true)
	records := make([]*evidence.EvidenceRecord, 100)
	for i := 0; i < 100; i++ {
		records[i] = createTestCSVRecord("bench-" + string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.Export(context.Background(), records, &buf)
	}
}

// BenchmarkCSVExport_1000Records benchmarks exporting 1000 records.
func BenchmarkCSVExport_1000Records(b *testing.B) {
	exporter := NewCSVExporter(true)
	records := make([]*evidence.EvidenceRecord, 1000)
	for i := 0; i < 1000; i++ {
		records[i] = createTestCSVRecord("bench-" + string(rune(i)))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.Export(context.Background(), records, &buf)
	}
}

// createTestCSVRecord creates a test record for CSV benchmarking.
func createTestCSVRecord(id string) *evidence.EvidenceRecord {
	now := time.Now()
	return &evidence.EvidenceRecord{
		ID:          id,
		RequestID:   "req-" + id,
		RequestTime: now,
		RequestHeaders: map[string]string{
			"user-agent": "test-agent",
		},
		Model:           "gpt-4",
		Provider:        "openai",
		Messages:        3,
		SystemPrompt:    "You are a helpful assistant",
		UserPrompt:      "Test question",
		ToolsUsed:       []string{"search", "calculator"},
		EstimatedTokens: 1000,
		EstimatedCost:   0.03,
		RiskScore:       5,
		ComplexityScore: 6,
		PIIDetected:     false,
		PIITypes:        []string{},
		PolicyDecision:  "allow",
		MatchedRules: []evidence.MatchedRuleRecord{
			{
				PolicyID: "policy-1",
				RuleID:   "rule-1",
				Action:   "allow",
			},
		},
		ResponseContent:  "Test response",
		PromptTokens:     800,
		CompletionTokens: 200,
		TotalTokens:      1000,
		ActualCost:       0.03,
		ProviderLatency:  100 * time.Millisecond,
		RecordedTime:     now,
	}
}

// TestCSVExporter_JSONMarshalError tests error handling when JSON fields are present.
func TestCSVExporter_JSONFields(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	now := time.Now()

	// Create a record with various JSON-encodable fields
	headers := map[string]string{
		"content-type": "application/json",
		"user-agent":   "test/1.0",
	}

	tools := []string{"function1", "function2"}
	piiTypes := []string{"email", "ssn"}

	matchedRules := []evidence.MatchedRuleRecord{
		{
			PolicyID:       "pol1",
			RuleID:         "rule1",
			Action:         "allow",
			Reason:         "test",
			EvaluationTime: 5 * time.Millisecond,
		},
	}

	record := &evidence.EvidenceRecord{
		ID:             "json-test",
		RequestID:      "req-json",
		RequestTime:    now,
		RequestHeaders: headers,
		ToolsUsed:      tools,
		PIITypes:       piiTypes,
		MatchedRules:   matchedRules,
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()

	// Verify JSON fields are marshaled correctly
	// The output should contain JSON-encoded versions of the fields
	if !strings.Contains(output, "content-type") {
		t.Error("Expected headers to be JSON-encoded in output")
	}

	// Verify the CSV has the correct structure
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines (header + data), got %d", len(lines))
	}
}
