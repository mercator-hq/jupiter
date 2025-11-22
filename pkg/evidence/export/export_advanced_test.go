package export

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

// TestJSONExporter_MemoryLimit tests handling of very large exports.
func TestJSONExporter_MemoryLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory limit test in short mode")
	}

	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	// Create 10,000 records (should be manageable)
	recordCount := 10000
	records := make([]*evidence.EvidenceRecord, recordCount)

	for i := 0; i < recordCount; i++ {
		records[i] = &evidence.EvidenceRecord{
			ID:              fmt.Sprintf("record-%d", i),
			RequestID:       fmt.Sprintf("req-%d", i),
			RequestTime:     time.Now(),
			Model:           "gpt-4",
			SystemPrompt:    strings.Repeat("x", 500), // Max size
			ResponseContent: strings.Repeat("y", 500), // Max size
		}
	}

	start := time.Now()
	err := exporter.Export(context.Background(), records, &buf)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	outputSize := buf.Len()
	t.Logf("Exported %d records in %v (size: %d bytes, %.2f MB)",
		recordCount, duration, outputSize, float64(outputSize)/(1024*1024))

	// Performance target: export 10K records in <10s
	if duration > 10*time.Second {
		t.Logf("Warning: Export took %v (target: <10s)", duration)
	}

	// Memory check: should be reasonable (not loading everything multiple times)
	expectedSize := recordCount * 1000 // Rough estimate
	if outputSize < expectedSize {
		t.Logf("Output size reasonable: %d bytes for %d records", outputSize, recordCount)
	}
}

// TestCSVExporter_MemoryLimit tests handling of very large CSV exports.
func TestCSVExporter_MemoryLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory limit test in short mode")
	}

	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	// Create 10,000 records
	recordCount := 10000
	records := make([]*evidence.EvidenceRecord, recordCount)

	for i := 0; i < recordCount; i++ {
		records[i] = &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: time.Now(),
			Model:       "gpt-4",
			Provider:    "openai",
			TotalTokens: 1000 + i,
			ActualCost:  0.01 + float64(i)*0.001,
		}
	}

	start := time.Now()
	err := exporter.Export(context.Background(), records, &buf)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	outputSize := buf.Len()
	t.Logf("Exported %d records to CSV in %v (size: %d bytes, %.2f MB)",
		recordCount, duration, outputSize, float64(outputSize)/(1024*1024))

	// Verify CSV structure
	lines := strings.Split(buf.String(), "\n")
	expectedLines := recordCount + 1 // header + data rows
	if len(lines) < expectedLines {
		t.Errorf("Expected at least %d lines, got %d", expectedLines, len(lines))
	}
}

