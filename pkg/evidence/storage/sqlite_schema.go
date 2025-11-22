package storage

// SchemaVersion is the current database schema version.
const SchemaVersion = 1

// Schema contains the SQL statements to create the evidence database schema.
const Schema = `
-- Evidence records table
CREATE TABLE IF NOT EXISTS evidence (
    id TEXT PRIMARY KEY,
    request_id TEXT NOT NULL,

    -- Timestamps
    request_time TIMESTAMP NOT NULL,
    policy_eval_time TIMESTAMP NOT NULL,
    provider_call_time TIMESTAMP,
    response_time TIMESTAMP,
    recorded_time TIMESTAMP NOT NULL,

    -- Request metadata
    request_hash TEXT NOT NULL,
    request_method TEXT NOT NULL,
    request_path TEXT NOT NULL,
    request_headers TEXT,

    -- Request content
    model TEXT NOT NULL,
    provider TEXT NOT NULL,
    messages INTEGER,
    system_prompt TEXT,
    user_prompt TEXT,
    tools_used TEXT,

    -- Request metadata (from processing)
    estimated_tokens INTEGER,
    estimated_cost REAL,
    risk_score INTEGER,
    complexity_score INTEGER,
    pii_detected BOOLEAN,
    pii_types TEXT,

    -- Policy decisions
    policy_decision TEXT NOT NULL,
    matched_rules TEXT,
    block_reason TEXT,
    policy_version TEXT,

    -- Response metadata
    response_hash TEXT,
    response_status INTEGER,

    -- Response content
    response_content TEXT,
    finish_reason TEXT,

    -- Actual usage
    prompt_tokens INTEGER,
    completion_tokens INTEGER,
    total_tokens INTEGER,
    actual_cost REAL,

    -- Provider info
    provider_latency INTEGER,
    provider_model TEXT,

    -- User/API key
    user_id TEXT,
    api_key TEXT,
    ip_address TEXT,

    -- Error info
    error TEXT,
    error_type TEXT,

    -- Conversation context
    turn_number INTEGER,
    context_usage REAL
);

-- Schema version table
CREATE TABLE IF NOT EXISTS schema_version (
    version INTEGER PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_evidence_request_time ON evidence(request_time);
CREATE INDEX IF NOT EXISTS idx_evidence_user_id ON evidence(user_id);
CREATE INDEX IF NOT EXISTS idx_evidence_provider ON evidence(provider);
CREATE INDEX IF NOT EXISTS idx_evidence_model ON evidence(model);
CREATE INDEX IF NOT EXISTS idx_evidence_policy_decision ON evidence(policy_decision);
CREATE INDEX IF NOT EXISTS idx_evidence_actual_cost ON evidence(actual_cost);
CREATE INDEX IF NOT EXISTS idx_evidence_total_tokens ON evidence(total_tokens);
CREATE INDEX IF NOT EXISTS idx_evidence_request_id ON evidence(request_id);
`

// InsertSchemaVersion inserts the schema version into the schema_version table.
const InsertSchemaVersion = `
INSERT INTO schema_version (version, applied_at)
VALUES (?, datetime('now'))
ON CONFLICT(version) DO NOTHING;
`

// GetSchemaVersion retrieves the current schema version from the database.
const GetSchemaVersion = `
SELECT version FROM schema_version ORDER BY version DESC LIMIT 1;
`
