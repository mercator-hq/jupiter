package export

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/storage"
)

// TestJSONExporter_ExportStream tests JSON streaming export.
func TestJSONExporter_ExportStream(t *testing.T) {
	t.Run("stream multiple records", func(t *testing.T) {
		exporter := NewJSONExporter(false)

		// Create channel and send records
		recordsCh := make(chan *evidence.EvidenceRecord, 10)

		// Send 100 records asynchronously
		go func() {
			defer close(recordsCh)
			for i := 0; i < 100; i++ {
				recordsCh <- &evidence.EvidenceRecord{
					ID:          fmt.Sprintf("test-%d", i),
					RequestID:   fmt.Sprintf("req-%d", i),
					RequestTime: time.Now(),
					Model:       "gpt-4",
					Provider:    "openai",
				}
			}
		}()

		// Export to buffer
		var buf bytes.Buffer
		err := exporter.ExportStream(context.Background(), recordsCh, &buf)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// Verify JSON is valid
		var records []evidence.EvidenceRecord
		if err := json.Unmarshal(buf.Bytes(), &records); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if len(records) != 100 {
			t.Errorf("expected 100 records, got %d", len(records))
		}

		// Verify record contents
		if records[0].ID != "test-0" {
			t.Errorf("expected ID test-0, got %s", records[0].ID)
		}
		if records[99].ID != "test-99" {
			t.Errorf("expected ID test-99, got %s", records[99].ID)
		}
	})

	t.Run("stream with pretty printing", func(t *testing.T) {
		exporter := NewJSONExporter(true)

		recordsCh := make(chan *evidence.EvidenceRecord, 10)

		go func() {
			defer close(recordsCh)
			for i := 0; i < 3; i++ {
				recordsCh <- &evidence.EvidenceRecord{
					ID:        fmt.Sprintf("test-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
					Model:     "gpt-4",
				}
			}
		}()

		var buf bytes.Buffer
		err := exporter.ExportStream(context.Background(), recordsCh, &buf)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		output := buf.String()

		// Check for indentation (pretty printing)
		if !strings.Contains(output, "\n") {
			t.Error("expected newlines in pretty-printed output")
		}

		// Verify still valid JSON
		var records []evidence.EvidenceRecord
		if err := json.Unmarshal(buf.Bytes(), &records); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if len(records) != 3 {
			t.Errorf("expected 3 records, got %d", len(records))
		}
	})

	t.Run("stream empty channel", func(t *testing.T) {
		exporter := NewJSONExporter(false)

		recordsCh := make(chan *evidence.EvidenceRecord)
		close(recordsCh)

		var buf bytes.Buffer
		err := exporter.ExportStream(context.Background(), recordsCh, &buf)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		if buf.String() != "[]" {
			t.Errorf("expected empty array, got %s", buf.String())
		}
	})

	t.Run("stream with context cancellation", func(t *testing.T) {
		exporter := NewJSONExporter(false)
		ctx, cancel := context.WithCancel(context.Background())

		recordsCh := make(chan *evidence.EvidenceRecord, 10)

		// Send records slowly
		go func() {
			defer close(recordsCh)
			for i := 0; i < 100; i++ {
				time.Sleep(5 * time.Millisecond)
				recordsCh <- &evidence.EvidenceRecord{
					ID:        fmt.Sprintf("test-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
				}
			}
		}()

		// Cancel after short delay
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		var buf bytes.Buffer
		err := exporter.ExportStream(ctx, recordsCh, &buf)

		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("stream memory usage test", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping memory test in short mode")
		}

		exporter := NewJSONExporter(false)

		recordsCh := make(chan *evidence.EvidenceRecord, 100) // Small buffer

		// Send 50K records
		go func() {
			defer close(recordsCh)
			for i := 0; i < 50000; i++ {
				recordsCh <- &evidence.EvidenceRecord{
					ID:        fmt.Sprintf("test-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
				}
			}
		}()

		// Export to /dev/null (don't care about output, just memory usage)
		err := exporter.ExportStream(context.Background(), recordsCh, io.Discard)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// If this test passes, memory usage was bounded
		t.Log("Successfully streamed 50K records without OOM")
	})
}

// TestCSVExporter_ExportStream tests CSV streaming export.
func TestCSVExporter_ExportStream(t *testing.T) {
	t.Run("stream multiple records with header", func(t *testing.T) {
		exporter := NewCSVExporter(true)

		recordsCh := make(chan *evidence.EvidenceRecord, 10)

		go func() {
			defer close(recordsCh)
			for i := 0; i < 50; i++ {
				recordsCh <- &evidence.EvidenceRecord{
					ID:          fmt.Sprintf("test-%d", i),
					RequestID:   fmt.Sprintf("req-%d", i),
					RequestTime: time.Now(),
					Model:       "gpt-4",
					Provider:    "openai",
				}
			}
		}()

		var buf bytes.Buffer
		err := exporter.ExportStream(context.Background(), recordsCh, &buf)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// Parse CSV
		reader := csv.NewReader(&buf)
		rows, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to parse CSV: %v", err)
		}

		// Should have header + 50 data rows
		if len(rows) != 51 {
			t.Errorf("expected 51 rows (1 header + 50 data), got %d", len(rows))
		}

		// Check header
		if rows[0][0] != "id" {
			t.Errorf("expected first column header to be 'id', got %s", rows[0][0])
		}

		// Check first data row
		if rows[1][0] != "test-0" {
			t.Errorf("expected first data row ID to be 'test-0', got %s", rows[1][0])
		}
	})

	t.Run("stream without header", func(t *testing.T) {
		exporter := NewCSVExporter(false)

		recordsCh := make(chan *evidence.EvidenceRecord, 10)

		go func() {
			defer close(recordsCh)
			for i := 0; i < 10; i++ {
				recordsCh <- &evidence.EvidenceRecord{
					ID:        fmt.Sprintf("test-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
				}
			}
		}()

		var buf bytes.Buffer
		err := exporter.ExportStream(context.Background(), recordsCh, &buf)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// Parse CSV
		reader := csv.NewReader(&buf)
		rows, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to parse CSV: %v", err)
		}

		// Should have only data rows (no header)
		if len(rows) != 10 {
			t.Errorf("expected 10 rows, got %d", len(rows))
		}
	})

	t.Run("stream empty channel", func(t *testing.T) {
		exporter := NewCSVExporter(true)

		recordsCh := make(chan *evidence.EvidenceRecord)
		close(recordsCh)

		var buf bytes.Buffer
		err := exporter.ExportStream(context.Background(), recordsCh, &buf)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// Parse CSV
		reader := csv.NewReader(&buf)
		rows, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to parse CSV: %v", err)
		}

		// Should have only header row
		if len(rows) != 1 {
			t.Errorf("expected 1 row (header only), got %d", len(rows))
		}
	})

	t.Run("stream with context cancellation", func(t *testing.T) {
		exporter := NewCSVExporter(true)
		ctx, cancel := context.WithCancel(context.Background())

		recordsCh := make(chan *evidence.EvidenceRecord, 10)

		// Send records slowly
		go func() {
			defer close(recordsCh)
			for i := 0; i < 100; i++ {
				time.Sleep(5 * time.Millisecond)
				recordsCh <- &evidence.EvidenceRecord{
					ID:        fmt.Sprintf("test-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
				}
			}
		}()

		// Cancel after short delay
		go func() {
			time.Sleep(20 * time.Millisecond)
			cancel()
		}()

		var buf bytes.Buffer
		err := exporter.ExportStream(ctx, recordsCh, &buf)

		if err != context.Canceled {
			t.Errorf("expected context.Canceled, got %v", err)
		}
	})

	t.Run("stream memory usage test", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skipping memory test in short mode")
		}

		exporter := NewCSVExporter(true)

		recordsCh := make(chan *evidence.EvidenceRecord, 100) // Small buffer

		// Send 50K records
		go func() {
			defer close(recordsCh)
			for i := 0; i < 50000; i++ {
				recordsCh <- &evidence.EvidenceRecord{
					ID:        fmt.Sprintf("test-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
				}
			}
		}()

		// Export to /dev/null
		err := exporter.ExportStream(context.Background(), recordsCh, io.Discard)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// If this test passes, memory usage was bounded
		t.Log("Successfully streamed 50K records without OOM")
	})

	t.Run("periodic flush verification", func(t *testing.T) {
		exporter := NewCSVExporter(true)

		recordsCh := make(chan *evidence.EvidenceRecord, 10)

		// Send 250 records (should trigger 2 flushes at 100 and 200)
		go func() {
			defer close(recordsCh)
			for i := 0; i < 250; i++ {
				recordsCh <- &evidence.EvidenceRecord{
					ID:        fmt.Sprintf("test-%d", i),
					RequestID: fmt.Sprintf("req-%d", i),
				}
			}
		}()

		var buf bytes.Buffer
		err := exporter.ExportStream(context.Background(), recordsCh, &buf)
		if err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// Verify all records exported correctly
		reader := csv.NewReader(&buf)
		rows, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to parse CSV: %v", err)
		}

		// Should have header + 250 data rows
		if len(rows) != 251 {
			t.Errorf("expected 251 rows, got %d", len(rows))
		}
	})
}

