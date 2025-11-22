package engine

import (
	"context"
	"fmt"
	"log/slog"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// DefaultExecutor is the default implementation of ActionExecutor.
type DefaultExecutor struct {
	logger *slog.Logger
}

// NewDefaultExecutor creates a new default action executor.
func NewDefaultExecutor(logger *slog.Logger) *DefaultExecutor {
	if logger == nil {
		logger = slog.Default()
	}
	return &DefaultExecutor{
		logger: logger,
	}
}

// Execute executes an action and returns the result.
func (e *DefaultExecutor) Execute(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	if action == nil {
		return nil, fmt.Errorf("action cannot be nil")
	}

	e.logger.Debug("executing action",
		"type", action.Type,
		"request_id", evalCtx.RequestID,
	)

	switch action.Type {
	case ast.ActionTypeAllow:
		return e.executeAllow(ctx, action, evalCtx)

	case ast.ActionTypeDeny:
		return e.executeDeny(ctx, action, evalCtx)

	case ast.ActionTypeLog:
		return e.executeLog(ctx, action, evalCtx)

	case ast.ActionTypeRedact:
		return e.executeRedact(ctx, action, evalCtx)

	case ast.ActionTypeModify:
		return e.executeModify(ctx, action, evalCtx)

	case ast.ActionTypeRoute:
		return e.executeRoute(ctx, action, evalCtx)

	case ast.ActionTypeAlert:
		return e.executeAlert(ctx, action, evalCtx)

	case ast.ActionTypeTag:
		return e.executeTag(ctx, action, evalCtx)

	case ast.ActionTypeRateLimit:
		return e.executeRateLimit(ctx, action, evalCtx)

	case ast.ActionTypeBudget:
		return e.executeBudget(ctx, action, evalCtx)

	default:
		return &ActionResult{
			ActionType: action.Type,
			Success:    false,
			Error:      fmt.Errorf("unknown action type: %q", action.Type),
		}, nil
	}
}

// executeAllow explicitly allows the request (short-circuit).
func (e *DefaultExecutor) executeAllow(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Stop further evaluation
	evalCtx.Stop()

	e.logger.Info("action allow: request explicitly allowed",
		"request_id", evalCtx.RequestID,
	)

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"action": "allow",
		},
	}, nil
}

// executeDeny blocks the request with a message.
func (e *DefaultExecutor) executeDeny(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Get deny parameters
	message := action.GetStringParameter("message")
	if message == "" {
		message = "Request denied by policy"
	}

	statusCode := int(action.GetNumberParameter("status_code"))
	if statusCode == 0 {
		statusCode = 403 // Default to Forbidden
	}

	// Set block in evaluation context
	evalCtx.SetBlock(message, statusCode)

	e.logger.Warn("action deny: blocking request",
		"request_id", evalCtx.RequestID,
		"message", message,
		"status_code", statusCode,
	)

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"message":     message,
			"status_code": statusCode,
		},
	}, nil
}

// executeLog logs an event.
func (e *DefaultExecutor) executeLog(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	message := action.GetStringParameter("message")
	level := action.GetStringParameter("level")

	if level == "" {
		level = "info"
	}

	// Log based on level
	switch level {
	case "debug":
		e.logger.Debug(message, "request_id", evalCtx.RequestID)
	case "info":
		e.logger.Info(message, "request_id", evalCtx.RequestID)
	case "warn":
		e.logger.Warn(message, "request_id", evalCtx.RequestID)
	case "error":
		e.logger.Error(message, "request_id", evalCtx.RequestID)
	default:
		e.logger.Info(message, "request_id", evalCtx.RequestID)
	}

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"message": message,
			"level":   level,
		},
	}, nil
}

