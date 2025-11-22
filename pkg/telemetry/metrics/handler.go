package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Handler returns an HTTP handler for the Prometheus metrics endpoint.
//
// This handler exposes all registered metrics in the standard Prometheus
// exposition format. It should be mounted at the path specified in the
// MetricsConfig (typically "/metrics").
//
// Example:
//
//	collector := metrics.NewCollector(cfg, nil)
//	http.Handle("/metrics", collector.Handler())
//	http.ListenAndServe(":8080", nil)
//
// The handler uses the promhttp.Handler with default options, which includes:
//   - Automatic metric collection on each scrape
//   - Compression support (gzip)
//   - Timeout handling
//   - Error reporting
func (c *Collector) Handler() http.Handler {
	return promhttp.HandlerFor(
		c.registry,
		promhttp.HandlerOpts{
			// Enable OpenMetrics encoding (preferred over Prometheus text format)
			EnableOpenMetrics: true,

			// Timeout for collecting metrics
			Timeout: 0, // No timeout (use server's timeout)

			// Maximum number of requests that can be served concurrently
			MaxRequestsInFlight: 0, // Unlimited

			// Error handling
			ErrorHandling: promhttp.ContinueOnError,

			// Error logger (nil = use default)
			ErrorLog: nil,
		},
	)
}

// HandlerWithOptions returns an HTTP handler with custom options.
//
// This allows for more control over the handler behavior, such as:
//   - Setting a timeout for metric collection
//   - Limiting concurrent scrape requests
//   - Custom error handling
//
// Example:
//
//	handler := collector.HandlerWithOptions(promhttp.HandlerOpts{
//		Timeout: 10 * time.Second,
//		MaxRequestsInFlight: 5,
//		ErrorHandling: promhttp.HTTPErrorOnError,
//	})
//	http.Handle("/metrics", handler)
func (c *Collector) HandlerWithOptions(opts promhttp.HandlerOpts) http.Handler {
	return promhttp.HandlerFor(c.registry, opts)
}