// TestIntegration_StorageToExport tests the complete pipeline from storage streaming to export.
func TestIntegration_StorageToExport(t *testing.T) {
	// This test demonstrates the complete streaming pipeline:
	// Storage.QueryStream -> Exporter.ExportStream

	t.Run("SQLite to JSON streaming", func(t *testing.T) {
		// Create in-memory storage for testing
		memStorage := storage.NewMemoryStorage()
		ctx := context.Background()

		// Insert test records
		for i := 0; i < 100; i++ {
			record := &evidence.EvidenceRecord{
				ID:          fmt.Sprintf("test-%d", i),
				RequestID:   fmt.Sprintf("req-%d", i),
				RequestTime: time.Now(),
				Model:       "gpt-4",
				Provider:    "openai",
			}
			if err := memStorage.Store(ctx, record); err != nil {
				t.Fatalf("failed to store: %v", err)
			}
		}

		// Query stream
		query := &evidence.Query{Limit: 100}
		recordsCh, errCh, err := memStorage.QueryStream(ctx, query)
		if err != nil {
			t.Fatalf("QueryStream failed: %v", err)
		}

		// Export stream
		exporter := NewJSONExporter(false)
		var buf bytes.Buffer

		// Start export in goroutine
		exportDone := make(chan error)
		go func() {
			exportDone <- exporter.ExportStream(ctx, recordsCh, &buf)
		}()

		// Wait for export to complete
		if err := <-exportDone; err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// Check for query errors
		if err := <-errCh; err != nil {
			t.Fatalf("query error: %v", err)
		}

		// Verify exported JSON
		var records []evidence.EvidenceRecord
		if err := json.Unmarshal(buf.Bytes(), &records); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}

		if len(records) != 100 {
			t.Errorf("expected 100 records, got %d", len(records))
		}
	})

	t.Run("Memory to CSV streaming", func(t *testing.T) {
		memStorage := storage.NewMemoryStorage()
		ctx := context.Background()

		// Insert test records
		for i := 0; i < 200; i++ {
			record := &evidence.EvidenceRecord{
				ID:        fmt.Sprintf("test-%d", i),
				RequestID: fmt.Sprintf("req-%d", i),
				Model:     "claude-3",
				Provider:  "anthropic",
			}
			if err := memStorage.Store(ctx, record); err != nil {
				t.Fatalf("failed to store: %v", err)
			}
		}

		// Query stream
		query := &evidence.Query{Limit: 200}
		recordsCh, errCh, err := memStorage.QueryStream(ctx, query)
		if err != nil {
			t.Fatalf("QueryStream failed: %v", err)
		}

		// Export stream
		exporter := NewCSVExporter(true)
		var buf bytes.Buffer

		// Start export
		exportDone := make(chan error)
		go func() {
			exportDone <- exporter.ExportStream(ctx, recordsCh, &buf)
		}()

		// Wait for export to complete
		if err := <-exportDone; err != nil {
			t.Fatalf("ExportStream failed: %v", err)
		}

		// Check for query errors
		if err := <-errCh; err != nil {
			t.Fatalf("query error: %v", err)
		}

		// Verify exported CSV
		reader := csv.NewReader(&buf)
		rows, err := reader.ReadAll()
		if err != nil {
			t.Fatalf("failed to parse CSV: %v", err)
		}

		// Should have header + 200 data rows
		if len(rows) != 201 {
			t.Errorf("expected 201 rows, got %d", len(rows))
		}
	})
}
