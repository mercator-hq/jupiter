// Package storage provides storage backends for evidence records.
//
// # Storage Backends
//
// The storage package defines the Storage interface and provides multiple
// implementations:
//
//   - SQLite: Embedded database for single-node deployments (MVP)
//   - Memory: In-memory storage for testing
//   - PostgreSQL: High-volume production deployments (Phase 2)
//   - S3: Long-term archival storage (Phase 2)
//
// # SQLite Backend
//
// The SQLite backend provides durable storage with:
//
//   - WAL mode for concurrent reads/writes
//   - Prepared statements for performance
//   - Indexes on frequently queried fields
//   - Connection pooling for concurrent access
//   - Busy timeout for handling locks
//
// # Basic Usage
//
//	// Create SQLite storage
//	storage, err := storage.NewSQLiteStorage(&storage.SQLiteConfig{
//	    Path: "data/evidence.db",
//	    MaxOpenConns: 10,
//	    MaxIdleConns: 5,
//	    WALMode: true,
//	    BusyTimeout: 5 * time.Second,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer storage.Close()
//
//	// Store evidence record
//	err = storage.Store(ctx, record)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Query evidence records
//	query := &evidence.Query{
//	    StartTime: &startTime,
//	    EndTime: &endTime,
//	    UserID: "user-123",
//	    Limit: 100,
//	}
//	records, err := storage.Query(ctx, query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// # Thread Safety
//
// All storage backends are thread-safe and support concurrent access:
//
//   - Store() can be called concurrently from multiple goroutines
//   - Query() can be called concurrently with Store()
//   - WAL mode enables concurrent readers and writers
//
// # Schema Migration
//
// The SQLite storage automatically initializes the database schema on first use.
// Schema version is tracked in the schema_version table for future migrations.
//
// # Performance
//
// The SQLite storage is optimized for high throughput:
//
//   - Prepared statements reduce SQL parsing overhead
//   - Indexes on frequently queried fields enable fast lookups
//   - WAL mode enables concurrent access without blocking
//   - Connection pooling reduces connection overhead
//
// Target performance:
//   - Store: <5ms per record
//   - Query: <100ms for typical queries (with indexes)
//   - Concurrent writes: >1000 writes/sec
package storage
