package recorder

import (
	"context"
	"testing"
	"time"

	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/storage"
	"mercator-hq/jupiter/pkg/policy/engine"
	"mercator-hq/jupiter/pkg/processing"
	"mercator-hq/jupiter/pkg/processing/costs"
	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// TestRecorder_RecordRequest tests recording a request.
func TestRecorder_RecordRequest(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.AsyncBuffer = 10

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	// Create test data
	requestMeta := &proxy.RequestMetadata{
		Timestamp:  now,
		Method:     "POST",
		Path:       "/v1/chat/completions",
		UserAgent:  "test-agent/1.0",
		UserID:     "user-123",
		APIKey:     "sk-test123456789",
		RemoteAddr: "192.168.1.1",
	}

	enrichedReq := &processing.EnrichedRequest{
		RequestID: "req-123",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []types.Message{
				{Role: "system", Content: "You are a helpful assistant"},
				{Role: "user", Content: "What is the weather?"},
			},
		},
		RiskScore:       5,
		ComplexityScore: 6,
		TokenEstimate: &processing.TokenEstimate{
			TotalTokens: 100,
		},
		CostEstimate: &costs.CostEstimate{
			TotalCost: 0.01,
		},
	}

	policyDecision := &engine.PolicyDecision{
		Action: engine.ActionAllow,
		MatchedRules: []*engine.MatchedRule{
			{
				PolicyID:       "policy-1",
				RuleID:         "rule-1",
				RuleName:       "Test rule",
				EvaluationTime: 5 * time.Millisecond,
				ActionsExecuted: []*engine.ActionResult{
					{ActionType: "allow"},
				},
			},
		},
	}

	// Record request
	err := recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)
	if err != nil {
		t.Fatalf("RecordRequest() failed: %v", err)
	}

	// Verify record is pending (not yet stored)
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 0 {
		t.Errorf("Expected 0 stored records (pending response), got %d", count)
	}

	// Verify record is in pending map
	value, ok := recorder.pendingRecords.Load(enrichedReq.RequestID)
	if !ok {
		t.Fatal("Record not found in pending map")
	}

	record := value.(*evidence.EvidenceRecord)
	if record.RequestID != "req-123" {
		t.Errorf("Expected RequestID 'req-123', got '%s'", record.RequestID)
	}
	if record.Model != "gpt-4" {
		t.Errorf("Expected Model 'gpt-4', got '%s'", record.Model)
	}
	if record.PolicyDecision != string(engine.ActionAllow) {
		t.Errorf("Expected PolicyDecision 'allow', got '%s'", record.PolicyDecision)
	}
}

