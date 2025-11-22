package evidence

import "fmt"

// EvidenceError is the base error type for all evidence-related errors.
type EvidenceError struct {
	Message string
	Cause   error
}

// Error implements the error interface.
func (e *EvidenceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause error.
func (e *EvidenceError) Unwrap() error {
	return e.Cause
}

// StorageError represents an error from the storage backend.
type StorageError struct {
	Backend   string // Storage backend type ("sqlite", "postgres", etc.)
	Operation string // Operation that failed ("store", "query", "delete", etc.)
	Cause     error  // Underlying error
}

// Error implements the error interface.
func (e *StorageError) Error() string {
	return fmt.Sprintf("storage error [backend=%s, operation=%s]: %v", e.Backend, e.Operation, e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *StorageError) Unwrap() error {
	return e.Cause
}

// NewStorageError creates a new StorageError.
func NewStorageError(backend, operation string, cause error) *StorageError {
	return &StorageError{
		Backend:   backend,
		Operation: operation,
		Cause:     cause,
	}
}

// QueryError represents an error during query execution or validation.
type QueryError struct {
	Query *Query // Query that failed
	Cause error  // Underlying error
}

// Error implements the error interface.
func (e *QueryError) Error() string {
	return fmt.Sprintf("query error: %v", e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *QueryError) Unwrap() error {
	return e.Cause
}

// NewQueryError creates a new QueryError.
func NewQueryError(query *Query, cause error) *QueryError {
	return &QueryError{
		Query: query,
		Cause: cause,
	}
}

// RecorderError represents an error during evidence recording.
type RecorderError struct {
	RecordID string // Evidence record ID
	Cause    error  // Underlying error
}

// Error implements the error interface.
func (e *RecorderError) Error() string {
	if e.RecordID != "" {
		return fmt.Sprintf("recorder error [record_id=%s]: %v", e.RecordID, e.Cause)
	}
	return fmt.Sprintf("recorder error: %v", e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *RecorderError) Unwrap() error {
	return e.Cause
}

// NewRecorderError creates a new RecorderError.
func NewRecorderError(recordID string, cause error) *RecorderError {
	return &RecorderError{
		RecordID: recordID,
		Cause:    cause,
	}
}

// RetentionError represents an error during retention policy enforcement.
type RetentionError struct {
	RetentionDays int   // Configured retention period
	Cause         error // Underlying error
}

// Error implements the error interface.
func (e *RetentionError) Error() string {
	return fmt.Sprintf("retention error [retention_days=%d]: %v", e.RetentionDays, e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *RetentionError) Unwrap() error {
	return e.Cause
}

// NewRetentionError creates a new RetentionError.
func NewRetentionError(retentionDays int, cause error) *RetentionError {
	return &RetentionError{
		RetentionDays: retentionDays,
		Cause:         cause,
	}
}

// ExportError represents an error during evidence export.
type ExportError struct {
	Format      string // Export format ("json", "csv", etc.)
	RecordCount int    // Number of records being exported
	Cause       error  // Underlying error
}

// Error implements the error interface.
func (e *ExportError) Error() string {
	return fmt.Sprintf("export error [format=%s, record_count=%d]: %v", e.Format, e.RecordCount, e.Cause)
}

// Unwrap returns the underlying cause error.
func (e *ExportError) Unwrap() error {
	return e.Cause
}

// NewExportError creates a new ExportError.
func NewExportError(format string, recordCount int, cause error) *ExportError {
	return &ExportError{
		Format:      format,
		RecordCount: recordCount,
		Cause:       cause,
	}
}
