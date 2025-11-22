package validator

import (
	"strings"

	"mercator-hq/jupiter/pkg/mpl/ast"
)

// FieldInfo describes a field in the MPL data model.
type FieldInfo struct {
	Name        string                // Field name (e.g., "request.model")
	Type        ast.ValueType         // Field type
	Description string                // Human-readable description
	Children    map[string]*FieldInfo // Child fields for objects
}

// DataModel defines all valid fields available in MPL conditions.
// It represents the request, response, processing, and context namespaces.
var DataModel = &FieldInfo{
	Name: "root",
	Type: ast.ValueTypeObject,
	Children: map[string]*FieldInfo{
		"request":    requestFields,
		"response":   responseFields,
		"processing": processingFields,
		"context":    contextFields,
	},
}

// requestFields defines the request.* namespace
var requestFields = &FieldInfo{
	Name:        "request",
	Type:        ast.ValueTypeObject,
	Description: "LLM request fields",
	Children: map[string]*FieldInfo{
		"model": {
			Name:        "request.model",
			Type:        ast.ValueTypeString,
			Description: "Model name (e.g., 'gpt-4', 'claude-3-sonnet')",
		},
		"temperature": {
			Name:        "request.temperature",
			Type:        ast.ValueTypeNumber,
			Description: "Sampling temperature (0.0-2.0)",
		},
		"max_tokens": {
			Name:        "request.max_tokens",
			Type:        ast.ValueTypeNumber,
			Description: "Maximum tokens to generate",
		},
		"stream": {
			Name:        "request.stream",
			Type:        ast.ValueTypeBoolean,
			Description: "Whether streaming is enabled",
		},
		"user": {
			Name:        "request.user",
			Type:        ast.ValueTypeString,
			Description: "User identifier",
		},
		"messages": {
			Name:        "request.messages",
			Type:        ast.ValueTypeArray,
			Description: "Array of message objects",
		},
		"tools": {
			Name:        "request.tools",
			Type:        ast.ValueTypeArray,
			Description: "Array of tool definitions",
		},
		"top_p": {
			Name:        "request.top_p",
			Type:        ast.ValueTypeNumber,
			Description: "Nucleus sampling parameter",
		},
		"n": {
			Name:        "request.n",
			Type:        ast.ValueTypeNumber,
			Description: "Number of completions to generate",
		},
	},
}

// responseFields defines the response.* namespace
var responseFields = &FieldInfo{
	Name:        "response",
	Type:        ast.ValueTypeObject,
	Description: "LLM response fields",
	Children: map[string]*FieldInfo{
		"content": {
			Name:        "response.content",
			Type:        ast.ValueTypeString,
			Description: "Generated response content",
		},
		"finish_reason": {
			Name:        "response.finish_reason",
			Type:        ast.ValueTypeString,
			Description: "Reason for completion (stop, length, tool_calls)",
		},
		"usage": {
			Name:        "response.usage",
			Type:        ast.ValueTypeObject,
			Description: "Token usage information",
			Children: map[string]*FieldInfo{
				"prompt_tokens": {
					Name:        "response.usage.prompt_tokens",
					Type:        ast.ValueTypeNumber,
					Description: "Tokens in prompt",
				},
				"completion_tokens": {
					Name:        "response.usage.completion_tokens",
					Type:        ast.ValueTypeNumber,
					Description: "Tokens in completion",
				},
				"total_tokens": {
					Name:        "response.usage.total_tokens",
					Type:        ast.ValueTypeNumber,
					Description: "Total tokens used",
				},
			},
		},
	},
}

