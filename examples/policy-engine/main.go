package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"time"

	"mercator-hq/jupiter/pkg/policy/engine"
	"mercator-hq/jupiter/pkg/policy/engine/source"
	"mercator-hq/jupiter/pkg/processing"
	"mercator-hq/jupiter/pkg/proxy/types"
)

func main() {
	// Create logger
	logger := slog.Default()

	// Configure engine
	config := engine.DefaultEngineConfig().
		WithFailSafeMode(engine.FailClosed).
		WithRuleTimeout(50 * time.Millisecond).
		WithPolicyTimeout(100 * time.Millisecond).
		WithTrace(true)

	// Create policy source (file-based)
	policySource := source.NewFileSource("policies/", logger)

	// Create engine
	eng, err := engine.NewInterpreterEngine(config, policySource, logger)
	if err != nil {
		log.Fatal("Failed to create engine:", err)
	}
	defer eng.Close()

	// Create a sample enriched request
	enrichedReq := &processing.EnrichedRequest{
		RequestID: "req-123",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
		},
		TokenEstimate: &processing.TokenEstimate{
			TotalTokens:  1000,
			PromptTokens: 800,
		},
		RiskScore:       3,
		ComplexityScore: 5,
	}

	// Evaluate policies
	ctx := context.Background()
	decision, err := eng.EvaluateRequest(ctx, enrichedReq)
	if err != nil {
		log.Fatal("Policy evaluation failed:", err)
	}

	// Check decision
	fmt.Printf("Policy Decision:\n")
	fmt.Printf("  Action: %s\n", decision.Action)
	fmt.Printf("  Matched Rules: %d\n", len(decision.MatchedRules))
	fmt.Printf("  Evaluation Time: %v\n", decision.EvaluationTime)

	if decision.Action == engine.ActionBlock {
		fmt.Printf("  Block Reason: %s\n", decision.BlockReason)
		fmt.Printf("  Status Code: %d\n", decision.BlockStatusCode)
	}

	if decision.RoutingTarget != nil {
		fmt.Printf("  Routing: provider=%s, model=%s\n",
			decision.RoutingTarget.Provider,
			decision.RoutingTarget.Model)
	}

	if len(decision.Transformations) > 0 {
		fmt.Printf("  Transformations:\n")
		for _, t := range decision.Transformations {
			fmt.Printf("    - %s: %v (%s)\n", t.Field, t.Value, t.Operation)
		}
	}

	if len(decision.Tags) > 0 {
		fmt.Printf("  Tags:\n")
		for k, v := range decision.Tags {
			fmt.Printf("    - %s: %s\n", k, v)
		}
	}

	// Print trace if enabled
	if decision.Trace != nil {
		fmt.Printf("\nEvaluation Trace:\n")
		for i, step := range decision.Trace.Steps {
			fmt.Printf("  %d. [%s] %s (policy=%s, rule=%s, duration=%v)\n",
				i+1, step.StepType, step.Details, step.PolicyID, step.RuleID, step.Duration)
		}
	}
}
