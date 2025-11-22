package retention

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/export"
)

// Config contains configuration for the retention pruner.
type Config struct {
	// RetentionDays is the number of days to retain evidence.
	// 0 means keep evidence forever (no pruning).
	RetentionDays int

	// PruneSchedule is a cron expression for scheduling pruning.
	// Example: "0 3 * * *" (daily at 3 AM)
	PruneSchedule string

	// ArchiveBeforeDelete enables archiving evidence before deletion.
	ArchiveBeforeDelete bool

	// ArchivePath is the directory to store archived evidence.
	ArchivePath string

	// MaxRecords is the maximum number of records to keep.
	// 0 means unlimited.
	MaxRecords int64
}

// DefaultConfig returns the default retention configuration.
func DefaultConfig() *Config {
	return &Config{
		RetentionDays:       90,
		PruneSchedule:       "0 3 * * *",
		ArchiveBeforeDelete: false,
		ArchivePath:         "data/archives/",
		MaxRecords:          0,
	}
}

// Pruner enforces retention policies on evidence records.
type Pruner struct {
	storage   evidence.Storage
	config    *Config
	logger    *slog.Logger
	scheduler *Scheduler
}

// NewPruner creates a new retention pruner.
func NewPruner(storage evidence.Storage, config *Config) *Pruner {
	if config == nil {
		config = DefaultConfig()
	}

	pruner := &Pruner{
		storage: storage,
		config:  config,
		logger:  slog.Default().With("component", "evidence.retention"),
	}

	// Create scheduler
	pruner.scheduler = NewScheduler(pruner)

	return pruner
}

// Prune deletes evidence records older than the retention period
// or exceeding the max record count.
//
// Pruning happens in two phases:
// 1. Age-based: Delete records older than retention_days
// 2. Count-based: If total records > max_records, delete oldest
//
// Both can run together (e.g., delete old records AND limit total count).
// Returns the total number of records deleted.
func (p *Pruner) Prune(ctx context.Context) (int64, error) {
	var totalDeleted int64

	// Phase 1: Prune by retention period
	if p.config.RetentionDays > 0 {
		deleted, err := p.pruneByAge(ctx)
		if err != nil {
			return totalDeleted, fmt.Errorf("prune by age failed: %w", err)
		}
		totalDeleted += deleted
		p.logger.Info("pruned records by age",
			"deleted_count", deleted,
			"retention_days", p.config.RetentionDays,
		)
	}

	// Phase 2: Prune by max record count
	if p.config.MaxRecords > 0 {
		deleted, err := p.pruneByCount(ctx)
		if err != nil {
			return totalDeleted, fmt.Errorf("prune by count failed: %w", err)
		}
		totalDeleted += deleted
		p.logger.Info("pruned records by count",
			"deleted_count", deleted,
			"max_records", p.config.MaxRecords,
		)
	}

	if totalDeleted == 0 {
		p.logger.Debug("no records pruned",
			"retention_days", p.config.RetentionDays,
			"max_records", p.config.MaxRecords,
		)
	} else {
		p.logger.Info("evidence pruning completed",
			"total_deleted", totalDeleted,
			"retention_days", p.config.RetentionDays,
			"max_records", p.config.MaxRecords,
		)
	}

	return totalDeleted, nil
}

// pruneByAge deletes records older than the retention period.
func (p *Pruner) pruneByAge(ctx context.Context) (int64, error) {
	cutoff := time.Now().AddDate(0, 0, -p.config.RetentionDays)

	p.logger.Debug("pruning by age",
		"cutoff_time", cutoff,
		"retention_days", p.config.RetentionDays,
	)

	// Query for records older than cutoff
	query := &evidence.Query{
		EndTime: &cutoff,
	}

	// Archive before delete if configured
	if p.config.ArchiveBeforeDelete {
		if err := p.archive(ctx, query); err != nil {
			return 0, evidence.NewRetentionError(p.config.RetentionDays, err)
		}
	}

	// Delete old records
	deleted, err := p.storage.Delete(ctx, query)
	if err != nil {
		return 0, evidence.NewRetentionError(p.config.RetentionDays, err)
	}

	return deleted, nil
}

