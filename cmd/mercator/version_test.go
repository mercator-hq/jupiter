package main

import (
	"runtime"
	"testing"
)

func TestVersionDefaults(t *testing.T) {
	// Test that version variables can be set and retrieved
	origVersion := Version
	origGitCommit := GitCommit
	origBuildDate := BuildDate

	// Set test values
	Version = "0.1.0-test"
	GitCommit = "abc123"
	BuildDate = "2025-11-20"

	// Verify values
	if Version != "0.1.0-test" {
		t.Errorf("Version = %q, want %q", Version, "0.1.0-test")
	}
	if GitCommit != "abc123" {
		t.Errorf("GitCommit = %q, want %q", GitCommit, "abc123")
	}
	if BuildDate != "2025-11-20" {
		t.Errorf("BuildDate = %q, want %q", BuildDate, "2025-11-20")
	}

	// Restore original values
	Version = origVersion
	GitCommit = origGitCommit
	BuildDate = origBuildDate
}

func TestVersionCommandExists(t *testing.T) {
	// Test that the version command is properly initialized
	if versionCmd == nil {
		t.Fatal("versionCmd is nil")
	}

	if versionCmd.Use != "version" {
		t.Errorf("versionCmd.Use = %q, want %q", versionCmd.Use, "version")
	}

	if versionCmd.Short == "" {
		t.Error("versionCmd.Short should not be empty")
	}

	if versionCmd.Run == nil {
		t.Error("versionCmd.Run should not be nil")
	}
}

func TestRuntimeInfo(t *testing.T) {
	// Test that we can get runtime information (used by version command)
	goVersion := runtime.Version()
	if goVersion == "" {
		t.Error("runtime.Version() should not be empty")
	}

	goos := runtime.GOOS
	if goos == "" {
		t.Error("runtime.GOOS should not be empty")
	}

	goarch := runtime.GOARCH
	if goarch == "" {
		t.Error("runtime.GOARCH should not be empty")
	}
}
