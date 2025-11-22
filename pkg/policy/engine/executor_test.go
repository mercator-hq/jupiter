package engine

import (
	"context"
	"testing"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/processing"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// TestExecutor_Tag tests the tag action execution.
func TestExecutor_Tag(t *testing.T) {
	tests := []struct {
		name      string
		action    *ast.Action
		evalCtx   *EvaluationContext
		wantTags  map[string]string
		wantError bool
	}{
		{
			name: "static tag value",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key":   {Type: ast.ValueTypeString, Value: "environment"},
					"value": {Type: ast.ValueTypeString, Value: "production"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
			},
			wantTags: map[string]string{
				"environment": "production",
			},
			wantError: false,
		},
		{
			name: "dynamic tag from model field",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key":        {Type: ast.ValueTypeString, Value: "model"},
					"value_from": {Type: ast.ValueTypeString, Value: "request.model"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
				},
			},
			wantTags: map[string]string{
				"model": "gpt-4",
			},
			wantError: false,
		},
		{
			name: "dynamic tag from user field",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key":        {Type: ast.ValueTypeString, Value: "user"},
					"value_from": {Type: ast.ValueTypeString, Value: "request.user"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						User: "user@example.com",
					},
				},
			},
			wantTags: map[string]string{
				"user": "user@example.com",
			},
			wantError: false,
		},
		{
			name: "tag from model_family",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key":        {Type: ast.ValueTypeString, Value: "model_family"},
					"value_from": {Type: ast.ValueTypeString, Value: "request.model_family"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
					ModelFamily: "GPT-4",
				},
			},
			wantTags: map[string]string{
				"model_family": "GPT-4",
			},
			wantError: false,
		},
		{
			name: "tag from risk_score",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key":        {Type: ast.ValueTypeString, Value: "risk_level"},
					"value_from": {Type: ast.ValueTypeString, Value: "request.risk_score"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
					RiskScore: 7,
				},
			},
			wantTags: map[string]string{
				"risk_level": "7",
			},
			wantError: false,
		},
		{
			name: "tag expensive models",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key":   {Type: ast.ValueTypeString, Value: "cost_tier"},
					"value": {Type: ast.ValueTypeString, Value: "expensive"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
			},
			wantTags: map[string]string{
				"cost_tier": "expensive",
			},
			wantError: false,
		},
		{
			name: "missing key parameter",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"value": {Type: ast.ValueTypeString, Value: "test"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
			},
			wantTags:  map[string]string{},
			wantError: true,
		},
		{
			name: "no value or value_from - uses default",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key": {Type: ast.ValueTypeString, Value: "flagged"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
			},
			wantTags: map[string]string{
				"flagged": "true",
			},
			wantError: false,
		},
		{
			name: "value_from field extraction fails - uses default",
			action: &ast.Action{
				Type: ast.ActionTypeTag,
				Parameters: map[string]*ast.ValueNode{
					"key":        {Type: ast.ValueTypeString, Value: "user"},
					"value_from": {Type: ast.ValueTypeString, Value: "request.user"},
				},
			},
			evalCtx: &EvaluationContext{
				RequestID: "test-123",
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
						// User is empty
					},
				},
			},
			wantTags: map[string]string{
				"user": "true",
			},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			result, err := executor.Execute(context.Background(), tt.action, tt.evalCtx)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				if result.Success {
					t.Errorf("expected error, got success")
				}
				return
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			// Verify tags were added
			for k, v := range tt.wantTags {
				if got, ok := tt.evalCtx.Tags[k]; !ok {
					t.Errorf("tag %q not found", k)
				} else if got != v {
					t.Errorf("tag %q = %q, want %q", k, got, v)
				}
			}
		})
	}
}

