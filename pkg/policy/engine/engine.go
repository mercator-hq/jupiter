package engine

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"mercator-hq/jupiter/pkg/mpl/ast"
	"mercator-hq/jupiter/pkg/processing"
)

// Engine is the main interface for policy evaluation.
type Engine interface {
	// EvaluateRequest evaluates policies against a request (pre-request pipeline).
	EvaluateRequest(ctx context.Context, enriched *processing.EnrichedRequest) (*PolicyDecision, error)

	// EvaluateResponse evaluates policies against a response (post-response pipeline).
	EvaluateResponse(ctx context.Context, enriched *processing.EnrichedResponse) (*PolicyDecision, error)

	// ReloadPolicies reloads policies from the source.
	ReloadPolicies(ctx context.Context) error

	// GetPolicies returns all loaded policies (for introspection).
	GetPolicies() []*ast.Policy

	// Close shuts down the engine and releases resources.
	Close() error
}

// ConditionMatcher evaluates policy conditions against evaluation context.
type ConditionMatcher interface {
	// Match evaluates a condition and returns whether it matched.
	Match(ctx context.Context, condition *ast.ConditionNode, evalCtx *EvaluationContext) (bool, error)
}

// ActionExecutor executes policy actions.
type ActionExecutor interface {
	// Execute executes an action and returns the result.
	Execute(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error)
}

// PolicySource provides policies to the engine.
type PolicySource interface {
	// LoadPolicies loads all policies from the source.
	LoadPolicies(ctx context.Context) ([]*ast.Policy, error)

	// Watch watches for policy changes and sends events on the returned channel.
	// The channel is closed when the context is cancelled.
	Watch(ctx context.Context) (<-chan PolicyEvent, error)
}

// PolicyEvent represents a policy file change event.
type PolicyEvent struct {
	// Type is the event type ("created", "modified", "deleted").
	Type PolicyEventType

	// Path is the file path that changed.
	Path string

	// Error is any error that occurred while processing the event.
	Error error
}

// PolicyEventType represents the type of policy file event.
type PolicyEventType string

const (
	PolicyEventCreated  PolicyEventType = "created"
	PolicyEventModified PolicyEventType = "modified"
	PolicyEventDeleted  PolicyEventType = "deleted"
)

// InterpreterEngine is the main implementation of the policy evaluation engine.
// It evaluates policies at runtime without compilation (interpreted mode).
type InterpreterEngine struct {
	// policies contains all loaded policies
	policies []*ast.Policy

	// policiesMu protects the policies slice for concurrent access
	policiesMu sync.RWMutex

	// matcher evaluates conditions
	matcher ConditionMatcher

	// executor executes actions
	executor ActionExecutor

	// config contains engine configuration
	config *EngineConfig

	// logger for structured logging
	logger *slog.Logger

	// source provides policies
	source PolicySource

	// stopCh signals shutdown
	stopCh chan struct{}

	// wg tracks background goroutines
	wg sync.WaitGroup
}

