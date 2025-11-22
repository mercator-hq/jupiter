package engine

import (
	"fmt"
	"regexp"
	"strings"
)

// ApplyRedaction applies a redaction to content based on the redaction configuration.
func ApplyRedaction(content string, redaction Redaction) (string, error) {
	switch redaction.Strategy {
	case "mask":
		return applyMaskRedaction(content, redaction)
	case "remove":
		return applyRemoveRedaction(content, redaction)
	case "replace":
		return applyReplaceRedaction(content, redaction)
	default:
		return content, fmt.Errorf("unknown redaction strategy: %q", redaction.Strategy)
	}
}

// applyMaskRedaction masks content by replacing characters with asterisks.
func applyMaskRedaction(content string, redaction Redaction) (string, error) {
	if redaction.Pattern == "" {
		// No pattern - mask the entire content
		return strings.Repeat("*", len(content)), nil
	}

	// Pattern-based masking
	re, err := regexp.Compile(redaction.Pattern)
	if err != nil {
		return content, fmt.Errorf("invalid regex pattern: %w", err)
	}

	replacement := redaction.Replacement
	if replacement == "" {
		replacement = "***"
	}

	return re.ReplaceAllString(content, replacement), nil
}

// applyRemoveRedaction removes matching content entirely.
func applyRemoveRedaction(content string, redaction Redaction) (string, error) {
	if redaction.Pattern == "" {
		// No pattern - remove all content
		return "", nil
	}

	// Pattern-based removal
	re, err := regexp.Compile(redaction.Pattern)
	if err != nil {
		return content, fmt.Errorf("invalid regex pattern: %w", err)
	}

	return re.ReplaceAllString(content, ""), nil
}

// applyReplaceRedaction replaces matching content with replacement text.
func applyReplaceRedaction(content string, redaction Redaction) (string, error) {
	if redaction.Pattern == "" {
		// No pattern - replace entire content
		return redaction.Replacement, nil
	}

	// Pattern-based replacement
	re, err := regexp.Compile(redaction.Pattern)
	if err != nil {
		return content, fmt.Errorf("invalid regex pattern: %w", err)
	}

	replacement := redaction.Replacement
	if replacement == "" {
		replacement = "[REDACTED]"
	}

	return re.ReplaceAllString(content, replacement), nil
}

// ApplyRedactions applies multiple redactions to content in order.
func ApplyRedactions(content string, redactions []Redaction) (string, error) {
	result := content

	for i, redaction := range redactions {
		var err error
		result, err = ApplyRedaction(result, redaction)
		if err != nil {
			return content, fmt.Errorf("failed to apply redaction %d: %w", i, err)
		}
	}

	return result, nil
}

// CountMatches counts how many times a pattern matches in content.
func CountMatches(content, pattern string) (int, error) {
	if pattern == "" {
		return 0, nil
	}

	re, err := regexp.Compile(pattern)
	if err != nil {
		return 0, fmt.Errorf("invalid regex pattern: %w", err)
	}

	matches := re.FindAllString(content, -1)
	return len(matches), nil
}
