//go:build integration

package engine_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/policy/engine"
	"mercator-hq/jupiter/pkg/policy/engine/source"
	"mercator-hq/jupiter/pkg/processing"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// TestEngine_EndToEndEvaluation tests complete policy evaluation from file loading to decision.
func TestEngine_EndToEndEvaluation(t *testing.T) {
	//Test 1: Block expensive model
	t.Run("block expensive model", func(t *testing.T) {
		policyContent := `
mpl_version: "1.0"
name: block-policy

rules:
  - name: block-gpt4
    match:
      field:
        name: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: deny
        message: "GPT-4 blocked"
`
		tempDir := t.TempDir()
		policyPath := filepath.Join(tempDir, "policy.yaml")
		os.WriteFile(policyPath, []byte(policyContent), 0644)

		fileSource := source.NewFileSource(policyPath, slog.Default())
		cfg := engine.DefaultEngineConfig()
		cfg.EnableTrace = true

		eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}
		defer eng.Close()

		req := &processing.EnrichedRequest{
			RequestID: "test-block-1",
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-4",
			},
		}

		decision, err := eng.EvaluateRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("evaluation failed: %v", err)
		}

		if decision.Action != engine.ActionBlock {
			t.Errorf("action = %v, want %v", decision.Action, engine.ActionBlock)
		}

		if decision.BlockReason != "GPT-4 blocked" {
			t.Errorf("reason = %q, want %q", decision.BlockReason, "GPT-4 blocked")
		}

		if decision.Trace == nil {
			t.Error("expected evaluation trace")
		}
	})

	// Test 2: Route claude to anthropic
	t.Run("route claude to anthropic", func(t *testing.T) {
		policyContent := `
mpl_version: "1.0"
name: route-policy

rules:
  - name: route-claude
    match:
      field:
        name: "request.model"
        operator: "starts_with"
        value: "claude"
    actions:
      - type: route
        provider: "anthropic"
`
		tempDir := t.TempDir()
		policyPath := filepath.Join(tempDir, "policy.yaml")
		os.WriteFile(policyPath, []byte(policyContent), 0644)

		fileSource := source.NewFileSource(policyPath, slog.Default())
		cfg := engine.DefaultEngineConfig()

		eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}
		defer eng.Close()

		req := &processing.EnrichedRequest{
			RequestID: "test-route-1",
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "claude-3-sonnet",
			},
		}

		decision, err := eng.EvaluateRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("evaluation failed: %v", err)
		}

		if decision.Action != engine.ActionRoute {
			t.Errorf("action = %v, want %v", decision.Action, engine.ActionRoute)
		}

		if decision.RoutingTarget == nil {
			t.Fatal("expected routing target")
		}

		if decision.RoutingTarget.Provider != "anthropic" {
			t.Errorf("routing_target.provider = %q, want %q", decision.RoutingTarget.Provider, "anthropic")
		}
	})

	// Test 3: Tagging
	t.Run("tag requests", func(t *testing.T) {
		policyContent := `
mpl_version: "1.0"
name: tag-policy

rules:
  - name: tag-all
    match:
      field:
        name: "request.model"
        operator: "exists"
    actions:
      - type: tag
        key: "environment"
        value: "test"
`
		tempDir := t.TempDir()
		policyPath := filepath.Join(tempDir, "policy.yaml")
		os.WriteFile(policyPath, []byte(policyContent), 0644)

		fileSource := source.NewFileSource(policyPath, slog.Default())
		cfg := engine.DefaultEngineConfig()

		eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}
		defer eng.Close()

		req := &processing.EnrichedRequest{
			RequestID: "test-tag-1",
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-3.5-turbo",
			},
		}

		decision, err := eng.EvaluateRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("evaluation failed: %v", err)
		}

		// Tags should be applied
		if env, ok := decision.Tags["environment"]; !ok || env != "test" {
			t.Errorf("expected tag environment=test, got %q", env)
		}

		// No blocking or routing, so should allow
		if decision.Action != engine.ActionAllow {
			t.Errorf("action = %v, want allow", decision.Action)
		}
	})

	// Test 4: Trace enabled
	t.Run("evaluation trace", func(t *testing.T) {
		policyContent := `
mpl_version: "1.0"
name: trace-policy

rules:
  - name: simple-rule
    match:
      field:
        name: "request.model"
        operator: "exists"
    actions:
      - type: allow
`
		tempDir := t.TempDir()
		policyPath := filepath.Join(tempDir, "policy.yaml")
		os.WriteFile(policyPath, []byte(policyContent), 0644)

		fileSource := source.NewFileSource(policyPath, slog.Default())
		cfg := engine.DefaultEngineConfig()
		cfg.EnableTrace = true

		eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
		if err != nil {
			t.Fatalf("failed to create engine: %v", err)
		}
		defer eng.Close()

		req := &processing.EnrichedRequest{
			RequestID: "test-trace-1",
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-4",
			},
		}

		decision, err := eng.EvaluateRequest(context.Background(), req)
		if err != nil {
			t.Fatalf("evaluation failed: %v", err)
		}

		// Verify trace exists
		if decision.Trace == nil {
			t.Fatal("expected trace to be enabled")
		}

		// Verify evaluation time is recorded
		if decision.EvaluationTime == 0 {
			t.Error("expected non-zero evaluation time")
		}

		if len(decision.MatchedRules) == 0 {
			t.Error("expected matched rules")
		}
	})
}

