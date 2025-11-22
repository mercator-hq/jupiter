package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"mercator-hq/jupiter/pkg/evidence"
)

// SQLiteConfig contains configuration for the SQLite storage backend.
type SQLiteConfig struct {
	// Path is the database file path.
	Path string

	// MaxOpenConns is the maximum number of open connections to the database.
	// Default: 10
	MaxOpenConns int

	// MaxIdleConns is the maximum number of idle connections.
	// Default: 5
	MaxIdleConns int

	// WALMode enables Write-Ahead Logging mode for better concurrency.
	// Default: true
	WALMode bool

	// BusyTimeout is the duration to wait when the database is locked.
	// Default: 5 seconds
	BusyTimeout time.Duration
}

// DefaultSQLiteConfig returns the default SQLite configuration.
func DefaultSQLiteConfig() *SQLiteConfig {
	return &SQLiteConfig{
		Path:         "data/evidence.db",
		MaxOpenConns: 10,
		MaxIdleConns: 5,
		WALMode:      true,
		BusyTimeout:  5 * time.Second,
	}
}

// SQLiteStorage implements the Storage interface using SQLite.
type SQLiteStorage struct {
	db            *sql.DB
	config        *SQLiteConfig
	preparedStmts map[string]*sql.Stmt
	mu            sync.RWMutex
	logger        *slog.Logger
}

