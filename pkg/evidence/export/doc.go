// Package export provides evidence record exporters for various formats.
//
// # Export Formats
//
// The export package provides exporters for:
//
//   - JSON: Single record or array, with optional pretty-printing
//   - CSV: Flattened schema with header row and proper escaping
//
// # JSON Export
//
// The JSON exporter outputs evidence records in JSON format:
//
//	// Create JSON exporter with pretty-printing
//	exporter := export.NewJSONExporter(true)
//
//	// Export records to stdout
//	err := exporter.Export(ctx, records, os.Stdout)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # CSV Export
//
// The CSV exporter outputs evidence records in CSV format with proper escaping:
//
//	// Create CSV exporter with header row
//	exporter := export.NewCSVExporter(true)
//
//	// Export records to file
//	f, _ := os.Create("evidence.csv")
//	defer f.Close()
//
//	err := exporter.Export(ctx, records, f)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Streaming
//
// All exporters support streaming large result sets without loading all records
// into memory. Records are written to the output writer as they are processed.
//
// # Error Handling
//
// Exporters return ExportError if the export fails:
//
//   - JSON encoding errors
//   - CSV escaping errors
//   - Writer errors
package export
