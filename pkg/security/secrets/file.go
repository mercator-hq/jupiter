package secrets

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// FileProvider loads secrets from individual files in a directory.
//
// This provider supports Kubernetes-style secret mounting where each
// secret is stored as a separate file. File permissions are validated
// to ensure secrets are properly protected (0600 or 0400 only).
//
// The provider can optionally watch for file changes and automatically
// reload secrets when files are modified.
type FileProvider struct {
	BasePath string // Directory containing secret files
	Watch    bool   // Enable file watching for auto-reload

	mu      sync.RWMutex
	cache   map[string]string
	watcher *fsnotify.Watcher
	stopCh  chan struct{}
}

// NewFileProvider creates a new file-based secret provider.
//
// If watch is enabled, the provider will monitor the directory for changes
// and automatically refresh secrets when files are modified.
func NewFileProvider(basePath string, watch bool) (*FileProvider, error) {
	p := &FileProvider{
		BasePath: basePath,
		Watch:    watch,
		cache:    make(map[string]string),
		stopCh:   make(chan struct{}),
	}

	// Verify base path exists and is a directory
	info, err := os.Stat(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat base path: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("base path is not a directory: %s", basePath)
	}

	// Set up file watching if enabled
	if watch {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			return nil, fmt.Errorf("failed to create file watcher: %w", err)
		}

		if err := watcher.Add(basePath); err != nil {
			_ = watcher.Close() // Best effort close on error path
			return nil, fmt.Errorf("failed to watch directory: %w", err)
		}

		p.watcher = watcher
		go p.watchLoop()

		slog.Info("file-based secret provider started with watching",
			"path", basePath,
		)
	} else {
		slog.Info("file-based secret provider started without watching",
			"path", basePath,
		)
	}

	return p, nil
}

// GetSecret retrieves a secret from a file.
//
// The secret name is used as the filename within the configured base path.
// For example, secret "openai-api-key" is read from "<basePath>/openai-api-key".
//
// File permissions are validated to ensure they are 0600 or 0400 only.
// This prevents accidental exposure of secrets through overly permissive permissions.
func (p *FileProvider) GetSecret(ctx context.Context, name string) (string, error) {
	// Check cache first
	p.mu.RLock()
	if value, ok := p.cache[name]; ok {
		p.mu.RUnlock()
		return value, nil
	}
	p.mu.RUnlock()

	// Read from file
	path := filepath.Join(p.BasePath, name)

	// Validate path is within BasePath (prevent directory traversal)
	absBase, err := filepath.Abs(p.BasePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve secret path: %w", err)
	}
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) && absPath != absBase {
		return "", fmt.Errorf("invalid secret path: directory traversal detected")
	}

	// Validate file exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("secret file not found: %s", name)
		}
		return "", fmt.Errorf("failed to stat secret file: %w", err)
	}

	// Ensure it's a regular file (not a directory or symlink)
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("secret path is not a regular file: %s", name)
	}

	// Validate permissions (must be 0600 or 0400)
	mode := info.Mode().Perm()
	if mode != 0600 && mode != 0400 {
		return "", fmt.Errorf("insecure permissions on %s: %o (expected 0600 or 0400)", path, mode)
	}

	// Read file contents
	// #nosec G304 - Path is validated above to prevent directory traversal
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read secret file: %w", err)
	}

	// Trim whitespace (common for file-based secrets)
	value := strings.TrimSpace(string(data))

	// Cache the value
	p.mu.Lock()
	p.cache[name] = value
	p.mu.Unlock()

	return value, nil
}

// ListSecrets returns all secret names (filenames) in the base directory.
//
// Only regular files are included. Directories, symlinks, and special files
// are excluded.
func (p *FileProvider) ListSecrets(ctx context.Context) ([]string, error) {
	entries, err := os.ReadDir(p.BasePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read secrets directory: %w", err)
	}

	var secrets []string
	for _, entry := range entries {
		// Only include regular files
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Skip non-regular files (symlinks, devices, etc.)
		if !info.Mode().IsRegular() {
			continue
		}

		secrets = append(secrets, entry.Name())
	}

	return secrets, nil
}

// Provider returns the provider name.
func (p *FileProvider) Provider() string {
	return "file"
}

// Supports indicates if this provider supports the given secret name.
//
// A secret is supported if a file with that name exists in the base directory.
func (p *FileProvider) Supports(name string) bool {
	path := filepath.Join(p.BasePath, name)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// Refresh clears the cache, forcing secrets to be re-read from files.
//
// This is typically called when file changes are detected or when
// secrets need to be rotated.
func (p *FileProvider) Refresh(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	slog.Debug("refreshing file-based secrets cache")
	p.cache = make(map[string]string)

	return nil
}

// Close stops the file watcher and cleans up resources.
//
// This should be called when the provider is no longer needed.
func (p *FileProvider) Close() error {
	if p.watcher != nil {
		close(p.stopCh)
		return p.watcher.Close()
	}
	return nil
}

// watchLoop monitors the directory for file changes and refreshes the cache.
//
// This runs in a background goroutine when watching is enabled.
func (p *FileProvider) watchLoop() {
	for {
		select {
		case event, ok := <-p.watcher.Events:
			if !ok {
				return
			}

			// Refresh cache on file writes or creates
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create {

				slog.Debug("file change detected, refreshing secrets",
					"file", filepath.Base(event.Name),
					"op", event.Op.String(),
				)

				// Clear cache to force re-read
				if err := p.Refresh(context.Background()); err != nil {
					slog.Error("failed to refresh secrets after file change",
						"error", err,
					)
				}
			}

		case err, ok := <-p.watcher.Errors:
			if !ok {
				return
			}

			slog.Error("file watcher error", "error", err)

		case <-p.stopCh:
			return
		}
	}
}