// executeRedact redacts sensitive content from the request.
func (e *DefaultExecutor) executeRedact(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Get redact parameters
	strategy := action.GetStringParameter("strategy")
	if strategy == "" {
		strategy = "mask" // Default to masking
	}

	field := action.GetStringParameter("field")
	if field == "" {
		field = "prompt" // Default to prompt
	}

	pattern := action.GetStringParameter("pattern")
	replacement := action.GetStringParameter("replacement")

	if replacement == "" {
		replacement = "***"
	}

	// Add redaction to evaluation context
	// Actual content redaction will be applied by the proxy when forwarding the request
	evalCtx.AddRedaction(field, strategy, pattern, replacement, 0)

	e.logger.Info("action redact: content redaction configured",
		"request_id", evalCtx.RequestID,
		"field", field,
		"strategy", strategy,
		"pattern", pattern,
	)

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"field":       field,
			"strategy":    strategy,
			"pattern":     pattern,
			"replacement": replacement,
		},
	}, nil
}

// executeModify modifies request parameters.
func (e *DefaultExecutor) executeModify(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Get modify parameters
	field := action.GetStringParameter("field")
	value := action.GetParameter("value")
	operation := action.GetStringParameter("operation")

	if operation == "" {
		operation = "set" // Default to set
	}

	if field == "" {
		return &ActionResult{
			ActionType: action.Type,
			Success:    false,
			Error:      fmt.Errorf("field parameter is required for modify action"),
		}, nil
	}

	// Add transformation to evaluation context
	var transformValue interface{}
	if value != nil {
		transformValue = value.Value
	}

	evalCtx.AddTransformation(field, operation, transformValue)

	e.logger.Info("action modify: adding transformation",
		"request_id", evalCtx.RequestID,
		"field", field,
		"operation", operation,
		"value", transformValue,
	)

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"field":     field,
			"operation": operation,
			"value":     transformValue,
		},
	}, nil
}

// executeRoute routes the request to a specific provider/model.
func (e *DefaultExecutor) executeRoute(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Get route parameters
	provider := action.GetStringParameter("provider")
	model := action.GetStringParameter("model")

	if provider == "" {
		return &ActionResult{
			ActionType: action.Type,
			Success:    false,
			Error:      fmt.Errorf("provider parameter is required for route action"),
		}, nil
	}

	// Get fallback providers (if any)
	var fallback []string
	if fallbackParam := action.GetParameter("fallback"); fallbackParam != nil {
		if fallbackSlice, ok := fallbackParam.Value.([]interface{}); ok {
			for _, f := range fallbackSlice {
				if str, ok := f.(string); ok {
					fallback = append(fallback, str)
				}
			}
		}
	}

	// Set routing in evaluation context
	evalCtx.SetRouting(provider, model, fallback)

	e.logger.Info("action route: setting routing target",
		"request_id", evalCtx.RequestID,
		"provider", provider,
		"model", model,
		"fallback", fallback,
	)

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"provider": provider,
			"model":    model,
			"fallback": fallback,
		},
	}, nil
}

// executeAlert sends an external alert/webhook.
func (e *DefaultExecutor) executeAlert(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Get alert parameters
	destination := action.GetStringParameter("destination")
	message := action.GetStringParameter("message")
	notifType := action.GetStringParameter("type")

	if destination == "" {
		return &ActionResult{
			ActionType: action.Type,
			Success:    false,
			Error:      fmt.Errorf("destination parameter is required for alert action"),
		}, nil
	}

	if notifType == "" {
		notifType = "webhook"
	}

	// Build notification payload
	payload := map[string]interface{}{
		"request_id": evalCtx.RequestID,
		"message":    message,
		"timestamp":  evalCtx.StartTime,
	}

	// Add notification to evaluation context
	evalCtx.AddNotification(notifType, destination, payload, true)

	e.logger.Info("action alert: adding notification",
		"request_id", evalCtx.RequestID,
		"type", notifType,
		"destination", destination,
	)

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"type":        notifType,
			"destination": destination,
			"message":     message,
		},
	}, nil
}

// executeRateLimit enforces rate limiting.
func (e *DefaultExecutor) executeRateLimit(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Get rate limit parameters
	limit := int(action.GetNumberParameter("limit"))
	window := action.GetStringParameter("window")

	// TODO: Implement actual rate limiting logic
	// For now, this is a placeholder that would integrate with a rate limiter

	e.logger.Info("action rate_limit: checking rate limit",
		"request_id", evalCtx.RequestID,
		"limit", limit,
		"window", window,
	)

	// Placeholder: assume rate limit not exceeded
	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"limit":  limit,
			"window": window,
		},
	}, nil
}

