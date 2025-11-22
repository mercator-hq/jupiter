package manager

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/mpl/parser"
)

// PolicyLoader handles loading policies from the file system.
// It supports single files and directory structures with validation.
type PolicyLoader struct {
	config *PolicyLoaderConfig
	parser *parser.Parser
}

// NewPolicyLoader creates a new policy loader with the given configuration.
func NewPolicyLoader(config *PolicyLoaderConfig, parser *parser.Parser) *PolicyLoader {
	if config == nil {
		config = DefaultLoaderConfig()
	}
	return &PolicyLoader{
		config: config,
		parser: parser,
	}
}

// LoadFromFile loads a single policy file from the given path.
// It performs file size validation, UTF-8 validation, and YAML parsing.
func (l *PolicyLoader) LoadFromFile(path string) (*ast.Policy, error) {
	// Check if file exists and get info
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &LoadError{
				FilePath: path,
				Message:  "file not found",
				Cause:    err,
			}
		}
		if os.IsPermission(err) {
			return nil, &LoadError{
				FilePath: path,
				Message:  "permission denied",
				Cause:    err,
			}
		}
		return nil, &LoadError{
			FilePath: path,
			Message:  "failed to access file",
			Cause:    err,
		}
	}

	// Check if it's a regular file
	if !fileInfo.Mode().IsRegular() {
		return nil, &LoadError{
			FilePath: path,
			Message:  "not a regular file",
		}
	}

	// Validate file size
	if fileInfo.Size() > l.config.MaxFileSize {
		return nil, &LoadError{
			FilePath: path,
			Message:  fmt.Sprintf("file size %d bytes exceeds maximum %d bytes", fileInfo.Size(), l.config.MaxFileSize),
		}
	}

	// Read file contents
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, &LoadError{
			FilePath: path,
			Message:  "failed to read file",
			Cause:    err,
		}
	}

	// Validate UTF-8 encoding
	if !utf8.Valid(data) {
		return nil, &LoadError{
			FilePath: path,
			Message:  "file contains invalid UTF-8 encoding",
		}
	}

	// Parse policy
	policy, err := l.parser.Parse(path)
	if err != nil {
		return nil, &ParseError{
			FilePath: path,
			Message:  "YAML parsing failed",
			Cause:    err,
		}
	}

	return policy, nil
}

// LoadFromDirectory loads all policy files from the given directory recursively.
// It returns a list of successfully loaded policies and any errors encountered.
func (l *PolicyLoader) LoadFromDirectory(dir string) ([]*ast.Policy, error) {
	// Check if directory exists
	fileInfo, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &LoadError{
				FilePath: dir,
				Message:  "directory not found",
				Cause:    err,
			}
		}
		return nil, &LoadError{
			FilePath: dir,
			Message:  "failed to access directory",
			Cause:    err,
		}
	}

	if !fileInfo.IsDir() {
		return nil, &LoadError{
			FilePath: dir,
			Message:  "not a directory",
		}
	}

	// Collect all policy files
	policyFiles, err := l.collectPolicyFiles(dir)
	if err != nil {
		return nil, err
	}

	if len(policyFiles) == 0 {
		return nil, &LoadError{
			FilePath: dir,
			Message:  "no policy files found in directory",
		}
	}

	// Load all policies
	var policies []*ast.Policy
	errList := &ErrorList{}

	for _, filePath := range policyFiles {
		policy, err := l.LoadFromFile(filePath)
		if err != nil {
			errList.Add(err)
			continue
		}
		policies = append(policies, policy)
	}

	// Return error if all files failed to load
	if len(policies) == 0 && errList.HasErrors() {
		return nil, errList
	}

	// Return policies with partial errors
	if errList.HasErrors() {
		return policies, errList
	}

	return policies, nil
}

// collectPolicyFiles collects all policy file paths in the given directory.
// It filters by extension and skips hidden files based on configuration.
func (l *PolicyLoader) collectPolicyFiles(dir string) ([]string, error) {
	var policyFiles []string
	visited := make(map[string]bool)

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden files/directories if configured
		if l.config.SkipHidden && strings.HasPrefix(d.Name(), ".") && path != dir {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Handle directories
		if d.IsDir() {
			return nil
		}

		// Handle symbolic links
		if d.Type()&fs.ModeSymlink != 0 {
			if !l.config.FollowSymlinks {
				return nil
			}

			// Resolve symlink
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return &LoadError{
					FilePath: path,
					Message:  "failed to resolve symlink",
					Cause:    err,
				}
			}

			// Detect symlink loops
			if visited[realPath] {
				return &LoadError{
					FilePath: path,
					Message:  "symlink loop detected",
				}
			}
			visited[realPath] = true

			// Check if symlink points to a file with valid extension
			if !l.hasValidExtension(realPath) {
				return nil
			}

			policyFiles = append(policyFiles, path)
			return nil
		}

		// Check file extension
		if !l.hasValidExtension(path) {
			return nil
		}

		policyFiles = append(policyFiles, path)
		return nil
	})

	if err != nil {
		return nil, &LoadError{
			FilePath: dir,
			Message:  "failed to walk directory",
			Cause:    err,
		}
	}

	return policyFiles, nil
}

// hasValidExtension checks if the file has a valid policy file extension.
func (l *PolicyLoader) hasValidExtension(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, validExt := range l.config.AllowedExtensions {
		if ext == strings.ToLower(validExt) {
			return true
		}
	}
	return false
}

// ValidateFileSize validates that a file does not exceed the maximum size.
func (l *PolicyLoader) ValidateFileSize(path string) error {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return &LoadError{
			FilePath: path,
			Message:  "failed to access file",
			Cause:    err,
		}
	}

	if fileInfo.Size() > l.config.MaxFileSize {
		return &LoadError{
			FilePath: path,
			Message:  fmt.Sprintf("file size %d bytes exceeds maximum %d bytes", fileInfo.Size(), l.config.MaxFileSize),
		}
	}

	return nil
}

// ValidateUTF8 validates that a file contains valid UTF-8 encoding.
func (l *PolicyLoader) ValidateUTF8(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return &LoadError{
			FilePath: path,
			Message:  "failed to read file",
			Cause:    err,
		}
	}

	if !utf8.Valid(data) {
		return &LoadError{
			FilePath: path,
			Message:  "file contains invalid UTF-8 encoding",
		}
	}

	return nil
}

// IsDirectory checks if the given path is a directory.
func (l *PolicyLoader) IsDirectory(path string) (bool, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, &LoadError{
				FilePath: path,
				Message:  "path does not exist",
				Cause:    err,
			}
		}
		return false, &LoadError{
			FilePath: path,
			Message:  "failed to access path",
			Cause:    err,
		}
	}

	return fileInfo.IsDir(), nil
}
