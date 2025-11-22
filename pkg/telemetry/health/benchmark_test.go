package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Benchmark_New benchmarks creating a new health checker.
func Benchmark_New(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = New(5 * time.Second)
	}
}

// Benchmark_RegisterCheck benchmarks registering health checks.
func Benchmark_RegisterCheck(b *testing.B) {
	checker := New(5 * time.Second)
	checkFunc := func(ctx context.Context) error { return nil }

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		checker.RegisterCheck("test", checkFunc)
	}
}

// Benchmark_CheckLiveness benchmarks the liveness check.
func Benchmark_CheckLiveness(b *testing.B) {
	checker := New(5 * time.Second)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckLiveness(ctx)
	}
}

// Benchmark_CheckReadiness_NoChecks benchmarks readiness with no checks.
func Benchmark_CheckReadiness_NoChecks(b *testing.B) {
	checker := New(5 * time.Second)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckReadiness(ctx)
	}
}

// Benchmark_CheckReadiness_OneCheck benchmarks readiness with one check.
func Benchmark_CheckReadiness_OneCheck(b *testing.B) {
	checker := New(5 * time.Second)
	checker.RegisterCheck("test", func(ctx context.Context) error { return nil })
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckReadiness(ctx)
	}
}

// Benchmark_CheckReadiness_FiveChecks benchmarks readiness with five checks.
func Benchmark_CheckReadiness_FiveChecks(b *testing.B) {
	checker := New(5 * time.Second)
	for i := 0; i < 5; i++ {
		checker.RegisterCheck("test", func(ctx context.Context) error { return nil })
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckReadiness(ctx)
	}
}

// Benchmark_CheckReadiness_TenChecks benchmarks readiness with ten checks.
func Benchmark_CheckReadiness_TenChecks(b *testing.B) {
	checker := New(5 * time.Second)
	for i := 0; i < 10; i++ {
		checker.RegisterCheck("test", func(ctx context.Context) error { return nil })
	}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckReadiness(ctx)
	}
}

// Benchmark_CheckReadiness_FailingCheck benchmarks readiness with a failing check.
func Benchmark_CheckReadiness_FailingCheck(b *testing.B) {
	checker := New(5 * time.Second)
	checker.RegisterCheck("failing", func(ctx context.Context) error {
		return errors.New("component unhealthy")
	})
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckReadiness(ctx)
	}
}

// Benchmark_CheckReadiness_SlowCheck benchmarks readiness with a slow check.
func Benchmark_CheckReadiness_SlowCheck(b *testing.B) {
	checker := New(5 * time.Second)
	checker.RegisterCheck("slow", func(ctx context.Context) error {
		time.Sleep(1 * time.Millisecond)
		return nil
	})
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckReadiness(ctx)
	}
}

// Benchmark_LivenessHandler benchmarks the liveness HTTP handler.
func Benchmark_LivenessHandler(b *testing.B) {
	checker := New(5 * time.Second)
	handler := checker.LivenessHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}

// Benchmark_ReadinessHandler benchmarks the readiness HTTP handler.
func Benchmark_ReadinessHandler(b *testing.B) {
	checker := New(5 * time.Second)
	checker.RegisterCheck("test", func(ctx context.Context) error { return nil })
	handler := checker.ReadinessHandler()
	req := httptest.NewRequest(http.MethodGet, "/ready", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}

// Benchmark_VersionHandler benchmarks the version HTTP handler.
func Benchmark_VersionHandler(b *testing.B) {
	handler := VersionHandler("1.0.0", "abc123", "2025-11-20")
	req := httptest.NewRequest(http.MethodGet, "/version", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		handler(rec, req)
	}
}

// Benchmark_GetCheck benchmarks retrieving a check function.
func Benchmark_GetCheck(b *testing.B) {
	checker := New(5 * time.Second)
	checker.RegisterCheck("test", func(ctx context.Context) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.GetCheck("test")
	}
}

// Benchmark_ListChecks benchmarks listing all checks.
func Benchmark_ListChecks(b *testing.B) {
	checker := New(5 * time.Second)
	for i := 0; i < 5; i++ {
		checker.RegisterCheck("test", func(ctx context.Context) error { return nil })
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.ListChecks()
	}
}

// Benchmark_CheckCount benchmarks counting checks.
func Benchmark_CheckCount(b *testing.B) {
	checker := New(5 * time.Second)
	checker.RegisterCheck("test", func(ctx context.Context) error { return nil })

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.CheckCount()
	}
}

// Benchmark_Parallel_CheckReadiness benchmarks concurrent readiness checks.
func Benchmark_Parallel_CheckReadiness(b *testing.B) {
	checker := New(5 * time.Second)
	checker.RegisterCheck("test", func(ctx context.Context) error { return nil })
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = checker.CheckReadiness(ctx)
		}
	})
}

// Benchmark_Parallel_LivenessHandler benchmarks concurrent liveness requests.
func Benchmark_Parallel_LivenessHandler(b *testing.B) {
	checker := New(5 * time.Second)
	handler := checker.LivenessHandler()
	req := httptest.NewRequest(http.MethodGet, "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rec := httptest.NewRecorder()
			handler(rec, req)
		}
	})
}

// Benchmark_FullHealthCheckCycle benchmarks a complete health check cycle.
// This simulates a realistic scenario with liveness, readiness, and version checks.
func Benchmark_FullHealthCheckCycle(b *testing.B) {
	checker := New(5 * time.Second)

	// Register typical component checks
	checker.RegisterCheck("config", func(ctx context.Context) error { return nil })
	checker.RegisterCheck("providers", func(ctx context.Context) error { return nil })
	checker.RegisterCheck("policy", func(ctx context.Context) error { return nil })

	livenessHandler := checker.LivenessHandler()
	readinessHandler := checker.ReadinessHandler()
	versionHandler := VersionHandler("1.0.0", "abc123", "2025-11-20")

	livenessReq := httptest.NewRequest(http.MethodGet, "/health", nil)
	readinessReq := httptest.NewRequest(http.MethodGet, "/ready", nil)
	versionReq := httptest.NewRequest(http.MethodGet, "/version", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate a monitoring system checking all endpoints
		livenessRec := httptest.NewRecorder()
		livenessHandler(livenessRec, livenessReq)

		readinessRec := httptest.NewRecorder()
		readinessHandler(readinessRec, readinessReq)

		versionRec := httptest.NewRecorder()
		versionHandler(versionRec, versionReq)
	}
}