// NewSQLiteStorage creates a new SQLite storage backend.
// It initializes the database schema and enables WAL mode if configured.
func NewSQLiteStorage(config *SQLiteConfig) (*SQLiteStorage, error) {
	if config == nil {
		config = DefaultSQLiteConfig()
	}

	logger := slog.Default().With("component", "evidence.storage.sqlite")

	// Open database connection
	db, err := sql.Open("sqlite3", config.Path)
	if err != nil {
		return nil, evidence.NewStorageError("sqlite", "open", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)

	s := &SQLiteStorage{
		db:            db,
		config:        config,
		preparedStmts: make(map[string]*sql.Stmt),
		logger:        logger,
	}

	// Initialize database
	if err := s.initialize(); err != nil {
		db.Close()
		return nil, err
	}

	logger.Info("SQLite storage initialized",
		"path", config.Path,
		"wal_mode", config.WALMode,
		"max_open_conns", config.MaxOpenConns,
	)

	return s, nil
}

// initialize sets up the database schema and enables WAL mode.
func (s *SQLiteStorage) initialize() error {
	// Enable WAL mode if configured
	if s.config.WALMode {
		_, err := s.db.Exec("PRAGMA journal_mode=WAL;")
		if err != nil {
			return evidence.NewStorageError("sqlite", "enable_wal", err)
		}
		s.logger.Debug("WAL mode enabled")
	}

	// Set busy timeout
	busyTimeoutMs := s.config.BusyTimeout.Milliseconds()
	_, err := s.db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d;", busyTimeoutMs))
	if err != nil {
		return evidence.NewStorageError("sqlite", "set_busy_timeout", err)
	}

	// Create schema
	_, err = s.db.Exec(Schema)
	if err != nil {
		return evidence.NewStorageError("sqlite", "create_schema", err)
	}
	s.logger.Debug("database schema created")

	// Insert schema version
	_, err = s.db.Exec(InsertSchemaVersion, SchemaVersion)
	if err != nil {
		return evidence.NewStorageError("sqlite", "insert_schema_version", err)
	}

	// Verify schema version
	var version int
	err = s.db.QueryRow(GetSchemaVersion).Scan(&version)
	if err != nil && err != sql.ErrNoRows {
		return evidence.NewStorageError("sqlite", "get_schema_version", err)
	}

	if version != SchemaVersion {
		return evidence.NewStorageError("sqlite", "schema_version_mismatch",
			fmt.Errorf("expected schema version %d, got %d", SchemaVersion, version))
	}

	s.logger.Debug("schema version verified", "version", version)

	return nil
}

// Store persists an evidence record to the database.
func (s *SQLiteStorage) Store(ctx context.Context, record *evidence.EvidenceRecord) error {
	// Marshal JSON fields
	requestHeaders, _ := json.Marshal(record.RequestHeaders)
	toolsUsed, _ := json.Marshal(record.ToolsUsed)
	piiTypes, _ := json.Marshal(record.PIITypes)
	matchedRules, _ := json.Marshal(record.MatchedRules)

	// Insert evidence record
	query := `
		INSERT INTO evidence (
			id, request_id,
			request_time, policy_eval_time, provider_call_time, response_time, recorded_time,
			request_hash, request_method, request_path, request_headers,
			model, provider, messages, system_prompt, user_prompt, tools_used,
			estimated_tokens, estimated_cost, risk_score, complexity_score, pii_detected, pii_types,
			policy_decision, matched_rules, block_reason, policy_version,
			response_hash, response_status,
			response_content, finish_reason,
			prompt_tokens, completion_tokens, total_tokens, actual_cost,
			provider_latency, provider_model,
			user_id, api_key, ip_address,
			error, error_type,
			turn_number, context_usage
		) VALUES (
			?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
		)
	`

	// Convert empty strings to NULL for optional fields
	var errorVal, errorTypeVal interface{}
	if record.Error == "" {
		errorVal = nil
	} else {
		errorVal = record.Error
	}
	if record.ErrorType == "" {
		errorTypeVal = nil
	} else {
		errorTypeVal = record.ErrorType
	}

	_, err := s.db.ExecContext(ctx, query,
		record.ID, record.RequestID,
		record.RequestTime, record.PolicyEvalTime, record.ProviderCallTime, record.ResponseTime, record.RecordedTime,
		record.RequestHash, record.RequestMethod, record.RequestPath, string(requestHeaders),
		record.Model, record.Provider, record.Messages, record.SystemPrompt, record.UserPrompt, string(toolsUsed),
		record.EstimatedTokens, record.EstimatedCost, record.RiskScore, record.ComplexityScore, record.PIIDetected, string(piiTypes),
		record.PolicyDecision, string(matchedRules), record.BlockReason, record.PolicyVersion,
		record.ResponseHash, record.ResponseStatus,
		record.ResponseContent, record.FinishReason,
		record.PromptTokens, record.CompletionTokens, record.TotalTokens, record.ActualCost,
		record.ProviderLatency.Milliseconds(), record.ProviderModel,
		record.UserID, record.APIKey, record.IPAddress,
		errorVal, errorTypeVal,
		record.TurnNumber, record.ContextUsage,
	)

	if err != nil {
		return evidence.NewStorageError("sqlite", "store", err)
	}

	return nil
}

// Query retrieves evidence records matching the query filters.
func (s *SQLiteStorage) Query(ctx context.Context, query *evidence.Query) ([]*evidence.EvidenceRecord, error) {
	// Build WHERE clause and collect args
	whereClause, args := s.buildWhereClause(query)

	// Build complete query
	sqlQuery := "SELECT * FROM evidence"
	if whereClause != "" {
		sqlQuery += " WHERE " + whereClause
	}

	// Add sorting
	sortBy := "request_time"
	sortOrder := "DESC"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	if query.SortOrder != "" {
		sortOrder = query.SortOrder
	}
	sqlQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	limit := 100
	if query.Limit > 0 {
		limit = query.Limit
	}
	sqlQuery += fmt.Sprintf(" LIMIT %d", limit)

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	// Execute query
	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, evidence.NewStorageError("sqlite", "query", err)
	}
	defer rows.Close()

	// Scan results
	records := []*evidence.EvidenceRecord{}
	for rows.Next() {
		record, err := s.scanRow(rows)
		if err != nil {
			return nil, evidence.NewStorageError("sqlite", "scan", err)
		}
		records = append(records, record)
	}

	if err := rows.Err(); err != nil {
		return nil, evidence.NewStorageError("sqlite", "query", err)
	}

	return records, nil
}

