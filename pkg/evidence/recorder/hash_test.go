package recorder

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestHashContent(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected string
	}{
		{
			name:     "empty content",
			content:  []byte{},
			expected: "",
		},
		{
			name:     "nil content",
			content:  nil,
			expected: "",
		},
		{
			name:     "small content",
			content:  []byte("hello world"),
			expected: computeSHA256("hello world"),
		},
		{
			name:     "json content",
			content:  []byte(`{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`),
			expected: computeSHA256(`{"model":"gpt-4","messages":[{"role":"user","content":"hello"}]}`),
		},
		{
			name:     "large content under limit",
			content:  bytes.Repeat([]byte("a"), MaxHashSize-1),
			expected: computeSHA256(string(bytes.Repeat([]byte("a"), MaxHashSize-1))),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashContent(tt.content)
			if result != tt.expected {
				t.Errorf("HashContent() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHashContent_LargeContent(t *testing.T) {
	// Test content larger than MaxHashSize
	largeContent := bytes.Repeat([]byte("a"), MaxHashSize+1000)

	// Should only hash first MaxHashSize bytes
	expectedContent := largeContent[:MaxHashSize]
	expected := computeSHA256(string(expectedContent))

	result := HashContent(largeContent)

	if result != expected {
		t.Errorf("HashContent() for large content = %v, want %v", result, expected)
	}
}

func TestHashString(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected string
	}{
		{
			name:     "empty string",
			content:  "",
			expected: "",
		},
		{
			name:     "simple string",
			content:  "test string",
			expected: computeSHA256("test string"),
		},
		{
			name:     "unicode string",
			content:  "Hello 世界 🌍",
			expected: computeSHA256("Hello 世界 🌍"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := HashString(tt.content)
			if result != tt.expected {
				t.Errorf("HashString() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHashContent_Deterministic(t *testing.T) {
	// Hashing the same content multiple times should produce the same result
	content := []byte("deterministic test")

	hash1 := HashContent(content)
	hash2 := HashContent(content)
	hash3 := HashContent(content)

	if hash1 != hash2 || hash2 != hash3 {
		t.Errorf("HashContent() not deterministic: %v, %v, %v", hash1, hash2, hash3)
	}
}

func TestHashContent_HexEncoding(t *testing.T) {
	content := []byte("test")
	result := HashContent(content)

	// Verify result is valid hex
	_, err := hex.DecodeString(result)
	if err != nil {
		t.Errorf("HashContent() returned invalid hex: %v", err)
	}

	// SHA-256 produces 32 bytes = 64 hex characters
	if len(result) != 64 {
		t.Errorf("HashContent() length = %d, want 64", len(result))
	}
}

func TestHashContent_Uniqueness(t *testing.T) {
	// Different content should produce different hashes
	hash1 := HashContent([]byte("content1"))
	hash2 := HashContent([]byte("content2"))
	hash3 := HashContent([]byte("content3"))

	if hash1 == hash2 || hash2 == hash3 || hash1 == hash3 {
		t.Errorf("HashContent() not unique: %v, %v, %v", hash1, hash2, hash3)
	}
}

func TestHashContent_MaxHashSizeConstant(t *testing.T) {
	// Verify MaxHashSize is 1MB as documented
	expected := 1024 * 1024
	if MaxHashSize != expected {
		t.Errorf("MaxHashSize = %d, want %d (1MB)", MaxHashSize, expected)
	}
}

// BenchmarkHashContent benchmarks hashing performance
func BenchmarkHashContent(b *testing.B) {
	sizes := []int{
		1024,            // 1KB
		10 * 1024,       // 10KB
		100 * 1024,      // 100KB
		MaxHashSize,     // 1MB
		MaxHashSize * 2, // 2MB (tests truncation)
	}

	for _, size := range sizes {
		b.Run(formatSize(size), func(b *testing.B) {
			content := bytes.Repeat([]byte("a"), size)
			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				_ = HashContent(content)
			}
		})
	}
}

func BenchmarkHashString(b *testing.B) {
	content := strings.Repeat("test string ", 1000)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = HashString(content)
	}
}

// Helper function to compute expected SHA-256 hash
func computeSHA256(content string) string {
	if content == "" {
		return ""
	}
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// Helper function to format size for benchmark names
func formatSize(bytes int) string {
	if bytes < 1024 {
		return string(rune(bytes)) + "B"
	}
	kb := bytes / 1024
	if kb < 1024 {
		return string(rune(kb)) + "KB"
	}
	mb := kb / 1024
	return string(rune(mb)) + "MB"
}