// NewInterpreterEngine creates a new policy evaluation engine.
func NewInterpreterEngine(config *EngineConfig, source PolicySource, logger *slog.Logger) (*InterpreterEngine, error) {
	if config == nil {
		config = DefaultEngineConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if source == nil {
		return nil, fmt.Errorf("policy source cannot be nil")
	}

	if logger == nil {
		logger = slog.Default()
	}

	engine := &InterpreterEngine{
		config: config,
		logger: logger,
		source: source,
		stopCh: make(chan struct{}),
	}

	// Initialize condition matcher and action executor
	// These will be created in the conditions and actions packages
	engine.matcher = NewDefaultMatcher(logger, config)
	engine.executor = NewDefaultExecutor(logger)

	// Load initial policies
	ctx := context.Background()
	if err := engine.ReloadPolicies(ctx); err != nil {
		return nil, fmt.Errorf("failed to load initial policies: %w", err)
	}

	// Start watching for policy changes
	engine.startWatching()

	return engine, nil
}

// EvaluateRequest evaluates policies against a request (pre-request pipeline).
func (e *InterpreterEngine) EvaluateRequest(ctx context.Context, enriched *processing.EnrichedRequest) (*PolicyDecision, error) {
	if enriched == nil {
		return nil, fmt.Errorf("enriched request cannot be nil")
	}

	// Create evaluation context
	evalCtx := &EvaluationContext{
		RequestID: enriched.RequestID,
		Request:   enriched,
		Tags:      make(map[string]string),
		StartTime: time.Now(),
	}

	// Enable trace if configured
	if e.config.EnableTrace {
		evalCtx.Trace = &EvaluationTrace{}
	}

	// Evaluate policies
	decision, err := e.evaluatePolicies(ctx, evalCtx)
	if err != nil {
		// Apply fail-safe mode
		return e.handleEvaluationError(err, evalCtx)
	}

	return decision, nil
}

// EvaluateResponse evaluates policies against a response (post-response pipeline).
func (e *InterpreterEngine) EvaluateResponse(ctx context.Context, enriched *processing.EnrichedResponse) (*PolicyDecision, error) {
	if enriched == nil {
		return nil, fmt.Errorf("enriched response cannot be nil")
	}

	// Create evaluation context
	evalCtx := &EvaluationContext{
		RequestID: enriched.RequestID,
		Response:  enriched,
		Tags:      make(map[string]string),
		StartTime: time.Now(),
	}

	// Enable trace if configured
	if e.config.EnableTrace {
		evalCtx.Trace = &EvaluationTrace{}
	}

	// Evaluate policies
	decision, err := e.evaluatePolicies(ctx, evalCtx)
	if err != nil {
		// Apply fail-safe mode
		return e.handleEvaluationError(err, evalCtx)
	}

	return decision, nil
}

// evaluatePolicies evaluates all loaded policies against the evaluation context.
func (e *InterpreterEngine) evaluatePolicies(ctx context.Context, evalCtx *EvaluationContext) (*PolicyDecision, error) {
	// Get policies (read lock)
	e.policiesMu.RLock()
	policies := e.policies
	e.policiesMu.RUnlock()

	if len(policies) == 0 {
		e.logger.Warn("no policies loaded")
		return e.buildDecision(evalCtx), nil
	}

	// Set policy timeout
	policyCtx, cancel := context.WithTimeout(ctx, e.config.PolicyTimeout)
	defer cancel()

	// Evaluate each policy
	for _, policy := range policies {
		// Check if context is cancelled
		select {
		case <-policyCtx.Done():
			return nil, &TimeoutError{
				PolicyID: policy.Name,
				Timeout:  e.config.PolicyTimeout,
			}
		default:
		}

		// Trace policy start
		policyStart := time.Now()
		evalCtx.AddTraceStep("policy_start", policy.Name, "", fmt.Sprintf("evaluating policy %q", policy.Name), 0)

		// Evaluate enabled rules in this policy
		for _, rule := range policy.EnabledRules() {
			if err := e.evaluateRule(policyCtx, policy, rule, evalCtx); err != nil {
				return nil, err
			}

			// Stop if evaluation is short-circuited
			if evalCtx.Stopped {
				evalCtx.AddTraceStep("policy_stop", policy.Name, rule.Name, "evaluation short-circuited", time.Since(policyStart))
				break
			}
		}

		evalCtx.AddTraceStep("policy_end", policy.Name, "", fmt.Sprintf("completed policy %q", policy.Name), time.Since(policyStart))

		// Stop if evaluation is short-circuited
		if evalCtx.Stopped {
			break
		}
	}

	return e.buildDecision(evalCtx), nil
}

// evaluateRule evaluates a single rule.
func (e *InterpreterEngine) evaluateRule(ctx context.Context, policy *ast.Policy, rule *ast.Rule, evalCtx *EvaluationContext) error {
	ruleStart := time.Now()

	// Set rule timeout
	ruleCtx, cancel := context.WithTimeout(ctx, e.config.RuleTimeout)
	defer cancel()

	matched := &MatchedRule{
		PolicyID:   policy.Name,
		PolicyName: policy.Name,
		RuleID:     rule.Name,
		RuleName:   rule.Name,
	}

	// Trace rule start
	evalCtx.AddTraceStep("rule_start", policy.Name, rule.Name, fmt.Sprintf("evaluating rule %q", rule.Name), 0)

	// Evaluate conditions
	if rule.HasConditions() {
		condMatched, err := e.matcher.Match(ruleCtx, rule.Conditions, evalCtx)
		if err != nil {
			// Check for timeout
			select {
			case <-ruleCtx.Done():
				return &TimeoutError{
					PolicyID: policy.Name,
					RuleID:   rule.Name,
					Timeout:  e.config.RuleTimeout,
				}
			default:
			}

			matched.Error = err
			matched.ConditionResult = false
			evalCtx.MatchedRules = append(evalCtx.MatchedRules, matched)
			return &ConditionError{
				PolicyID: policy.Name,
				RuleID:   rule.Name,
				Cause:    err,
			}
		}

		matched.ConditionResult = condMatched
		evalCtx.AddTraceStep("condition_eval", policy.Name, rule.Name, fmt.Sprintf("condition matched: %v", condMatched), time.Since(ruleStart))

		// If condition didn't match, skip actions
		if !condMatched {
			matched.EvaluationTime = time.Since(ruleStart)
			evalCtx.MatchedRules = append(evalCtx.MatchedRules, matched)
			return nil
		}
	} else {
		// No conditions means always match
		matched.ConditionResult = true
	}

	// Execute actions
	if rule.HasActions() {
		for _, action := range rule.Actions {
			actionStart := time.Now()

			result, err := e.executor.Execute(ruleCtx, action, evalCtx)
			if err != nil {
				// Check for timeout
				select {
				case <-ruleCtx.Done():
					return &TimeoutError{
						PolicyID: policy.Name,
						RuleID:   rule.Name,
						Timeout:  e.config.RuleTimeout,
					}
				default:
				}

				result = &ActionResult{
					ActionType: action.Type,
					Success:    false,
					Error:      err,
				}
			}

			matched.ActionsExecuted = append(matched.ActionsExecuted, result)
			evalCtx.AddTraceStep("action_exec", policy.Name, rule.Name, fmt.Sprintf("executed action %s: success=%v", action.Type, result.Success), time.Since(actionStart))

			// Stop if action failed and is blocking
			if !result.Success && isBlockingAction(action.Type) {
				matched.EvaluationTime = time.Since(ruleStart)
				evalCtx.MatchedRules = append(evalCtx.MatchedRules, matched)
				return &ActionError{
					PolicyID:   policy.Name,
					RuleID:     rule.Name,
					ActionType: string(action.Type),
					Cause:      err,
				}
			}
		}
	}

	matched.EvaluationTime = time.Since(ruleStart)
	evalCtx.MatchedRules = append(evalCtx.MatchedRules, matched)

	return nil
}

// buildDecision constructs the final policy decision from the evaluation context.
func (e *InterpreterEngine) buildDecision(evalCtx *EvaluationContext) *PolicyDecision {
	decision := &PolicyDecision{
		MatchedRules:    evalCtx.MatchedRules,
		Tags:            evalCtx.Tags,
		Transformations: evalCtx.Transformations,
		Redactions:      evalCtx.Redactions,
		Notifications:   evalCtx.Notifications,
		EvaluationTime:  time.Since(evalCtx.StartTime),
		Trace:           evalCtx.Trace,
	}

	// Determine final action
	if evalCtx.BlockReason != "" {
		decision.Action = ActionBlock
		decision.BlockReason = evalCtx.BlockReason
		decision.BlockStatusCode = evalCtx.BlockStatusCode
		if decision.BlockStatusCode == 0 {
			decision.BlockStatusCode = 403 // Default to Forbidden
		}
	} else if evalCtx.RoutingTarget != nil {
		decision.Action = ActionRoute
		decision.RoutingTarget = evalCtx.RoutingTarget
	} else if len(evalCtx.Transformations) > 0 {
		decision.Action = ActionTransform
	} else {
		decision.Action = ActionAllow
	}

	// Update trace total time
	if decision.Trace != nil {
		decision.Trace.TotalTime = decision.EvaluationTime
	}

	return decision
}

// handleEvaluationError handles evaluation errors according to the fail-safe mode.
func (e *InterpreterEngine) handleEvaluationError(err error, evalCtx *EvaluationContext) (*PolicyDecision, error) {
	e.logger.Error("policy evaluation error",
		"error", err,
		"request_id", evalCtx.RequestID,
		"fail_safe_mode", e.config.FailSafeMode,
	)

	switch e.config.FailSafeMode {
	case FailOpen:
		// Allow request
		e.logger.Info("fail-open: allowing request after evaluation error",
			"request_id", evalCtx.RequestID,
		)
		decision := e.buildDecision(evalCtx)
		decision.Action = ActionAllow
		return decision, nil

	case FailClosed:
		// Block request
		e.logger.Info("fail-closed: blocking request after evaluation error",
			"request_id", evalCtx.RequestID,
		)
		decision := e.buildDecision(evalCtx)
		decision.Action = ActionBlock
		decision.BlockReason = "Policy evaluation error"
		decision.BlockStatusCode = 500
		return decision, nil

	case FailSafeDefault:
		// Apply default action
		e.logger.Info("fail-safe-default: applying default action after evaluation error",
			"request_id", evalCtx.RequestID,
			"default_action", e.config.DefaultAction,
		)
		decision := e.buildDecision(evalCtx)
		decision.Action = e.config.DefaultAction
		if e.config.DefaultAction == ActionBlock {
			decision.BlockReason = "Policy evaluation error"
			decision.BlockStatusCode = 500
		}
		return decision, nil

	default:
		// This should never happen if config is validated
		return nil, fmt.Errorf("unknown fail-safe mode: %q", e.config.FailSafeMode)
	}
}

// ReloadPolicies reloads policies from the source.
func (e *InterpreterEngine) ReloadPolicies(ctx context.Context) error {
	e.logger.Info("reloading policies")

	// Load policies from source
	policies, err := e.source.LoadPolicies(ctx)
	if err != nil {
		return &ReloadError{
			Path:  "source",
			Cause: err,
		}
	}

	// Validate policy count
	if len(policies) > e.config.MaxPolicies {
		return &ValidationError{
			PolicyID: "global",
			Errors: []string{
				fmt.Sprintf("too many policies: %d (max: %d)", len(policies), e.config.MaxPolicies),
			},
		}
	}

	// Validate each policy
	totalRules := 0
	for _, policy := range policies {
		if len(policy.Rules) > e.config.MaxRulesPerPolicy {
			return &ValidationError{
				PolicyID: policy.Name,
				Errors: []string{
					fmt.Sprintf("too many rules: %d (max: %d)", len(policy.Rules), e.config.MaxRulesPerPolicy),
				},
			}
		}
		totalRules += len(policy.Rules)
	}

	// Normalize priorities (sort policies and rules by priority)
	NormalizePolicyPriorities(policies)

	// Atomically replace policies (write lock)
	e.policiesMu.Lock()
	e.policies = policies
	e.policiesMu.Unlock()

	e.logger.Info("policies reloaded successfully",
		"policy_count", len(policies),
		"rule_count", totalRules,
	)

	return nil
}

// GetPolicies returns all loaded policies (for introspection).
func (e *InterpreterEngine) GetPolicies() []*ast.Policy {
	e.policiesMu.RLock()
	defer e.policiesMu.RUnlock()

	// Return a copy to prevent external modification
	policies := make([]*ast.Policy, len(e.policies))
	copy(policies, e.policies)
	return policies
}

// startWatching starts watching for policy changes.
func (e *InterpreterEngine) startWatching() {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()

		ctx := context.Background()
		eventCh, err := e.source.Watch(ctx)
		if err != nil {
			e.logger.Error("failed to start policy watcher", "error", err)
			return
		}

		for {
			select {
			case <-e.stopCh:
				return
			case event, ok := <-eventCh:
				if !ok {
					return
				}
				e.handlePolicyEvent(event)
			}
		}
	}()
}

// handlePolicyEvent handles a policy file change event.
func (e *InterpreterEngine) handlePolicyEvent(event PolicyEvent) {
	e.logger.Info("policy file changed",
		"type", event.Type,
		"path", event.Path,
	)

	// Reload policies
	ctx := context.Background()
	if err := e.ReloadPolicies(ctx); err != nil {
		e.logger.Error("failed to reload policies after file change",
			"error", err,
			"path", event.Path,
		)
	}
}

// Close shuts down the engine and releases resources.
func (e *InterpreterEngine) Close() error {
	close(e.stopCh)
	e.wg.Wait()
	return nil
}

// isBlockingAction returns true if the action type is blocking (deny, rate_limit, budget).
func isBlockingAction(actionType ast.ActionType) bool {
	return actionType == ast.ActionTypeDeny ||
		actionType == ast.ActionTypeRateLimit ||
		actionType == ast.ActionTypeBudget
}