// pruneByCount deletes oldest records if total count exceeds max_records.
func (p *Pruner) pruneByCount(ctx context.Context) (int64, error) {
	// Count total records
	count, err := p.storage.Count(ctx, &evidence.Query{})
	if err != nil {
		return 0, fmt.Errorf("failed to count records: %w", err)
	}

	if count <= p.config.MaxRecords {
		p.logger.Debug("record count within limit",
			"current", count,
			"max", p.config.MaxRecords,
		)
		return 0, nil
	}

	// Calculate how many to delete
	toDelete := count - p.config.MaxRecords

	p.logger.Info("record count exceeds limit, pruning oldest",
		"current_count", count,
		"max_records", p.config.MaxRecords,
		"to_delete", toDelete,
	)

	// Query ALL records since storage backend may not support sorting
	allRecords, err := p.storage.Query(ctx, &evidence.Query{})
	if err != nil {
		return 0, fmt.Errorf("failed to query records: %w", err)
	}

	if len(allRecords) == 0 {
		p.logger.Debug("no records found to delete")
		return 0, nil
	}

	// Sort records by timestamp (oldest first)
	sortRecordsByTime(allRecords)

	// Determine how many to actually delete (in case count changed)
	actualToDelete := len(allRecords) - int(p.config.MaxRecords)
	if actualToDelete <= 0 {
		p.logger.Debug("record count within limit after query")
		return 0, nil
	}
	if actualToDelete > len(allRecords) {
		actualToDelete = len(allRecords)
	}

	// Get the cutoff time: time of the last record to delete
	cutoffTime := allRecords[actualToDelete-1].RequestTime

	p.logger.Debug("calculated cutoff time for count-based pruning",
		"cutoff_time", cutoffTime,
		"records_to_delete", actualToDelete,
	)

	// Create delete query for records older than or equal to cutoff
	deleteQuery := &evidence.Query{
		EndTime: &cutoffTime,
	}

	// Archive if configured
	if p.config.ArchiveBeforeDelete {
		recordsToArchive := allRecords[:actualToDelete]
		if err := p.archiveRecords(ctx, recordsToArchive); err != nil {
			return 0, fmt.Errorf("archive failed: %w", err)
		}
	}

	// Delete records
	deleted, err := p.storage.Delete(ctx, deleteQuery)
	if err != nil {
		return 0, fmt.Errorf("delete failed: %w", err)
	}

	return deleted, nil
}

// sortRecordsByTime sorts evidence records by RequestTime in ascending order (oldest first).
func sortRecordsByTime(records []*evidence.EvidenceRecord) {
	// Simple bubble sort - fine for test environment
	// In production, consider using sort.Slice
	for i := 0; i < len(records)-1; i++ {
		for j := i + 1; j < len(records); j++ {
			if records[i].RequestTime.After(records[j].RequestTime) {
				records[i], records[j] = records[j], records[i]
			}
		}
	}
}

// archiveRecords exports a list of evidence records to JSON before deletion.
func (p *Pruner) archiveRecords(ctx context.Context, records []*evidence.EvidenceRecord) error {
	if len(records) == 0 {
		return nil
	}

	p.logger.Info("archiving evidence records before deletion",
		"record_count", len(records),
	)

	// Create archive directory if it doesn't exist
	if err := os.MkdirAll(p.config.ArchivePath, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Create archive file
	archiveFile := filepath.Join(p.config.ArchivePath, fmt.Sprintf("evidence-count-%s.json", time.Now().Format("2006-01-02-150405")))
	f, err := os.Create(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer f.Close()

	// Export records to JSON
	exporter := export.NewJSONExporter(true)
	if err := exporter.Export(ctx, records, f); err != nil {
		return fmt.Errorf("failed to export records to archive: %w", err)
	}

	p.logger.Info("evidence records archived",
		"archive_file", archiveFile,
		"record_count", len(records),
	)

	return nil
}

// archive exports evidence records to JSON before deletion.
func (p *Pruner) archive(ctx context.Context, query *evidence.Query) error {
	p.logger.Info("archiving evidence before deletion")

	// Query records to archive
	records, err := p.storage.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query records for archiving: %w", err)
	}

	if len(records) == 0 {
		p.logger.Debug("no records to archive")
		return nil
	}

	// Create archive directory if it doesn't exist
	if err := os.MkdirAll(p.config.ArchivePath, 0755); err != nil {
		return fmt.Errorf("failed to create archive directory: %w", err)
	}

	// Create archive file
	archiveFile := filepath.Join(p.config.ArchivePath, fmt.Sprintf("evidence-%s.json", time.Now().Format("2006-01-02")))
	f, err := os.Create(archiveFile)
	if err != nil {
		return fmt.Errorf("failed to create archive file: %w", err)
	}
	defer f.Close()

	// Export records to JSON
	exporter := export.NewJSONExporter(true)
	if err := exporter.Export(ctx, records, f); err != nil {
		return fmt.Errorf("failed to export records to archive: %w", err)
	}

	p.logger.Info("evidence archived",
		"archive_file", archiveFile,
		"record_count", len(records),
	)

	return nil
}

// Start starts the automatic pruning scheduler.
// Call this when starting the application.
func (p *Pruner) Start(ctx context.Context) error {
	return p.scheduler.Start(ctx)
}

// Stop stops the automatic pruning scheduler.
// Call this during graceful shutdown.
func (p *Pruner) Stop() {
	p.scheduler.Stop()
}

// NextPruning returns the time of the next scheduled pruning.
func (p *Pruner) NextPruning() *time.Time {
	return p.scheduler.NextRun()
}