// TestRecorder_RecordResponse tests recording a response.
func TestRecorder_RecordResponse(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.AsyncBuffer = 10
	config.WriteTimeout = 1 * time.Second

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	// First, record a request
	requestMeta := &proxy.RequestMetadata{
		Timestamp:  now,
		Method:     "POST",
		Path:       "/v1/chat/completions",
		UserID:     "user-123",
		APIKey:     "sk-test123",
		RemoteAddr: "192.168.1.1",
	}

	enrichedReq := &processing.EnrichedRequest{
		RequestID: "req-123",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []types.Message{
				{Role: "user", Content: "Test question"},
			},
		},
	}

	policyDecision := &engine.PolicyDecision{
		Action: engine.ActionAllow,
	}

	_ = recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)

	// Now record the response
	responseMeta := &proxy.ResponseMetadata{
		Timestamp:       now.Add(100 * time.Millisecond),
		StatusCode:      200,
		ProviderName:    "openai",
		ProviderLatency: 100 * time.Millisecond,
		Error:           nil,
	}

	enrichedResp := &processing.EnrichedResponse{
		RequestID: "req-123",
		OriginalResponse: &providers.CompletionResponse{
			Model:        "gpt-4",
			Content:      "The weather is sunny today",
			FinishReason: "stop",
		},
		TokenUsage: &costs.TokenUsage{
			PromptTokens:     50,
			CompletionTokens: 20,
			TotalTokens:      70,
		},
		CostEstimate: &costs.CostEstimate{
			TotalCost: 0.007,
		},
	}

	err := recorder.RecordResponse(ctx, responseMeta, enrichedResp)
	if err != nil {
		t.Fatalf("RecordResponse() failed: %v", err)
	}

	// Wait for async write to complete
	time.Sleep(100 * time.Millisecond)

	// Verify record was stored
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 1 {
		t.Fatalf("Expected 1 stored record, got %d", count)
	}

	// Query and verify the record
	results, err := store.Query(ctx, &evidence.Query{})
	if err != nil {
		t.Fatalf("Query() failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	record := results[0]

	// Verify request data
	if record.RequestID != "req-123" {
		t.Errorf("Expected RequestID 'req-123', got '%s'", record.RequestID)
	}
	if record.Model != "gpt-4" {
		t.Errorf("Expected Model 'gpt-4', got '%s'", record.Model)
	}

	// Verify response data
	if record.Provider != "openai" {
		t.Errorf("Expected Provider 'openai', got '%s'", record.Provider)
	}
	if record.ResponseStatus != 200 {
		t.Errorf("Expected ResponseStatus 200, got %d", record.ResponseStatus)
	}
	if record.TotalTokens != 70 {
		t.Errorf("Expected TotalTokens 70, got %d", record.TotalTokens)
	}
	if record.ActualCost != 0.007 {
		t.Errorf("Expected ActualCost 0.007, got %f", record.ActualCost)
	}
}

// TestRecorder_HashingEnabled tests that request/response hashing works.
func TestRecorder_HashingEnabled(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.HashRequest = true
	config.HashResponse = true

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	// Record request
	requestMeta := &proxy.RequestMetadata{
		Timestamp: now,
		Method:    "POST",
		Path:      "/v1/chat/completions",
	}

	enrichedReq := &processing.EnrichedRequest{
		RequestID: "req-123",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []types.Message{
				{Role: "user", Content: "Test"},
			},
		},
	}

	policyDecision := &engine.PolicyDecision{Action: engine.ActionAllow}

	_ = recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)

	// Record response
	responseMeta := &proxy.ResponseMetadata{
		Timestamp:    now.Add(100 * time.Millisecond),
		StatusCode:   200,
		ProviderName: "openai",
	}

	enrichedResp := &processing.EnrichedResponse{
		RequestID: "req-123",
		OriginalResponse: &providers.CompletionResponse{
			Model:   "gpt-4",
			Content: "Response",
		},
	}

	_ = recorder.RecordResponse(ctx, responseMeta, enrichedResp)

	// Wait for async write
	time.Sleep(100 * time.Millisecond)

	// Verify hashes were computed
	results, _ := store.Query(ctx, &evidence.Query{})
	if len(results) != 1 {
		t.Fatalf("Expected 1 result, got %d", len(results))
	}

	record := results[0]

	if record.RequestHash == "" {
		t.Error("Expected RequestHash to be set")
	}
	if record.ResponseHash == "" {
		t.Error("Expected ResponseHash to be set")
	}

	// Verify hash format (should be hex string)
	if len(record.RequestHash) != 64 {
		t.Errorf("Expected RequestHash length 64 (SHA-256 hex), got %d", len(record.RequestHash))
	}
}

// TestRecorder_APIKeyRedaction tests API key redaction.
func TestRecorder_APIKeyRedaction(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.RedactAPIKeys = true

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	apiKey := "sk-proj-1234567890abcdefghij"

	// Record request
	requestMeta := &proxy.RequestMetadata{
		Timestamp: now,
		Method:    "POST",
		Path:      "/v1/chat/completions",
		APIKey:    apiKey,
	}

	enrichedReq := &processing.EnrichedRequest{
		RequestID: "req-123",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
		},
	}

	policyDecision := &engine.PolicyDecision{Action: engine.ActionAllow}

	_ = recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)

	// Record response
	responseMeta := &proxy.ResponseMetadata{
		Timestamp:  now.Add(100 * time.Millisecond),
		StatusCode: 200,
	}

	enrichedResp := &processing.EnrichedResponse{
		RequestID: "req-123",
		OriginalResponse: &providers.CompletionResponse{
			Model: "gpt-4",
		},
	}

	_ = recorder.RecordResponse(ctx, responseMeta, enrichedResp)

	// Wait for async write
	time.Sleep(100 * time.Millisecond)

	// Verify API key was redacted
	results, _ := store.Query(ctx, &evidence.Query{})
	record := results[0]

	if record.APIKey == apiKey {
		t.Error("API key should be redacted, but found original key")
	}

	// Verify redaction format (should be SHA-256 hash with "sha256:" prefix)
	if len(record.APIKey) != 71 { // "sha256:" (7) + 64 hex chars
		t.Logf("API key redacted to: %s (length: %d)", record.APIKey, len(record.APIKey))
	}

	// Verify it starts with "sha256:"
	if record.APIKey[:7] != "sha256:" {
		t.Errorf("Expected redacted key to start with 'sha256:', got '%s'", record.APIKey[:7])
	}
}

