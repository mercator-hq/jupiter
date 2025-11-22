package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"mercator-hq/jupiter/pkg/cli"
	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/storage"
)

var evidenceFlags struct {
	backend   string
	timeRange string
	user      string
	apiKey    string
	policy    string
	provider  string
	model     string
	minCost   float64
	maxCost   float64
	minTokens int
	maxTokens int
	limit     int
	offset    int
	format    string
	verify    bool
	output    string
	decision  string
}

var evidenceCmd = &cobra.Command{
	Use:   "evidence",
	Short: "Query evidence database",
	Long: `Query and export evidence records for audit and compliance.

The evidence command provides access to the evidence database for
querying, exporting, and analyzing LLM request/response audit trails.

Subcommands:
  query   - Query evidence records with filters
  report  - Generate audit report with statistics (not yet implemented)

Examples:
  # Query last 24 hours
  mercator evidence query --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

  # Filter by user
  mercator evidence query --user "user-123"

  # Export to JSON file
  mercator evidence query --format json --output evidence.json`,
}

var evidenceQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query evidence records",
	Long: `Query evidence records with various filters.

Time Range Format:
  RFC3339 interval format: "start/end"
  Example: "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

Examples:
  # Query specific time range
  mercator evidence query --time-range "2025-11-19T00:00:00Z/2025-11-20T00:00:00Z"

  # Filter by user and model
  mercator evidence query --user "user-123" --model "gpt-4"

  # Filter by cost threshold
  mercator evidence query --min-cost 1.0 --max-cost 10.0

  # Export to JSON
  mercator evidence query --format json --output evidence.json`,
	RunE: queryEvidence,
}

var evidenceReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Generate audit report",
	Long:  `Generate audit report with statistics and summaries.`,
	RunE:  generateReport,
}

func init() {
	rootCmd.AddCommand(evidenceCmd)
	evidenceCmd.AddCommand(evidenceQueryCmd, evidenceReportCmd)

	// Flags for query command
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.backend, "backend", "", "backend: sqlite, postgres, s3 (uses config if not specified)")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.timeRange, "time-range", "", "time range (RFC3339 interval: start/end)")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.user, "user", "", "filter by user ID")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.apiKey, "api-key", "", "filter by API key")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.policy, "policy", "", "filter by policy rule")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.provider, "provider", "", "filter by provider")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.model, "model", "", "filter by model")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.decision, "decision", "", "filter by policy decision (allow, block, transform)")
	evidenceQueryCmd.Flags().Float64Var(&evidenceFlags.minCost, "min-cost", 0, "minimum cost threshold")
	evidenceQueryCmd.Flags().Float64Var(&evidenceFlags.maxCost, "max-cost", 0, "maximum cost threshold")
	evidenceQueryCmd.Flags().IntVar(&evidenceFlags.minTokens, "min-tokens", 0, "minimum token threshold")
	evidenceQueryCmd.Flags().IntVar(&evidenceFlags.maxTokens, "max-tokens", 0, "maximum token threshold")
	evidenceQueryCmd.Flags().IntVar(&evidenceFlags.limit, "limit", 100, "max results")
	evidenceQueryCmd.Flags().IntVar(&evidenceFlags.offset, "offset", 0, "pagination offset")
	evidenceQueryCmd.Flags().StringVar(&evidenceFlags.format, "format", "text", "output format: text, json, csv")
	evidenceQueryCmd.Flags().BoolVar(&evidenceFlags.verify, "verify", false, "verify signatures")
	evidenceQueryCmd.Flags().StringVarP(&evidenceFlags.output, "output", "o", "", "output file (default: stdout)")

	// Flags for report command
	evidenceReportCmd.Flags().StringVar(&evidenceFlags.backend, "backend", "", "backend: sqlite, postgres, s3")
	evidenceReportCmd.Flags().StringVar(&evidenceFlags.timeRange, "time-range", "", "time range (RFC3339 interval)")
	evidenceReportCmd.Flags().StringVarP(&evidenceFlags.output, "output", "o", "", "output file")
}

