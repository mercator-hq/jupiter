package recorder

import (
	"crypto/sha256"
	"encoding/hex"
)

const (
	// MaxHashSize is the maximum number of bytes to hash from large bodies.
	// Hashing only the first 1MB prevents memory exhaustion while still
	// providing reasonable collision resistance for integrity verification.
	MaxHashSize = 1024 * 1024 // 1MB
)

// HashContent computes the SHA-256 hash of the content and returns it as a
// hex-encoded string. For large content exceeding MaxHashSize, only the first
// MaxHashSize bytes are hashed.
//
// Returns an empty string if content is empty.
func HashContent(content []byte) string {
	if len(content) == 0 {
		return ""
	}

	// Limit hash size to prevent memory exhaustion on large bodies
	contentToHash := content
	if len(content) > MaxHashSize {
		contentToHash = content[:MaxHashSize]
	}

	// Compute SHA-256 hash
	hash := sha256.Sum256(contentToHash)

	// Return hex-encoded hash
	return hex.EncodeToString(hash[:])
}

// HashString is a convenience function that hashes a string and returns the
// hex-encoded SHA-256 hash.
func HashString(content string) string {
	return HashContent([]byte(content))
}
