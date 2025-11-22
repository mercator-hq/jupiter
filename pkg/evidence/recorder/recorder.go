package recorder

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"

	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/policy/engine"
	"mercator-hq/jupiter/pkg/processing"
	"mercator-hq/jupiter/pkg/proxy"
	"mercator-hq/jupiter/pkg/proxy/types"
)

// Config contains configuration for the evidence recorder.
type Config struct {
	// Enabled enables evidence recording.
	Enabled bool

	// AsyncBuffer is the size of the async write channel buffer.
	// Default: 1000
	AsyncBuffer int

	// WriteTimeout is the timeout for writing evidence to storage.
	// Default: 5 seconds
	WriteTimeout time.Duration

	// HashRequest enables hashing of request bodies.
	// Default: true
	HashRequest bool

	// HashResponse enables hashing of response bodies.
	// Default: true
	HashResponse bool

	// RedactAPIKeys enables API key redaction.
	// Default: true
	RedactAPIKeys bool

	// MaxFieldLength is the maximum length for text fields before truncation.
	// Default: 500
	MaxFieldLength int
}

// DefaultConfig returns the default recorder configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:        true,
		AsyncBuffer:    1000,
		WriteTimeout:   5 * time.Second,
		HashRequest:    true,
		HashResponse:   true,
		RedactAPIKeys:  true,
		MaxFieldLength: 500,
	}
}

// Recorder records evidence for LLM proxy requests and responses.
// It creates evidence records asynchronously to avoid blocking proxy requests.
type Recorder struct {
	storage    evidence.Storage
	config     *Config
	recordChan chan *evidence.EvidenceRecord
	wg         sync.WaitGroup
	done       chan struct{}
	logger     *slog.Logger

	// pendingRecords tracks partial evidence records that are waiting for response data
	pendingRecords sync.Map // map[requestID]*evidence.EvidenceRecord
}

// NewRecorder creates a new evidence recorder with the provided storage backend and configuration.
func NewRecorder(storage evidence.Storage, config *Config) *Recorder {
	if config == nil {
		config = DefaultConfig()
	}

	r := &Recorder{
		storage:    storage,
		config:     config,
		recordChan: make(chan *evidence.EvidenceRecord, config.AsyncBuffer),
		done:       make(chan struct{}),
		logger:     slog.Default().With("component", "evidence.recorder"),
	}

	// Start background worker to drain channel
	r.wg.Add(1)
	go r.worker()

	r.logger.Info("evidence recorder initialized",
		"async_buffer", config.AsyncBuffer,
		"write_timeout", config.WriteTimeout,
		"hash_request", config.HashRequest,
		"hash_response", config.HashResponse,
	)

	return r
}

// RecordRequest creates an evidence record from an enriched request and policy decision.
// The evidence record is enqueued for async writing to storage.
//
// This method returns immediately and does not block on storage writes.
func (r *Recorder) RecordRequest(ctx context.Context, requestMeta *proxy.RequestMetadata, enrichedReq *processing.EnrichedRequest, policyDecision *engine.PolicyDecision) error {
	if !r.config.Enabled {
		return nil
	}

	// Create evidence record
	record := r.createEvidenceRecord(requestMeta, enrichedReq, policyDecision)

	// Store in pending map (will be updated when response arrives)
	r.pendingRecords.Store(enrichedReq.RequestID, record)

	r.logger.Debug("evidence record created (awaiting response)",
		"record_id", record.ID,
		"request_id", record.RequestID,
		"policy_decision", record.PolicyDecision,
	)

	return nil
}

// RecordResponse updates an evidence record with response data and enqueues it for async writing.
//
// This method returns immediately and does not block on storage writes.
func (r *Recorder) RecordResponse(ctx context.Context, responseMeta *proxy.ResponseMetadata, enrichedResp *processing.EnrichedResponse) error {
	if !r.config.Enabled {
		return nil
	}

	// Retrieve pending record
	value, ok := r.pendingRecords.LoadAndDelete(enrichedResp.RequestID)
	if !ok {
		r.logger.Warn("no pending evidence record found for response",
			"request_id", enrichedResp.RequestID,
		)
		return nil
	}

	record := value.(*evidence.EvidenceRecord)

	// Update record with response data
	r.updateEvidenceWithResponse(record, responseMeta, enrichedResp)

	// Enqueue for async writing
	select {
	case r.recordChan <- record:
		r.logger.Debug("evidence record enqueued for writing",
			"record_id", record.ID,
			"request_id", record.RequestID,
		)
	case <-time.After(r.config.WriteTimeout):
		r.logger.Error("evidence record channel full, dropping record",
			"record_id", record.ID,
			"request_id", record.RequestID,
			"channel_capacity", r.config.AsyncBuffer,
		)
		return evidence.NewRecorderError(record.ID, context.DeadlineExceeded)
	case <-r.done:
		r.logger.Warn("recorder shutting down, dropping record",
			"record_id", record.ID,
			"request_id", record.RequestID,
		)
		return evidence.NewRecorderError(record.ID, context.Canceled)
	}

	return nil
}