// TestRecorder_GracefulShutdown tests that Close() drains pending records.
func TestRecorder_GracefulShutdown(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.AsyncBuffer = 100

	recorder := NewRecorder(store, config)

	ctx := context.Background()
	now := time.Now()

	// Record multiple requests and responses
	for i := 0; i < 10; i++ {
		requestMeta := &proxy.RequestMetadata{
			Timestamp: now,
			Method:    "POST",
			Path:      "/v1/chat/completions",
		}

		enrichedReq := &processing.EnrichedRequest{
			RequestID: "req-" + string(rune('0'+i)),
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-4",
			},
		}

		policyDecision := &engine.PolicyDecision{Action: engine.ActionAllow}

		_ = recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)

		responseMeta := &proxy.ResponseMetadata{
			Timestamp:  now.Add(100 * time.Millisecond),
			StatusCode: 200,
		}

		enrichedResp := &processing.EnrichedResponse{
			RequestID: enrichedReq.RequestID,
			OriginalResponse: &providers.CompletionResponse{
				Model: "gpt-4",
			},
		}

		_ = recorder.RecordResponse(ctx, responseMeta, enrichedResp)
	}

	// Close immediately (should drain channel)
	recorder.Close()

	// Verify all records were stored
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 10 {
		t.Errorf("Expected 10 stored records after graceful shutdown, got %d", count)
	}
}

// TestRecorder_DisabledRecording tests that recording can be disabled.
func TestRecorder_DisabledRecording(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.Enabled = false

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	// Try to record
	requestMeta := &proxy.RequestMetadata{
		Timestamp: now,
		Method:    "POST",
		Path:      "/v1/chat/completions",
	}

	enrichedReq := &processing.EnrichedRequest{
		RequestID: "req-123",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
		},
	}

	policyDecision := &engine.PolicyDecision{Action: engine.ActionAllow}

	err := recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)
	if err != nil {
		t.Fatalf("RecordRequest() should not fail when disabled: %v", err)
	}

	// Verify nothing was stored
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 0 {
		t.Errorf("Expected 0 stored records when recording disabled, got %d", count)
	}
}

// TestRecorder_ResponseWithoutRequest tests handling response without prior request.
func TestRecorder_ResponseWithoutRequest(t *testing.T) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	// Try to record a response without a request
	responseMeta := &proxy.ResponseMetadata{
		Timestamp:  now,
		StatusCode: 200,
	}

	enrichedResp := &processing.EnrichedResponse{
		RequestID: "nonexistent-req",
		OriginalResponse: &providers.CompletionResponse{
			Model: "gpt-4",
		},
	}

	err := recorder.RecordResponse(ctx, responseMeta, enrichedResp)

	// Should not return error, but should log warning
	if err != nil {
		t.Errorf("RecordResponse() should not fail for missing request: %v", err)
	}

	// Wait for any async operations
	time.Sleep(100 * time.Millisecond)

	// Verify nothing was stored
	count, _ := store.Count(ctx, &evidence.Query{})
	if count != 0 {
		t.Errorf("Expected 0 stored records, got %d", count)
	}
}

// BenchmarkRecorder_RecordRequest benchmarks recording requests.
func BenchmarkRecorder_RecordRequest(b *testing.B) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.AsyncBuffer = 10000

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	requestMeta := &proxy.RequestMetadata{
		Timestamp: now,
		Method:    "POST",
		Path:      "/v1/chat/completions",
	}

	enrichedReq := &processing.EnrichedRequest{
		RequestID: "req-bench",
		OriginalRequest: &types.ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []types.Message{
				{Role: "user", Content: "Test"},
			},
		},
	}

	policyDecision := &engine.PolicyDecision{Action: engine.ActionAllow}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)
	}
}

// BenchmarkRecorder_EndToEnd benchmarks full request/response cycle.
func BenchmarkRecorder_EndToEnd(b *testing.B) {
	store := storage.NewMemoryStorage()
	config := DefaultConfig()
	config.AsyncBuffer = 10000

	recorder := NewRecorder(store, config)
	defer recorder.Close()

	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		requestID := "req-" + string(rune(i))

		requestMeta := &proxy.RequestMetadata{
			Timestamp: now,
			Method:    "POST",
			Path:      "/v1/chat/completions",
		}

		enrichedReq := &processing.EnrichedRequest{
			RequestID: requestID,
			OriginalRequest: &types.ChatCompletionRequest{
				Model: "gpt-4",
			},
		}

		policyDecision := &engine.PolicyDecision{Action: engine.ActionAllow}

		_ = recorder.RecordRequest(ctx, requestMeta, enrichedReq, policyDecision)

		responseMeta := &proxy.ResponseMetadata{
			Timestamp:  now.Add(100 * time.Millisecond),
			StatusCode: 200,
		}

		enrichedResp := &processing.EnrichedResponse{
			RequestID: requestID,
			OriginalResponse: &providers.CompletionResponse{
				Model: "gpt-4",
			},
		}

		_ = recorder.RecordResponse(ctx, responseMeta, enrichedResp)
	}
}
