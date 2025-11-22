package engine

import (
	"context"
	"log/slog"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/processing"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// TestMatchSimple_FieldConditions tests simple field condition matching
func TestMatchSimple_FieldConditions(t *testing.T) {
	tests := []struct {
		name          string
		field         string
		operator      ast.Operator
		expectedValue interface{}
		actualValue   interface{}
		wantMatch     bool
		wantError     bool
	}{
		{
			name:          "exact match - string",
			field:         "request.model",
			operator:      ast.OperatorEqual,
			expectedValue: "gpt-4",
			actualValue:   "gpt-4",
			wantMatch:     true,
		},
		{
			name:          "exact match - number",
			field:         "request.tokens",
			operator:      ast.OperatorEqual,
			expectedValue: float64(1000),
			actualValue:   float64(1000),
			wantMatch:     true,
		},
		{
			name:          "not equal",
			field:         "request.model",
			operator:      ast.OperatorNotEqual,
			expectedValue: "gpt-3.5",
			actualValue:   "gpt-4",
			wantMatch:     true,
		},
		{
			name:          "greater than",
			field:         "request.tokens",
			operator:      ast.OperatorGreaterThan,
			expectedValue: float64(500),
			actualValue:   1000,
			wantMatch:     true,
		},
		{
			name:          "less than",
			field:         "request.tokens",
			operator:      ast.OperatorLessThan,
			expectedValue: float64(2000),
			actualValue:   1000,
			wantMatch:     true,
		},
		{
			name:          "greater or equal",
			field:         "request.tokens",
			operator:      ast.OperatorGreaterEqual,
			expectedValue: float64(1000),
			actualValue:   1000,
			wantMatch:     true,
		},
		{
			name:          "less or equal",
			field:         "request.tokens",
			operator:      ast.OperatorLessEqual,
			expectedValue: float64(1000),
			actualValue:   1000,
			wantMatch:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultEngineConfig()
			matcher := NewDefaultMatcher(slog.Default(), config)

			// Create evaluation context with test data
			evalCtx := createTestEvalContext(tt.actualValue)

			// Create condition node
			condition := &ast.ConditionNode{
				Type:     ast.ConditionTypeSimple,
				Field:    tt.field,
				Operator: tt.operator,
				Value: &ast.ValueNode{
					Type:  ast.ValueTypeNumber,
					Value: tt.expectedValue,
				},
			}

			// Match condition
			matched, err := matcher.matchSimple(context.Background(), condition, evalCtx)

			if (err != nil) != tt.wantError {
				t.Errorf("matchSimple() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if matched != tt.wantMatch {
				t.Errorf("matchSimple() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestMatchSimple_PatternConditions tests pattern matching (regex, substring)
func TestMatchSimple_PatternConditions(t *testing.T) {
	tests := []struct {
		name      string
		operator  ast.Operator
		pattern   string
		content   string
		wantMatch bool
		wantError bool
	}{
		{
			name:      "contains substring",
			operator:  ast.OperatorContains,
			pattern:   "secret",
			content:   "this contains a secret word",
			wantMatch: true,
		},
		{
			name:      "does not contain",
			operator:  ast.OperatorContains,
			pattern:   "password",
			content:   "this is safe content",
			wantMatch: false,
		},
		{
			name:      "matches regex - SSN",
			operator:  ast.OperatorMatches,
			pattern:   `\d{3}-\d{2}-\d{4}`,
			content:   "SSN: 123-45-6789",
			wantMatch: true,
		},
		{
			name:      "matches regex - email",
			operator:  ast.OperatorMatches,
			pattern:   `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
			content:   "contact: user@example.com",
			wantMatch: true,
		},
		{
			name:      "starts with",
			operator:  ast.OperatorStartsWith,
			pattern:   "Hello",
			content:   "Hello, world!",
			wantMatch: true,
		},
		{
			name:      "ends with",
			operator:  ast.OperatorEndsWith,
			pattern:   "world",
			content:   "Hello world",
			wantMatch: true,
		},
		{
			name:      "invalid regex",
			operator:  ast.OperatorMatches,
			pattern:   `[invalid(regex`,
			content:   "test",
			wantMatch: false,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matched, err := evaluateOperator(tt.operator, tt.content, tt.pattern)

			if (err != nil) != tt.wantError {
				t.Errorf("evaluateOperator() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError && matched != tt.wantMatch {
				t.Errorf("evaluateOperator() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestMatchAll_BooleanLogic tests AND logic
func TestMatchAll_BooleanLogic(t *testing.T) {
	config := DefaultEngineConfig()
	matcher := NewDefaultMatcher(slog.Default(), config)
	evalCtx := createTestEvalContext(1000)

	tests := []struct {
		name      string
		children  []*ast.ConditionNode
		wantMatch bool
	}{
		{
			name: "all match",
			children: []*ast.ConditionNode{
				createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(500)),
				createSimpleCondition("request.tokens", ast.OperatorLessThan, float64(2000)),
			},
			wantMatch: true,
		},
		{
			name: "one does not match",
			children: []*ast.ConditionNode{
				createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(500)),
				createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(2000)),
			},
			wantMatch: false,
		},
		{
			name: "none match",
			children: []*ast.ConditionNode{
				createSimpleCondition("request.tokens", ast.OperatorLessThan, float64(500)),
				createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(2000)),
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := &ast.ConditionNode{
				Type:     ast.ConditionTypeAll,
				Children: tt.children,
			}

			matched, err := matcher.matchAll(context.Background(), condition, evalCtx)
			if err != nil {
				t.Fatalf("matchAll() error = %v", err)
			}

			if matched != tt.wantMatch {
				t.Errorf("matchAll() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestMatchAny_BooleanLogic tests OR logic
func TestMatchAny_BooleanLogic(t *testing.T) {
	config := DefaultEngineConfig()
	matcher := NewDefaultMatcher(slog.Default(), config)
	evalCtx := createTestEvalContext(1000)

	tests := []struct {
		name      string
		children  []*ast.ConditionNode
		wantMatch bool
	}{
		{
			name: "at least one matches",
			children: []*ast.ConditionNode{
				createSimpleCondition("request.tokens", ast.OperatorLessThan, float64(500)),
				createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(500)),
			},
			wantMatch: true,
		},
		{
			name: "all match",
			children: []*ast.ConditionNode{
				createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(500)),
				createSimpleCondition("request.tokens", ast.OperatorLessThan, float64(2000)),
			},
			wantMatch: true,
		},
		{
			name: "none match",
			children: []*ast.ConditionNode{
				createSimpleCondition("request.tokens", ast.OperatorLessThan, float64(500)),
				createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(2000)),
			},
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := &ast.ConditionNode{
				Type:     ast.ConditionTypeAny,
				Children: tt.children,
			}

			matched, err := matcher.matchAny(context.Background(), condition, evalCtx)
			if err != nil {
				t.Fatalf("matchAny() error = %v", err)
			}

			if matched != tt.wantMatch {
				t.Errorf("matchAny() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestMatchNot_BooleanLogic tests NOT logic
func TestMatchNot_BooleanLogic(t *testing.T) {
	config := DefaultEngineConfig()
	matcher := NewDefaultMatcher(slog.Default(), config)
	evalCtx := createTestEvalContext(1000)

	tests := []struct {
		name      string
		child     *ast.ConditionNode
		wantMatch bool
	}{
		{
			name:      "negates true to false",
			child:     createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(500)),
			wantMatch: false,
		},
		{
			name:      "negates false to true",
			child:     createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(2000)),
			wantMatch: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			condition := &ast.ConditionNode{
				Type:     ast.ConditionTypeNot,
				Children: []*ast.ConditionNode{tt.child},
			}

			matched, err := matcher.matchNot(context.Background(), condition, evalCtx)
			if err != nil {
				t.Fatalf("matchNot() error = %v", err)
			}

			if matched != tt.wantMatch {
				t.Errorf("matchNot() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestMatchFunction_HasPII tests PII detection function
func TestMatchFunction_HasPII(t *testing.T) {
	config := DefaultEngineConfig()
	matcher := NewDefaultMatcher(slog.Default(), config)

	tests := []struct {
		name      string
		hasPII    bool
		wantMatch bool
	}{
		{
			name:      "has PII",
			hasPII:    true,
			wantMatch: true,
		},
		{
			name:      "no PII",
			hasPII:    false,
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evalCtx := &EvaluationContext{
				Request: &processing.EnrichedRequest{
					ContentAnalysis: &processing.ContentAnalysis{
						PIIDetection: &processing.PIIDetection{
							HasPII: tt.hasPII,
						},
					},
				},
			}

			condition := &ast.ConditionNode{
				Type:     ast.ConditionTypeFunction,
				Function: "has_pii",
			}

			matched, err := matcher.hasPII(condition, evalCtx)
			if err != nil {
				t.Fatalf("hasPII() error = %v", err)
			}

			if matched != tt.wantMatch {
				t.Errorf("hasPII() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestMatchFunction_InBusinessHours tests business hours check
func TestMatchFunction_InBusinessHours(t *testing.T) {
	tests := []struct {
		name      string
		testTime  time.Time
		wantMatch bool
	}{
		{
			name:      "weekday during business hours",
			testTime:  time.Date(2025, 11, 18, 10, 0, 0, 0, time.UTC), // Tuesday 10am
			wantMatch: true,
		},
		{
			name:      "weekday before business hours",
			testTime:  time.Date(2025, 11, 18, 8, 0, 0, 0, time.UTC), // Tuesday 8am
			wantMatch: false,
		},
		{
			name:      "weekday after business hours",
			testTime:  time.Date(2025, 11, 18, 18, 0, 0, 0, time.UTC), // Tuesday 6pm
			wantMatch: false,
		},
		{
			name:      "weekend",
			testTime:  time.Date(2025, 11, 16, 10, 0, 0, 0, time.UTC), // Sunday 10am
			wantMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultEngineConfig()
			config.BusinessHours = &BusinessHoursConfig{
				Timezone:   "UTC",
				DaysOfWeek: []int{1, 2, 3, 4, 5}, // Mon-Fri
				StartHour:  9,
				EndHour:    17,
			}

			matcher := NewDefaultMatcher(slog.Default(), config)
			evalCtx := &EvaluationContext{
				StartTime: tt.testTime,
			}

			condition := &ast.ConditionNode{
				Type:     ast.ConditionTypeFunction,
				Function: "in_business_hours",
			}

			matched, err := matcher.inBusinessHours(condition, evalCtx)
			if err != nil {
				t.Fatalf("inBusinessHours() error = %v", err)
			}

			if matched != tt.wantMatch {
				t.Errorf("inBusinessHours() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// TestFailSafeMode_MissingFields tests fail-safe behavior for missing fields
func TestFailSafeMode_MissingFields(t *testing.T) {
	tests := []struct {
		name         string
		failSafeMode FailSafeMode
		wantMatch    bool
		wantError    bool
	}{
		{
			name:         "fail-open treats missing field as match",
			failSafeMode: FailOpen,
			wantMatch:    true,
			wantError:    false,
		},
		{
			name:         "fail-closed treats missing field as error",
			failSafeMode: FailClosed,
			wantMatch:    false,
			wantError:    true,
		},
		{
			name:         "fail-safe-default treats missing field as no match",
			failSafeMode: FailSafeDefault,
			wantMatch:    false,
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := DefaultEngineConfig()
			config.FailSafeMode = tt.failSafeMode
			matcher := NewDefaultMatcher(slog.Default(), config)

			// Create eval context without the field we're looking for
			evalCtx := &EvaluationContext{
				Request: &processing.EnrichedRequest{
					RequestID: "test-123",
				},
			}

			// Try to match a field that doesn't exist
			condition := &ast.ConditionNode{
				Type:     ast.ConditionTypeSimple,
				Field:    "request.nonexistent_field",
				Operator: ast.OperatorEqual,
				Value: &ast.ValueNode{
					Type:  ast.ValueTypeString,
					Value: "test",
				},
			}

			matched, err := matcher.matchSimple(context.Background(), condition, evalCtx)

			if (err != nil) != tt.wantError {
				t.Errorf("matchSimple() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if matched != tt.wantMatch {
				t.Errorf("matchSimple() matched = %v, want %v", matched, tt.wantMatch)
			}
		})
	}
}

// Helper functions

func createTestEvalContext(tokenValue interface{}) *EvaluationContext {
	tokens := 0
	switch v := tokenValue.(type) {
	case int:
		tokens = v
	case float64:
		tokens = int(v)
	}

	return &EvaluationContext{
		RequestID: "test-123",
		Request: &processing.EnrichedRequest{
			RequestID: "test-123",
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-4",
			},
			TokenEstimate: &processing.TokenEstimate{
				TotalTokens: tokens,
			},
		},
		Tags:      make(map[string]string),
		StartTime: time.Now(),
	}
}

func createSimpleCondition(field string, operator ast.Operator, value interface{}) *ast.ConditionNode {
	return &ast.ConditionNode{
		Type:     ast.ConditionTypeSimple,
		Field:    field,
		Operator: operator,
		Value: &ast.ValueNode{
			Type:  ast.ValueTypeNumber,
			Value: value,
		},
	}
}
