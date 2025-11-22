package manager

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// FileWatcher watches policy files for changes and triggers reloads.
// It implements debouncing to prevent reload storms.
type FileWatcher struct {
	watcher  *fsnotify.Watcher
	logger   *slog.Logger
	config   *FileWatcherConfig
	debounce *Debouncer

	// State
	mu      sync.RWMutex
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

// FileWatcherConfig contains configuration for the file watcher.
type FileWatcherConfig struct {
	// Path is the file or directory to watch
	Path string

	// DebounceInterval is the time to wait before triggering a reload
	// after detecting file changes (default: 100ms)
	DebounceInterval time.Duration

	// Extensions is the list of file extensions to watch (e.g., ".yaml", ".yml")
	Extensions []string

	// FollowSymlinks controls whether to follow symbolic links
	FollowSymlinks bool

	// SkipHidden controls whether to skip hidden files
	SkipHidden bool
}

// DefaultFileWatcherConfig returns the default watcher configuration.
func DefaultFileWatcherConfig() *FileWatcherConfig {
	return &FileWatcherConfig{
		DebounceInterval: 100 * time.Millisecond,
		Extensions:       []string{".yaml", ".yml"},
		FollowSymlinks:   true,
		SkipHidden:       true,
	}
}

// NewFileWatcher creates a new file watcher.
func NewFileWatcher(config *FileWatcherConfig, logger *slog.Logger) (*FileWatcher, error) {
	if config == nil {
		config = DefaultFileWatcherConfig()
	}

	if logger == nil {
		logger = slog.Default()
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	fw := &FileWatcher{
		watcher:  watcher,
		logger:   logger,
		config:   config,
		debounce: NewDebouncer(config.DebounceInterval),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}

	return fw, nil
}

// Watch starts watching for file changes and sends reload events.
// This is a blocking operation that runs until the context is cancelled or Stop is called.
func (fw *FileWatcher) Watch(ctx context.Context, onReload func() error) error {
	fw.mu.Lock()
	if fw.running {
		fw.mu.Unlock()
		return fmt.Errorf("watcher already running")
	}
	fw.running = true
	fw.mu.Unlock()

	defer func() {
		fw.mu.Lock()
		fw.running = false
		fw.mu.Unlock()
		close(fw.doneCh)
	}()

	// Add path to watcher
	if err := fw.addPath(fw.config.Path); err != nil {
		return fmt.Errorf("failed to watch path: %w", err)
	}

	fw.logger.Info("File watcher started",
		"path", fw.config.Path,
		"debounce_ms", fw.config.DebounceInterval.Milliseconds(),
	)

	// Event processing loop
	for {
		select {
		case <-ctx.Done():
			fw.logger.Info("File watcher stopped (context cancelled)")
			return nil

		case <-fw.stopCh:
			fw.logger.Info("File watcher stopped")
			return nil

		case event, ok := <-fw.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			// Filter events
			if !fw.shouldProcessEvent(event) {
				continue
			}

			fw.logger.Debug("File event detected",
				"path", event.Name,
				"op", event.Op.String(),
			)

			// Debounce and trigger reload
			fw.debounce.Trigger(func() {
				fw.logger.Info("Triggering policy reload",
					"path", event.Name,
					"op", event.Op.String(),
				)

				if err := onReload(); err != nil {
					fw.logger.Error("Policy reload failed",
						"error", err,
					)
				}
			})

		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}

			fw.logger.Error("File watcher error", "error", err)
			// Continue watching despite errors
		}
	}
}

// Stop stops the file watcher.
func (fw *FileWatcher) Stop() error {
	fw.mu.Lock()
	if !fw.running {
		fw.mu.Unlock()
		return nil
	}
	fw.mu.Unlock()

	// Signal stop
	close(fw.stopCh)

	// Wait for watcher to stop
	<-fw.doneCh

	// Stop debouncer
	fw.debounce.Stop()

	// Close fsnotify watcher
	if err := fw.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	return nil
}

// addPath adds a file or directory to the watcher.
func (fw *FileWatcher) addPath(path string) error {
	// Check if path is a directory
	isDir, err := isDirectory(path)
	if err != nil {
		return err
	}

	if isDir {
		// Watch directory and all subdirectories
		return fw.addDirectory(path)
	}

	// Watch single file
	return fw.watcher.Add(path)
}

// addDirectory adds a directory and all subdirectories to the watcher.
func (fw *FileWatcher) addDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/directories if configured
		if fw.config.SkipHidden && strings.HasPrefix(filepath.Base(path), ".") && path != dir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Only watch directories
		if info.IsDir() {
			if err := fw.watcher.Add(path); err != nil {
				return fmt.Errorf("failed to watch directory %q: %w", path, err)
			}
			fw.logger.Debug("Watching directory", "path", path)
		}

		return nil
	})
}

// shouldProcessEvent determines if an event should trigger a reload.
func (fw *FileWatcher) shouldProcessEvent(event fsnotify.Event) bool {
	// Skip events we don't care about
	if event.Op&fsnotify.Chmod == fsnotify.Chmod {
		return false
	}

	// Check file extension
	ext := strings.ToLower(filepath.Ext(event.Name))
	if !fw.hasValidExtension(ext) {
		return false
	}

	// Skip hidden files if configured
	if fw.config.SkipHidden && strings.HasPrefix(filepath.Base(event.Name), ".") {
		return false
	}

	return true
}

// hasValidExtension checks if a file extension should be watched.
func (fw *FileWatcher) hasValidExtension(ext string) bool {
	for _, validExt := range fw.config.Extensions {
		if ext == strings.ToLower(validExt) {
			return true
		}
	}
	return false
}

// Debouncer implements event debouncing to prevent reload storms.
// It collects rapid events and triggers the callback only after a quiet period.
type Debouncer struct {
	interval time.Duration
	timer    *time.Timer
	mu       sync.Mutex
	callback func()
	stopCh   chan struct{}
}

// NewDebouncer creates a new debouncer.
func NewDebouncer(interval time.Duration) *Debouncer {
	return &Debouncer{
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

// Trigger triggers the debouncer with a new event.
// The callback will be called after the debounce interval if no new events occur.
func (d *Debouncer) Trigger(callback func()) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Store the callback
	d.callback = callback

	// Reset or create timer
	if d.timer != nil {
		d.timer.Stop()
	}

	d.timer = time.AfterFunc(d.interval, func() {
		select {
		case <-d.stopCh:
			return
		default:
			d.mu.Lock()
			cb := d.callback
			d.mu.Unlock()

			if cb != nil {
				cb()
			}
		}
	})
}

// Stop stops the debouncer and cancels any pending callbacks.
func (d *Debouncer) Stop() {
	close(d.stopCh)

	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}
	d.callback = nil
}

// Helper function to check if path is a directory
func isDirectory(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