// executeBudget enforces budget constraints.
func (e *DefaultExecutor) executeBudget(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	// Get budget parameters
	limit := action.GetNumberParameter("limit")

	// TODO: Implement actual budget checking logic
	// For now, this is a placeholder that would integrate with a budget tracker

	e.logger.Info("action budget: checking budget",
		"request_id", evalCtx.RequestID,
		"limit", limit,
	)

	// Placeholder: assume budget not exceeded
	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"limit": limit,
		},
	}, nil
}

// executeTag adds metadata tags to the evaluation context.
// Tags can be used for tracking, analytics, routing decisions, and auditing.
//
// Parameters:
//   - key: Tag key (required)
//   - value: Static tag value (optional if value_from provided)
//   - value_from: Extract value from request field
//
// Supported value_from paths:
//   - request.model - The model name (e.g., "gpt-4")
//   - request.user - The user identifier
//   - request.model_family - The model family (e.g., "GPT-4")
//   - request.pricing_tier - The pricing tier
//   - request.risk_score - The computed risk score
//   - request.complexity_score - The computed complexity score
//
// Examples:
//   - tag: {key: "model", value_from: "request.model"}
//   - tag: {key: "cost_tier", value: "expensive"}
//   - tag: {key: "environment", value: "production"}
func (e *DefaultExecutor) executeTag(ctx context.Context, action *ast.Action, evalCtx *EvaluationContext) (*ActionResult, error) {
	key := action.GetStringParameter("key")
	if key == "" {
		return &ActionResult{
			ActionType: action.Type,
			Success:    false,
			Error:      fmt.Errorf("key parameter is required for tag action"),
		}, nil
	}

	value := action.GetStringParameter("value")

	// Support value_from for dynamic values
	if value == "" {
		if valueFrom := action.GetStringParameter("value_from"); valueFrom != "" {
			// Extract value from request field (e.g., "request.metadata.department")
			extractedValue, err := extractFieldValue(evalCtx, valueFrom)
			if err != nil {
				e.logger.Warn("failed to extract tag value from field",
					"request_id", evalCtx.RequestID,
					"field", valueFrom,
					"error", err,
				)
				// Continue with empty value rather than failing
				value = ""
			} else {
				value = fmt.Sprintf("%v", extractedValue)
			}
		}
	}

	if value == "" {
		value = "true" // Default value if nothing specified
	}

	// Add tag to evaluation context
	evalCtx.AddTag(key, value)

	e.logger.Info("action tag: added metadata tag",
		"request_id", evalCtx.RequestID,
		"key", key,
		"value", value,
	)

	return &ActionResult{
		ActionType: action.Type,
		Success:    true,
		Details: map[string]interface{}{
			"key":   key,
			"value": value,
		},
	}, nil
}

// extractFieldValue extracts a value from the request using field path
// (e.g., "request.model" -> evalCtx.Request.OriginalRequest.Model).
func extractFieldValue(evalCtx *EvaluationContext, fieldPath string) (interface{}, error) {
	if evalCtx.Request == nil || evalCtx.Request.OriginalRequest == nil {
		return nil, fmt.Errorf("request data not available")
	}

	// Handle supported field paths
	switch fieldPath {
	case "request.model":
		return evalCtx.Request.OriginalRequest.Model, nil

	case "request.user":
		if evalCtx.Request.OriginalRequest.User != "" {
			return evalCtx.Request.OriginalRequest.User, nil
		}
		return nil, fmt.Errorf("user field is empty")

	case "request.model_family":
		if evalCtx.Request.ModelFamily != "" {
			return evalCtx.Request.ModelFamily, nil
		}
		return nil, fmt.Errorf("model_family not available")

	case "request.pricing_tier":
		if evalCtx.Request.PricingTier != "" {
			return evalCtx.Request.PricingTier, nil
		}
		return nil, fmt.Errorf("pricing_tier not available")

	case "request.risk_score":
		return evalCtx.Request.RiskScore, nil

	case "request.complexity_score":
		return evalCtx.Request.ComplexityScore, nil

	default:
		return nil, fmt.Errorf("unsupported field path: %s", fieldPath)
	}
}
