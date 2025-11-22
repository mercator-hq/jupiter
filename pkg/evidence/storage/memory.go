package storage

import (
	"context"
	"sync"

	"mercator-hq/jupiter/pkg/evidence"
)

// MemoryStorage implements the Storage interface using an in-memory map.
// This implementation is intended for testing only and should not be used in production.
type MemoryStorage struct {
	records map[string]*evidence.EvidenceRecord
	mu      sync.RWMutex
}

// NewMemoryStorage creates a new in-memory storage backend.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		records: make(map[string]*evidence.EvidenceRecord),
	}
}

// Store persists an evidence record to memory.
func (s *MemoryStorage) Store(ctx context.Context, record *evidence.EvidenceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create a copy to avoid mutation
	recordCopy := *record
	s.records[record.ID] = &recordCopy

	return nil
}

// Query retrieves evidence records matching the query filters.
func (s *MemoryStorage) Query(ctx context.Context, query *evidence.Query) ([]*evidence.EvidenceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*evidence.EvidenceRecord

	// Filter records
	for _, record := range s.records {
		if s.matchesQuery(record, query) {
			// Create a copy to avoid mutation
			recordCopy := *record
			results = append(results, &recordCopy)
		}
	}

	// Sort results (simple implementation for testing)
	// In production, use more sophisticated sorting

	// Apply pagination
	start := query.Offset
	if start > len(results) {
		return []*evidence.EvidenceRecord{}, nil
	}

	end := start + query.Limit
	if end > len(results) {
		end = len(results)
	}

	if query.Limit > 0 {
		results = results[start:end]
	}

	return results, nil
}

// QueryStream returns a channel of evidence records for memory-efficient streaming.
// Use this for large result sets to avoid loading everything in memory.
// The channels will be closed when the query completes or errors.
func (s *MemoryStorage) QueryStream(ctx context.Context, query *evidence.Query) (<-chan *evidence.EvidenceRecord, <-chan error, error) {
	recordsCh := make(chan *evidence.EvidenceRecord, 100) // Buffer 100 records
	errCh := make(chan error, 1)

	// Start goroutine to stream results
	go func() {
		defer close(recordsCh)
		defer close(errCh)

		s.mu.RLock()
		defer s.mu.RUnlock()

		// Stream filtered records
		count := 0
		for _, record := range s.records {
			// Check for context cancellation
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			default:
			}

			// Check if record matches query
			if !s.matchesQuery(record, query) {
				continue
			}

			// Apply offset
			if count < query.Offset {
				count++
				continue
			}

			// Apply limit
			if query.Limit > 0 && count >= query.Offset+query.Limit {
				break
			}

			// Create a copy to avoid mutation
			recordCopy := *record

			// Send record to channel
			select {
			case <-ctx.Done():
				errCh <- ctx.Err()
				return
			case recordsCh <- &recordCopy:
				count++
			}
		}
	}()

	return recordsCh, errCh, nil
}

// Count returns the number of evidence records matching the query filters.
func (s *MemoryStorage) Count(ctx context.Context, query *evidence.Query) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64

	for _, record := range s.records {
		if s.matchesQuery(record, query) {
			count++
		}
	}

	return count, nil
}

// Delete removes evidence records matching the query filters.
func (s *MemoryStorage) Delete(ctx context.Context, query *evidence.Query) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var deleted int64

	// Find records to delete
	toDelete := []string{}
	for id, record := range s.records {
		if s.matchesQuery(record, query) {
			toDelete = append(toDelete, id)
		}
	}

	// Delete records
	for _, id := range toDelete {
		delete(s.records, id)
		deleted++
	}

	return deleted, nil
}

// Close releases resources held by the storage backend.
func (s *MemoryStorage) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = make(map[string]*evidence.EvidenceRecord)
	return nil
}

// matchesQuery checks if a record matches the query filters.
func (s *MemoryStorage) matchesQuery(record *evidence.EvidenceRecord, query *evidence.Query) bool {
	// Time range filter
	if query.StartTime != nil && record.RequestTime.Before(*query.StartTime) {
		return false
	}
	if query.EndTime != nil && record.RequestTime.After(*query.EndTime) {
		return false
	}

	// User/API key filter
	if query.UserID != "" && record.UserID != query.UserID {
		return false
	}
	if query.APIKey != "" && record.APIKey != query.APIKey {
		return false
	}

	// Provider/model filter
	if query.Provider != "" && record.Provider != query.Provider {
		return false
	}
	if query.Model != "" && record.Model != query.Model {
		return false
	}

	// Policy filter
	if query.PolicyDecision != "" && record.PolicyDecision != query.PolicyDecision {
		return false
	}

	// Cost thresholds
	if query.MinCost != nil && record.ActualCost < *query.MinCost {
		return false
	}
	if query.MaxCost != nil && record.ActualCost > *query.MaxCost {
		return false
	}

	// Token thresholds
	if query.MinTokens != nil && record.TotalTokens < *query.MinTokens {
		return false
	}
	if query.MaxTokens != nil && record.TotalTokens > *query.MaxTokens {
		return false
	}

	// Status filter
	if query.Status != "" {
		switch query.Status {
		case "success":
			if record.Error != "" {
				return false
			}
		case "error":
			if record.Error == "" {
				return false
			}
		case "blocked":
			if record.PolicyDecision != "block" {
				return false
			}
		}
	}

	return true
}

// Clear removes all records from storage (for testing).
func (s *MemoryStorage) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.records = make(map[string]*evidence.EvidenceRecord)
}

// GetByID retrieves a single evidence record by ID (for testing).
func (s *MemoryStorage) GetByID(id string) *evidence.EvidenceRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.records[id]
	if !ok {
		return nil
	}

	// Return a copy
	recordCopy := *record
	return &recordCopy
}

// Size returns the number of records in storage (for testing).
func (s *MemoryStorage) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return len(s.records)
}
