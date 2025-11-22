// Package evidence provides comprehensive evidence generation and storage for
// LLM proxy activity. It records all request/response pairs as immutable evidence
// records for compliance, audit, and forensics.
//
// # Architecture
//
// The evidence system consists of three layers:
//
//  1. Evidence Recorder - Creates evidence records from proxy events
//  2. Storage Backend - Persists evidence records (SQLite, PostgreSQL, S3)
//  3. Query Engine - Retrieves and filters evidence records
//
// # Evidence Records
//
// Each evidence record captures:
//   - Request metadata (model, provider, user, IP address)
//   - Response metadata (tokens, cost, finish reason)
//   - Policy decisions (matched rules, actions taken)
//   - Cryptographic hashes (SHA-256 of request/response bodies)
//   - Timestamps (request, policy eval, provider call, response)
//   - Error information (if request failed)
//
// # Recording Flow
//
// Evidence is recorded asynchronously to avoid blocking proxy requests:
//
//	Proxy Request → Enriched Request → Policy Decision
//	     ↓
//	Evidence Recorder (async)
//	     ↓
//	Build Evidence Record
//	     ↓
//	Hash Request/Response
//	     ↓
//	Storage Backend (SQLite)
//	     ↓
//	Write to Database (WAL mode)
//
// # Basic Usage
//
//	// Initialize storage backend
//	storage, err := storage.NewSQLiteStorage(&storage.SQLiteConfig{
//	    Path: "data/evidence.db",
//	    WALMode: true,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer storage.Close()
//
//	// Create evidence recorder
//	recorder := recorder.NewRecorder(storage, &recorder.Config{
//	    Enabled: true,
//	    AsyncBuffer: 1000,
//	    HashRequest: true,
//	    HashResponse: true,
//	})
//	defer recorder.Close()
//
//	// Record evidence (async, non-blocking)
//	recorder.RecordRequest(ctx, enrichedReq, policyDecision)
//	recorder.RecordResponse(ctx, enrichedResp)
//
// # Querying Evidence
//
//	// Build query
//	query := &evidence.Query{
//	    StartTime: &startTime,
//	    EndTime: &endTime,
//	    UserID: "user-123",
//	    PolicyDecision: "block",
//	    Limit: 100,
//	}
//
//	// Execute query
//	records, err := storage.Query(ctx, query)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Export to JSON
//	exporter := export.NewJSONExporter(true) // pretty-print
//	exporter.Export(ctx, records, os.Stdout)
//
// # Retention Policies
//
// Evidence can be automatically pruned based on age:
//
//	// Create retention pruner
//	pruner := retention.NewPruner(storage, &retention.Config{
//	    RetentionDays: 90,
//	    PruneSchedule: "0 3 * * *", // Daily at 3 AM
//	    ArchiveBeforeDelete: true,
//	})
//
//	// Start background pruning
//	pruner.Start(ctx)
//	defer pruner.Stop()
//
// # Performance
//
// The evidence system is designed for high throughput:
//   - Async recording: >1000 writes/sec, <5ms per record
//   - Indexed queries: <100ms for typical queries
//   - WAL mode: Concurrent reads/writes without blocking
//   - Prepared statements: Reduced query overhead
//
// # Thread Safety
//
// All evidence types are safe for concurrent use:
//   - Recorder: Thread-safe async channel
//   - Storage: Thread-safe with connection pooling
//   - Query: Stateless, can be executed concurrently
//
// # Storage Backends
//
// The evidence system supports multiple storage backends via the Storage interface:
//   - SQLite (MVP): Single-node, embedded database
//   - PostgreSQL (Phase 2): High-volume production deployments
//   - S3 (Phase 2): Long-term archival storage
//
// Custom storage backends can be implemented by satisfying the Storage interface.
package evidence