// Close gracefully shuts down the recorder by draining the async channel and
// waiting for all pending writes to complete.
func (r *Recorder) Close() error {
	r.logger.Info("shutting down evidence recorder")

	// Signal shutdown
	close(r.done)

	// Wait for worker to finish draining channel
	r.wg.Wait()

	r.logger.Info("evidence recorder shut down complete")
	return nil
}

// worker is the background goroutine that drains the evidence channel and
// writes records to storage.
func (r *Recorder) worker() {
	defer r.wg.Done()

	for {
		select {
		case record := <-r.recordChan:
			r.writeRecord(record)

		case <-r.done:
			// Drain remaining records from channel before exit
			r.logger.Info("draining evidence channel before shutdown",
				"pending_count", len(r.recordChan),
			)

			for {
				select {
				case record := <-r.recordChan:
					r.writeRecord(record)
				default:
					// Channel is empty, we can exit
					r.logger.Info("evidence channel drained")
					return
				}
			}
		}
	}
}

// writeRecord writes a single evidence record to storage.
func (r *Recorder) writeRecord(record *evidence.EvidenceRecord) {
	ctx, cancel := context.WithTimeout(context.Background(), r.config.WriteTimeout)
	defer cancel()

	start := time.Now()

	err := r.storage.Store(ctx, record)
	if err != nil {
		r.logger.Error("failed to store evidence record",
			"record_id", record.ID,
			"request_id", record.RequestID,
			"error", err,
		)
		return
	}

	duration := time.Since(start)

	r.logger.Info("evidence recorded",
		"record_id", record.ID,
		"request_id", record.RequestID,
		"policy_decision", record.PolicyDecision,
		"duration_ms", duration.Milliseconds(),
	)

	// Warn if write was slow
	if duration > r.config.WriteTimeout/2 {
		r.logger.Warn("slow evidence write",
			"record_id", record.ID,
			"duration_ms", duration.Milliseconds(),
			"threshold_ms", (r.config.WriteTimeout / 2).Milliseconds(),
		)
	}
}

// createEvidenceRecord creates an evidence record from enriched request and policy decision.
func (r *Recorder) createEvidenceRecord(requestMeta *proxy.RequestMetadata, enrichedReq *processing.EnrichedRequest, policyDecision *engine.PolicyDecision) *evidence.EvidenceRecord {
	now := time.Now()

	record := &evidence.EvidenceRecord{
		ID:        uuid.New().String(),
		RequestID: enrichedReq.RequestID,

		// Timestamps
		RequestTime:    requestMeta.Timestamp,
		PolicyEvalTime: now, // Approximate policy eval time
		RecordedTime:   now,

		// Request metadata
		RequestMethod:  requestMeta.Method,
		RequestPath:    requestMeta.Path,
		RequestHeaders: r.extractHeaders(requestMeta),

		// Request content
		Model:    enrichedReq.OriginalRequest.Model,
		Provider: "", // Will be set when response arrives
		Messages: len(enrichedReq.OriginalRequest.Messages),

		// Request metadata (from processing)
		RiskScore:       enrichedReq.RiskScore,
		ComplexityScore: enrichedReq.ComplexityScore,
	}

	// Hash request body if configured
	if r.config.HashRequest {
		requestBody, _ := json.Marshal(enrichedReq.OriginalRequest)
		record.RequestHash = HashContent(requestBody)
	}

	// Extract system and user prompts
	r.extractPrompts(record, enrichedReq.OriginalRequest)

	// Extract tools used
	record.ToolsUsed = r.extractTools(enrichedReq.OriginalRequest)

	// Extract token estimates
	if enrichedReq.TokenEstimate != nil {
		record.EstimatedTokens = enrichedReq.TokenEstimate.TotalTokens
	}

	// Extract cost estimate
	if enrichedReq.CostEstimate != nil {
		record.EstimatedCost = enrichedReq.CostEstimate.TotalCost
	}

	// Extract PII detection
	if enrichedReq.ContentAnalysis != nil && enrichedReq.ContentAnalysis.PIIDetection != nil {
		record.PIIDetected = enrichedReq.ContentAnalysis.PIIDetection.HasPII
		record.PIITypes = enrichedReq.ContentAnalysis.PIIDetection.PIITypes
	}

	// Extract policy decision
	r.extractPolicyDecision(record, policyDecision)

	// Extract user/API key
	record.UserID = requestMeta.UserID
	if r.config.RedactAPIKeys {
		record.APIKey = RedactAPIKey(requestMeta.APIKey)
	} else {
		record.APIKey = requestMeta.APIKey
	}
	record.IPAddress = requestMeta.RemoteAddr

	return record
}

