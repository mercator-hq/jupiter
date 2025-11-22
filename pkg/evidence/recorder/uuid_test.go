package recorder

import (
	"sync"
	"testing"

	"github.com/google/uuid"
)

// TestUUIDv4_Generation tests UUID v4 generation.
func TestUUIDv4_Generation(t *testing.T) {
	// Generate a UUID
	id := uuid.New()

	if id == uuid.Nil {
		t.Error("Generated UUID should not be nil")
	}

	// Verify format (UUID v4 has specific version bits)
	idStr := id.String()
	if len(idStr) != 36 {
		t.Errorf("Expected UUID string length 36, got %d", len(idStr))
	}

	// Verify it's a valid UUID
	parsed, err := uuid.Parse(idStr)
	if err != nil {
		t.Fatalf("Failed to parse generated UUID: %v", err)
	}

	if parsed != id {
		t.Error("Parsed UUID does not match original")
	}

	t.Logf("Generated UUID: %s", idStr)
}

// TestUUIDv4_Uniqueness tests UUID uniqueness.
func TestUUIDv4_Uniqueness(t *testing.T) {
	// Generate multiple UUIDs and verify they're all unique
	count := 10000
	uuids := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		id := uuid.New().String()

		if uuids[id] {
			t.Fatalf("Duplicate UUID found: %s", id)
		}

		uuids[id] = true
	}

	if len(uuids) != count {
		t.Errorf("Expected %d unique UUIDs, got %d", count, len(uuids))
	}

	t.Logf("Generated %d unique UUIDs", count)
}

// TestUUIDv4_ConcurrentGeneration tests UUID generation under concurrent load.
func TestUUIDv4_ConcurrentGeneration(t *testing.T) {
	count := 1000
	goroutines := 10
	totalUUIDs := count * goroutines

	uuidChan := make(chan string, totalUUIDs)
	var wg sync.WaitGroup

	// Launch multiple goroutines generating UUIDs concurrently
	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < count; i++ {
				uuidChan <- uuid.New().String()
			}
		}()
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(uuidChan)

	// Verify all UUIDs are unique
	uuids := make(map[string]bool, totalUUIDs)
	for id := range uuidChan {
		if uuids[id] {
			t.Fatalf("Duplicate UUID found in concurrent generation: %s", id)
		}
		uuids[id] = true
	}

	if len(uuids) != totalUUIDs {
		t.Errorf("Expected %d unique UUIDs, got %d", totalUUIDs, len(uuids))
	}

	t.Logf("Generated %d unique UUIDs across %d concurrent goroutines", totalUUIDs, goroutines)
}

// TestUUIDv4_Format tests UUID format compliance.
func TestUUIDv4_Format(t *testing.T) {
	id := uuid.New()

	// Verify version field (should be 4 for UUID v4)
	version := id.Version()
	if version != 4 {
		t.Errorf("Expected UUID version 4, got %d", version)
	}

	// Verify variant field
	variant := id.Variant()
	if variant != uuid.RFC4122 {
		t.Errorf("Expected RFC4122 variant, got %d", variant)
	}

	t.Logf("UUID v4 format verified: %s (version=%d, variant=%d)", id, version, variant)
}

// TestUUIDv4_ParseError tests error handling for invalid UUIDs.
func TestUUIDv4_ParseError(t *testing.T) {
	invalidUUIDs := []string{
		"",
		"invalid",
		"123",
		"12345678-1234-1234-1234-12345678901",   // Too short
		"12345678-1234-1234-1234-1234567890123", // Too long
		"xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx",  // Invalid hex
	}

	for _, invalid := range invalidUUIDs {
		_, err := uuid.Parse(invalid)
		if err == nil {
			t.Errorf("Expected error parsing invalid UUID '%s', got nil", invalid)
		} else {
			t.Logf("Correctly rejected invalid UUID '%s': %v", invalid, err)
		}
	}
}

// TestUUIDv4_Collision tests extremely unlikely collision scenario.
func TestUUIDv4_Collision(t *testing.T) {
	// This test documents that UUID collision is astronomically unlikely
	// UUID v4 has 122 bits of randomness, collision probability is negligible

	// For 1 trillion UUIDs, collision probability is ~0.000000001
	// We'll generate 1 million to demonstrate uniqueness

	if testing.Short() {
		t.Skip("Skipping collision test in short mode")
	}

	count := 1000000 // 1 million
	uuids := make(map[string]bool, count)

	for i := 0; i < count; i++ {
		id := uuid.New().String()

		if uuids[id] {
			// This would be extremely unexpected
			t.Fatalf("UUID collision detected after %d generations: %s", i, id)
		}

		uuids[id] = true
	}

	t.Logf("Generated %d unique UUIDs with no collisions", count)
	t.Logf("Collision probability for %d UUIDs: ~%.15f", count, float64(count)*float64(count)/(2*5.316911983139664e+36))
}

// TestUUIDv4_PerformanceTarget tests UUID generation performance.
func TestUUIDv4_PerformanceTarget(t *testing.T) {
	// Target: Generate UUIDs fast enough for 1000+ evidence records/sec
	// This means <1ms per UUID

	count := 10000

	start := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < count; i++ {
			_ = uuid.New()
		}
	})

	nsPerUUID := start.NsPerOp()
	msPerUUID := float64(nsPerUUID) / 1000000

	t.Logf("UUID generation: %.3f ms per UUID", msPerUUID)
	t.Logf("UUID throughput: %.0f UUIDs/sec", 1000/msPerUUID)

	if msPerUUID > 1.0 {
		t.Logf("Warning: UUID generation took %.3f ms (target: <1ms)", msPerUUID)
	}
}

// BenchmarkUUIDv4_Generation benchmarks UUID generation.
func BenchmarkUUIDv4_Generation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}

// BenchmarkUUIDv4_ConcurrentGeneration benchmarks concurrent UUID generation.
func BenchmarkUUIDv4_ConcurrentGeneration(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = uuid.New()
		}
	})
}

// BenchmarkUUIDv4_StringConversion benchmarks UUID to string conversion.
func BenchmarkUUIDv4_StringConversion(b *testing.B) {
	id := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = id.String()
	}
}

// BenchmarkUUIDv4_Parsing benchmarks UUID parsing from string.
func BenchmarkUUIDv4_Parsing(b *testing.B) {
	idStr := uuid.New().String()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = uuid.Parse(idStr)
	}
}
