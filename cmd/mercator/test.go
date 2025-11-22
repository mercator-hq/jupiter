package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"mercator-hq/jupiter/pkg/cli"
	"mercator-hq/jupiter/pkg/policy/engine"
	"mercator-hq/jupiter/pkg/policy/engine/source"
	"mercator-hq/jupiter/pkg/processing"
	"mercator-hq/jupiter/pkg/processing/content"
	"mercator-hq/jupiter/pkg/processing/conversation"
	"mercator-hq/jupiter/pkg/processing/costs"
	"mercator-hq/jupiter/pkg/proxy/types"
)

var testFlags struct {
	policyFile string
	testsFile  string
	dryRun     bool
	logsFile   string
	coverage   bool
	baseline   string
	format     string
}

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "Run policy unit tests",
	Long: `Execute policy unit tests or dry-run against traffic logs.

The test command loads policy files and test case files, then executes
each test case against the policy engine to verify expected behavior.

Test Case Format (YAML):
  tests:
    - name: "Test case description"
      request:
        model: "gpt-4"
        messages:
          - role: "user"
            content: "Hello"
      metadata:
        user_id: "test-user"
        estimated_cost: 1.50
      expect:
        action: "allow"  # allow, block, transform, route
        reason: ""       # optional: expected block reason

Examples:
  # Run unit tests
  mercator test --policy policies.yaml --tests policy_tests.yaml

  # Dry-run against logs (not yet implemented)
  mercator test --policy policies.yaml --dry-run --logs traffic.jsonl

  # Generate coverage report (not yet implemented)
  mercator test --policy policies.yaml --tests policy_tests.yaml --coverage`,
	RunE: runTests,
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().StringVarP(&testFlags.policyFile, "policy", "p", "", "policy file to test")
	testCmd.Flags().StringVarP(&testFlags.testsFile, "tests", "t", "", "test case file")
	testCmd.Flags().BoolVar(&testFlags.dryRun, "dry-run", false, "evaluate against traffic logs")
	testCmd.Flags().StringVar(&testFlags.logsFile, "logs", "", "traffic logs file (for dry-run)")
	testCmd.Flags().BoolVar(&testFlags.coverage, "coverage", false, "generate coverage report")
	testCmd.Flags().StringVar(&testFlags.baseline, "baseline", "", "baseline results file")
	testCmd.Flags().StringVar(&testFlags.format, "format", "text", "output format: text, json, junit")

	// Mark required flags - panic if this fails as it's a programming error
	if err := testCmd.MarkFlagRequired("policy"); err != nil {
		panic(fmt.Sprintf("failed to mark policy flag as required: %v", err))
	}
}

func runTests(cmd *cobra.Command, args []string) error {
	if testFlags.testsFile == "" && !testFlags.dryRun {
		return fmt.Errorf("either --tests or --dry-run must be specified")
	}

	if testFlags.dryRun {
		return fmt.Errorf("dry-run mode not yet implemented")
	}

	// Load test cases
	testSuite, err := loadTestCases(testFlags.testsFile)
	if err != nil {
		return cli.NewCommandError("test", fmt.Errorf("failed to load test cases: %w", err))
	}

	if len(testSuite.Tests) == 0 {
		return fmt.Errorf("no test cases found in %s", testFlags.testsFile)
	}

	// Create policy engine
	logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError, // Suppress logs during testing
	}))

	policySource := source.NewFileSource(testFlags.policyFile, logger)
	engineConfig := engine.DefaultEngineConfig()
	engineConfig.EnableTrace = false // Disable trace for testing
	engineConfig.FailSafeMode = engine.FailClosed
	engineConfig.DefaultAction = engine.ActionAllow

	policyEngine, err := engine.NewInterpreterEngine(engineConfig, policySource, logger)
	if err != nil {
		return cli.NewCommandError("test", fmt.Errorf("failed to create policy engine: %w", err))
	}
	defer policyEngine.Close()

	fmt.Println("Running policy tests...")
	fmt.Println()

	// Run tests
	results := make([]TestResult, 0, len(testSuite.Tests))
	passed := 0
	failed := 0

	for _, testCase := range testSuite.Tests {
		result := runTestCase(policyEngine, testCase)
		results = append(results, result)

		if result.Passed {
			passed++
			fmt.Printf("✓ %s (%.1fms)\n", testCase.Name, result.Duration.Seconds()*1000)
		} else {
			failed++
			fmt.Printf("✗ %s\n", testCase.Name)
			if result.Error != "" {
				fmt.Printf("  Error: %s\n", result.Error)
			} else {
				fmt.Printf("  Expected: action=%s", testCase.Expect.Action)
				if testCase.Expect.Reason != "" {
					fmt.Printf(", reason=%q", testCase.Expect.Reason)
				}
				fmt.Println()
				fmt.Printf("  Actual:   action=%s", result.ActualAction)
				if result.ActualReason != "" {
					fmt.Printf(", reason=%q", result.ActualReason)
				}
				fmt.Println()
			}
		}
	}

	fmt.Println()
	fmt.Println("Summary:")
	fmt.Printf("  %d tests run, %d passed, %d failed\n", len(testSuite.Tests), passed, failed)

	if testFlags.coverage {
		fmt.Println("  Coverage report not yet implemented")
	}

	if failed > 0 {
		fmt.Println()
		fmt.Println("Failed tests:")
		for _, result := range results {
			if !result.Passed {
				fmt.Printf("  - %s\n", result.TestName)
			}
		}
		return cli.NewCommandError("test", fmt.Errorf("test failures"))
	}

	return nil
}

