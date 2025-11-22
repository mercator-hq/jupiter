package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLintPoliciesValidFile(t *testing.T) {
	// Set flags
	lintFlags.file = "testdata/valid-policy.yaml"
	lintFlags.dir = ""
	lintFlags.strict = false
	lintFlags.format = "text"

	// Run lint command
	err := lintPolicies(nil, []string{})
	if err != nil {
		t.Errorf("lintPolicies() with valid file returned error: %v", err)
	}
}

func TestLintPoliciesInvalidFile(t *testing.T) {
	// Set flags
	lintFlags.file = "testdata/invalid-policy.yaml"
	lintFlags.dir = ""
	lintFlags.strict = false
	lintFlags.format = "text"

	// Run lint command - should return error for invalid policy
	err := lintPolicies(nil, []string{})
	if err == nil {
		t.Error("lintPolicies() with invalid file should return error")
	}
}

func TestLintPoliciesNonexistentFile(t *testing.T) {
	// Set flags
	lintFlags.file = "testdata/nonexistent.yaml"
	lintFlags.dir = ""
	lintFlags.strict = false
	lintFlags.format = "text"

	// Run lint command - should return error
	err := lintPolicies(nil, []string{})
	if err == nil {
		t.Error("lintPolicies() with nonexistent file should return error")
	}
}

func TestLintPoliciesNoFileOrDir(t *testing.T) {
	// Set flags - neither file nor dir specified
	lintFlags.file = ""
	lintFlags.dir = ""
	lintFlags.strict = false
	lintFlags.format = "text"

	// Run lint command - should return error
	err := lintPolicies(nil, []string{})
	if err == nil {
		t.Error("lintPolicies() without file or dir should return error")
	}
}

func TestLintPoliciesJSONFormat(t *testing.T) {
	// Set flags
	lintFlags.file = "testdata/valid-policy.yaml"
	lintFlags.dir = ""
	lintFlags.strict = false
	lintFlags.format = "json"

	// Run lint command
	err := lintPolicies(nil, []string{})
	if err != nil {
		t.Errorf("lintPolicies() with JSON format returned error: %v", err)
	}
}

func TestValidatePolicyFile(t *testing.T) {
	tests := []struct {
		name      string
		file      string
		wantValid bool
	}{
		{
			name:      "valid policy",
			file:      "testdata/valid-policy.yaml",
			wantValid: true,
		},
		{
			name:      "invalid policy",
			file:      "testdata/invalid-policy.yaml",
			wantValid: false,
		},
		{
			name:      "nonexistent file",
			file:      "testdata/nonexistent.yaml",
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validatePolicyFile(tt.file)
			if result.Valid != tt.wantValid {
				t.Errorf("validatePolicyFile(%q).Valid = %v, want %v",
					tt.file, result.Valid, tt.wantValid)
			}
		})
	}
}

func TestLintPoliciesDirectory(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "lint-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy valid policy to temp dir
	validPolicy := filepath.Join(tmpDir, "valid.yaml")
	data, _ := os.ReadFile("testdata/valid-policy.yaml")
	_ = os.WriteFile(validPolicy, data, 0644)

	// Set flags to lint directory
	lintFlags.file = ""
	lintFlags.dir = tmpDir
	lintFlags.strict = false
	lintFlags.format = "text"

	// Run lint command
	err = lintPolicies(nil, []string{})
	if err != nil {
		t.Errorf("lintPolicies() with valid directory returned error: %v", err)
	}
}