// updateEvidenceWithResponse updates an evidence record with response data.
func (r *Recorder) updateEvidenceWithResponse(record *evidence.EvidenceRecord, responseMeta *proxy.ResponseMetadata, enrichedResp *processing.EnrichedResponse) {
	// Update timestamps
	record.ResponseTime = responseMeta.Timestamp
	record.RecordedTime = time.Now()

	// Hash response body if configured
	if r.config.HashResponse && enrichedResp.OriginalResponse != nil {
		responseBody, _ := json.Marshal(enrichedResp.OriginalResponse)
		record.ResponseHash = HashContent(responseBody)
	}

	// Update response metadata
	record.ResponseStatus = responseMeta.StatusCode

	// Extract provider info
	record.Provider = responseMeta.ProviderName
	record.ProviderModel = enrichedResp.OriginalResponse.Model
	record.ProviderLatency = responseMeta.ProviderLatency

	// Extract response content
	if enrichedResp.OriginalResponse != nil {
		record.ResponseContent = TruncateString(enrichedResp.OriginalResponse.Content, r.config.MaxFieldLength)
		record.FinishReason = enrichedResp.OriginalResponse.FinishReason
	}

	// Extract actual token usage
	if enrichedResp.TokenUsage != nil {
		record.PromptTokens = enrichedResp.TokenUsage.PromptTokens
		record.CompletionTokens = enrichedResp.TokenUsage.CompletionTokens
		record.TotalTokens = enrichedResp.TokenUsage.TotalTokens
	}

	// Extract actual cost
	if enrichedResp.CostEstimate != nil {
		record.ActualCost = enrichedResp.CostEstimate.TotalCost
	}

	// Extract error info
	if responseMeta.Error != nil {
		record.Error = responseMeta.Error.Error()
		record.ErrorType = r.classifyError(responseMeta.Error)
	}

	// Extract conversation context
	if enrichedResp.OriginalResponse != nil {
		record.TurnNumber = 1 // TODO: Extract from conversation context
		record.ContextUsage = 0.0 // TODO: Calculate from token usage
	}
}

// extractHeaders extracts selected headers from the request metadata.
func (r *Recorder) extractHeaders(requestMeta *proxy.RequestMetadata) map[string]string {
	// Only store selected headers (user-agent)
	selected := make(map[string]string)

	if requestMeta.UserAgent != "" {
		selected["user-agent"] = requestMeta.UserAgent
	}

	return selected
}

// extractPrompts extracts system and user prompts from the request.
func (r *Recorder) extractPrompts(record *evidence.EvidenceRecord, req *types.ChatCompletionRequest) {
	for _, msg := range req.Messages {
		// Extract content as string (handle both string and structured content)
		content := ""
		if str, ok := msg.Content.(string); ok {
			content = str
		}

		if msg.Role == "system" && record.SystemPrompt == "" {
			record.SystemPrompt = TruncateString(content, r.config.MaxFieldLength)
		}
		if msg.Role == "user" && record.UserPrompt == "" {
			record.UserPrompt = TruncateString(content, r.config.MaxFieldLength)
		}
	}
}

// extractTools extracts tool names from the request.
func (r *Recorder) extractTools(req *types.ChatCompletionRequest) []string {
	tools := []string{}
	for _, tool := range req.Tools {
		if tool.Function.Name != "" {
			tools = append(tools, tool.Function.Name)
		}
	}
	return tools
}

// extractPolicyDecision extracts policy decision data from the engine decision.
func (r *Recorder) extractPolicyDecision(record *evidence.EvidenceRecord, policyDecision *engine.PolicyDecision) {
	if policyDecision == nil {
		record.PolicyDecision = string(engine.ActionAllow)
		return
	}

	record.PolicyDecision = string(policyDecision.Action)
	record.BlockReason = policyDecision.BlockReason

	// Convert matched rules
	record.MatchedRules = make([]evidence.MatchedRuleRecord, 0, len(policyDecision.MatchedRules))
	for _, rule := range policyDecision.MatchedRules {
		record.MatchedRules = append(record.MatchedRules, evidence.MatchedRuleRecord{
			PolicyID:       rule.PolicyID,
			RuleID:         rule.RuleID,
			Action:         r.extractRuleAction(rule),
			Reason:         rule.RuleName,
			EvaluationTime: rule.EvaluationTime,
		})
	}

	// TODO: Extract policy version (Git commit hash) once available
	record.PolicyVersion = "unknown"
}

// extractRuleAction extracts the action from a matched rule.
func (r *Recorder) extractRuleAction(rule *engine.MatchedRule) string {
	if len(rule.ActionsExecuted) > 0 {
		return string(rule.ActionsExecuted[0].ActionType)
	}
	return "unknown"
}

// classifyError classifies an error by type.
func (r *Recorder) classifyError(err error) string {
	// TODO: Implement error classification based on error type
	// For now, return generic "error"
	return "error"
}
