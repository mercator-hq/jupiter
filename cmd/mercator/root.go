package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Global flags
	cfgFile string
	verbose bool
)

var rootCmd = &cobra.Command{
	Use:   "mercator",
	Short: "Mercator Jupiter - GitOps-native LLM governance runtime",
	Long: `Mercator Jupiter is an open-source LLM governance runtime that provides
policy-based control, evidence generation, and cost management for LLM applications.

It acts as an HTTP proxy for LLM API requests, providing:
  - Policy-based request governance and routing
  - Multi-provider LLM routing (OpenAI, Anthropic, etc.)
  - Cryptographic evidence generation for audit trails
  - Cost tracking and budget enforcement
  - Content analysis and PII detection

For more information, visit: https://github.com/mercator-hq/jupiter`,
	Version: Version,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global persistent flags (available to all subcommands)
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "config.yaml", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Disable default completion command (we'll add our own)
	rootCmd.CompletionOptions.DisableDefaultCmd = false
}