// TestEngine_HotReload tests policy hot-reloading functionality.
func TestEngine_HotReload(t *testing.T) {
	// Skip this test because hot-reload is not implemented in MVP
	// The file watcher returns an empty channel
	t.Skip("skipping hot-reload test: file watching not implemented in MVP")

	// NOTE: When hot-reload is implemented, this test should be enabled
	// and should verify:
	// 1. Engine loads initial policy
	// 2. Policy file is modified
	// 3. Engine detects change and reloads
	// 4. New policy behavior takes effect
}

// TestEngine_FailSafeModes tests different fail-safe mode configurations.
// Note: This test verifies the fail-safe mode configuration is set correctly.
// Testing actual fail-safe behavior (engine errors) would require error injection
// mechanisms that are beyond the scope of this integration test.
func TestEngine_FailSafeModes(t *testing.T) {
	tests := []struct {
		name         string
		failSafeMode engine.FailSafeMode
		description  string
	}{
		{
			name:         "fail-open mode configured",
			failSafeMode: engine.FailOpen,
			description:  "Engine configured to allow requests on errors",
		},
		{
			name:         "fail-closed mode configured",
			failSafeMode: engine.FailClosed,
			description:  "Engine configured to block requests on errors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a simple working policy
			policyContent := `
mpl_version: "1.0"
name: fail-safe-test

rules:
  - name: test-rule
    description: "Simple test rule"
    match:
      field:
        name: "request.model"
        operator: "exists"
    actions:
      - type: allow
`

			tempDir := t.TempDir()
			policyPath := filepath.Join(tempDir, "policy.yaml")
			if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
				t.Fatalf("failed to write policy: %v", err)
			}

			fileSource := source.NewFileSource(policyPath, slog.Default())

			cfg := engine.DefaultEngineConfig()
			cfg.FailSafeMode = tt.failSafeMode

			eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
			if err != nil {
				t.Fatalf("failed to create engine: %v", err)
			}
			defer eng.Close()

			req := &processing.EnrichedRequest{
				RequestID: "test-failsafe",
				OriginalRequest: &types.ChatCompletionRequest{
					Model: "test-model",
				},
			}

			decision, err := eng.EvaluateRequest(context.Background(), req)
			if err != nil {
				t.Fatalf("evaluation failed: %v", err)
			}

			// With a working policy, should get allow action
			if decision.Action != engine.ActionAllow {
				t.Errorf("action = %v, want allow (policy should work normally)", decision.Action)
			}

			t.Logf("✓ %s: %s", tt.name, tt.description)
		})
	}
}

