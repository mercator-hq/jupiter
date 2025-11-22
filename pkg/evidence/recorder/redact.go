package recorder

import (
	"crypto/sha256"
	"encoding/hex"
)

// RedactAPIKey redacts an API key by hashing it with SHA-256.
// This prevents storing API keys in plaintext while allowing for
// identification of which key was used.
//
// The hash cannot be reversed, so the original API key cannot be
// recovered from the evidence record.
//
// Returns an empty string if the API key is empty.
func RedactAPIKey(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	// Hash the API key with SHA-256
	hash := sha256.Sum256([]byte(apiKey))

	// Return hex-encoded hash
	return "sha256:" + hex.EncodeToString(hash[:])
}

// RedactAPIKeyTruncated redacts an API key by showing only the first
// and last 4 characters, with the middle replaced by asterisks.
//
// This allows visual identification of the key while preventing full exposure.
// For keys shorter than 12 characters, returns all asterisks.
//
// Example: "sk-abc123xyz789" -> "sk-a***9789"
//
// Returns an empty string if the API key is empty.
func RedactAPIKeyTruncated(apiKey string) string {
	if apiKey == "" {
		return ""
	}

	// For short keys, just redact everything
	if len(apiKey) < 12 {
		return "****"
	}

	// Show first 4 and last 4 characters
	return apiKey[:4] + "***" + apiKey[len(apiKey)-4:]
}

// TruncateString truncates a string to the specified maximum length.
// If the string is longer than maxLen, it is truncated and "..." is appended.
//
// Returns the original string if it's shorter than maxLen.
func TruncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}

	// Truncate and append ellipsis
	if maxLen <= 3 {
		return s[:maxLen]
	}

	return s[:maxLen-3] + "..."
}