// QueryStream returns a channel of evidence records for memory-efficient streaming.
// Use this for large result sets to avoid loading everything in memory.
// The channels will be closed when the query completes or errors.
func (s *SQLiteStorage) QueryStream(ctx context.Context, query *evidence.Query) (<-chan *evidence.EvidenceRecord, <-chan error, error) {
	recordsCh := make(chan *evidence.EvidenceRecord, 100) // Buffer 100 records
	errCh := make(chan error, 1)

	// Build WHERE clause and collect args
	whereClause, args := s.buildWhereClause(query)

	// Build complete query
	sqlQuery := "SELECT * FROM evidence"
	if whereClause != "" {
		sqlQuery += " WHERE " + whereClause
	}

	// Add sorting
	sortBy := "request_time"
	sortOrder := "DESC"
	if query.SortBy != "" {
		sortBy = query.SortBy
	}
	if query.SortOrder != "" {
		sortOrder = query.SortOrder
	}
	sqlQuery += fmt.Sprintf(" ORDER BY %s %s", sortBy, sortOrder)

	// Add pagination
	limit := 100
	if query.Limit > 0 {
		limit = query.Limit
	}
	sqlQuery += fmt.Sprintf(" LIMIT %d", limit)

	if query.Offset > 0 {
		sqlQuery += fmt.Sprintf(" OFFSET %d", query.Offset)
	}

	// Start goroutine to stream results
	go func() {
		defer close(recordsCh)
		defer close(errCh)

		// Execute query
		rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
		if err != nil {
			errCh <- evidence.NewStorageError("sqlite", "query_stream", err)
			return
		}
		defer rows.Close()

		// Stream rows
		for rows.Next() {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			record, err := s.scanRow(rows)
			if err != nil {
				errCh <- evidence.NewStorageError("sqlite", "scan", err)
				return
			}

			// Send record to channel (also check for context cancellation)
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case recordsCh <- record:
				// Record sent successfully
			}
		}

		// Check for any row iteration errors
		if err := rows.Err(); err != nil {
			errCh <- evidence.NewStorageError("sqlite", "query_stream", err)
		}
	}()

	return recordsCh, errCh, nil
}

// Count returns the number of evidence records matching the query filters.
func (s *SQLiteStorage) Count(ctx context.Context, query *evidence.Query) (int64, error) {
	// Build WHERE clause and collect args
	whereClause, args := s.buildWhereClause(query)

	// Build count query
	sqlQuery := "SELECT COUNT(*) FROM evidence"
	if whereClause != "" {
		sqlQuery += " WHERE " + whereClause
	}

	// Execute query
	var count int64
	err := s.db.QueryRowContext(ctx, sqlQuery, args...).Scan(&count)
	if err != nil {
		return 0, evidence.NewStorageError("sqlite", "count", err)
	}

	return count, nil
}

// Delete removes evidence records matching the query filters.
// Returns the number of records deleted.
func (s *SQLiteStorage) Delete(ctx context.Context, query *evidence.Query) (int64, error) {
	// Build WHERE clause and collect args
	whereClause, args := s.buildWhereClause(query)

	// Build delete query
	sqlQuery := "DELETE FROM evidence"
	if whereClause != "" {
		sqlQuery += " WHERE " + whereClause
	}

	// Execute query
	result, err := s.db.ExecContext(ctx, sqlQuery, args...)
	if err != nil {
		return 0, evidence.NewStorageError("sqlite", "delete", err)
	}

	// Get number of rows deleted
	count, err := result.RowsAffected()
	if err != nil {
		return 0, evidence.NewStorageError("sqlite", "delete", err)
	}

	return count, nil
}

// Close releases resources held by the storage backend.
func (s *SQLiteStorage) Close() error {
	// Close prepared statements
	s.mu.Lock()
	for _, stmt := range s.preparedStmts {
		stmt.Close()
	}
	s.mu.Unlock()

	// Close database connection
	if err := s.db.Close(); err != nil {
		return evidence.NewStorageError("sqlite", "close", err)
	}

	s.logger.Info("SQLite storage closed")
	return nil
}