// TestSuite represents a collection of test cases.
type TestSuite struct {
	Tests []TestCase `yaml:"tests"`
}

// TestCase represents a single policy test case.
type TestCase struct {
	Name     string                 `yaml:"name"`
	Request  TestRequest            `yaml:"request"`
	Metadata map[string]interface{} `yaml:"metadata"`
	Expect   TestExpectation        `yaml:"expect"`
}

// TestRequest represents the request portion of a test case.
type TestRequest struct {
	Model       string          `yaml:"model"`
	Messages    []types.Message `yaml:"messages"`
	MaxTokens   int             `yaml:"max_tokens,omitempty"`
	Temperature float64         `yaml:"temperature,omitempty"`
}

// TestExpectation represents the expected result of a test case.
type TestExpectation struct {
	Action string `yaml:"action"` // allow, block, transform, route
	Reason string `yaml:"reason,omitempty"`
}

// TestResult represents the result of executing a single test case.
type TestResult struct {
	TestName     string
	Passed       bool
	ActualAction string
	ActualReason string
	Error        string
	Duration     time.Duration
}

func loadTestCases(path string) (*TestSuite, error) {
	// #nosec G304 - User-specified test file path is expected behavior for a CLI tool.
	// The test command intentionally reads user-provided test case files.
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var suite TestSuite
	if err := yaml.Unmarshal(data, &suite); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return &suite, nil
}

func runTestCase(policyEngine *engine.InterpreterEngine, testCase TestCase) TestResult {
	start := time.Now()

	result := TestResult{
		TestName: testCase.Name,
	}

	// Build enriched request
	chatReq := &types.ChatCompletionRequest{
		Model:    testCase.Request.Model,
		Messages: testCase.Request.Messages,
	}
	if testCase.Request.MaxTokens > 0 {
		chatReq.MaxTokens = &testCase.Request.MaxTokens
	}
	if testCase.Request.Temperature > 0 {
		chatReq.Temperature = &testCase.Request.Temperature
	}

	// Create enriched request with metadata
	enriched := &processing.EnrichedRequest{
		RequestID:       "test-request",
		OriginalRequest: chatReq,
		TokenEstimate: &processing.TokenEstimate{
			PromptTokens: 10, // Dummy value for testing
			TotalTokens:  10,
			Model:        testCase.Request.Model,
		},
		ContentAnalysis: &content.ContentAnalysis{
			PIIDetection: &content.PIIDetection{
				HasPII:   false,
				PIITypes: []string{},
			},
			SensitiveContent: &content.SensitiveContent{
				HasSensitiveContent: false,
				Severity:            "low",
			},
		},
		ConversationContext: &conversation.ConversationContext{
			TurnCount:              len(chatReq.Messages),
			MessageCount:           len(chatReq.Messages),
			HasConversationHistory: len(chatReq.Messages) > 1,
		},
		CostEstimate: &costs.CostEstimate{
			PromptCost:     0.005,
			CompletionCost: 0.005,
			TotalCost:      0.01,
			Currency:       "USD",
			Model:          testCase.Request.Model,
		},
		ModelFamily:     "gpt",
		ComplexityScore: 1,
		RiskScore:       1,
	}

	// Apply metadata from test case
	if cost, ok := testCase.Metadata["estimated_cost"].(float64); ok {
		enriched.CostEstimate.TotalCost = cost
	}

	// Evaluate policy
	ctx := context.Background()
	decision, err := policyEngine.EvaluateRequest(ctx, enriched)
	if err != nil {
		result.Error = err.Error()
		result.Duration = time.Since(start)
		return result
	}

	// Check result
	result.ActualAction = string(decision.Action)
	result.ActualReason = decision.BlockReason
	result.Duration = time.Since(start)

	// Compare with expectation
	if result.ActualAction == testCase.Expect.Action {
		if testCase.Expect.Reason == "" || result.ActualReason == testCase.Expect.Reason {
			result.Passed = true
		}
	}

	return result
}
