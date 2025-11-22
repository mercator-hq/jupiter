package manager

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"mercator-hq/jupiter/pkg/mpl/parser"
)

func TestPolicyLoader_LoadFromFile_Success(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	path := filepath.Join("testdata", "valid", "simple.yaml")
	policy, err := loader.LoadFromFile(path)

	if err != nil {
		t.Fatalf("LoadFromFile() error = %v, want nil", err)
	}

	if policy == nil {
		t.Fatal("LoadFromFile() returned nil policy")
	}

	if policy.Name != "simple-policy" {
		t.Errorf("Policy name = %q, want %q", policy.Name, "simple-policy")
	}

	if policy.Version != "1.0.0" {
		t.Errorf("Policy version = %q, want %q", policy.Version, "1.0.0")
	}

	if len(policy.Rules) != 1 {
		t.Errorf("Policy rules count = %d, want 1", len(policy.Rules))
	}
}

func TestPolicyLoader_LoadFromFile_FileNotFound(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	path := filepath.Join("testdata", "nonexistent.yaml")
	_, err := loader.LoadFromFile(path)

	if err == nil {
		t.Fatal("LoadFromFile() error = nil, want error")
	}

	var loadErr *LoadError
	if !errorAs(err, &loadErr) {
		t.Fatalf("LoadFromFile() error type = %T, want *LoadError", err)
	}

	if !strings.Contains(loadErr.Message, "file not found") {
		t.Errorf("LoadError message = %q, want to contain 'file not found'", loadErr.Message)
	}
}

func TestPolicyLoader_LoadFromFile_InvalidYAML(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	path := filepath.Join("testdata", "invalid", "malformed.yaml")
	_, err := loader.LoadFromFile(path)

	if err == nil {
		t.Fatal("LoadFromFile() error = nil, want error")
	}

	var parseErr *ParseError
	if !errorAs(err, &parseErr) {
		t.Fatalf("LoadFromFile() error type = %T, want *ParseError", err)
	}
}

func TestPolicyLoader_LoadFromFile_FileSizeExceeded(t *testing.T) {
	// Create a config with a very small max file size
	config := DefaultLoaderConfig()
	config.MaxFileSize = 10 // 10 bytes

	loader := NewPolicyLoader(config, parser.NewParser())

	path := filepath.Join("testdata", "valid", "simple.yaml")
	_, err := loader.LoadFromFile(path)

	if err == nil {
		t.Fatal("LoadFromFile() error = nil, want error for file size exceeded")
	}

	var loadErr *LoadError
	if !errorAs(err, &loadErr) {
		t.Fatalf("LoadFromFile() error type = %T, want *LoadError", err)
	}

	if !strings.Contains(loadErr.Message, "exceeds maximum") {
		t.Errorf("LoadError message = %q, want to contain 'exceeds maximum'", loadErr.Message)
	}
}

func TestPolicyLoader_LoadFromFile_InvalidUTF8(t *testing.T) {
	// Create a temporary file with invalid UTF-8
	tmpFile, err := os.CreateTemp("", "invalid-utf8-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid UTF-8 sequence
	invalidUTF8 := []byte{0xff, 0xfe, 0xfd}
	if _, err := tmpFile.Write(invalidUTF8); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())
	_, err = loader.LoadFromFile(tmpFile.Name())

	if err == nil {
		t.Fatal("LoadFromFile() error = nil, want error for invalid UTF-8")
	}

	var loadErr *LoadError
	if !errorAs(err, &loadErr) {
		t.Fatalf("LoadFromFile() error type = %T, want *LoadError", err)
	}

	if !strings.Contains(loadErr.Message, "invalid UTF-8") {
		t.Errorf("LoadError message = %q, want to contain 'invalid UTF-8'", loadErr.Message)
	}
}

func TestPolicyLoader_LoadFromDirectory_Success(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	dir := filepath.Join("testdata", "multi")
	policies, err := loader.LoadFromDirectory(dir)

	if err != nil {
		t.Fatalf("LoadFromDirectory() error = %v, want nil", err)
	}

	if len(policies) != 2 {
		t.Errorf("LoadFromDirectory() loaded %d policies, want 2", len(policies))
	}

	// Verify policy names
	names := make(map[string]bool)
	for _, policy := range policies {
		names[policy.Name] = true
	}

	if !names["policy-1"] {
		t.Error("LoadFromDirectory() missing policy-1")
	}
	if !names["policy-2"] {
		t.Error("LoadFromDirectory() missing policy-2")
	}
}

func TestPolicyLoader_LoadFromDirectory_DirectoryNotFound(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	dir := filepath.Join("testdata", "nonexistent")
	_, err := loader.LoadFromDirectory(dir)

	if err == nil {
		t.Fatal("LoadFromDirectory() error = nil, want error")
	}

	var loadErr *LoadError
	if !errorAs(err, &loadErr) {
		t.Fatalf("LoadFromDirectory() error type = %T, want *LoadError", err)
	}

	if !strings.Contains(loadErr.Message, "not found") {
		t.Errorf("LoadError message = %q, want to contain 'not found'", loadErr.Message)
	}
}

