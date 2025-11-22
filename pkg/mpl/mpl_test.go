package mpl

import (
	"testing"
)

// TestParseAndValidate tests the high-level API
func TestParseAndValidate(t *testing.T) {
	policy, err := ParseAndValidate("../../internal/mpl/testdata/valid/simple.yaml")
	if err != nil {
		t.Fatalf("ParseAndValidate() failed: %v", err)
	}

	if policy.Name != "simple-policy" {
		t.Errorf("Policy name = %q, want %q", policy.Name, "simple-policy")
	}
}

// TestParseAndValidateBytes tests parsing from bytes
func TestParseAndValidateBytes(t *testing.T) {
	yaml := []byte(`
mpl_version: "1.0"
name: "test-policy"
version: "1.0.0"

rules:
  - name: "test-rule"
    conditions:
      - field: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: "allow"
`)

	policy, err := ParseAndValidateBytes(yaml, "memory://test")
	if err != nil {
		t.Fatalf("ParseAndValidateBytes() failed: %v", err)
	}

	if policy.Name != "test-policy" {
		t.Errorf("Policy name = %q, want %q", policy.Name, "test-policy")
	}
}

// BenchmarkParse benchmarks policy parsing
func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := Parse("../../internal/mpl/testdata/valid/simple.yaml")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseAndValidate benchmarks parsing + validation
func BenchmarkParseAndValidate(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseAndValidate("../../internal/mpl/testdata/valid/simple.yaml")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseWithVariables benchmarks parsing policies with variables
func BenchmarkParseWithVariables(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := ParseAndValidate("../../internal/mpl/testdata/valid/with-variables.yaml")
		if err != nil {
			b.Fatal(err)
		}
	}
}
