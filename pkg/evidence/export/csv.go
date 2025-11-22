package export

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
)

// CSVExporter exports evidence records to CSV format.
type CSVExporter struct {
	// IncludeHeader includes a header row with column names.
	IncludeHeader bool
}

// NewCSVExporter creates a new CSV exporter.
func NewCSVExporter(includeHeader bool) *CSVExporter {
	return &CSVExporter{
		IncludeHeader: includeHeader,
	}
}

// Export writes evidence records to the provided writer in CSV format.
// The CSV format flattens nested structures (JSON arrays become comma-separated strings).
func (e *CSVExporter) Export(ctx context.Context, records []*evidence.EvidenceRecord, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header row if configured
	if e.IncludeHeader {
		header := e.getHeaderRow()
		if err := writer.Write(header); err != nil {
			return evidence.NewExportError("csv", len(records), err)
		}
	}

	// Write data rows
	for _, record := range records {
		row, err := e.recordToRow(record)
		if err != nil {
			return evidence.NewExportError("csv", len(records), err)
		}
		if err := writer.Write(row); err != nil {
			return evidence.NewExportError("csv", len(records), err)
		}
	}

	return nil
}

// ExportStream exports evidence records from a channel to CSV format.
// This is memory-efficient for large result sets as it streams records
// one at a time instead of loading all records in memory.
//
// The CSV writer flushes periodically to provide progress feedback
// for long-running exports.
func (e *CSVExporter) ExportStream(ctx context.Context, recordsCh <-chan *evidence.EvidenceRecord, w io.Writer) error {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header if configured
	if e.IncludeHeader {
		header := e.getHeaderRow()
		if err := writer.Write(header); err != nil {
			return evidence.NewExportError("csv", 0, err)
		}
	}

	recordCount := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case record, ok := <-recordsCh:
			if !ok {
				// Channel closed - flush and return
				writer.Flush()
				if err := writer.Error(); err != nil {
					return evidence.NewExportError("csv", recordCount, err)
				}
				return nil
			}

			// Convert record to CSV row
			row, err := e.recordToRow(record)
			if err != nil {
				return evidence.NewExportError("csv", recordCount, err)
			}

			// Write row
			if err := writer.Write(row); err != nil {
				return evidence.NewExportError("csv", recordCount, err)
			}

			recordCount++

			// Flush periodically (every 100 records)
			// This provides progress feedback for long exports
			if recordCount%100 == 0 {
				writer.Flush()
				if err := writer.Error(); err != nil {
					return evidence.NewExportError("csv", recordCount, err)
				}
			}
		}
	}
}

// getHeaderRow returns the CSV header row.
func (e *CSVExporter) getHeaderRow() []string {
	return []string{
		"id", "request_id",
		"request_time", "policy_eval_time", "provider_call_time", "response_time", "recorded_time",
		"request_hash", "request_method", "request_path", "request_headers",
		"model", "provider", "messages", "system_prompt", "user_prompt", "tools_used",
		"estimated_tokens", "estimated_cost", "risk_score", "complexity_score", "pii_detected", "pii_types",
		"policy_decision", "matched_rules", "block_reason", "policy_version",
		"response_hash", "response_status",
		"response_content", "finish_reason",
		"prompt_tokens", "completion_tokens", "total_tokens", "actual_cost",
		"provider_latency_ms", "provider_model",
		"user_id", "api_key", "ip_address",
		"error", "error_type",
		"turn_number", "context_usage",
	}
}

// recordToRow converts an evidence record to a CSV row.
func (e *CSVExporter) recordToRow(record *evidence.EvidenceRecord) ([]string, error) {
	// Helper function to format timestamps
	formatTime := func(t time.Time) string {
		if t.IsZero() {
			return ""
		}
		return t.Format(time.RFC3339)
	}

	// Helper function to format JSON fields
	formatJSON := func(v interface{}) string {
		data, _ := json.Marshal(v)
		return string(data)
	}

	row := []string{
		record.ID,
		record.RequestID,
		formatTime(record.RequestTime),
		formatTime(record.PolicyEvalTime),
		formatTime(record.ProviderCallTime),
		formatTime(record.ResponseTime),
		formatTime(record.RecordedTime),
		record.RequestHash,
		record.RequestMethod,
		record.RequestPath,
		formatJSON(record.RequestHeaders),
		record.Model,
		record.Provider,
		fmt.Sprintf("%d", record.Messages),
		record.SystemPrompt,
		record.UserPrompt,
		formatJSON(record.ToolsUsed),
		fmt.Sprintf("%d", record.EstimatedTokens),
		fmt.Sprintf("%.6f", record.EstimatedCost),
		fmt.Sprintf("%d", record.RiskScore),
		fmt.Sprintf("%d", record.ComplexityScore),
		fmt.Sprintf("%t", record.PIIDetected),
		formatJSON(record.PIITypes),
		record.PolicyDecision,
		formatJSON(record.MatchedRules),
		record.BlockReason,
		record.PolicyVersion,
		record.ResponseHash,
		fmt.Sprintf("%d", record.ResponseStatus),
		record.ResponseContent,
		record.FinishReason,
		fmt.Sprintf("%d", record.PromptTokens),
		fmt.Sprintf("%d", record.CompletionTokens),
		fmt.Sprintf("%d", record.TotalTokens),
		fmt.Sprintf("%.6f", record.ActualCost),
		fmt.Sprintf("%d", record.ProviderLatency.Milliseconds()),
		record.ProviderModel,
		record.UserID,
		record.APIKey,
		record.IPAddress,
		record.Error,
		record.ErrorType,
		fmt.Sprintf("%d", record.TurnNumber),
		fmt.Sprintf("%.2f", record.ContextUsage),
	}

	return row, nil
}