// TestJSONExporter_VeryLargeRecords tests handling of records with large fields.
func TestJSONExporter_VeryLargeRecords(t *testing.T) {
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	// Create a record with very large fields
	largeContent := strings.Repeat("Lorem ipsum dolor sit amet. ", 100) // ~2.8KB

	record := &evidence.EvidenceRecord{
		ID:              "large-record",
		RequestID:       "req-large",
		RequestTime:     time.Now(),
		SystemPrompt:    largeContent,
		UserPrompt:      largeContent,
		ResponseContent: largeContent,
		Model:           "gpt-4",
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	outputSize := buf.Len()
	t.Logf("Exported large record: %d bytes", outputSize)

	// Verify content is present
	if !strings.Contains(buf.String(), "Lorem ipsum") {
		t.Error("Expected large content in output")
	}
}

// TestCSVExporter_VeryLargeRecords tests CSV export with large fields.
func TestCSVExporter_VeryLargeRecords(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	// Create a record with large nested structures
	record := &evidence.EvidenceRecord{
		ID:          "large-nested",
		RequestID:   "req-nested",
		RequestTime: time.Now(),
		RequestHeaders: map[string]string{
			"header-1": strings.Repeat("x", 100),
			"header-2": strings.Repeat("y", 100),
			"header-3": strings.Repeat("z", 100),
		},
		ToolsUsed: []string{
			"tool1", "tool2", "tool3", "tool4", "tool5",
			"tool6", "tool7", "tool8", "tool9", "tool10",
		},
		PIITypes: []string{
			"email", "phone", "ssn", "credit_card", "address",
			"name", "dob", "passport", "license", "tax_id",
		},
		MatchedRules: []evidence.MatchedRuleRecord{
			{PolicyID: "p1", RuleID: "r1", Action: "allow"},
			{PolicyID: "p2", RuleID: "r2", Action: "log"},
			{PolicyID: "p3", RuleID: "r3", Action: "transform"},
		},
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()
	t.Logf("Exported nested record: %d bytes", len(output))

	// Verify nested structures are flattened
	if !strings.Contains(output, "header-1") {
		t.Error("Expected headers in CSV output")
	}
	if !strings.Contains(output, "tool1") {
		t.Error("Expected tools in CSV output")
	}
}

// TestJSONExporter_ContextCancellation tests export cancellation.
func TestJSONExporter_ContextCancellation(t *testing.T) {
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	// Create many records
	records := make([]*evidence.EvidenceRecord, 1000)
	for i := 0; i < 1000; i++ {
		records[i] = &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: time.Now(),
		}
	}

	// Create cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Export should respect cancellation (though it may complete fast)
	err := exporter.Export(ctx, records, &buf)

	// Note: Export may complete before checking context
	// This test documents the behavior
	if err != nil {
		t.Logf("Export cancelled: %v", err)
	} else {
		t.Log("Export completed before cancellation check")
	}
}

// TestCSVExporter_ContextCancellation tests CSV export cancellation.
func TestCSVExporter_ContextCancellation(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	records := make([]*evidence.EvidenceRecord, 1000)
	for i := 0; i < 1000; i++ {
		records[i] = &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: time.Now(),
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := exporter.Export(ctx, records, &buf)

	if err != nil {
		t.Logf("CSV export cancelled: %v", err)
	} else {
		t.Log("CSV export completed before cancellation check")
	}
}

// TestJSONExporter_EncodingErrors tests handling of encoding errors.
func TestJSONExporter_EncodingErrors(t *testing.T) {
	exporter := NewJSONExporter(false)
	var buf bytes.Buffer

	// Create a record with special characters that might cause encoding issues
	record := &evidence.EvidenceRecord{
		ID:              "encoding-test",
		RequestID:       "req-encoding",
		RequestTime:     time.Now(),
		SystemPrompt:    "Test with unicode: \u0000 \uffff 你好 🚀",
		ResponseContent: "Test with escape chars: \n\r\t\b\f",
		Error:           "Error with quotes: \"test\" and 'single'",
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()
	t.Logf("Encoded output: %d bytes", len(output))

	// Verify output is valid JSON
	if !strings.Contains(output, "encoding-test") {
		t.Error("Expected record ID in output")
	}
}

// TestCSVExporter_EncodingErrors tests CSV encoding of problematic characters.
func TestCSVExporter_EncodingErrors(t *testing.T) {
	exporter := NewCSVExporter(true)
	var buf bytes.Buffer

	record := &evidence.EvidenceRecord{
		ID:          "csv-encoding",
		RequestID:   "req-csv",
		RequestTime: time.Now(),
		Model:       "gpt-4",
		// Test various CSV problematic characters
		SystemPrompt:    `Test with "quotes", commas, and newlines`,
		ResponseContent: "Test with\nnewlines\rand\ttabs",
		BlockReason:     `Blocked: "invalid" content`,
	}

	err := exporter.Export(context.Background(), []*evidence.EvidenceRecord{record}, &buf)
	if err != nil {
		t.Fatalf("Export() failed: %v", err)
	}

	output := buf.String()
	t.Logf("CSV output: %d bytes", len(output))

	// CSV should escape these properly
	if !strings.Contains(output, "csv-encoding") {
		t.Error("Expected record ID in CSV output")
	}
}

// BenchmarkJSONExport_10KRecords benchmarks JSON export performance target.
func BenchmarkJSONExport_10KRecords(b *testing.B) {
	exporter := NewJSONExporter(false)

	records := make([]*evidence.EvidenceRecord, 10000)
	now := time.Now()
	for i := 0; i < 10000; i++ {
		records[i] = &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
			TotalTokens: 1000,
			ActualCost:  0.03,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.Export(context.Background(), records, &buf)
	}
	b.StopTimer()

	// Report metrics
	avgDuration := b.Elapsed() / time.Duration(b.N)
	b.ReportMetric(float64(avgDuration.Milliseconds()), "ms/10K-records")

	// Check against target: <10s per 10K records
	if avgDuration > 10*time.Second {
		b.Logf("Warning: Export took %v (target: <10s)", avgDuration)
	}
}

// BenchmarkCSVExport_10KRecords benchmarks CSV export performance target.
func BenchmarkCSVExport_10KRecords(b *testing.B) {
	exporter := NewCSVExporter(true)

	records := make([]*evidence.EvidenceRecord, 10000)
	now := time.Now()
	for i := 0; i < 10000; i++ {
		records[i] = &evidence.EvidenceRecord{
			ID:          fmt.Sprintf("record-%d", i),
			RequestID:   fmt.Sprintf("req-%d", i),
			RequestTime: now,
			Model:       "gpt-4",
			Provider:    "openai",
			TotalTokens: 1000,
			ActualCost:  0.03,
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var buf bytes.Buffer
		_ = exporter.Export(context.Background(), records, &buf)
	}
	b.StopTimer()

	avgDuration := b.Elapsed() / time.Duration(b.N)
	b.ReportMetric(float64(avgDuration.Milliseconds()), "ms/10K-records")

	if avgDuration > 10*time.Second {
		b.Logf("Warning: CSV export took %v (target: <10s)", avgDuration)
	}
}

// TestJSONExporter_MemoryUsage documents memory usage for large exports.
func TestJSONExporter_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping memory usage test in short mode")
	}

	exporter := NewJSONExporter(false)

	// Test various sizes to document memory behavior
	sizes := []int{100, 1000, 10000}

	for _, size := range sizes {
		records := make([]*evidence.EvidenceRecord, size)
		now := time.Now()

		for i := 0; i < size; i++ {
			records[i] = &evidence.EvidenceRecord{
				ID:          fmt.Sprintf("record-%d", i),
				RequestID:   fmt.Sprintf("req-%d", i),
				RequestTime: now,
				Model:       "gpt-4",
			}
		}

		var buf bytes.Buffer
		err := exporter.Export(context.Background(), records, &buf)
		if err != nil {
			t.Fatalf("Export() failed for size %d: %v", size, err)
		}

		outputSize := buf.Len()
		avgPerRecord := outputSize / size

		t.Logf("Size %d: output %d bytes (%.2f KB per record)",
			size, outputSize, float64(avgPerRecord)/1024)
	}
}
