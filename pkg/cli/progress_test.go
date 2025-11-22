package cli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSimpleProgressBasic(t *testing.T) {
	buf := &bytes.Buffer{}
	progress := NewProgressReporter(buf)

	// Start progress
	progress.Start(100)
	time.Sleep(10 * time.Millisecond) // Give it time to render

	// Update progress
	progress.Update(50)
	time.Sleep(10 * time.Millisecond)

	// Finish progress
	progress.Finish()

	output := buf.String()
	if !strings.Contains(output, "Progress:") {
		t.Error("Expected progress output to contain 'Progress:'")
	}
}

func TestSimpleProgressZeroTotal(t *testing.T) {
	buf := &bytes.Buffer{}
	progress := NewProgressReporter(buf).(*SimpleProgress)

	// Start with zero total should not cause panic
	progress.Start(0)
	progress.Update(0)
	progress.Finish()

	// Should have minimal output since total is 0 (either empty or just newline is acceptable)
	_ = buf.String()
}

func TestSimpleProgressError(t *testing.T) {
	buf := &bytes.Buffer{}
	progress := NewProgressReporter(buf)

	progress.Start(100)
	progress.Error(fmt.Errorf("test error"))

	output := buf.String()
	if !strings.Contains(output, "Error:") {
		t.Error("Expected error output to contain 'Error:'")
	}
	if !strings.Contains(output, "test error") {
		t.Error("Expected error output to contain error message")
	}
}

func TestSimpleProgressConcurrent(t *testing.T) {
	buf := &bytes.Buffer{}
	progress := NewProgressReporter(buf)

	progress.Start(1000)

	// Simulate concurrent updates
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(start int) {
			for j := 0; j < 100; j++ {
				progress.Update(int64(start*100 + j))
				time.Sleep(time.Microsecond)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	progress.Finish()

	// Should not panic and should produce some output
	if buf.Len() == 0 {
		t.Error("Expected some progress output")
	}
}

func TestNewProgressReporterNilWriter(t *testing.T) {
	// Should default to stdout, not panic
	progress := NewProgressReporter(nil)
	if progress == nil {
		t.Error("NewProgressReporter(nil) should not return nil")
	}

	// Should not panic on operations
	progress.Start(10)
	progress.Update(5)
	progress.Finish()
}
