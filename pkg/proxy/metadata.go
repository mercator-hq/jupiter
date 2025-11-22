package proxy

import (
	"net/http"
	"time"

	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// RequestMetadata contains extracted metadata from an HTTP request.
// This is used for logging, tracing, and policy evaluation.
type RequestMetadata struct {
	// RequestID is a unique identifier for the request.
	RequestID string

	// Model is the requested model name.
	Model string

	// Messages is the conversation history.
	Messages []types.Message

	// Stream indicates whether streaming is requested.
	Stream bool

	// MaxTokens is the maximum number of tokens to generate.
	MaxTokens int

	// Temperature controls response randomness.
	Temperature float64

	// UserID is the identifier for the end-user making the request.
	UserID string

	// APIKey is the authentication key (redacted for logging).
	APIKey string

	// Method is the HTTP method (GET, POST, etc.).
	Method string

	// Path is the HTTP request path.
	Path string

	// UserAgent is the client's user agent string.
	UserAgent string

	// RemoteAddr is the client's IP address.
	RemoteAddr string

	// Timestamp is when the request was received.
	Timestamp time.Time
}

// ResponseMetadata contains extracted metadata from a response.
// This is used for logging, metrics, and evidence generation.
type ResponseMetadata struct {
	// RequestID is the unique identifier for the request.
	RequestID string

	// StatusCode is the HTTP response status code.
	StatusCode int

	// Latency is the total request processing time.
	Latency time.Duration

	// ProviderName is the name of the provider that handled the request.
	ProviderName string

	// ProviderLatency is the time spent waiting for the provider.
	ProviderLatency time.Duration

	// TokensPrompt is the number of prompt tokens.
	TokensPrompt int

	// TokensCompletion is the number of completion tokens.
	TokensCompletion int

	// TokensTotal is the total number of tokens.
	TokensTotal int

	// FinishReason explains why the model stopped generating.
	FinishReason string

	// Error contains any error that occurred.
	Error error

	// Timestamp is when the response was completed.
	Timestamp time.Time
}

// ExtractRequestMetadata extracts metadata from an HTTP request and chat completion request.
// This provides a unified view of request information for logging and tracing.
func ExtractRequestMetadata(r *http.Request, req *types.ChatCompletionRequest) *RequestMetadata {
	metadata := &RequestMetadata{
		RequestID:  ExtractRequestID(r),
		Model:      req.Model,
		Messages:   req.Messages,
		Stream:     req.Stream,
		UserID:     ExtractUserID(r),
		APIKey:     RedactAPIKey(ExtractAPIKey(r)),
		Method:     r.Method,
		Path:       r.URL.Path,
		UserAgent:  r.UserAgent(),
		RemoteAddr: r.RemoteAddr,
		Timestamp:  time.Now(),
	}

	// Extract optional parameters with defaults
	if req.MaxTokens != nil {
		metadata.MaxTokens = *req.MaxTokens
	}

	if req.Temperature != nil {
		metadata.Temperature = *req.Temperature
	} else {
		metadata.Temperature = 1.0 // OpenAI default
	}

	return metadata
}

// ExtractResponseMetadata extracts metadata from a provider response.
// This provides a unified view of response information for logging and metrics.
func ExtractResponseMetadata(requestID string, resp *providers.CompletionResponse, latency time.Duration, providerName string) *ResponseMetadata {
	metadata := &ResponseMetadata{
		RequestID:        requestID,
		StatusCode:       200, // Success status
		Latency:          latency,
		ProviderName:     providerName,
		TokensPrompt:     resp.Usage.PromptTokens,
		TokensCompletion: resp.Usage.CompletionTokens,
		TokensTotal:      resp.Usage.TotalTokens,
		FinishReason:     resp.FinishReason,
		Timestamp:        time.Now(),
	}

	return metadata
}

// ExtractErrorMetadata creates response metadata for an error response.
func ExtractErrorMetadata(requestID string, statusCode int, err error, latency time.Duration) *ResponseMetadata {
	return &ResponseMetadata{
		RequestID:  requestID,
		StatusCode: statusCode,
		Latency:    latency,
		Error:      err,
		Timestamp:  time.Now(),
	}
}

// RedactAPIKey redacts an API key for safe logging.
// It shows only the first 4 and last 4 characters.
//
// Example:
//
//	sk-1234567890abcdef -> sk-1234...cdef
func RedactAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	// Show first 7 characters (e.g., "sk-1234") and last 4
	if len(apiKey) < 12 {
		return "***"
	}

	return apiKey[:7] + "..." + apiKey[len(apiKey)-4:]
}

// MessageCount returns the number of messages in the request.
func (m *RequestMetadata) MessageCount() int {
	return len(m.Messages)
}

// EstimatedPromptTokens estimates the number of prompt tokens.
// This is a rough estimate based on character count (4 chars ≈ 1 token).
func (m *RequestMetadata) EstimatedPromptTokens() int {
	totalChars := 0
	for _, msg := range m.Messages {
		if content, ok := msg.Content.(string); ok {
			totalChars += len(content)
		}
	}
	return totalChars / 4
}

// IsStreaming returns true if the request is a streaming request.
func (m *RequestMetadata) IsStreaming() bool {
	return m.Stream
}

// TokenEfficiency returns the ratio of completion tokens to total tokens.
// Higher values indicate more efficient responses.
func (m *ResponseMetadata) TokenEfficiency() float64 {
	if m.TokensTotal == 0 {
		return 0.0
	}
	return float64(m.TokensCompletion) / float64(m.TokensTotal)
}

// IsSuccess returns true if the response was successful (2xx status code).
func (m *ResponseMetadata) IsSuccess() bool {
	return m.StatusCode >= 200 && m.StatusCode < 300
}

// IsError returns true if an error occurred.
func (m *ResponseMetadata) IsError() bool {
	return m.Error != nil || m.StatusCode >= 400
}
