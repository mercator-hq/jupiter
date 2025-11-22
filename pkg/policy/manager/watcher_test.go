package manager

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
)

func TestNewFileWatcher(t *testing.T) {
	config := DefaultFileWatcherConfig()
	config.Path = "testdata"

	watcher, err := NewFileWatcher(config, nil)

	if err != nil {
		t.Fatalf("NewFileWatcher() error = %v, want nil", err)
	}

	if watcher == nil {
		t.Fatal("NewFileWatcher() returned nil")
	}

	if watcher.watcher == nil {
		t.Error("watcher.watcher is nil")
	}

	if watcher.debounce == nil {
		t.Error("watcher.debounce is nil")
	}

	// Cleanup
	_ = watcher.Stop()
}

func TestDefaultFileWatcherConfig(t *testing.T) {
	config := DefaultFileWatcherConfig()

	if config.DebounceInterval != 100*time.Millisecond {
		t.Errorf("config.DebounceInterval = %v, want 100ms", config.DebounceInterval)
	}

	if len(config.Extensions) != 2 {
		t.Errorf("config.Extensions count = %d, want 2", len(config.Extensions))
	}

	if !config.FollowSymlinks {
		t.Error("config.FollowSymlinks = false, want true")
	}

	if !config.SkipHidden {
		t.Error("config.SkipHidden = false, want true")
	}
}

func TestFileWatcher_Watch_SingleFile(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "test-policy"
version: "1.0.0"
rules: []
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create watcher
	config := DefaultFileWatcherConfig()
	config.Path = tmpFile
	config.DebounceInterval = 50 * time.Millisecond

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Stop() }()

	// Track reload calls
	var reloadCount atomic.Int32
	var reloadMu sync.Mutex
	reloadCalled := make(chan struct{}, 10)

	onReload := func() error {
		reloadMu.Lock()
		defer reloadMu.Unlock()
		reloadCount.Add(1)
		select {
		case reloadCalled <- struct{}{}:
		default:
		}
		return nil
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = watcher.Watch(ctx, onReload)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify file
	newContent := `
mpl_version: "1.0"
name: "test-policy-modified"
version: "1.0.1"
rules: []
`
	if err := os.WriteFile(tmpFile, []byte(newContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for reload to be called (with timeout)
	select {
	case <-reloadCalled:
		// Success!
	case <-time.After(500 * time.Millisecond):
		t.Error("Reload not called after file modification")
	}

	// Stop watching
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Verify reload was called
	if reloadCount.Load() == 0 {
		t.Error("Reload was never called")
	}
}

func TestFileWatcher_Watch_Directory(t *testing.T) {
	// Create temporary directory with policy file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "test-policy"
version: "1.0.0"
rules: []
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create watcher for directory
	config := DefaultFileWatcherConfig()
	config.Path = tmpDir
	config.DebounceInterval = 50 * time.Millisecond

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Stop() }()

	// Track reload calls
	var reloadCount atomic.Int32
	reloadCalled := make(chan struct{}, 10)

	onReload := func() error {
		reloadCount.Add(1)
		select {
		case reloadCalled <- struct{}{}:
		default:
		}
		return nil
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = watcher.Watch(ctx, onReload)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Create new file in directory
	newFile := filepath.Join(tmpDir, "policy2.yaml")
	if err := os.WriteFile(newFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait for reload to be called (with timeout)
	select {
	case <-reloadCalled:
		// Success!
	case <-time.After(500 * time.Millisecond):
		t.Error("Reload not called after creating new file")
	}

	// Verify reload was called
	if reloadCount.Load() == 0 {
		t.Error("Reload was never called")
	}
}

func TestFileWatcher_Debouncing(t *testing.T) {
	// Create temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "policy.yaml")

	content := `
mpl_version: "1.0"
name: "test-policy"
version: "1.0.0"
rules: []
`
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Create watcher with longer debounce interval
	config := DefaultFileWatcherConfig()
	config.Path = tmpFile
	config.DebounceInterval = 200 * time.Millisecond

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Stop() }()

	// Track reload calls
	var reloadCount atomic.Int32

	onReload := func() error {
		reloadCount.Add(1)
		return nil
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = watcher.Watch(ctx, onReload)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Make multiple rapid modifications
	for i := 0; i < 5; i++ {
		newContent := content + "\n# modification " + string(rune('0'+i))
		if err := os.WriteFile(tmpFile, []byte(newContent), 0644); err != nil {
			t.Fatal(err)
		}
		time.Sleep(30 * time.Millisecond) // Less than debounce interval
	}

	// Wait for debounce interval + some buffer
	time.Sleep(300 * time.Millisecond)

	// Reload should be called only once (or at most twice) due to debouncing
	count := reloadCount.Load()
	if count == 0 {
		t.Error("Reload was never called")
	}
	if count > 2 {
		t.Errorf("Reload called %d times, want <= 2 (debouncing failed)", count)
	}
}

func TestFileWatcher_Stop(t *testing.T) {
	config := DefaultFileWatcherConfig()
	config.Path = "testdata"

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = watcher.Watch(ctx, func() error { return nil })
	}()

	// Wait for watcher to start
	time.Sleep(50 * time.Millisecond)

	// Stop watcher
	err = watcher.Stop()

	if err != nil {
		t.Errorf("Stop() error = %v, want nil", err)
	}

	// Verify watcher is not running
	watcher.mu.RLock()
	running := watcher.running
	watcher.mu.RUnlock()

	if running {
		t.Error("Watcher still running after Stop()")
	}
}

func TestFileWatcher_DoubleStart(t *testing.T) {
	config := DefaultFileWatcherConfig()
	config.Path = "testdata"

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Stop() }()

	// Start first watch
	ctx1, cancel1 := context.WithCancel(context.Background())
	defer cancel1()

	go func() {
		_ = watcher.Watch(ctx1, func() error { return nil })
	}()

	// Wait for watcher to start
	time.Sleep(50 * time.Millisecond)

	// Try to start second watch (should fail)
	ctx2, cancel2 := context.WithCancel(context.Background())
	defer cancel2()

	err = watcher.Watch(ctx2, func() error { return nil })

	if err == nil {
		t.Error("Second Watch() call error = nil, want error")
	}
}

func TestFileWatcher_SkipHiddenFiles(t *testing.T) {
	// Create temporary directory
	tmpDir := t.TempDir()

	// Create hidden file
	hiddenFile := filepath.Join(tmpDir, ".hidden.yaml")
	if err := os.WriteFile(hiddenFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create watcher
	config := DefaultFileWatcherConfig()
	config.Path = tmpDir
	config.SkipHidden = true
	config.DebounceInterval = 50 * time.Millisecond

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Stop() }()

	// Track reload calls
	reloadCalled := false
	var mu sync.Mutex

	onReload := func() error {
		mu.Lock()
		reloadCalled = true
		mu.Unlock()
		return nil
	}

	// Start watching
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = watcher.Watch(ctx, onReload)
	}()

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify hidden file
	if err := os.WriteFile(hiddenFile, []byte("modified"), 0644); err != nil {
		t.Fatal(err)
	}

	// Wait to see if reload is called (it shouldn't be)
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	called := reloadCalled
	mu.Unlock()

	if called {
		t.Error("Reload was called for hidden file (should be skipped)")
	}
}