// TestExecutor_MultipleTagActions tests that multiple tag actions accumulate tags.
func TestExecutor_MultipleTagActions(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	evalCtx := &EvaluationContext{
		RequestID: "test-multi",
		Request: &processing.EnrichedRequest{
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-4",
				User:  "user@example.com",
			},
		},
	}

	// Execute first tag action
	action1 := &ast.Action{
		Type: ast.ActionTypeTag,
		Parameters: map[string]*ast.ValueNode{
			"key":   {Type: ast.ValueTypeString, Value: "environment"},
			"value": {Type: ast.ValueTypeString, Value: "production"},
		},
	}

	result1, err := executor.Execute(context.Background(), action1, evalCtx)
	if err != nil {
		t.Fatalf("first tag action failed: %v", err)
	}
	if !result1.Success {
		t.Fatalf("first tag action not successful: %v", result1.Error)
	}

	// Execute second tag action
	action2 := &ast.Action{
		Type: ast.ActionTypeTag,
		Parameters: map[string]*ast.ValueNode{
			"key":        {Type: ast.ValueTypeString, Value: "model"},
			"value_from": {Type: ast.ValueTypeString, Value: "request.model"},
		},
	}

	result2, err := executor.Execute(context.Background(), action2, evalCtx)
	if err != nil {
		t.Fatalf("second tag action failed: %v", err)
	}
	if !result2.Success {
		t.Fatalf("second tag action not successful: %v", result2.Error)
	}

	// Verify both tags are present
	expectedTags := map[string]string{
		"environment": "production",
		"model":       "gpt-4",
	}

	if len(evalCtx.Tags) != len(expectedTags) {
		t.Errorf("expected %d tags, got %d", len(expectedTags), len(evalCtx.Tags))
	}

	for k, v := range expectedTags {
		if got, ok := evalCtx.Tags[k]; !ok {
			t.Errorf("tag %q not found", k)
		} else if got != v {
			t.Errorf("tag %q = %q, want %q", k, got, v)
		}
	}
}