// processingFields defines the processing.* namespace
var processingFields = &FieldInfo{
	Name:        "processing",
	Type:        ast.ValueTypeObject,
	Description: "Processing metadata and analysis",
	Children: map[string]*FieldInfo{
		"risk_score": {
			Name:        "processing.risk_score",
			Type:        ast.ValueTypeNumber,
			Description: "Overall risk score (0-10)",
		},
		"complexity_score": {
			Name:        "processing.complexity_score",
			Type:        ast.ValueTypeNumber,
			Description: "Request complexity score (0-10)",
		},
		"token_estimate": {
			Name:        "processing.token_estimate",
			Type:        ast.ValueTypeObject,
			Description: "Estimated token usage",
			Children: map[string]*FieldInfo{
				"prompt_tokens": {
					Name:        "processing.token_estimate.prompt_tokens",
					Type:        ast.ValueTypeNumber,
					Description: "Estimated prompt tokens",
				},
				"completion_tokens": {
					Name:        "processing.token_estimate.completion_tokens",
					Type:        ast.ValueTypeNumber,
					Description: "Estimated completion tokens",
				},
				"total_tokens": {
					Name:        "processing.token_estimate.total_tokens",
					Type:        ast.ValueTypeNumber,
					Description: "Estimated total tokens",
				},
			},
		},
		"cost_estimate": {
			Name:        "processing.cost_estimate",
			Type:        ast.ValueTypeObject,
			Description: "Estimated cost",
			Children: map[string]*FieldInfo{
				"prompt_cost": {
					Name:        "processing.cost_estimate.prompt_cost",
					Type:        ast.ValueTypeNumber,
					Description: "Estimated prompt cost (USD)",
				},
				"completion_cost": {
					Name:        "processing.cost_estimate.completion_cost",
					Type:        ast.ValueTypeNumber,
					Description: "Estimated completion cost (USD)",
				},
				"total_cost": {
					Name:        "processing.cost_estimate.total_cost",
					Type:        ast.ValueTypeNumber,
					Description: "Estimated total cost (USD)",
				},
			},
		},
		"content_analysis": {
			Name:        "processing.content_analysis",
			Type:        ast.ValueTypeObject,
			Description: "Content analysis results",
			Children: map[string]*FieldInfo{
				"pii_detection": {
					Name:        "processing.content_analysis.pii_detection",
					Type:        ast.ValueTypeObject,
					Description: "PII detection results",
					Children: map[string]*FieldInfo{
						"detected": {
							Name:        "processing.content_analysis.pii_detection.detected",
							Type:        ast.ValueTypeBoolean,
							Description: "Whether PII was detected",
						},
						"types": {
							Name:        "processing.content_analysis.pii_detection.types",
							Type:        ast.ValueTypeArray,
							Description: "Types of PII detected",
						},
					},
				},
				"sentiment": {
					Name:        "processing.content_analysis.sentiment",
					Type:        ast.ValueTypeObject,
					Description: "Sentiment analysis",
					Children: map[string]*FieldInfo{
						"score": {
							Name:        "processing.content_analysis.sentiment.score",
							Type:        ast.ValueTypeNumber,
							Description: "Sentiment score (-1 to 1)",
						},
						"label": {
							Name:        "processing.content_analysis.sentiment.label",
							Type:        ast.ValueTypeString,
							Description: "Sentiment label (positive, negative, neutral)",
						},
					},
				},
				"sensitive_content": {
					Name:        "processing.content_analysis.sensitive_content",
					Type:        ast.ValueTypeObject,
					Description: "Sensitive content detection",
					Children: map[string]*FieldInfo{
						"detected": {
							Name:        "processing.content_analysis.sensitive_content.detected",
							Type:        ast.ValueTypeBoolean,
							Description: "Whether sensitive content was detected",
						},
						"severity": {
							Name:        "processing.content_analysis.sensitive_content.severity",
							Type:        ast.ValueTypeString,
							Description: "Severity level (low, medium, high, critical)",
						},
						"categories": {
							Name:        "processing.content_analysis.sensitive_content.categories",
							Type:        ast.ValueTypeArray,
							Description: "Categories of sensitive content detected",
						},
					},
				},
				"prompt_injection": {
					Name:        "processing.content_analysis.prompt_injection",
					Type:        ast.ValueTypeObject,
					Description: "Prompt injection detection",
					Children: map[string]*FieldInfo{
						"detected": {
							Name:        "processing.content_analysis.prompt_injection.detected",
							Type:        ast.ValueTypeBoolean,
							Description: "Whether prompt injection was detected",
						},
						"confidence": {
							Name:        "processing.content_analysis.prompt_injection.confidence",
							Type:        ast.ValueTypeNumber,
							Description: "Confidence score (0.0-1.0)",
						},
						"patterns": {
							Name:        "processing.content_analysis.prompt_injection.patterns",
							Type:        ast.ValueTypeArray,
							Description: "Injection patterns detected",
						},
					},
				},
			},
		},
		"conversation_context": {
			Name:        "processing.conversation_context",
			Type:        ast.ValueTypeObject,
			Description: "Conversation context",
			Children: map[string]*FieldInfo{
				"turn_count": {
					Name:        "processing.conversation_context.turn_count",
					Type:        ast.ValueTypeNumber,
					Description: "Number of conversation turns",
				},
				"total_tokens": {
					Name:        "processing.conversation_context.total_tokens",
					Type:        ast.ValueTypeNumber,
					Description: "Total tokens in conversation",
				},
				"context_window_percent": {
					Name:        "processing.conversation_context.context_window_percent",
					Type:        ast.ValueTypeNumber,
					Description: "Percentage of context window used (0-100)",
				},
			},
		},
	},
}

