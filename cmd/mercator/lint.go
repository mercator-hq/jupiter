package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"mercator-hq/jupiter/pkg/cli"
	mplErrors "mercator-hq/jupiter/pkg/mpl/errors"
	"mercator-hq/jupiter/pkg/mpl/parser"
	"mercator-hq/jupiter/pkg/mpl/validator"
)

var lintFlags struct {
	file   string
	dir    string
	strict bool
	format string
}

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Validate policy files",
	Long: `Validate MPL policy files for syntax and semantic errors.

The lint command parses policy files and performs comprehensive validation:
  - YAML syntax validation
  - Policy structure validation
  - Semantic validation (condition types, field references)
  - Action validation (parameter types and constraints)

Examples:
  # Lint single file
  mercator lint --file policies.yaml

  # Lint directory
  mercator lint --dir policies/

  # Strict mode (warnings as errors)
  mercator lint --file policies.yaml --strict

  # JSON output for CI/CD
  mercator lint --file policies.yaml --format json`,
	RunE: lintPolicies,
}

func init() {
	rootCmd.AddCommand(lintCmd)

	lintCmd.Flags().StringVarP(&lintFlags.file, "file", "f", "", "policy file to validate")
	lintCmd.Flags().StringVarP(&lintFlags.dir, "dir", "d", "", "directory of policy files")
	lintCmd.Flags().BoolVar(&lintFlags.strict, "strict", false, "treat warnings as errors")
	lintCmd.Flags().StringVar(&lintFlags.format, "format", "text", "output format: text, json")
}

func lintPolicies(cmd *cobra.Command, args []string) error {
	if lintFlags.file == "" && lintFlags.dir == "" {
		return fmt.Errorf("either --file or --dir must be specified")
	}

	var files []string

	if lintFlags.file != "" {
		files = append(files, lintFlags.file)
	}

	if lintFlags.dir != "" {
		matches, err := filepath.Glob(filepath.Join(lintFlags.dir, "*.yaml"))
		if err != nil {
			return fmt.Errorf("failed to list policy files: %w", err)
		}
		yamlMatches, err := filepath.Glob(filepath.Join(lintFlags.dir, "*.yml"))
		if err != nil {
			return fmt.Errorf("failed to list policy files: %w", err)
		}
		files = append(files, matches...)
		files = append(files, yamlMatches...)
	}

	if len(files) == 0 {
		return fmt.Errorf("no policy files found")
	}

	results := make([]ValidationResult, 0, len(files))

	for _, file := range files {
		result := validatePolicyFile(file)
		results = append(results, result)
	}

	// Output results
	if lintFlags.format == "json" {
		return outputJSON(results)
	}
	return outputText(results, lintFlags.strict)
}

// ValidationResult represents the validation result for a single policy file.
type ValidationResult struct {
	File     string            `json:"file"`
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors,omitempty"`
	Warnings []ValidationError `json:"warnings,omitempty"`
}

// ValidationError represents a single validation error or warning.
type ValidationError struct {
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Rule     string `json:"rule,omitempty"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Type     string `json:"type,omitempty"`
}

func validatePolicyFile(path string) ValidationResult {
	result := ValidationResult{
		File:  path,
		Valid: true,
	}

	// Create parser
	p := parser.NewParser()
	if lintFlags.strict {
		p.WithStrictMode(true)
	}

	// Parse policy
	policy, err := p.Parse(path)
	if err != nil {
		result.Valid = false

		// Handle MPL errors
		if errList, ok := err.(*mplErrors.ErrorList); ok {
			for _, e := range errList.Errors {
				validationErr := ValidationError{
					Line:     e.Location.Line,
					Column:   e.Location.Column,
					Message:  e.Message,
					Severity: "error",
					Type:     string(e.Type),
				}
				result.Errors = append(result.Errors, validationErr)
			}
		} else if mplErr, ok := err.(*mplErrors.Error); ok {
			validationErr := ValidationError{
				Line:     mplErr.Location.Line,
				Column:   mplErr.Location.Column,
				Message:  mplErr.Message,
				Severity: "error",
				Type:     string(mplErr.Type),
			}
			result.Errors = append(result.Errors, validationErr)
		} else {
			// Generic error
			validationErr := ValidationError{
				Message:  err.Error(),
				Severity: "error",
			}
			result.Errors = append(result.Errors, validationErr)
		}
		return result
	}

	// Validate policy
	v := validator.NewValidator()
	if err := v.Validate(policy); err != nil {
		result.Valid = false

		// Handle MPL errors
		if errList, ok := err.(*mplErrors.ErrorList); ok {
			for _, e := range errList.Errors {
				validationErr := ValidationError{
					Line:     e.Location.Line,
					Column:   e.Location.Column,
					Message:  e.Message,
					Severity: "error",
					Type:     string(e.Type),
				}

				// Classify warnings vs errors
				// For now, all validation errors are errors
				// In the future, we could have a severity field in mplErrors.Error
				result.Errors = append(result.Errors, validationErr)
			}
		} else if mplErr, ok := err.(*mplErrors.Error); ok {
			validationErr := ValidationError{
				Line:     mplErr.Location.Line,
				Column:   mplErr.Location.Column,
				Message:  mplErr.Message,
				Severity: "error",
				Type:     string(mplErr.Type),
			}
			result.Errors = append(result.Errors, validationErr)
		} else {
			// Generic error
			validationErr := ValidationError{
				Message:  err.Error(),
				Severity: "error",
			}
			result.Errors = append(result.Errors, validationErr)
		}
	}

	return result
}

func outputText(results []ValidationResult, strict bool) error {
	totalErrors := 0
	totalWarnings := 0

	for _, result := range results {
		fmt.Printf("Validating %s...\n", result.File)

		if len(result.Errors) == 0 && len(result.Warnings) == 0 {
			fmt.Println("✓ Syntax valid")
			fmt.Println("✓ All rules have valid conditions")
		}

		for _, err := range result.Errors {
			fmt.Printf("✗ Error: %s", err.Message)
			if err.Line > 0 {
				fmt.Printf(" (line %d", err.Line)
				if err.Column > 0 {
					fmt.Printf(", col %d", err.Column)
				}
				fmt.Print(")")
			}
			if err.Type != "" {
				fmt.Printf(" [%s]", err.Type)
			}
			fmt.Println()
			totalErrors++
		}

		for _, warn := range result.Warnings {
			fmt.Printf("⚠  Warning: %s", warn.Message)
			if warn.Line > 0 {
				fmt.Printf(" (line %d", warn.Line)
				if warn.Column > 0 {
					fmt.Printf(", col %d", warn.Column)
				}
				fmt.Print(")")
			}
			fmt.Println()
			totalWarnings++
		}

		fmt.Println()
	}

	fmt.Println("Summary:")
	fmt.Printf("  %d error(s), %d warning(s)\n", totalErrors, totalWarnings)

	if strict && totalWarnings > 0 {
		fmt.Println("  Strict mode enabled: treating warnings as errors")
		return cli.NewCommandError("lint", fmt.Errorf("validation failed"))
	}

	if totalErrors > 0 {
		return cli.NewCommandError("lint", fmt.Errorf("validation failed"))
	}

	return nil
}

func outputJSON(results []ValidationResult) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(results)
}
