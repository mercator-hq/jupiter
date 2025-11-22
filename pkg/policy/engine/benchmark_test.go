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

// BenchmarkConditionMatching benchmarks simple condition matching
func BenchmarkConditionMatching(b *testing.B) {
	config := DefaultEngineConfig()
	matcher := NewDefaultMatcher(slog.Default(), config)
	evalCtx := createBenchEvalContext()
	condition := createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(500))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = matcher.matchSimple(context.Background(), condition, evalCtx)
	}
}

// BenchmarkBooleanLogic benchmarks AND/OR logic
func BenchmarkBooleanLogic(b *testing.B) {
	config := DefaultEngineConfig()
	matcher := NewDefaultMatcher(slog.Default(), config)
	evalCtx := createBenchEvalContext()

	condition := &ast.ConditionNode{
		Type: ast.ConditionTypeAll,
		Children: []*ast.ConditionNode{
			createSimpleCondition("request.tokens", ast.OperatorGreaterThan, float64(500)),
			createSimpleCondition("request.tokens", ast.OperatorLessThan, float64(2000)),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = matcher.matchAll(context.Background(), condition, evalCtx)
	}
}

// BenchmarkPatternMatching benchmarks regex matching
func BenchmarkPatternMatching(b *testing.B) {
	pattern := `\d{3}-\d{2}-\d{4}`
	content := "SSN: 123-45-6789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = evaluateOperator(ast.OperatorMatches, content, pattern)
	}
}

// BenchmarkActionExecution benchmarks action execution
func BenchmarkActionExecution(b *testing.B) {
	executor := NewDefaultExecutor(slog.Default())
	evalCtx := createBenchEvalContext()

	action := &ast.Action{
		Type: ast.ActionTypeDeny,
		Parameters: map[string]*ast.ValueNode{
			"message": {
				Type:  ast.ValueTypeString,
				Value: "Request denied",
			},
			"status_code": {
				Type:  ast.ValueTypeNumber,
				Value: float64(403),
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = executor.Execute(context.Background(), action, evalCtx)
	}
}

// Note: Engine-level benchmarks removed to avoid import cycles
// Run integration benchmarks in a separate package

// BenchmarkPrioritySort benchmarks policy priority sorting
func BenchmarkPrioritySort(b *testing.B) {
	policies := make([]*ast.Policy, 100)
	for i := 0; i < 100; i++ {
		policies[i] = createBenchPolicy(1)
		policies[i].Name = policies[i].Name + string(rune(i))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy to sort
		policiesCopy := make([]*ast.Policy, len(policies))
		copy(policiesCopy, policies)
		SortPoliciesByPriority(policiesCopy)
	}
}

// BenchmarkRedaction benchmarks content redaction
func BenchmarkRedaction(b *testing.B) {
	content := "Contact me at john@example.com or call 123-456-7890. My SSN is 123-45-6789."
	redaction := Redaction{
		Field:       "prompt",
		Strategy:    "mask",
		Pattern:     `\d{3}-\d{2}-\d{4}`,
		Replacement: "***-**-****",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ApplyRedaction(content, redaction)
	}
}

// Helper functions for benchmarks

func createBenchEvalContext() *EvaluationContext {
	return &EvaluationContext{
		RequestID: "bench-123",
		Request: &processing.EnrichedRequest{
			RequestID: "bench-123",
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-4",
			},
			TokenEstimate: &processing.TokenEstimate{
				TotalTokens:  1000,
				PromptTokens: 800,
			},
			RiskScore:       3,
			ComplexityScore: 5,
		},
		Tags:      make(map[string]string),
		StartTime: time.Now(),
	}
}

func createBenchPolicy(numRules int) *ast.Policy {
	policy := &ast.Policy{
		MPLVersion:  "1.0",
		Name:        "bench-policy",
		Version:     "1.0.0",
		Description: "Benchmark policy",
		Created:     time.Now(),
		Updated:     time.Now(),
		Rules:       make([]*ast.Rule, numRules),
	}

	for i := 0; i < numRules; i++ {
		policy.Rules[i] = &ast.Rule{
			Name:        "bench-rule-" + string(rune('A'+i)),
			Description: "Benchmark rule",
			Enabled:     true,
			Conditions: &ast.ConditionNode{
				Type:     ast.ConditionTypeSimple,
				Field:    "request.tokens",
				Operator: ast.OperatorGreaterThan,
				Value: &ast.ValueNode{
					Type:  ast.ValueTypeNumber,
					Value: float64(500),
				},
			},
			Actions: []*ast.Action{
				{
					Type: ast.ActionTypeLog,
					Parameters: map[string]*ast.ValueNode{
						"message": {
							Type:  ast.ValueTypeString,
							Value: "Benchmark log",
						},
					},
				},
			},
		}
	}

	return policy
}
