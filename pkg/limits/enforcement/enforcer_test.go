package enforcement

import (
	"context"
	"testing"
	"time"
)

func TestNewEnforcer_Defaults(t *testing.T) {
	enforcer := NewEnforcer(Config{})

	config := enforcer.GetConfig()
	if config.DefaultAction != ActionBlock {
		t.Errorf("Expected default action Block, got %s", config.DefaultAction)
	}
	if config.QueueDepth != 100 {
		t.Errorf("Expected queue depth 100, got %d", config.QueueDepth)
	}
	if config.QueueTimeout != 30*time.Second {
		t.Errorf("Expected queue timeout 30s, got %v", config.QueueTimeout)
	}
}

func TestEnforcer_Allow(t *testing.T) {
	enforcer := NewEnforcer(Config{})
	ctx := context.Background()

	result, err := enforcer.Enforce(ctx, ActionAllow, "", "", 0)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed")
	}
	if result.Action != ActionAllow {
		t.Errorf("Expected action Allow, got %s", result.Action)
	}
}

func TestEnforcer_Block(t *testing.T) {
	enforcer := NewEnforcer(Config{})
	ctx := context.Background()

	result, err := enforcer.Enforce(ctx, ActionBlock, "rate limit exceeded", "gpt-4", 30*time.Second)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if result.Allowed {
		t.Error("Expected request to be blocked")
	}
	if result.Action != ActionBlock {
		t.Errorf("Expected action Block, got %s", result.Action)
	}
	if result.Reason != "rate limit exceeded" {
		t.Errorf("Expected reason 'rate limit exceeded', got %s", result.Reason)
	}
	if result.RetryAfter != 30*time.Second {
		t.Errorf("Expected retry after 30s, got %v", result.RetryAfter)
	}
}

func TestEnforcer_Queue(t *testing.T) {
	enforcer := NewEnforcer(Config{
		QueueDepth:   50,
		QueueTimeout: 10 * time.Second,
	})
	ctx := context.Background()

	result, err := enforcer.Enforce(ctx, ActionQueue, "rate limit exceeded", "gpt-4", 5*time.Second)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if result.Action != ActionQueue {
		t.Errorf("Expected action Queue, got %s", result.Action)
	}
	if result.Allowed {
		t.Error("Expected queue action to indicate not immediately allowed")
	}
}

func TestEnforcer_Downgrade_Success(t *testing.T) {
	enforcer := NewEnforcer(Config{
		ModelDowngrades: map[string]string{
			"gpt-4":         "gpt-3.5-turbo",
			"claude-3-opus": "claude-3-sonnet",
		},
	})
	ctx := context.Background()

	result, err := enforcer.Enforce(ctx, ActionDowngrade, "budget exceeded", "gpt-4", 0)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed with downgrade")
	}
	if result.Action != ActionDowngrade {
		t.Errorf("Expected action Downgrade, got %s", result.Action)
	}
	if result.DowngradedModel != "gpt-3.5-turbo" {
		t.Errorf("Expected downgraded model gpt-3.5-turbo, got %s", result.DowngradedModel)
	}
}

func TestEnforcer_Downgrade_NoMapping(t *testing.T) {
	enforcer := NewEnforcer(Config{
		ModelDowngrades: map[string]string{
			"gpt-4": "gpt-3.5-turbo",
		},
	})
	ctx := context.Background()

	// Try to downgrade a model with no mapping
	result, err := enforcer.Enforce(ctx, ActionDowngrade, "budget exceeded", "unknown-model", 0)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if result.Allowed {
		t.Error("Expected request to be blocked when no downgrade available")
	}
	if result.Action != ActionBlock {
		t.Errorf("Expected fallback to Block action, got %s", result.Action)
	}
}

func TestEnforcer_Alert(t *testing.T) {
	enforcer := NewEnforcer(Config{})
	ctx := context.Background()

	result, err := enforcer.Enforce(ctx, ActionAlert, "80% budget used", "gpt-4", 0)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if !result.Allowed {
		t.Error("Expected request to be allowed with alert")
	}
	if result.Action != ActionAlert {
		t.Errorf("Expected action Alert, got %s", result.Action)
	}
	if result.AlertMessage != "80% budget used" {
		t.Errorf("Expected alert message '80%% budget used', got %s", result.AlertMessage)
	}
}

func TestEnforcer_DefaultAction(t *testing.T) {
	enforcer := NewEnforcer(Config{
		DefaultAction: ActionAlert,
	})
	ctx := context.Background()

	// Enforce with empty action - should use default
	result, err := enforcer.Enforce(ctx, "", "some reason", "gpt-4", 0)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if result.Action != ActionAlert {
		t.Errorf("Expected default action Alert, got %s", result.Action)
	}
}

func TestEnforcer_InvalidAction(t *testing.T) {
	enforcer := NewEnforcer(Config{})
	ctx := context.Background()

	// Invalid action should fall back to default (Block)
	result, err := enforcer.Enforce(ctx, Action("invalid"), "some reason", "gpt-4", 0)
	if err != nil {
		t.Fatalf("Enforce failed: %v", err)
	}

	if result.Action != ActionBlock {
		t.Errorf("Expected fallback to Block action, got %s", result.Action)
	}
}
