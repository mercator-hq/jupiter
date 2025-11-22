package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/spf13/cobra"
	"mercator-hq/jupiter/pkg/cli"
)

var benchmarkFlags struct {
	target      string
	duration    time.Duration
	rate        int
	concurrency int
	model       string
	template    string
	report      string
	format      string
}

var benchmarkCmd = &cobra.Command{
	Use:   "benchmark",
	Short: "Load test the proxy",
	Long: `Perform load testing and performance profiling.

The benchmark command generates synthetic LLM requests and sends them to the
proxy server at a configurable rate to measure performance characteristics.

Metrics Collected:
  - Request throughput (requests/sec)
  - Latency percentiles (p50, p95, p99, max)
  - Success/error rates
  - Policy evaluation latency

Note: This is a basic MVP implementation. Full benchmarking with detailed
metrics and reports will be implemented in a future release.

Examples:
  # Basic benchmark
  mercator benchmark --target http://localhost:8080

  # High load test
  mercator benchmark --duration 60s --rate 100 --concurrency 10

  # Profile specific model
  mercator benchmark --model gpt-4 --duration 30s`,
	RunE: runBenchmark,
}

func init() {
	rootCmd.AddCommand(benchmarkCmd)

	benchmarkCmd.Flags().StringVar(&benchmarkFlags.target, "target", "http://localhost:8080", "proxy URL")
	benchmarkCmd.Flags().DurationVar(&benchmarkFlags.duration, "duration", 30*time.Second, "test duration")
	benchmarkCmd.Flags().IntVar(&benchmarkFlags.rate, "rate", 10, "requests per second")
	benchmarkCmd.Flags().IntVar(&benchmarkFlags.concurrency, "concurrency", 1, "concurrent clients")
	benchmarkCmd.Flags().StringVar(&benchmarkFlags.model, "model", "gpt-3.5-turbo", "model to use")
	benchmarkCmd.Flags().StringVar(&benchmarkFlags.template, "template", "", "request template file (JSON)")
	benchmarkCmd.Flags().StringVar(&benchmarkFlags.report, "report", "", "output file for results")
	benchmarkCmd.Flags().StringVar(&benchmarkFlags.format, "format", "text", "output format: text, json")
}

func runBenchmark(cmd *cobra.Command, args []string) error {
	fmt.Println("Mercator Benchmark")
	fmt.Println("==================")
	fmt.Printf("Target: %s\n", benchmarkFlags.target)
	fmt.Printf("Duration: %s\n", benchmarkFlags.duration)
	fmt.Printf("Rate: %d req/s\n", benchmarkFlags.rate)
	fmt.Printf("Concurrency: %d\n", benchmarkFlags.concurrency)
	fmt.Println()

	// Calculate total requests
	totalRequests := int(benchmarkFlags.duration.Seconds()) * benchmarkFlags.rate

	fmt.Println("Running...")
	fmt.Println()

	// Run benchmark
	results := runLoadTest(totalRequests)

	// Display results
	displayResults(results)

	return nil
}

type benchmarkResults struct {
	totalRequests   int
	successfulReqs  int
	failedReqs      int
	duration        time.Duration
	latencies       []time.Duration
	errors          []error
}