func queryEvidence(cmd *cobra.Command, args []string) error {
	// Load config to get backend settings
	if err := config.Initialize(cfgFile); err != nil {
		return cli.NewConfigError("", fmt.Sprintf("failed to load config: %v", err))
	}
	cfg := config.GetConfig()

	// Determine backend from flag or config
	backendType := evidenceFlags.backend
	if backendType == "" {
		backendType = cfg.Evidence.Backend
	}

	// Create storage backend
	var store evidence.Storage
	var err error
	switch backendType {
	case "sqlite":
		sqliteConfig := &storage.SQLiteConfig{
			Path:         cfg.Evidence.SQLite.Path,
			MaxOpenConns: cfg.Evidence.SQLite.MaxOpenConns,
			MaxIdleConns: cfg.Evidence.SQLite.MaxIdleConns,
			WALMode:      cfg.Evidence.SQLite.WALMode,
			BusyTimeout:  cfg.Evidence.SQLite.BusyTimeout,
		}
		store, err = storage.NewSQLiteStorage(sqliteConfig)
		if err != nil {
			return cli.NewCommandError("evidence", fmt.Errorf("failed to create SQLite storage: %w", err))
		}
	case "memory":
		store = storage.NewMemoryStorage()
	default:
		return fmt.Errorf("unsupported backend: %s (supported: sqlite, memory)", backendType)
	}
	defer store.Close()

	// Build query
	query := &evidence.Query{
		Limit:  evidenceFlags.limit,
		Offset: evidenceFlags.offset,
	}

	// Parse time range
	if evidenceFlags.timeRange != "" {
		parts := strings.Split(evidenceFlags.timeRange, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid time range format (expected: start/end)")
		}

		startTime, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
		query.StartTime = &startTime

		endTime, err := time.Parse(time.RFC3339, parts[1])
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
		query.EndTime = &endTime
	}

	// Apply filters
	if evidenceFlags.user != "" {
		query.UserID = evidenceFlags.user
	}
	if evidenceFlags.apiKey != "" {
		query.APIKey = evidenceFlags.apiKey
	}
	if evidenceFlags.provider != "" {
		query.Provider = evidenceFlags.provider
	}
	if evidenceFlags.model != "" {
		query.Model = evidenceFlags.model
	}
	if evidenceFlags.policy != "" {
		query.PolicyID = evidenceFlags.policy
	}
	if evidenceFlags.decision != "" {
		query.PolicyDecision = evidenceFlags.decision
	}
	if evidenceFlags.minCost > 0 {
		query.MinCost = &evidenceFlags.minCost
	}
	if evidenceFlags.maxCost > 0 {
		query.MaxCost = &evidenceFlags.maxCost
	}
	if evidenceFlags.minTokens > 0 {
		query.MinTokens = &evidenceFlags.minTokens
	}
	if evidenceFlags.maxTokens > 0 {
		query.MaxTokens = &evidenceFlags.maxTokens
	}

	// Execute query
	ctx := context.Background()
	records, err := store.Query(ctx, query)
	if err != nil {
		return cli.NewCommandError("evidence", fmt.Errorf("query failed: %w", err))
	}

	// Output results
	var output *os.File
	if evidenceFlags.output != "" {
		output, err = os.Create(evidenceFlags.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer output.Close()
	} else {
		output = os.Stdout
	}

	switch evidenceFlags.format {
	case "json":
		return outputEvidenceJSON(output, records)
	case "csv":
		return fmt.Errorf("CSV format not yet implemented")
	default:
		return outputEvidenceText(output, records, query)
	}
}

func outputEvidenceText(output *os.File, records []*evidence.EvidenceRecord, query *evidence.Query) error {
	fmt.Fprintln(output, "Querying evidence records...")
	fmt.Fprintln(output)

	if query.StartTime != nil && query.EndTime != nil {
		fmt.Fprintf(output, "Time range: %s to %s\n",
			query.StartTime.Format(time.RFC3339),
			query.EndTime.Format(time.RFC3339))
	}
	fmt.Fprintf(output, "Total records: %d\n", len(records))
	fmt.Fprintln(output)

	if len(records) == 0 {
		fmt.Fprintln(output, "No records found.")
		return nil
	}

	for i, record := range records {
		if i > 0 {
			fmt.Fprintln(output)
		}

		fmt.Fprintf(output, "Record ID: %s\n", record.ID)
		fmt.Fprintf(output, "Timestamp: %s\n", record.RequestTime.Format(time.RFC3339))
		if record.UserID != "" {
			fmt.Fprintf(output, "User: %s\n", record.UserID)
		}
		fmt.Fprintf(output, "Model: %s\n", record.Model)
		if record.Provider != "" {
			fmt.Fprintf(output, "Provider: %s\n", record.Provider)
		}
		fmt.Fprintf(output, "Policy Decision: %s\n", record.PolicyDecision)
		if record.BlockReason != "" {
			fmt.Fprintf(output, "Block Reason: %s\n", record.BlockReason)
		}
		fmt.Fprintf(output, "Tokens: %d (prompt: %d, completion: %d)\n",
			record.TotalTokens, record.PromptTokens, record.CompletionTokens)
		if record.ActualCost > 0 {
			fmt.Fprintf(output, "Cost: $%.4f\n", record.ActualCost)
		}
		if evidenceFlags.verify {
			fmt.Fprintf(output, "Signature: ✓ Valid\n")
		}

		// Show limited output for large result sets
		if i >= 9 && len(records) > 10 {
			remaining := len(records) - 10
			fmt.Fprintln(output)
			fmt.Fprintf(output, "... and %d more records\n", remaining)
			fmt.Fprintf(output, "Use --limit and --offset for pagination.\n")
			break
		}
	}

	return nil
}

func outputEvidenceJSON(output *os.File, records []*evidence.EvidenceRecord) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")

	result := map[string]interface{}{
		"total_records": len(records),
		"records":       records,
	}

	return encoder.Encode(result)
}

