package engine

import (
	"fmt"
	"reflect"
	"strings"
)

// extractField extracts a field value from the evaluation context.
// Field names use dot notation: request.model, request.content_analysis.has_pii, etc.
func extractField(fieldPath string, evalCtx *EvaluationContext) (interface{}, error) {
	// Split field path into parts
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid field path: %q (must be at least two parts)", fieldPath)
	}

	// First part determines the source (request, response, metadata)
	source := parts[0]
	fieldName := parts[1:]

	switch source {
	case "request":
		return extractRequestField(fieldName, evalCtx)

	case "response":
		return extractResponseField(fieldName, evalCtx)

	case "metadata":
		return extractMetadataField(fieldName, evalCtx)

	default:
		return nil, fmt.Errorf("unknown field source: %q", source)
	}
}

// extractRequestField extracts a field from the enriched request.
func extractRequestField(fieldPath []string, evalCtx *EvaluationContext) (interface{}, error) {
	if evalCtx.Request == nil {
		return nil, fmt.Errorf("request not available in evaluation context")
	}

	// Handle common request fields
	if len(fieldPath) == 1 {
		switch fieldPath[0] {
		case "request_id":
			return evalCtx.Request.RequestID, nil

		case "model":
			if evalCtx.Request.OriginalRequest != nil {
				return evalCtx.Request.OriginalRequest.Model, nil
			}
			return nil, fmt.Errorf("original request not available")

		case "model_family":
			return evalCtx.Request.ModelFamily, nil

		case "tokens":
			if evalCtx.Request.TokenEstimate != nil {
				return evalCtx.Request.TokenEstimate.TotalTokens, nil
			}
			return 0, nil

		case "prompt_tokens":
			if evalCtx.Request.TokenEstimate != nil {
				return evalCtx.Request.TokenEstimate.PromptTokens, nil
			}
			return 0, nil

		case "estimated_cost":
			if evalCtx.Request.CostEstimate != nil {
				return evalCtx.Request.CostEstimate.TotalCost, nil
			}
			return 0.0, nil

		case "risk_score":
			return evalCtx.Request.RiskScore, nil

		case "complexity_score":
			return evalCtx.Request.ComplexityScore, nil

		case "temperature":
			if evalCtx.Request.OriginalRequest != nil {
				if evalCtx.Request.OriginalRequest.Temperature != nil {
					return *evalCtx.Request.OriginalRequest.Temperature, nil
				}
			}
			return nil, fmt.Errorf("temperature not set")

		case "max_tokens":
			if evalCtx.Request.OriginalRequest != nil {
				if evalCtx.Request.OriginalRequest.MaxTokens != nil {
					return *evalCtx.Request.OriginalRequest.MaxTokens, nil
				}
			}
			return nil, fmt.Errorf("max_tokens not set")

		case "stream":
			if evalCtx.Request.OriginalRequest != nil {
				return evalCtx.Request.OriginalRequest.Stream, nil
			}
			return false, nil
		}
	}

	// Handle nested fields
	if len(fieldPath) >= 2 {
		switch fieldPath[0] {
		case "content_analysis":
			return extractContentAnalysisField(fieldPath[1:], evalCtx.Request.ContentAnalysis)

		case "conversation_context":
			return extractConversationField(fieldPath[1:], evalCtx.Request.ConversationContext)

		case "token_estimate":
			return extractTokenEstimateField(fieldPath[1:], evalCtx.Request.TokenEstimate)

		case "cost_estimate":
			return extractCostEstimateField(fieldPath[1:], evalCtx.Request.CostEstimate)
		}
	}

	// Fallback to reflection for other fields
	return extractFieldReflection(evalCtx.Request, fieldPath)
}

