package source

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/policy/engine"
)

// FileSource loads policies from YAML files on disk.
type FileSource struct {
	path   string
	logger *slog.Logger
}

// NewFileSource creates a new file-based policy source.
// The path can be either a single file or a directory.
// If it's a directory, all .yaml and .yml files will be loaded.
func NewFileSource(path string, logger *slog.Logger) *FileSource {
	if logger == nil {
		logger = slog.Default()
	}
	return &FileSource{
		path:   path,
		logger: logger,
	}
}

// LoadPolicies loads all policies from the configured path.
func (s *FileSource) LoadPolicies(ctx context.Context) ([]*ast.Policy, error) {
	// Check if path exists
	info, err := os.Stat(s.path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat path %q: %w", s.path, err)
	}

	var policies []*ast.Policy

	if info.IsDir() {
		// Load all policy files from directory
		policies, err = s.loadDirectory(ctx)
		if err != nil {
			return nil, err
		}
	} else {
		// Load single policy file
		policy, err := s.loadFile(ctx, s.path)
		if err != nil {
			return nil, err
		}
		policies = []*ast.Policy{policy}
	}

	s.logger.Info("loaded policies from source",
		"path", s.path,
		"policy_count", len(policies),
	)

	return policies, nil
}

// loadDirectory loads all policy files from a directory.
func (s *FileSource) loadDirectory(ctx context.Context) ([]*ast.Policy, error) {
	var policies []*ast.Policy

	// Walk the directory
	err := filepath.Walk(s.path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process YAML files
		ext := filepath.Ext(path)
		if ext != ".yaml" && ext != ".yml" {
			return nil
		}

		// Load policy file
		policy, err := s.loadFile(ctx, path)
		if err != nil {
			s.logger.Warn("failed to load policy file, skipping",
				"path", path,
				"error", err,
			)
			return nil // Skip invalid files
		}

		policies = append(policies, policy)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory %q: %w", s.path, err)
	}

	return policies, nil
}

// loadFile loads a single policy file.
func (s *FileSource) loadFile(ctx context.Context, path string) (*ast.Policy, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %q: %w", path, err)
	}

	// Parse policy
	p := parser.NewParser()
	policy, err := p.ParseBytes(data, path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse policy file %q: %w", path, err)
	}

	// Set source file
	policy.SourceFile = path

	s.logger.Debug("loaded policy file",
		"path", path,
		"policy_name", policy.Name,
		"rule_count", len(policy.Rules),
	)

	return policy, nil
}

// Watch watches for file system changes and sends events on the returned channel.
// For MVP, this is a basic implementation that doesn't use fsnotify.
// The channel is closed when the context is cancelled.
func (s *FileSource) Watch(ctx context.Context) (<-chan engine.PolicyEvent, error) {
	eventCh := make(chan engine.PolicyEvent)

	// For MVP, we don't implement actual file watching
	// This would require adding fsnotify dependency
	// Just return an empty channel that closes when context is cancelled
	go func() {
		<-ctx.Done()
		close(eventCh)
	}()

	s.logger.Info("policy file watcher started (file watching not implemented in MVP)")

	return eventCh, nil
}
