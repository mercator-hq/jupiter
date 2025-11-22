package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite" // SQLite driver
)

// SQLiteBackend implements Backend using SQLite for persistence.
// This backend provides durable storage with periodic snapshots and is suitable
// for single-instance deployments where persistence across restarts is required.
//
// SQLiteBackend uses a write-ahead log (WAL) for better concurrent performance
// and automatic checkpointing to balance write performance with durability.
type SQLiteBackend struct {
	db               *sql.DB
	dbPath           string
	snapshotInterval time.Duration
	done             chan struct{}
	mu               sync.RWMutex
	closeOnce        sync.Once

	// preparedStatements contains pre-compiled SQL statements for performance
	saveStmt    *sql.Stmt
	loadStmt    *sql.Stmt
	deleteStmt  *sql.Stmt
	listStmt    *sql.Stmt
	cleanupStmt *sql.Stmt
}

// SQLiteBackendConfig configures the SQLite backend.
type SQLiteBackendConfig struct {
	// DBPath is the path to the SQLite database file.
	DBPath string

	// SnapshotInterval is how often to checkpoint the WAL.
	// Default: 5 minutes
	SnapshotInterval time.Duration

	// BusyTimeout is how long to wait for locks before failing.
	// Default: 5 seconds
	BusyTimeout time.Duration
}

// NewSQLiteBackend creates a new SQLite storage backend with default settings.
func NewSQLiteBackend(dbPath string) (*SQLiteBackend, error) {
	return NewSQLiteBackendWithConfig(SQLiteBackendConfig{
		DBPath:           dbPath,
		SnapshotInterval: 5 * time.Minute,
		BusyTimeout:      5 * time.Second,
	})
}

// NewSQLiteBackendWithConfig creates a new SQLite backend with custom configuration.
func NewSQLiteBackendWithConfig(cfg SQLiteBackendConfig) (*SQLiteBackend, error) {
	// Apply defaults
	if cfg.DBPath == "" {
		return nil, fmt.Errorf("db path cannot be empty")
	}
	if cfg.SnapshotInterval == 0 {
		cfg.SnapshotInterval = 5 * time.Minute
	}
	if cfg.BusyTimeout == 0 {
		cfg.BusyTimeout = 5 * time.Second
	}

	// Open database with WAL mode and busy timeout
	dsn := fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=%d&_synchronous=NORMAL",
		cfg.DBPath, int(cfg.BusyTimeout.Milliseconds()))

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(1) // SQLite only supports single writer
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	backend := &SQLiteBackend{
		db:               db,
		dbPath:           cfg.DBPath,
		snapshotInterval: cfg.SnapshotInterval,
		done:             make(chan struct{}),
	}

	// Initialize schema
	if err := backend.initSchema(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	// Prepare statements
	if err := backend.prepareStatements(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to prepare statements: %w", err)
	}

	// Start background checkpoint goroutine
	go backend.checkpointLoop()

	return backend, nil
}

// initSchema creates the database schema if it doesn't exist.
func (s *SQLiteBackend) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS limit_states (
		identifier TEXT NOT NULL,
		dimension TEXT NOT NULL,
		rate_limit_state TEXT,
		budget_state TEXT,
		last_updated INTEGER NOT NULL,
		created_at INTEGER NOT NULL,
		PRIMARY KEY (dimension, identifier)
	);

	CREATE INDEX IF NOT EXISTS idx_last_updated ON limit_states(last_updated);
	CREATE INDEX IF NOT EXISTS idx_dimension ON limit_states(dimension);
	`

	_, err := s.db.Exec(schema)
	return err
}

// prepareStatements prepares SQL statements for reuse.
func (s *SQLiteBackend) prepareStatements() error {
	var err error

	s.saveStmt, err = s.db.Prepare(`
		INSERT INTO limit_states (identifier, dimension, rate_limit_state, budget_state, last_updated, created_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (dimension, identifier) DO UPDATE SET
			rate_limit_state = excluded.rate_limit_state,
			budget_state = excluded.budget_state,
			last_updated = excluded.last_updated
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare save statement: %w", err)
	}

	s.loadStmt, err = s.db.Prepare(`
		SELECT identifier, dimension, rate_limit_state, budget_state, last_updated, created_at
		FROM limit_states
		WHERE identifier = ? AND dimension = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare load statement: %w", err)
	}

	s.deleteStmt, err = s.db.Prepare(`
		DELETE FROM limit_states
		WHERE identifier = ? AND dimension = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}

	s.listStmt, err = s.db.Prepare(`
		SELECT identifier, dimension, rate_limit_state, budget_state, last_updated, created_at
		FROM limit_states
		WHERE dimension = ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare list statement: %w", err)
	}

	s.cleanupStmt, err = s.db.Prepare(`
		DELETE FROM limit_states
		WHERE last_updated < ?
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare cleanup statement: %w", err)
	}

	return nil
}

// Save persists the limit state for an identifier.
func (s *SQLiteBackend) Save(ctx context.Context, state *LimitState) error {
	if state == nil {
		return fmt.Errorf("state cannot be nil")
	}
	if state.Identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if state.Dimension == "" {
		return fmt.Errorf("dimension cannot be empty")
	}

	// Serialize rate limit state
	var rateLimitJSON []byte
	var err error
	if state.RateLimit != nil {
		rateLimitJSON, err = json.Marshal(state.RateLimit)
		if err != nil {
			return fmt.Errorf("failed to marshal rate limit state: %w", err)
		}
	}

	// Serialize budget state
	var budgetJSON []byte
	if state.Budget != nil {
		budgetJSON, err = json.Marshal(state.Budget)
		if err != nil {
			return fmt.Errorf("failed to marshal budget state: %w", err)
		}
	}

	// Update timestamps
	now := time.Now()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = now
	}
	if state.LastUpdated.IsZero() {
		state.LastUpdated = now
	}

	// Execute save
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err = s.saveStmt.ExecContext(ctx,
		state.Identifier,
		state.Dimension,
		string(rateLimitJSON),
		string(budgetJSON),
		state.LastUpdated.Unix(),
		state.CreatedAt.Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to save state: %w", err)
	}

	return nil
}