// extractResponseField extracts a field from the enriched response.
func extractResponseField(fieldPath []string, evalCtx *EvaluationContext) (interface{}, error) {
	if evalCtx.Response == nil {
		return nil, fmt.Errorf("response not available in evaluation context")
	}

	// Handle common response fields
	if len(fieldPath) == 1 {
		switch fieldPath[0] {
		case "request_id":
			return evalCtx.Response.RequestID, nil

		case "tokens":
			if evalCtx.Response.TokenUsage != nil {
				return evalCtx.Response.TokenUsage.TotalTokens, nil
			}
			return 0, nil

		case "prompt_tokens":
			if evalCtx.Response.TokenUsage != nil {
				return evalCtx.Response.TokenUsage.PromptTokens, nil
			}
			return 0, nil

		case "completion_tokens":
			if evalCtx.Response.TokenUsage != nil {
				return evalCtx.Response.TokenUsage.CompletionTokens, nil
			}
			return 0, nil

		case "finish_reason":
			if evalCtx.Response.OriginalResponse != nil {
				return evalCtx.Response.OriginalResponse.FinishReason, nil
			}
			return "", nil

		case "actual_cost":
			if evalCtx.Response.CostEstimate != nil {
				return evalCtx.Response.CostEstimate.TotalCost, nil
			}
			return 0.0, nil
		}
	}

	// Handle nested fields
	if len(fieldPath) >= 2 {
		switch fieldPath[0] {
		case "content_analysis":
			return extractContentAnalysisField(fieldPath[1:], evalCtx.Response.ContentAnalysis)

		case "token_usage":
			return extractTokenUsageField(fieldPath[1:], evalCtx.Response.TokenUsage)

		case "cost_estimate":
			return extractCostEstimateField(fieldPath[1:], evalCtx.Response.CostEstimate)
		}
	}

	// Fallback to reflection
	return extractFieldReflection(evalCtx.Response, fieldPath)
}

// extractMetadataField extracts a metadata field.
func extractMetadataField(fieldPath []string, evalCtx *EvaluationContext) (interface{}, error) {
	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("empty metadata field path")
	}

	switch fieldPath[0] {
	case "request_id":
		return evalCtx.RequestID, nil

	default:
		return nil, fmt.Errorf("unknown metadata field: %q", fieldPath[0])
	}
}

// extractContentAnalysisField extracts a field from content analysis.
func extractContentAnalysisField(fieldPath []string, analysis interface{}) (interface{}, error) {
	if analysis == nil {
		return nil, fmt.Errorf("content analysis not available")
	}

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("empty content analysis field path")
	}

	// Use reflection for content analysis fields
	return extractFieldReflection(analysis, fieldPath)
}

// extractConversationField extracts a field from conversation context.
func extractConversationField(fieldPath []string, conversation interface{}) (interface{}, error) {
	if conversation == nil {
		return nil, fmt.Errorf("conversation context not available")
	}

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("empty conversation field path")
	}

	// Use reflection for conversation fields
	return extractFieldReflection(conversation, fieldPath)
}

// extractTokenEstimateField extracts a field from token estimate.
func extractTokenEstimateField(fieldPath []string, estimate interface{}) (interface{}, error) {
	if estimate == nil {
		return nil, fmt.Errorf("token estimate not available")
	}

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("empty token estimate field path")
	}

	// Use reflection for token estimate fields
	return extractFieldReflection(estimate, fieldPath)
}

// extractCostEstimateField extracts a field from cost estimate.
func extractCostEstimateField(fieldPath []string, estimate interface{}) (interface{}, error) {
	if estimate == nil {
		return nil, fmt.Errorf("cost estimate not available")
	}

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("empty cost estimate field path")
	}

	// Use reflection for cost estimate fields
	return extractFieldReflection(estimate, fieldPath)
}

// extractTokenUsageField extracts a field from token usage.
func extractTokenUsageField(fieldPath []string, usage interface{}) (interface{}, error) {
	if usage == nil {
		return nil, fmt.Errorf("token usage not available")
	}

	if len(fieldPath) == 0 {
		return nil, fmt.Errorf("empty token usage field path")
	}

	// Use reflection for token usage fields
	return extractFieldReflection(usage, fieldPath)
}

// extractFieldReflection uses reflection to extract nested fields.
// This is a fallback for fields not explicitly handled above.
func extractFieldReflection(obj interface{}, fieldPath []string) (interface{}, error) {
	if obj == nil {
		return nil, fmt.Errorf("nil object")
	}

	v := reflect.ValueOf(obj)

	// Dereference pointers
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, fmt.Errorf("nil pointer in field path")
		}
		v = v.Elem()
	}

	// Traverse field path
	for _, fieldName := range fieldPath {
		if v.Kind() != reflect.Struct {
			return nil, fmt.Errorf("cannot access field %q on non-struct type %s", fieldName, v.Kind())
		}

		// Find field (case-insensitive)
		f := v.FieldByNameFunc(func(name string) bool {
			return strings.EqualFold(name, fieldName)
		})

		if !f.IsValid() {
			return nil, fmt.Errorf("field %q not found", fieldName)
		}

		v = f
	}

	// Return the value
	if !v.CanInterface() {
		return nil, fmt.Errorf("cannot access unexported field")
	}

	return v.Interface(), nil
}