// TestExecutor_Allow tests the allow action execution.
func TestExecutor_Allow(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	evalCtx := &EvaluationContext{
		RequestID: "test-allow",
	}

	action := &ast.Action{
		Type:       ast.ActionTypeAllow,
		Parameters: map[string]*ast.ValueNode{},
	}

	result, err := executor.Execute(context.Background(), action, evalCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	// Verify evaluation was stopped
	if !evalCtx.Stopped {
		t.Error("expected Stopped=true after allow action")
	}
}

// TestExecutor_Deny tests the deny action execution.
func TestExecutor_Deny(t *testing.T) {
	tests := []struct {
		name           string
		action         *ast.Action
		wantBlocked    bool
		wantMessage    string
		wantStatusCode int
	}{
		{
			name: "deny with default message",
			action: &ast.Action{
				Type:       ast.ActionTypeDeny,
				Parameters: map[string]*ast.ValueNode{},
			},
			wantBlocked:    true,
			wantMessage:    "Request denied by policy",
			wantStatusCode: 403,
		},
		{
			name: "deny with custom message",
			action: &ast.Action{
				Type: ast.ActionTypeDeny,
				Parameters: map[string]*ast.ValueNode{
					"message":     {Type: ast.ValueTypeString, Value: "PII detected"},
					"status_code": {Type: ast.ValueTypeNumber, Value: float64(451)},
				},
			},
			wantBlocked:    true,
			wantMessage:    "PII detected",
			wantStatusCode: 451,
		},
		{
			name: "deny with custom message only",
			action: &ast.Action{
				Type: ast.ActionTypeDeny,
				Parameters: map[string]*ast.ValueNode{
					"message": {Type: ast.ValueTypeString, Value: "Rate limit exceeded"},
				},
			},
			wantBlocked:    true,
			wantMessage:    "Rate limit exceeded",
			wantStatusCode: 403,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			evalCtx := &EvaluationContext{
				RequestID: "test-deny",
			}

			result, err := executor.Execute(context.Background(), tt.action, evalCtx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			if evalCtx.BlockReason == "" {
				t.Error("expected BlockReason to be set")
			}

			if evalCtx.BlockReason != tt.wantMessage {
				t.Errorf("BlockReason = %q, want %q", evalCtx.BlockReason, tt.wantMessage)
			}

			if evalCtx.BlockStatusCode != tt.wantStatusCode {
				t.Errorf("BlockStatusCode = %d, want %d", evalCtx.BlockStatusCode, tt.wantStatusCode)
			}
		})
	}
}

// TestExecutor_Log tests the log action execution.
func TestExecutor_Log(t *testing.T) {
	tests := []struct {
		name    string
		action  *ast.Action
		wantMsg string
		wantLvl string
	}{
		{
			name: "log with default level",
			action: &ast.Action{
				Type: ast.ActionTypeLog,
				Parameters: map[string]*ast.ValueNode{
					"message": {Type: ast.ValueTypeString, Value: "Test message"},
				},
			},
			wantMsg: "Test message",
			wantLvl: "info",
		},
		{
			name: "log with warn level",
			action: &ast.Action{
				Type: ast.ActionTypeLog,
				Parameters: map[string]*ast.ValueNode{
					"message": {Type: ast.ValueTypeString, Value: "Warning message"},
					"level":   {Type: ast.ValueTypeString, Value: "warn"},
				},
			},
			wantMsg: "Warning message",
			wantLvl: "warn",
		},
		{
			name: "log with error level",
			action: &ast.Action{
				Type: ast.ActionTypeLog,
				Parameters: map[string]*ast.ValueNode{
					"message": {Type: ast.ValueTypeString, Value: "Error occurred"},
					"level":   {Type: ast.ValueTypeString, Value: "error"},
				},
			},
			wantMsg: "Error occurred",
			wantLvl: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			evalCtx := &EvaluationContext{
				RequestID: "test-log",
			}

			result, err := executor.Execute(context.Background(), tt.action, evalCtx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			// Verify details
			if msg, ok := result.Details["message"].(string); !ok || msg != tt.wantMsg {
				t.Errorf("message = %q, want %q", msg, tt.wantMsg)
			}

			if lvl, ok := result.Details["level"].(string); !ok || lvl != tt.wantLvl {
				t.Errorf("level = %q, want %q", lvl, tt.wantLvl)
			}
		})
	}
}

// TestExecutor_Redact tests the redact action execution.
func TestExecutor_Redact(t *testing.T) {
	tests := []struct {
		name         string
		action       *ast.Action
		wantField    string
		wantStrategy string
		wantPattern  string
	}{
		{
			name: "redact with defaults",
			action: &ast.Action{
				Type:       ast.ActionTypeRedact,
				Parameters: map[string]*ast.ValueNode{},
			},
			wantField:    "prompt",
			wantStrategy: "mask",
		},
		{
			name: "redact with pattern",
			action: &ast.Action{
				Type: ast.ActionTypeRedact,
				Parameters: map[string]*ast.ValueNode{
					"field":       {Type: ast.ValueTypeString, Value: "messages.0.content"},
					"pattern":     {Type: ast.ValueTypeString, Value: "\\b\\d{3}-\\d{2}-\\d{4}\\b"},
					"replacement": {Type: ast.ValueTypeString, Value: "[SSN-REDACTED]"},
					"strategy":    {Type: ast.ValueTypeString, Value: "replace"},
				},
			},
			wantField:    "messages.0.content",
			wantStrategy: "replace",
			wantPattern:  "\\b\\d{3}-\\d{2}-\\d{4}\\b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			evalCtx := &EvaluationContext{
				RequestID: "test-redact",
			}

			result, err := executor.Execute(context.Background(), tt.action, evalCtx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			// Verify redaction was added
			if len(evalCtx.Redactions) == 0 {
				t.Error("expected redaction to be added")
			}
		})
	}
}

