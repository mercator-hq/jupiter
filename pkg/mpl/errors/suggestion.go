package errors

import (
	"fmt"
	"strings"
)

// SuggestFieldName suggests possible field names when an unknown field is referenced.
// It uses Levenshtein distance to find similar field names.
func SuggestFieldName(unknown string, validFields []string) string {
	if len(validFields) == 0 {
		return ""
	}

	// Find the closest match
	minDistance := 1000
	var bestMatch string

	for _, field := range validFields {
		dist := levenshteinDistance(unknown, field)
		if dist < minDistance {
			minDistance = dist
			bestMatch = field
		}
	}

	// Only suggest if the distance is reasonable (< 5 edits)
	if minDistance < 5 {
		return fmt.Sprintf("Did you mean '%s'?", bestMatch)
	}

	// If no close match, suggest a few common fields
	if len(validFields) > 5 {
		return fmt.Sprintf("Valid fields include: %s, ...", strings.Join(validFields[:5], ", "))
	}
	return fmt.Sprintf("Valid fields: %s", strings.Join(validFields, ", "))
}

// SuggestOperator suggests valid operators for a field type.
func SuggestOperator(fieldType string) string {
	switch fieldType {
	case "string":
		return "Valid operators: ==, !=, contains, matches, starts_with, ends_with, in, not_in"
	case "number":
		return "Valid operators: ==, !=, <, >, <=, >=, in, not_in"
	case "boolean":
		return "Valid operators: ==, !="
	case "array":
		return "Valid operators: contains, in, not_in"
	default:
		return "Valid operators: ==, !=, <, >, <=, >=, contains, matches, starts_with, ends_with, in, not_in"
	}
}

// SuggestActionType suggests valid action types when an unknown action is specified.
func SuggestActionType(unknown string, validActions []string) string {
	if len(validActions) == 0 {
		return ""
	}

	// Find the closest match
	minDistance := 1000
	var bestMatch string

	for _, action := range validActions {
		dist := levenshteinDistance(unknown, action)
		if dist < minDistance {
			minDistance = dist
			bestMatch = action
		}
	}

	// Only suggest if the distance is reasonable
	if minDistance < 5 {
		return fmt.Sprintf("Did you mean '%s'?", bestMatch)
	}

	return fmt.Sprintf("Valid action types: %s", strings.Join(validActions, ", "))
}

// SuggestMissingField suggests adding a required field.
func SuggestMissingField(fieldName string, exampleValue string) string {
	if exampleValue != "" {
		return fmt.Sprintf("Add '%s: %s' to the policy", fieldName, exampleValue)
	}
	return fmt.Sprintf("Add '%s' field to the policy", fieldName)
}

// levenshteinDistance computes the Levenshtein distance between two strings.
// This is used for finding similar field/action names for suggestions.
func levenshteinDistance(s1, s2 string) int {
	if s1 == s2 {
		return 0
	}

	len1 := len(s1)
	len2 := len(s2)

	// Create distance matrix
	matrix := make([][]int, len1+1)
	for i := range matrix {
		matrix[i] = make([]int, len2+1)
	}

	// Initialize first column and row
	for i := 0; i <= len1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= len2; j++ {
		matrix[0][j] = j
	}

	// Compute distances
	for i := 1; i <= len1; i++ {
		for j := 1; j <= len2; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}

			matrix[i][j] = min(
				matrix[i-1][j]+1,      // Deletion
				matrix[i][j-1]+1,      // Insertion
				matrix[i-1][j-1]+cost, // Substitution
			)
		}
	}

	return matrix[len1][len2]
}

// min returns the minimum of three integers.
func min(a, b, c int) int {
	if a < b {
		if a < c {
			return a
		}
		return c
	}
	if b < c {
		return b
	}
	return c
}