func TestPolicyLoader_LoadFromDirectory_EmptyDirectory(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	dir := filepath.Join("testdata", "empty")
	_, err := loader.LoadFromDirectory(dir)

	if err == nil {
		t.Fatal("LoadFromDirectory() error = nil, want error for empty directory")
	}

	var loadErr *LoadError
	if !errorAs(err, &loadErr) {
		t.Fatalf("LoadFromDirectory() error type = %T, want *LoadError", err)
	}

	if !strings.Contains(loadErr.Message, "no policy files found") {
		t.Errorf("LoadError message = %q, want to contain 'no policy files found'", loadErr.Message)
	}
}

func TestPolicyLoader_LoadFromDirectory_NotADirectory(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	// Try to load from a file as if it were a directory
	path := filepath.Join("testdata", "valid", "simple.yaml")
	_, err := loader.LoadFromDirectory(path)

	if err == nil {
		t.Fatal("LoadFromDirectory() error = nil, want error")
	}

	var loadErr *LoadError
	if !errorAs(err, &loadErr) {
		t.Fatalf("LoadFromDirectory() error type = %T, want *LoadError", err)
	}

	if !strings.Contains(loadErr.Message, "not a directory") {
		t.Errorf("LoadError message = %q, want to contain 'not a directory'", loadErr.Message)
	}
}

func TestPolicyLoader_HasValidExtension(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	tests := []struct {
		name  string
		path  string
		valid bool
	}{
		{
			name:  "yaml extension",
			path:  "policy.yaml",
			valid: true,
		},
		{
			name:  "yml extension",
			path:  "policy.yml",
			valid: true,
		},
		{
			name:  "YAML uppercase",
			path:  "policy.YAML",
			valid: true,
		},
		{
			name:  "txt extension",
			path:  "policy.txt",
			valid: false,
		},
		{
			name:  "no extension",
			path:  "policy",
			valid: false,
		},
		{
			name:  "json extension",
			path:  "policy.json",
			valid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := loader.hasValidExtension(tt.path)
			if got != tt.valid {
				t.Errorf("hasValidExtension(%q) = %v, want %v", tt.path, got, tt.valid)
			}
		})
	}
}

func TestPolicyLoader_ValidateFileSize(t *testing.T) {
	config := DefaultLoaderConfig()
	config.MaxFileSize = 1024 // 1KB
	loader := NewPolicyLoader(config, parser.NewParser())

	path := filepath.Join("testdata", "valid", "simple.yaml")
	err := loader.ValidateFileSize(path)

	if err != nil {
		t.Errorf("ValidateFileSize() error = %v, want nil", err)
	}
}

func TestPolicyLoader_ValidateUTF8(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	path := filepath.Join("testdata", "valid", "simple.yaml")
	err := loader.ValidateUTF8(path)

	if err != nil {
		t.Errorf("ValidateUTF8() error = %v, want nil", err)
	}
}

func TestPolicyLoader_IsDirectory(t *testing.T) {
	loader := NewPolicyLoader(DefaultLoaderConfig(), parser.NewParser())

	tests := []struct {
		name  string
		path  string
		isDir bool
		hasErr bool
	}{
		{
			name:   "valid directory",
			path:   "testdata",
			isDir:  true,
			hasErr: false,
		},
		{
			name:   "valid file",
			path:   filepath.Join("testdata", "valid", "simple.yaml"),
			isDir:  false,
			hasErr: false,
		},
		{
			name:   "nonexistent path",
			path:   filepath.Join("testdata", "nonexistent"),
			isDir:  false,
			hasErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isDir, err := loader.IsDirectory(tt.path)

			if tt.hasErr && err == nil {
				t.Error("IsDirectory() error = nil, want error")
			}

			if !tt.hasErr && err != nil {
				t.Errorf("IsDirectory() error = %v, want nil", err)
			}

			if !tt.hasErr && isDir != tt.isDir {
				t.Errorf("IsDirectory() = %v, want %v", isDir, tt.isDir)
			}
		})
	}
}

func TestDefaultLoaderConfig(t *testing.T) {
	config := DefaultLoaderConfig()

	if config.MaxFileSize != 10*1024*1024 {
		t.Errorf("MaxFileSize = %d, want %d", config.MaxFileSize, 10*1024*1024)
	}

	if config.MaxIncludeDepth != 10 {
		t.Errorf("MaxIncludeDepth = %d, want 10", config.MaxIncludeDepth)
	}

	if len(config.AllowedExtensions) != 2 {
		t.Errorf("AllowedExtensions count = %d, want 2", len(config.AllowedExtensions))
	}

	if !config.FollowSymlinks {
		t.Error("FollowSymlinks = false, want true")
	}

	if !config.SkipHidden {
		t.Error("SkipHidden = false, want true")
	}
}

// Helper function to check error type using errors.As pattern
func errorAs(err error, target interface{}) bool {
	if err == nil {
		return false
	}

	switch target := target.(type) {
	case **LoadError:
		le, ok := err.(*LoadError)
		if ok {
			*target = le
			return true
		}
	case **ParseError:
		pe, ok := err.(*ParseError)
		if ok {
			*target = pe
			return true
		}
	}

	return false
}