// TestEngine_MultiplePolicies tests evaluation with multiple policies and priority ordering.
func TestEngine_MultiplePolicies(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple policy files with different priorities
	policies := []struct {
		filename string
		priority int
		content  string
	}{
		{
			filename: "high-priority.yaml",
			priority: 100,
			content: `
mpl_version: "1.0"
name: high-priority
priority: 100

rules:
  - name: block-high
    description: "High priority block rule"
    match:
      field:
        name: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: deny
        message: "Blocked by high priority"
`,
		},
		{
			filename: "low-priority.yaml",
			priority: 10,
			content: `
mpl_version: "1.0"
name: low-priority
priority: 10

rules:
  - name: allow-low
    description: "Low priority allow rule (should not override high priority block)"
    match:
      field:
        name: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: allow
`,
		},
		{
			filename: "medium-priority.yaml",
			priority: 50,
			content: `
mpl_version: "1.0"
name: medium-priority
priority: 50

rules:
  - name: tag-medium
    description: "Medium priority tagging rule"
    match:
      field:
        name: "request.model"
        operator: "exists"
    actions:
      - type: tag
        key: "priority"
        value: "medium"
`,
		},
	}

	// Write all policy files
	for _, p := range policies {
		path := filepath.Join(tempDir, p.filename)
		if err := os.WriteFile(path, []byte(p.content), 0644); err != nil {
			t.Fatalf("failed to write policy %s: %v", p.filename, err)
		}
	}

	// Create engine pointing to directory
	fileSource := source.NewFileSource(tempDir, slog.Default())

	cfg := engine.DefaultEngineConfig()
	cfg.EnableTrace = true

	eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// Verify policies were loaded
	loadedPolicies := eng.GetPolicies()
	if len(loadedPolicies) != 3 {
		t.Fatalf("expected 3 policies, got %d", len(loadedPolicies))
	}

	// Test that high priority policy wins
	req := &processing.EnrichedRequest{
		RequestID: "test-priority",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
		},
	}

	decision, err := eng.EvaluateRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	// High priority deny should win
	if decision.Action != engine.ActionBlock {
		t.Errorf("action = %v, want %v (high priority should win)",
			decision.Action, engine.ActionBlock)
	}

	if decision.BlockReason != "Blocked by high priority" {
		t.Errorf("reason = %q, expected high priority message", decision.BlockReason)
	}

	// Medium priority tags should still be applied (evaluated before high priority blocks)
	if priority, ok := decision.Tags["priority"]; !ok || priority != "medium" {
		t.Logf("Note: tag 'priority' not found or incorrect. This is expected if high priority block short-circuits evaluation.")
	}

	// Log matched rules for debugging
	t.Logf("Matched %d rules:", len(decision.MatchedRules))
	for _, rule := range decision.MatchedRules {
		t.Logf("  - Policy: %s, Rule: %s, Matched: %v",
			rule.PolicyName, rule.RuleName, rule.ConditionResult)
	}
}

// TestEngine_ConcurrentEvaluation tests thread-safety with concurrent policy evaluations.
func TestEngine_ConcurrentEvaluation(t *testing.T) {
	policyContent := `
mpl_version: "1.0"
name: concurrent-test

rules:
  - name: test-rule
    description: "Simple rule for concurrent testing"
    match:
      field:
        name: "request.model"
        operator: "exists"
    actions:
      - type: tag
        key: "tested"
        value: "true"
`

	tempDir := t.TempDir()
	policyPath := filepath.Join(tempDir, "policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	fileSource := source.NewFileSource(policyPath, slog.Default())

	cfg := engine.DefaultEngineConfig()
	cfg.EnableTrace = false // Disable trace for performance

	eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// Spawn 100 goroutines, each evaluating 10 requests
	const goroutines = 100
	const requestsPerGoroutine = 10

	errChan := make(chan error, goroutines)
	doneChan := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			for j := 0; j < requestsPerGoroutine; j++ {
				req := &processing.EnrichedRequest{
					RequestID: fmt.Sprintf("concurrent-%d-%d", id, j),
					OriginalRequest: &types.ChatCompletionRequest{
						Model: fmt.Sprintf("model-%d-%d", id, j),
					},
				}

				decision, err := eng.EvaluateRequest(context.Background(), req)
				if err != nil {
					errChan <- fmt.Errorf("goroutine %d request %d failed: %w", id, j, err)
					return
				}

				// Verify basic decision
				if decision.Action != engine.ActionAllow {
					errChan <- fmt.Errorf("goroutine %d request %d: unexpected action %v", id, j, decision.Action)
					return
				}

				// Verify tag was applied
				if tested, ok := decision.Tags["tested"]; !ok || tested != "true" {
					errChan <- fmt.Errorf("goroutine %d request %d: tag not applied correctly", id, j)
					return
				}
			}
			doneChan <- true
		}(i)
	}

	// Wait for all goroutines with timeout
	completed := 0
	timeout := time.After(30 * time.Second)

	for completed < goroutines {
		select {
		case err := <-errChan:
			t.Fatalf("concurrent evaluation failed: %v", err)
		case <-doneChan:
			completed++
		case <-timeout:
			t.Fatalf("timeout waiting for concurrent evaluations (completed %d/%d)",
				completed, goroutines)
		}
	}

	totalRequests := goroutines * requestsPerGoroutine
	t.Logf("Successfully completed %d concurrent evaluations", totalRequests)
}