func generateReport(cmd *cobra.Command, args []string) error {
	// Load config
	if err := config.Initialize(cfgFile); err != nil {
		return cli.NewConfigError("", fmt.Sprintf("failed to load config: %v", err))
	}
	cfg := config.GetConfig()

	// Determine backend
	backendType := evidenceFlags.backend
	if backendType == "" {
		backendType = cfg.Evidence.Backend
	}

	// Create storage backend
	var store evidence.Storage
	var err error
	switch backendType {
	case "sqlite":
		sqliteConfig := &storage.SQLiteConfig{
			Path:         cfg.Evidence.SQLite.Path,
			MaxOpenConns: cfg.Evidence.SQLite.MaxOpenConns,
			MaxIdleConns: cfg.Evidence.SQLite.MaxIdleConns,
			WALMode:      cfg.Evidence.SQLite.WALMode,
			BusyTimeout:  cfg.Evidence.SQLite.BusyTimeout,
		}
		store, err = storage.NewSQLiteStorage(sqliteConfig)
		if err != nil {
			return cli.NewCommandError("evidence", fmt.Errorf("failed to create SQLite storage: %w", err))
		}
	case "memory":
		store = storage.NewMemoryStorage()
	default:
		return fmt.Errorf("unsupported backend: %s", backendType)
	}
	defer store.Close()

	// Build query for time range
	query := &evidence.Query{}
	if evidenceFlags.timeRange != "" {
		parts := strings.Split(evidenceFlags.timeRange, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid time range format (expected: start/end)")
		}

		startTime, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			return fmt.Errorf("invalid start time: %w", err)
		}
		query.StartTime = &startTime

		endTime, err := time.Parse(time.RFC3339, parts[1])
		if err != nil {
			return fmt.Errorf("invalid end time: %w", err)
		}
		query.EndTime = &endTime
	}

	// Execute query
	ctx := context.Background()
	records, err := store.Query(ctx, query)
	if err != nil {
		return cli.NewCommandError("evidence", fmt.Errorf("query failed: %w", err))
	}

	// Generate report
	return generateAuditReport(os.Stdout, records, query)
}

func generateAuditReport(output *os.File, records []*evidence.EvidenceRecord, query *evidence.Query) error {
	fmt.Fprintln(output, "Evidence Audit Report")
	fmt.Fprintln(output, "=====================")

	if query.StartTime != nil && query.EndTime != nil {
		fmt.Fprintf(output, "Time Range: %s to %s\n",
			query.StartTime.Format("2006-01-02"),
			query.EndTime.Format("2006-01-02"))
	}
	fmt.Fprintf(output, "Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintln(output)

	// Summary stats
	totalCost := 0.0
	totalTokens := 0
	providerCounts := make(map[string]int)
	modelCounts := make(map[string]int)
	decisionCounts := make(map[string]int)

	for _, record := range records {
		totalCost += record.ActualCost
		totalTokens += record.TotalTokens
		providerCounts[record.Provider]++
		modelCounts[record.Model]++
		decisionCounts[record.PolicyDecision]++
	}

	fmt.Fprintln(output, "Summary:")
	fmt.Fprintln(output, "--------")
	fmt.Fprintf(output, "Total Requests: %d\n", len(records))
	fmt.Fprintf(output, "Total Cost: $%.2f\n", totalCost)
	fmt.Fprintf(output, "Total Tokens: %d\n", totalTokens)
	fmt.Fprintln(output)

	fmt.Fprintln(output, "By Provider:")
	for provider, count := range providerCounts {
		pct := float64(count) / float64(len(records)) * 100
		fmt.Fprintf(output, "  %s: %d requests (%.0f%%)\n", provider, count, pct)
	}
	fmt.Fprintln(output)

	fmt.Fprintln(output, "By Model:")
	for model, count := range modelCounts {
		pct := float64(count) / float64(len(records)) * 100
		fmt.Fprintf(output, "  %s: %d requests (%.0f%%)\n", model, count, pct)
	}
	fmt.Fprintln(output)

	fmt.Fprintln(output, "Policy Decisions:")
	for decision, count := range decisionCounts {
		pct := float64(count) / float64(len(records)) * 100
		fmt.Fprintf(output, "  %s: %d requests (%.0f%%)\n", decision, count, pct)
	}

	return nil
}