// contextFields defines the context.* namespace
var contextFields = &FieldInfo{
	Name:        "context",
	Type:        ast.ValueTypeObject,
	Description: "Request context and environment",
	Children: map[string]*FieldInfo{
		"environment": {
			Name:        "context.environment",
			Type:        ast.ValueTypeString,
			Description: "Environment name (dev, staging, prod)",
		},
		"time": {
			Name:        "context.time",
			Type:        ast.ValueTypeObject,
			Description: "Request timestamp information",
			Children: map[string]*FieldInfo{
				"hour": {
					Name:        "context.time.hour",
					Type:        ast.ValueTypeNumber,
					Description: "Hour of day (0-23)",
				},
				"day_of_week": {
					Name:        "context.time.day_of_week",
					Type:        ast.ValueTypeNumber,
					Description: "Day of week (0-6, Sunday=0)",
				},
				"timestamp": {
					Name:        "context.time.timestamp",
					Type:        ast.ValueTypeNumber,
					Description: "Unix timestamp",
				},
			},
		},
		"user_attributes": {
			Name:        "context.user_attributes",
			Type:        ast.ValueTypeObject,
			Description: "User attributes",
			Children: map[string]*FieldInfo{
				"tier": {
					Name:        "context.user_attributes.tier",
					Type:        ast.ValueTypeString,
					Description: "User tier (free, premium, enterprise)",
				},
				"department": {
					Name:        "context.user_attributes.department",
					Type:        ast.ValueTypeString,
					Description: "User department",
				},
				"region": {
					Name:        "context.user_attributes.region",
					Type:        ast.ValueTypeString,
					Description: "User region (e.g., 'us', 'eu', 'asia')",
				},
				"requests_this_hour": {
					Name:        "context.user_attributes.requests_this_hour",
					Type:        ast.ValueTypeNumber,
					Description: "Requests made this hour",
				},
				"daily_cost": {
					Name:        "context.user_attributes.daily_cost",
					Type:        ast.ValueTypeNumber,
					Description: "Cost incurred today (USD)",
				},
				"monthly_cost": {
					Name:        "context.user_attributes.monthly_cost",
					Type:        ast.ValueTypeNumber,
					Description: "Cost incurred this month (USD)",
				},
				"daily_token_usage": {
					Name:        "context.user_attributes.daily_token_usage",
					Type:        ast.ValueTypeNumber,
					Description: "Tokens used today",
				},
			},
		},
	},
}

// LookupField finds a field in the data model by its path.
// Returns the field info and true if found, nil and false otherwise.
func LookupField(path string) (*FieldInfo, bool) {
	parts := strings.Split(path, ".")
	current := DataModel

	for _, part := range parts {
		if current.Children == nil {
			return nil, false
		}
		next, ok := current.Children[part]
		if !ok {
			return nil, false
		}
		current = next
	}

	return current, true
}

// GetAllFieldPaths returns all valid field paths in the data model.
// This is used for error suggestions.
func GetAllFieldPaths() []string {
	var paths []string
	collectPaths(DataModel, "", &paths)
	return paths
}

// collectPaths recursively collects all field paths.
func collectPaths(field *FieldInfo, prefix string, paths *[]string) {
	if field.Name != "root" && field.Name != "" {
		*paths = append(*paths, field.Name)
	}

	for _, child := range field.Children {
		collectPaths(child, field.Name, paths)
	}
}