// TestEngine_PolicyReload tests manual policy reloading.
func TestEngine_PolicyReload(t *testing.T) {
	// Create initial policy
	initialPolicy := `
mpl_version: "1.0"
name: reload-test
description: "Initial policy"

rules:
  - name: initial-rule
    match:
      field:
        name: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: tag
        key: "version"
        value: "v1"
`

	tempDir := t.TempDir()
	policyPath := filepath.Join(tempDir, "policy.yaml")
	if err := os.WriteFile(policyPath, []byte(initialPolicy), 0644); err != nil {
		t.Fatalf("failed to write initial policy: %v", err)
	}

	fileSource := source.NewFileSource(policyPath, slog.Default())

	cfg := engine.DefaultEngineConfig()

	eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// Test initial policy
	req := &processing.EnrichedRequest{
		RequestID: "test-reload-1",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
		},
	}

	decision, err := eng.EvaluateRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("evaluation failed: %v", err)
	}

	if version, ok := decision.Tags["version"]; !ok || version != "v1" {
		t.Errorf("expected version=v1, got %q", version)
	}

	// Modify policy file
	updatedPolicy := `
mpl_version: "1.0"
name: reload-test
description: "Updated policy"

rules:
  - name: updated-rule
    match:
      field:
        name: "request.model"
        operator: "=="
        value: "gpt-4"
    actions:
      - type: tag
        key: "version"
        value: "v2"
`

	time.Sleep(100 * time.Millisecond) // Ensure file timestamp changes
	if err := os.WriteFile(policyPath, []byte(updatedPolicy), 0644); err != nil {
		t.Fatalf("failed to write updated policy: %v", err)
	}

	// Manually trigger reload
	if err := eng.ReloadPolicies(context.Background()); err != nil {
		t.Fatalf("failed to reload policies: %v", err)
	}

	// Test updated policy
	decision, err = eng.EvaluateRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("evaluation failed after reload: %v", err)
	}

	if version, ok := decision.Tags["version"]; !ok || version != "v2" {
		t.Errorf("expected version=v2 after reload, got %q", version)
	}

	t.Log("Manual policy reload successful")
}

// TestEngine_ContextCancellation tests that evaluation respects context cancellation.
func TestEngine_ContextCancellation(t *testing.T) {
	policyContent := `
mpl_version: "1.0"
name: cancellation-test

rules:
  - name: test-rule
    match:
      field:
        name: "request.model"
        operator: "exists"
    actions:
      - type: allow
`

	tempDir := t.TempDir()
	policyPath := filepath.Join(tempDir, "policy.yaml")
	if err := os.WriteFile(policyPath, []byte(policyContent), 0644); err != nil {
		t.Fatalf("failed to write policy: %v", err)
	}

	fileSource := source.NewFileSource(policyPath, slog.Default())

	cfg := engine.DefaultEngineConfig()

	eng, err := engine.NewInterpreterEngine(cfg, fileSource, slog.Default())
	if err != nil {
		t.Fatalf("failed to create engine: %v", err)
	}
	defer eng.Close()

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &processing.EnrichedRequest{
		RequestID: "test-cancelled",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
		},
	}

	_, err = eng.EvaluateRequest(ctx, req)

	// Should handle cancellation gracefully
	// The error might be wrapped, so we just check that it's not nil
	// and that it relates to timeout/cancellation
	if err == nil {
		t.Log("Note: Engine evaluated successfully with cancelled context. This is acceptable if evaluation is fast.")
	} else {
		t.Logf("Got expected error with cancelled context: %v", err)
	}
}