// TestExecutor_Modify tests the modify action execution.
func TestExecutor_Modify(t *testing.T) {
	tests := []struct {
		name          string
		action        *ast.Action
		wantField     string
		wantOperation string
		wantError     bool
	}{
		{
			name: "modify temperature",
			action: &ast.Action{
				Type: ast.ActionTypeModify,
				Parameters: map[string]*ast.ValueNode{
					"field": {Type: ast.ValueTypeString, Value: "temperature"},
					"value": {Type: ast.ValueTypeNumber, Value: 0.5},
				},
			},
			wantField:     "temperature",
			wantOperation: "set",
			wantError:     false,
		},
		{
			name: "modify with operation",
			action: &ast.Action{
				Type: ast.ActionTypeModify,
				Parameters: map[string]*ast.ValueNode{
					"field":     {Type: ast.ValueTypeString, Value: "max_tokens"},
					"value":     {Type: ast.ValueTypeNumber, Value: 100},
					"operation": {Type: ast.ValueTypeString, Value: "add"},
				},
			},
			wantField:     "max_tokens",
			wantOperation: "add",
			wantError:     false,
		},
		{
			name: "missing field parameter",
			action: &ast.Action{
				Type: ast.ActionTypeModify,
				Parameters: map[string]*ast.ValueNode{
					"value": {Type: ast.ValueTypeNumber, Value: 1.0},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			evalCtx := &EvaluationContext{
				RequestID: "test-modify",
			}

			result, err := executor.Execute(context.Background(), tt.action, evalCtx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				if result.Success {
					t.Error("expected error, got success")
				}
				return
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			// Verify transformation was added
			if len(evalCtx.Transformations) == 0 {
				t.Error("expected transformation to be added")
			}
		})
	}
}

// TestExecutor_Route tests the route action execution.
func TestExecutor_Route(t *testing.T) {
	tests := []struct {
		name         string
		action       *ast.Action
		wantProvider string
		wantModel    string
		wantFallback []string
		wantError    bool
	}{
		{
			name: "route to provider",
			action: &ast.Action{
				Type: ast.ActionTypeRoute,
				Parameters: map[string]*ast.ValueNode{
					"provider": {Type: ast.ValueTypeString, Value: "anthropic"},
				},
			},
			wantProvider: "anthropic",
			wantError:    false,
		},
		{
			name: "route with model override",
			action: &ast.Action{
				Type: ast.ActionTypeRoute,
				Parameters: map[string]*ast.ValueNode{
					"provider": {Type: ast.ValueTypeString, Value: "openai"},
					"model":    {Type: ast.ValueTypeString, Value: "gpt-4-turbo"},
				},
			},
			wantProvider: "openai",
			wantModel:    "gpt-4-turbo",
			wantError:    false,
		},
		{
			name: "route with fallback providers",
			action: &ast.Action{
				Type: ast.ActionTypeRoute,
				Parameters: map[string]*ast.ValueNode{
					"provider": {Type: ast.ValueTypeString, Value: "openai"},
					"fallback": {
						Type:  ast.ValueTypeArray,
						Value: []interface{}{"anthropic", "cohere"},
					},
				},
			},
			wantProvider: "openai",
			wantFallback: []string{"anthropic", "cohere"},
			wantError:    false,
		},
		{
			name: "missing provider parameter",
			action: &ast.Action{
				Type: ast.ActionTypeRoute,
				Parameters: map[string]*ast.ValueNode{
					"model": {Type: ast.ValueTypeString, Value: "gpt-4"},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			evalCtx := &EvaluationContext{
				RequestID: "test-route",
			}

			result, err := executor.Execute(context.Background(), tt.action, evalCtx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				if result.Success {
					t.Error("expected error, got success")
				}
				return
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			if evalCtx.RoutingTarget == nil {
				t.Error("expected RoutingTarget to be set")
			} else if evalCtx.RoutingTarget.Provider != tt.wantProvider {
				t.Errorf("RoutingTarget.Provider = %q, want %q", evalCtx.RoutingTarget.Provider, tt.wantProvider)
			}

			if tt.wantModel != "" {
				if evalCtx.RoutingTarget == nil {
					t.Error("expected RoutingTarget to be set for model override")
				} else if evalCtx.RoutingTarget.Model != tt.wantModel {
					t.Errorf("RoutingTarget.Model = %q, want %q", evalCtx.RoutingTarget.Model, tt.wantModel)
				}
			}

			if len(tt.wantFallback) > 0 {
				if evalCtx.RoutingTarget == nil {
					t.Error("expected RoutingTarget to be set for fallback")
				} else if len(evalCtx.RoutingTarget.Fallback) != len(tt.wantFallback) {
					t.Errorf("Fallback length = %d, want %d", len(evalCtx.RoutingTarget.Fallback), len(tt.wantFallback))
				} else {
					for i, fb := range tt.wantFallback {
						if evalCtx.RoutingTarget.Fallback[i] != fb {
							t.Errorf("Fallback[%d] = %q, want %q", i, evalCtx.RoutingTarget.Fallback[i], fb)
						}
					}
				}
			}
		})
	}
}

// TestExecutor_Alert tests the alert action execution.
func TestExecutor_Alert(t *testing.T) {
	tests := []struct {
		name            string
		action          *ast.Action
		wantDestination string
		wantType        string
		wantError       bool
	}{
		{
			name: "alert with webhook",
			action: &ast.Action{
				Type: ast.ActionTypeAlert,
				Parameters: map[string]*ast.ValueNode{
					"destination": {Type: ast.ValueTypeString, Value: "https://example.com/webhook"},
					"message":     {Type: ast.ValueTypeString, Value: "Policy alert"},
				},
			},
			wantDestination: "https://example.com/webhook",
			wantType:        "webhook",
			wantError:       false,
		},
		{
			name: "alert with email type",
			action: &ast.Action{
				Type: ast.ActionTypeAlert,
				Parameters: map[string]*ast.ValueNode{
					"destination": {Type: ast.ValueTypeString, Value: "admin@example.com"},
					"message":     {Type: ast.ValueTypeString, Value: "High risk detected"},
					"type":        {Type: ast.ValueTypeString, Value: "email"},
				},
			},
			wantDestination: "admin@example.com",
			wantType:        "email",
			wantError:       false,
		},
		{
			name: "missing destination parameter",
			action: &ast.Action{
				Type: ast.ActionTypeAlert,
				Parameters: map[string]*ast.ValueNode{
					"message": {Type: ast.ValueTypeString, Value: "Test"},
				},
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			evalCtx := &EvaluationContext{
				RequestID: "test-alert",
			}

			result, err := executor.Execute(context.Background(), tt.action, evalCtx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if tt.wantError {
				if result.Success {
					t.Error("expected error, got success")
				}
				return
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}

			// Verify notification was added
			if len(evalCtx.Notifications) == 0 {
				t.Error("expected notification to be added")
			}
		})
	}
}

// TestExecutor_RateLimit tests the rate_limit action execution.
func TestExecutor_RateLimit(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	evalCtx := &EvaluationContext{
		RequestID: "test-rate-limit",
	}

	action := &ast.Action{
		Type: ast.ActionTypeRateLimit,
		Parameters: map[string]*ast.ValueNode{
			"limit":  {Type: ast.ValueTypeNumber, Value: float64(100)},
			"window": {Type: ast.ValueTypeString, Value: "1h"},
		},
	}

	result, err := executor.Execute(context.Background(), action, evalCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	// Verify details
	if limit, ok := result.Details["limit"].(int); !ok || limit != 100 {
		t.Errorf("limit = %v, want 100", limit)
	}
}

// TestExecutor_Budget tests the budget action execution.
func TestExecutor_Budget(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	evalCtx := &EvaluationContext{
		RequestID: "test-budget",
	}

	action := &ast.Action{
		Type: ast.ActionTypeBudget,
		Parameters: map[string]*ast.ValueNode{
			"limit": {Type: ast.ValueTypeNumber, Value: float64(100.50)},
		},
	}

	result, err := executor.Execute(context.Background(), action, evalCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %v", result.Error)
	}

	// Verify details
	if limit, ok := result.Details["limit"].(float64); !ok || limit != 100.50 {
		t.Errorf("limit = %v, want 100.50", limit)
	}
}

// TestExecutor_UnknownAction tests handling of unknown action types.
func TestExecutor_UnknownAction(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	evalCtx := &EvaluationContext{
		RequestID: "test-unknown",
	}

	action := &ast.Action{
		Type:       "unknown_action",
		Parameters: map[string]*ast.ValueNode{},
	}

	result, err := executor.Execute(context.Background(), action, evalCtx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Success {
		t.Error("expected failure for unknown action type")
	}

	if result.Error == nil {
		t.Error("expected error message for unknown action type")
	}
}

// TestExecutor_NilAction tests handling of nil actions.
func TestExecutor_NilAction(t *testing.T) {
	executor := NewDefaultExecutor(nil)
	evalCtx := &EvaluationContext{
		RequestID: "test-nil",
	}

	result, err := executor.Execute(context.Background(), nil, evalCtx)
	if err == nil {
		t.Error("expected error for nil action")
	}

	if result != nil {
		t.Error("expected nil result for nil action")
	}
}

// TestExtractFieldValue tests the extractFieldValue helper function.
func TestExtractFieldValue(t *testing.T) {
	tests := []struct {
		name      string
		evalCtx   *EvaluationContext
		fieldPath string
		wantValue interface{}
		wantError bool
	}{
		{
			name: "extract model",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
				},
			},
			fieldPath: "request.model",
			wantValue: "gpt-4",
			wantError: false,
		},
		{
			name: "extract user",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						User: "user@example.com",
					},
				},
			},
			fieldPath: "request.user",
			wantValue: "user@example.com",
			wantError: false,
		},
		{
			name: "extract model_family",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
					ModelFamily: "GPT-4",
				},
			},
			fieldPath: "request.model_family",
			wantValue: "GPT-4",
			wantError: false,
		},
		{
			name: "extract pricing_tier",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
					PricingTier: "premium",
				},
			},
			fieldPath: "request.pricing_tier",
			wantValue: "premium",
			wantError: false,
		},
		{
			name: "extract risk_score",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
					RiskScore: 8,
				},
			},
			fieldPath: "request.risk_score",
			wantValue: 8,
			wantError: false,
		},
		{
			name: "extract complexity_score",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
					ComplexityScore: 5,
				},
			},
			fieldPath: "request.complexity_score",
			wantValue: 5,
			wantError: false,
		},
		{
			name: "extract empty user - error",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
				},
			},
			fieldPath: "request.user",
			wantError: true,
		},
		{
			name: "extract empty model_family - error",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
				},
			},
			fieldPath: "request.model_family",
			wantError: true,
		},
		{
			name: "unsupported field path",
			evalCtx: &EvaluationContext{
				Request: &processing.EnrichedRequest{
					OriginalRequest: &types.ChatCompletionRequest{
						Model: "gpt-4",
					},
				},
			},
			fieldPath: "request.unsupported.field",
			wantError: true,
		},
		{
			name:      "nil request",
			evalCtx:   &EvaluationContext{},
			fieldPath: "request.model",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := extractFieldValue(tt.evalCtx, tt.fieldPath)

			if tt.wantError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if value != tt.wantValue {
				t.Errorf("value = %v, want %v", value, tt.wantValue)
			}
		})
	}
}

// TestExecutor_LogLevels tests all log levels.
func TestExecutor_LogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error", "unknown"}

	for _, level := range levels {
		t.Run("log_level_"+level, func(t *testing.T) {
			executor := NewDefaultExecutor(nil)
			evalCtx := &EvaluationContext{
				RequestID: "test-log-levels",
			}

			action := &ast.Action{
				Type: ast.ActionTypeLog,
				Parameters: map[string]*ast.ValueNode{
					"message": {Type: ast.ValueTypeString, Value: "Test message"},
					"level":   {Type: ast.ValueTypeString, Value: level},
				},
			}

			result, err := executor.Execute(context.Background(), action, evalCtx)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if !result.Success {
				t.Errorf("expected success, got error: %v", result.Error)
			}
		})
	}
}