func runLoadTest(totalRequests int) *benchmarkResults {
	results := &benchmarkResults{
		totalRequests: totalRequests,
		latencies:     make([]time.Duration, 0, totalRequests),
	}

	var (
		successful int64
		failed     int64
		mu         sync.Mutex
	)

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), benchmarkFlags.duration)
	defer cancel()

	// Create progress reporter
	progress := cli.NewProgressReporter(nil)
	progress.Start(int64(totalRequests))

	// Simulate requests (MVP - no actual HTTP calls)
	// In production, this would make real HTTP requests to the proxy
	requestInterval := time.Second / time.Duration(benchmarkFlags.rate)
	ticker := time.NewTicker(requestInterval)
	defer ticker.Stop()

	requestsSent := 0
	for requestsSent < totalRequests {
		select {
		case <-ctx.Done():
			// Timeout reached
			goto done
		case <-ticker.C:
			// Send request (simulated)
			go func() {
				reqStart := time.Now()
				// Simulate request processing
				time.Sleep(time.Millisecond * time.Duration(10+requestsSent%50)) // 10-60ms simulated latency
				latency := time.Since(reqStart)

				// Record result
				mu.Lock()
				results.latencies = append(results.latencies, latency)
				mu.Unlock()

				atomic.AddInt64(&successful, 1)
				progress.Update(atomic.LoadInt64(&successful) + atomic.LoadInt64(&failed))
			}()

			requestsSent++
			if requestsSent >= totalRequests {
				break
			}
		}
	}

done:
	// Wait a bit for in-flight requests to complete
	time.Sleep(100 * time.Millisecond)
	progress.Finish()

	results.duration = time.Since(start)
	results.successfulReqs = int(atomic.LoadInt64(&successful))
	results.failedReqs = int(atomic.LoadInt64(&failed))

	return results
}

func displayResults(results *benchmarkResults) {
	fmt.Println()
	fmt.Println("Results:")
	fmt.Println("--------")
	fmt.Printf("Requests:        %d total, %d successful, %d failed\n",
		results.totalRequests, results.successfulReqs, results.failedReqs)
	fmt.Printf("Duration:        %.1fs\n", results.duration.Seconds())

	if results.successfulReqs > 0 {
		throughput := float64(results.successfulReqs) / results.duration.Seconds()
		fmt.Printf("Throughput:      %.2f req/s\n", throughput)
	}

	if len(results.latencies) > 0 {
		// Calculate percentiles
		latencies := results.latencies
		min, mean, median, p95, p99, max := calculatePercentiles(latencies)

		fmt.Println()
		fmt.Println("Latency:")
		fmt.Printf("  Min:     %.1fms\n", float64(min.Microseconds())/1000)
		fmt.Printf("  Mean:    %.1fms\n", float64(mean.Microseconds())/1000)
		fmt.Printf("  Median:  %.1fms\n", float64(median.Microseconds())/1000)
		fmt.Printf("  p95:     %.1fms\n", float64(p95.Microseconds())/1000)
		fmt.Printf("  p99:     %.1fms\n", float64(p99.Microseconds())/1000)
		fmt.Printf("  Max:     %.1fms\n", float64(max.Microseconds())/1000)
	}

	if results.successfulReqs > 0 {
		successRate := float64(results.successfulReqs) / float64(results.totalRequests) * 100
		fmt.Println()
		fmt.Printf("Status Codes:\n")
		fmt.Printf("  200:     %d (%.0f%%)\n", results.successfulReqs, successRate)
		if results.failedReqs > 0 {
			failRate := float64(results.failedReqs) / float64(results.totalRequests) * 100
			fmt.Printf("  Errors:  %d (%.0f%%)\n", results.failedReqs, failRate)
		}
	}

	fmt.Println()
	fmt.Println("Note: This is a simulated benchmark (MVP). Real HTTP requests")
	fmt.Println("and detailed metrics will be implemented in a future release.")
}

func calculatePercentiles(latencies []time.Duration) (min, mean, median, p95, p99, max time.Duration) {
	if len(latencies) == 0 {
		return
	}

	// Sort latencies (simple bubble sort for MVP)
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	min = sorted[0]
	max = sorted[len(sorted)-1]

	// Calculate mean
	var sum time.Duration
	for _, lat := range sorted {
		sum += lat
	}
	mean = sum / time.Duration(len(sorted))

	// Calculate percentiles
	median = sorted[len(sorted)/2]
	p95 = sorted[int(float64(len(sorted))*0.95)]
	p99 = sorted[int(float64(len(sorted))*0.99)]

	return
}