func TestDebouncer_Trigger(t *testing.T) {
	debouncer := NewDebouncer(100 * time.Millisecond)
	defer debouncer.Stop()

	var callCount atomic.Int32
	callback := func() {
		callCount.Add(1)
	}

	// Trigger multiple times
	for i := 0; i < 5; i++ {
		debouncer.Trigger(callback)
		time.Sleep(20 * time.Millisecond) // Less than debounce interval
	}

	// Wait for debounce interval
	time.Sleep(150 * time.Millisecond)

	// Callback should be called once
	count := callCount.Load()
	if count != 1 {
		t.Errorf("Callback called %d times, want 1", count)
	}
}

func TestDebouncer_Stop(t *testing.T) {
	debouncer := NewDebouncer(100 * time.Millisecond)

	var callCount atomic.Int32
	callback := func() {
		callCount.Add(1)
	}

	// Trigger
	debouncer.Trigger(callback)

	// Stop immediately
	debouncer.Stop()

	// Wait
	time.Sleep(150 * time.Millisecond)

	// Callback should not be called
	count := callCount.Load()
	if count != 0 {
		t.Errorf("Callback called %d times after Stop(), want 0", count)
	}
}

func TestFileWatcher_FilterExtensions(t *testing.T) {
	config := DefaultFileWatcherConfig()
	config.Extensions = []string{".yaml", ".yml"}

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Stop() }()

	tests := []struct {
		ext   string
		valid bool
	}{
		{".yaml", true},
		{".yml", true},
		{".txt", false},
		{".json", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := watcher.hasValidExtension(tt.ext)
			if got != tt.valid {
				t.Errorf("hasValidExtension(%q) = %v, want %v", tt.ext, got, tt.valid)
			}
		})
	}
}

func TestFileWatcher_ShouldProcessEvent_CaseInsensitive(t *testing.T) {
	config := DefaultFileWatcherConfig()
	config.Extensions = []string{".yaml", ".yml"}
	config.SkipHidden = true

	watcher, err := NewFileWatcher(config, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = watcher.Stop() }()

	tests := []struct {
		name        string
		eventName   string
		shouldAllow bool
	}{
		{"lowercase yaml", "/path/to/policy.yaml", true},
		{"uppercase YAML", "/path/to/policy.YAML", true},
		{"mixed case Yaml", "/path/to/policy.Yaml", true},
		{"yml extension", "/path/to/policy.yml", true},
		{"txt extension", "/path/to/policy.txt", false},
		{"hidden file", "/path/to/.hidden.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a WRITE event (which we care about)
			event := fsnotify.Event{
				Name: tt.eventName,
				Op:   fsnotify.Write,
			}

			got := watcher.shouldProcessEvent(event)
			if got != tt.shouldAllow {
				t.Errorf("shouldProcessEvent(%q) = %v, want %v", tt.eventName, got, tt.shouldAllow)
			}
		})
	}
}
