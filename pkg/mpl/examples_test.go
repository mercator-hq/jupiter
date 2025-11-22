package mpl

import (
	"path/filepath"
	"testing"
)

// TestParseAllExamples tests parsing all 21 example policies from Feature 5A
func TestParseAllExamples(t *testing.T) {
	examples := []string{
		"01-basic-deny.yaml",
		"02-pii-detection.yaml",
		"03-token-limits.yaml",
		"04-model-routing.yaml",
		"05-rate-limiting.yaml",
		"06-prompt-injection.yaml",
		"07-cost-control.yaml",
		"08-compliance.yaml",
		"09-data-residency.yaml",
		"10-multi-turn.yaml",
		"11-sensitive-content.yaml",
		"12-user-attributes.yaml",
		"13-time-based.yaml",
		"14-environment.yaml",
		"15-model-allowlist.yaml",
		"16-response-filtering.yaml",
		"17-tool-calling.yaml",
		"18-streaming.yaml",
		"19-multimodal.yaml",
		"20-audit-trail.yaml",
		"21-department-based.yaml",
	}

	examplesDir := "../../docs/mpl/examples"

	for _, example := range examples {
		t.Run(example, func(t *testing.T) {
			path := filepath.Join(examplesDir, example)
			policy, err := ParseAndValidate(path)
			if err != nil {
				t.Errorf("Failed to parse %s: %v", example, err)
				return
			}

			// Basic validation
			if policy.MPLVersion != "1.0" {
				t.Errorf("%s: mpl_version = %q, want %q", example, policy.MPLVersion, "1.0")
			}
			if policy.Name == "" {
				t.Errorf("%s: missing policy name", example)
			}
			if policy.Version == "" {
				t.Errorf("%s: missing policy version", example)
			}
			if len(policy.Rules) == 0 {
				t.Errorf("%s: no rules defined", example)
			}

			t.Logf("✅ %s: %d rules, %d variables", example, len(policy.Rules), len(policy.Variables))
		})
	}
}
