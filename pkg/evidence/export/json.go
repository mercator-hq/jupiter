package export

import (
	"context"
	"encoding/json"
	"io"

	"mercator-hq/jupiter/pkg/evidence"
)

// JSONExporter exports evidence records to JSON format.
type JSONExporter struct {
	// Pretty enables pretty-printing with indentation.
	Pretty bool
}

// NewJSONExporter creates a new JSON exporter.
func NewJSONExporter(pretty bool) *JSONExporter {
	return &JSONExporter{
		Pretty: pretty,
	}
}

// Export writes evidence records to the provided writer in JSON format.
// If Pretty is true, the JSON will be indented for readability.
//
// For a single record, exports the record as a JSON object.
// For multiple records, exports an array of JSON objects.
func (e *JSONExporter) Export(ctx context.Context, records []*evidence.EvidenceRecord, w io.Writer) error {
	if len(records) == 0 {
		// Write empty array
		_, err := w.Write([]byte("[]"))
		return err
	}

	var data []byte
	var err error

	// Export single record or array
	if len(records) == 1 {
		if e.Pretty {
			data, err = json.MarshalIndent(records[0], "", "  ")
		} else {
			data, err = json.Marshal(records[0])
		}
	} else {
		if e.Pretty {
			data, err = json.MarshalIndent(records, "", "  ")
		} else {
			data, err = json.Marshal(records)
		}
	}

	if err != nil {
		return evidence.NewExportError("json", len(records), err)
	}

	// Write to output
	_, err = w.Write(data)
	if err != nil {
		return evidence.NewExportError("json", len(records), err)
	}

	return nil
}

// ExportStream exports evidence records from a channel to JSON format.
// This is memory-efficient for large result sets as it streams records
// one at a time instead of loading all records in memory.
//
// The records are exported as a JSON array. The stream processes records
// as they arrive on the channel, making it suitable for very large exports.
func (e *JSONExporter) ExportStream(ctx context.Context, recordsCh <-chan *evidence.EvidenceRecord, w io.Writer) error {
	// Write opening bracket
	if _, err := w.Write([]byte("[")); err != nil {
		return evidence.NewExportError("json", 0, err)
	}

	first := true
	recordCount := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case record, ok := <-recordsCh:
			if !ok {
				// Channel closed - write closing bracket and return
				if _, err := w.Write([]byte("]")); err != nil {
					return evidence.NewExportError("json", recordCount, err)
				}
				return nil
			}

			// Write comma and newline before all but first record
			if !first {
				if _, err := w.Write([]byte(",")); err != nil {
					return evidence.NewExportError("json", recordCount, err)
				}
				if e.Pretty {
					if _, err := w.Write([]byte("\n")); err != nil {
						return evidence.NewExportError("json", recordCount, err)
					}
				}
			}
			first = false

			// Serialize record
			data, err := e.serializeRecord(record)
			if err != nil {
				return evidence.NewExportError("json", recordCount, err)
			}

			if _, err := w.Write(data); err != nil {
				return evidence.NewExportError("json", recordCount, err)
			}

			recordCount++
		}
	}
}

// serializeRecord serializes a single evidence record to JSON.
func (e *JSONExporter) serializeRecord(record *evidence.EvidenceRecord) ([]byte, error) {
	if e.Pretty {
		return json.MarshalIndent(record, "  ", "  ")
	}
	return json.Marshal(record)
}
