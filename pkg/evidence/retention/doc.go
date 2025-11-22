// Package retention provides retention policy enforcement for evidence records.
//
// # Retention Policy
//
// The retention package automatically prunes old evidence records based on age:
//
//   - Configurable retention period (days)
//   - Scheduled pruning (cron expression)
//   - Optional archiving before deletion
//   - Configurable max record count
//
// # Basic Usage
//
//	// Create retention pruner
//	pruner := retention.NewPruner(storage, &retention.Config{
//	    RetentionDays: 90,
//	    PruneSchedule: "0 3 * * *", // Daily at 3 AM
//	    ArchiveBeforeDelete: true,
//	    ArchivePath: "data/archives/",
//	})
//
//	// Start background pruning
//	if err := pruner.Start(ctx); err != nil {
//	    log.Fatal(err)
//	}
//	defer pruner.Stop()
//
//	// Check next scheduled pruning time
//	if next := pruner.NextPruning(); next != nil {
//	    log.Printf("Next pruning scheduled for: %s", next)
//	}
//
// # Manual Pruning
//
// You can also trigger pruning manually:
//
//	// Prune records older than retention period
//	deleted, err := pruner.Prune(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	log.Printf("Deleted %d old evidence records", deleted)
//
// # Archiving
//
// If archiving is enabled, evidence records are exported to JSON before deletion:
//
//   - Archives are stored in the configured archive path
//   - Archive files are named by date: evidence-2024-01-15.json
//   - Archives contain all deleted records in JSON format
//
// # Retention Period
//
// The retention period is specified in days:
//
//   - 0 days: Keep evidence forever (no pruning)
//   - 30 days: Delete evidence older than 30 days
//   - 90 days: Delete evidence older than 90 days (default)
//   - 365 days: Delete evidence older than 1 year
//
// # Scheduling
//
// The pruner runs on a cron schedule:
//
//   - "0 3 * * *": Daily at 3 AM (default)
//   - "0 0 * * 0": Weekly on Sunday at midnight
//   - "0 0 1 * *": Monthly on the 1st at midnight
//   - "0 */6 * * *": Every 6 hours
//   - "*/1 * * * *": Every minute (testing only)
//
// # Scheduler Features
//
// The scheduler provides:
//
//   - Automatic pruning based on cron schedule
//   - Graceful shutdown (waits for running jobs to complete)
//   - Context-based cancellation support
//   - Next run time queries for monitoring
//
// If no schedule is configured (empty PruneSchedule), the scheduler
// does nothing and Start() returns immediately without error.
package retention