// Load retrieves the limit state for an identifier and dimension.
func (s *SQLiteBackend) Load(ctx context.Context, identifier string, dimension string) (*LimitState, error) {
	if identifier == "" {
		return nil, fmt.Errorf("identifier cannot be empty")
	}
	if dimension == "" {
		return nil, fmt.Errorf("dimension cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	var (
		rateLimitJSON string
		budgetJSON    string
		lastUpdated   int64
		createdAt     int64
	)

	err := s.loadStmt.QueryRowContext(ctx, identifier, dimension).Scan(
		&identifier,
		&dimension,
		&rateLimitJSON,
		&budgetJSON,
		&lastUpdated,
		&createdAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load state: %w", err)
	}

	state := &LimitState{
		Identifier:  identifier,
		Dimension:   dimension,
		LastUpdated: time.Unix(lastUpdated, 0),
		CreatedAt:   time.Unix(createdAt, 0),
	}

	// Deserialize rate limit state
	if rateLimitJSON != "" {
		state.RateLimit = &RateLimitState{}
		if err := json.Unmarshal([]byte(rateLimitJSON), state.RateLimit); err != nil {
			return nil, fmt.Errorf("failed to unmarshal rate limit state: %w", err)
		}
	}

	// Deserialize budget state
	if budgetJSON != "" {
		state.Budget = &BudgetState{}
		if err := json.Unmarshal([]byte(budgetJSON), state.Budget); err != nil {
			return nil, fmt.Errorf("failed to unmarshal budget state: %w", err)
		}
	}

	return state, nil
}

// Delete removes the limit state for an identifier and dimension.
func (s *SQLiteBackend) Delete(ctx context.Context, identifier string, dimension string) error {
	if identifier == "" {
		return fmt.Errorf("identifier cannot be empty")
	}
	if dimension == "" {
		return fmt.Errorf("dimension cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.deleteStmt.ExecContext(ctx, identifier, dimension)
	if err != nil {
		return fmt.Errorf("failed to delete state: %w", err)
	}

	return nil
}

// List returns all limit states for a dimension.
func (s *SQLiteBackend) List(ctx context.Context, dimension string) ([]*LimitState, error) {
	if dimension == "" {
		return nil, fmt.Errorf("dimension cannot be empty")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.listStmt.QueryContext(ctx, dimension)
	if err != nil {
		return nil, fmt.Errorf("failed to list states: %w", err)
	}
	defer rows.Close()

	var states []*LimitState
	for rows.Next() {
		var (
			identifier    string
			dim           string
			rateLimitJSON string
			budgetJSON    string
			lastUpdated   int64
			createdAt     int64
		)

		if err := rows.Scan(&identifier, &dim, &rateLimitJSON, &budgetJSON, &lastUpdated, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		state := &LimitState{
			Identifier:  identifier,
			Dimension:   dim,
			LastUpdated: time.Unix(lastUpdated, 0),
			CreatedAt:   time.Unix(createdAt, 0),
		}

		// Deserialize rate limit state
		if rateLimitJSON != "" {
			state.RateLimit = &RateLimitState{}
			if err := json.Unmarshal([]byte(rateLimitJSON), state.RateLimit); err != nil {
				return nil, fmt.Errorf("failed to unmarshal rate limit state: %w", err)
			}
		}

		// Deserialize budget state
		if budgetJSON != "" {
			state.Budget = &BudgetState{}
			if err := json.Unmarshal([]byte(budgetJSON), state.Budget); err != nil {
				return nil, fmt.Errorf("failed to unmarshal budget state: %w", err)
			}
		}

		states = append(states, state)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return states, nil
}

// Cleanup removes expired state entries based on retention policy.
func (s *SQLiteBackend) Cleanup(ctx context.Context, olderThan time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result, err := s.cleanupStmt.ExecContext(ctx, olderThan.Unix())
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(deleted), nil
}

// Close releases any resources held by the backend.
// Close is idempotent and safe to call multiple times.
func (s *SQLiteBackend) Close() error {
	var closeErr error

	s.closeOnce.Do(func() {
		// Signal checkpoint goroutine to stop
		close(s.done)

		// Close prepared statements
		if s.saveStmt != nil {
			s.saveStmt.Close()
		}
		if s.loadStmt != nil {
			s.loadStmt.Close()
		}
		if s.deleteStmt != nil {
			s.deleteStmt.Close()
		}
		if s.listStmt != nil {
			s.listStmt.Close()
		}
		if s.cleanupStmt != nil {
			s.cleanupStmt.Close()
		}

		// Close database
		if s.db != nil {
			// Run final checkpoint
			_, _ = s.db.Exec("PRAGMA wal_checkpoint(TRUNCATE)")
			closeErr = s.db.Close()
		}
	})

	return closeErr
}

// checkpointLoop runs periodic WAL checkpoints.
func (s *SQLiteBackend) checkpointLoop() {
	ticker := time.NewTicker(s.snapshotInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Run checkpoint
			_, _ = s.db.Exec("PRAGMA wal_checkpoint(PASSIVE)")
		case <-s.done:
			return
		}
	}
}