// buildWhereClause builds a SQL WHERE clause from query filters.
// Returns the WHERE clause (without "WHERE" keyword) and the query arguments.
func (s *SQLiteStorage) buildWhereClause(query *evidence.Query) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	// Time range filter
	if query.StartTime != nil {
		conditions = append(conditions, "request_time >= ?")
		args = append(args, *query.StartTime)
	}
	if query.EndTime != nil {
		conditions = append(conditions, "request_time <= ?")
		args = append(args, *query.EndTime)
	}

	// User/API key filter
	if query.UserID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, query.UserID)
	}
	if query.APIKey != "" {
		conditions = append(conditions, "api_key = ?")
		args = append(args, query.APIKey)
	}

	// Provider/model filter
	if query.Provider != "" {
		conditions = append(conditions, "provider = ?")
		args = append(args, query.Provider)
	}
	if query.Model != "" {
		conditions = append(conditions, "model = ?")
		args = append(args, query.Model)
	}

	// Policy filter
	if query.PolicyDecision != "" {
		conditions = append(conditions, "policy_decision = ?")
		args = append(args, query.PolicyDecision)
	}
	if query.PolicyID != "" {
		conditions = append(conditions, "matched_rules LIKE ?")
		args = append(args, "%"+query.PolicyID+"%")
	}
	if query.RuleID != "" {
		conditions = append(conditions, "matched_rules LIKE ?")
		args = append(args, "%"+query.RuleID+"%")
	}

	// Cost thresholds
	if query.MinCost != nil {
		conditions = append(conditions, "actual_cost >= ?")
		args = append(args, *query.MinCost)
	}
	if query.MaxCost != nil {
		conditions = append(conditions, "actual_cost <= ?")
		args = append(args, *query.MaxCost)
	}

	// Token thresholds
	if query.MinTokens != nil {
		conditions = append(conditions, "total_tokens >= ?")
		args = append(args, *query.MinTokens)
	}
	if query.MaxTokens != nil {
		conditions = append(conditions, "total_tokens <= ?")
		args = append(args, *query.MaxTokens)
	}

	// Status filter
	if query.Status != "" {
		switch query.Status {
		case "success":
			conditions = append(conditions, "error IS NULL")
		case "error":
			conditions = append(conditions, "error IS NOT NULL")
		case "blocked":
			conditions = append(conditions, "policy_decision = ?")
			args = append(args, "block")
		}
	}

	// Join conditions with AND
	whereClause := ""
	if len(conditions) > 0 {
		for i, condition := range conditions {
			if i > 0 {
				whereClause += " AND "
			}
			whereClause += condition
		}
	}

	return whereClause, args
}

// scanRow scans a database row into an EvidenceRecord.
func (s *SQLiteStorage) scanRow(row *sql.Rows) (*evidence.EvidenceRecord, error) {
	var record evidence.EvidenceRecord
	var requestHeaders, toolsUsed, piiTypes, matchedRules string
	var providerLatencyMs int64
	var errorVal, errorTypeVal sql.NullString

	err := row.Scan(
		&record.ID, &record.RequestID,
		&record.RequestTime, &record.PolicyEvalTime, &record.ProviderCallTime, &record.ResponseTime, &record.RecordedTime,
		&record.RequestHash, &record.RequestMethod, &record.RequestPath, &requestHeaders,
		&record.Model, &record.Provider, &record.Messages, &record.SystemPrompt, &record.UserPrompt, &toolsUsed,
		&record.EstimatedTokens, &record.EstimatedCost, &record.RiskScore, &record.ComplexityScore, &record.PIIDetected, &piiTypes,
		&record.PolicyDecision, &matchedRules, &record.BlockReason, &record.PolicyVersion,
		&record.ResponseHash, &record.ResponseStatus,
		&record.ResponseContent, &record.FinishReason,
		&record.PromptTokens, &record.CompletionTokens, &record.TotalTokens, &record.ActualCost,
		&providerLatencyMs, &record.ProviderModel,
		&record.UserID, &record.APIKey, &record.IPAddress,
		&errorVal, &errorTypeVal,
		&record.TurnNumber, &record.ContextUsage,
	)
	if err != nil {
		return nil, err
	}

	// Convert NULL strings back to empty strings
	if errorVal.Valid {
		record.Error = errorVal.String
	}
	if errorTypeVal.Valid {
		record.ErrorType = errorTypeVal.String
	}

	// Unmarshal JSON fields
	if requestHeaders != "" {
		json.Unmarshal([]byte(requestHeaders), &record.RequestHeaders)
	}
	if toolsUsed != "" {
		json.Unmarshal([]byte(toolsUsed), &record.ToolsUsed)
	}
	if piiTypes != "" {
		json.Unmarshal([]byte(piiTypes), &record.PIITypes)
	}
	if matchedRules != "" {
		json.Unmarshal([]byte(matchedRules), &record.MatchedRules)
	}

	// Convert provider latency from milliseconds
	record.ProviderLatency = time.Duration(providerLatencyMs) * time.Millisecond

	return &record, nil
}
