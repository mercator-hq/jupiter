// Package engine provides a runtime policy evaluation engine that interprets
// parsed MPL policies and evaluates them against enriched LLM requests and responses.
//
// This is the core governance mechanism that enforces policies by matching conditions
// and executing actions (block, redact, route, transform, notify, etc.). The interpreted
// mode evaluates policies at runtime without compilation, prioritizing implementation
// simplicity over maximum performance.
//
// # Architecture
//
// The engine uses a three-layer design:
//
//  1. Condition Matcher - Evaluates individual conditions against request/response metadata
//  2. Action Executor - Executes policy actions and returns results
//  3. Evaluation Engine - Orchestrates condition matching, action execution, and decision recording
//
// # Evaluation Flow
//
//	EnrichedRequest/Response
//	       ↓
//	Evaluation Engine (load policies)
//	       ↓
//	For each policy in priority order:
//	  For each rule in policy:
//	    Evaluate condition → Match?
//	      Yes → Execute actions → Record decision
//	      No → Skip to next rule
//	       ↓
//	Return PolicyDecision (matched rules, actions, metadata)
//
// # Basic Usage
//
//	// Initialize engine
//	cfg := &engine.EngineConfig{
//	    FailSafeMode:  engine.FailClosed,
//	    RuleTimeout:   50 * time.Millisecond,
//	    PolicyTimeout: 100 * time.Millisecond,
//	}
//
//	source := source.NewFileSource("policies/")
//	eng, err := engine.NewInterpreterEngine(cfg, source)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Evaluate request
//	decision, err := eng.EvaluateRequest(ctx, enrichedReq)
//	if err != nil {
//	    log.Error("evaluation failed", "error", err)
//	}
//
//	// Apply decision
//	if decision.Action == engine.ActionBlock {
//	    return fmt.Errorf("blocked: %s", decision.BlockReason)
//	}
//
// # Fail-Safe Modes
//
// The engine supports three fail-safe modes for error handling:
//
//   - fail-open: On policy error, allow request (log error)
//   - fail-closed: On policy error, block request (return 500)
//   - fail-safe-default: On policy error, apply default action from config
//
// # Performance Targets
//
//   - Single rule: <50ms p99 latency (interpreted mode)
//   - 10 rules: <100ms p99
//   - 100 rules (10 policies): <200ms p99
//
// WASM compilation (targeting <10ms p99) is deferred to Phase 2.
//
// # Thread Safety
//
// The engine is safe for concurrent use. Multiple goroutines can evaluate
// policies simultaneously. Policy hot-reload uses RWMutex for atomic updates.
package engine
