package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

// ProgressReporter reports progress for long-running operations.
type ProgressReporter interface {
	Start(total int64)
	Update(current int64)
	Finish()
	Error(err error)
}

// SimpleProgress implements a simple text-based progress reporter.
type SimpleProgress struct {
	mu      sync.Mutex
	total   int64
	current int64
	started time.Time
	writer  io.Writer
}

// NewProgressReporter creates a new progress reporter that writes to w.
// If w is nil, it defaults to os.Stdout.
func NewProgressReporter(w io.Writer) ProgressReporter {
	if w == nil {
		w = os.Stdout
	}
	return &SimpleProgress{
		writer: w,
	}
}

// Start initializes the progress reporter with the total number of items.
func (p *SimpleProgress) Start(total int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.total = total
	p.current = 0
	p.started = time.Now()

	p.render()
}

// Update updates the current progress.
func (p *SimpleProgress) Update(current int64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = current
	p.render()
}

// Finish marks the progress as complete.
func (p *SimpleProgress) Finish() {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.current = p.total
	p.render()
	fmt.Fprintln(p.writer)
}

// Error reports an error during progress.
func (p *SimpleProgress) Error(err error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	fmt.Fprintf(p.writer, "\n✗ Error: %v\n", err)
}

func (p *SimpleProgress) render() {
	if p.total == 0 {
		return
	}

	percent := float64(p.current) / float64(p.total) * 100
	barWidth := 40
	filled := int(float64(barWidth) * percent / 100)

	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)

	elapsed := time.Since(p.started)
	rate := float64(p.current) / elapsed.Seconds()

	fmt.Fprintf(p.writer, "\rProgress: [%s] %.1f%% (%d/%d) %.1f req/s",
		bar, percent, p.current, p.total, rate)
}
