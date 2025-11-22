package recorder

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

func TestRedactAPIKey(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "empty key",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "standard OpenAI key",
			apiKey:   "sk-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "sha256:" + hashString("sk-1234567890abcdefghijklmnopqrstuvwxyz"),
		},
		{
			name:     "short key",
			apiKey:   "short",
			expected: "sha256:" + hashString("short"),
		},
		{
			name:     "long key",
			apiKey:   strings.Repeat("x", 100),
			expected: "sha256:" + hashString(strings.Repeat("x", 100)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactAPIKey(tt.apiKey)
			if result != tt.expected {
				t.Errorf("RedactAPIKey() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRedactAPIKey_Irreversible(t *testing.T) {
	// Verify that redacted API keys cannot be reversed
	apiKey := "sk-1234567890abcdefghijklmnopqrstuvwxyz"
	redacted := RedactAPIKey(apiKey)

	// Redacted string should not contain original key
	if strings.Contains(redacted, apiKey) {
		t.Errorf("RedactAPIKey() contains original key: %v", redacted)
	}

	// Should start with sha256 prefix
	if !strings.HasPrefix(redacted, "sha256:") {
		t.Errorf("RedactAPIKey() missing sha256 prefix: %v", redacted)
	}
}

func TestRedactAPIKey_Deterministic(t *testing.T) {
	// Same API key should always produce same redacted value
	apiKey := "sk-test123456789"

	result1 := RedactAPIKey(apiKey)
	result2 := RedactAPIKey(apiKey)
	result3 := RedactAPIKey(apiKey)

	if result1 != result2 || result2 != result3 {
		t.Errorf("RedactAPIKey() not deterministic: %v, %v, %v", result1, result2, result3)
	}
}

func TestRedactAPIKey_Unique(t *testing.T) {
	// Different API keys should produce different redacted values
	key1 := RedactAPIKey("sk-key1")
	key2 := RedactAPIKey("sk-key2")
	key3 := RedactAPIKey("sk-key3")

	if key1 == key2 || key2 == key3 || key1 == key3 {
		t.Errorf("RedactAPIKey() not unique: %v, %v, %v", key1, key2, key3)
	}
}

func TestRedactAPIKeyTruncated(t *testing.T) {
	tests := []struct {
		name     string
		apiKey   string
		expected string
	}{
		{
			name:     "empty key",
			apiKey:   "",
			expected: "",
		},
		{
			name:     "short key",
			apiKey:   "sk-12",
			expected: "****",
		},
		{
			name:     "11 char key",
			apiKey:   "sk-12345678",
			expected: "****",
		},
		{
			name:     "12 char key (boundary)",
			apiKey:   "sk-123456789",
			expected: "sk-1***6789",
		},
		{
			name:     "standard OpenAI key",
			apiKey:   "sk-1234567890abcdefghijklmnopqr",
			expected: "sk-1***opqr",
		},
		{
			name:     "long key",
			apiKey:   "sk-proj-1234567890abcdefghijklmnopqrstuvwxyz",
			expected: "sk-p***wxyz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RedactAPIKeyTruncated(tt.apiKey)
			if result != tt.expected {
				t.Errorf("RedactAPIKeyTruncated() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestRedactAPIKeyTruncated_PartialVisibility(t *testing.T) {
	// Verify that truncated keys show first 4 and last 4 characters
	apiKey := "sk-abcdefghijklmnopqrstuvwxyz"
	result := RedactAPIKeyTruncated(apiKey)

	// Should show first 4 characters
	if !strings.HasPrefix(result, "sk-a") {
		t.Errorf("RedactAPIKeyTruncated() missing first 4 chars: %v", result)
	}

	// Should show last 4 characters
	if !strings.HasSuffix(result, "wxyz") {
		t.Errorf("RedactAPIKeyTruncated() missing last 4 chars: %v", result)
	}

	// Should contain redaction marker
	if !strings.Contains(result, "***") {
		t.Errorf("RedactAPIKeyTruncated() missing redaction marker: %v", result)
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "helloworld",
			maxLen:   10,
			expected: "helloworld",
		},
		{
			name:     "needs truncation",
			input:    "hello world this is a long string",
			maxLen:   20,
			expected: "hello world this ...",
		},
		{
			name:     "very short maxLen",
			input:    "hello",
			maxLen:   3,
			expected: "hel",
		},
		{
			name:     "maxLen = 1",
			input:    "hello",
			maxLen:   1,
			expected: "h",
		},
		{
			name:     "unicode string",
			input:    "Hello 世界 🌍 " + strings.Repeat("x", 100),
			maxLen:   20,
			expected: "Hello 世界 🌍...",
		},
		{
			name:     "truncate to 500 chars (default)",
			input:    strings.Repeat("a", 600),
			maxLen:   500,
			expected: strings.Repeat("a", 497) + "...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := TruncateString(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("TruncateString() = %v (len=%d), want %v (len=%d)",
					result, len(result), tt.expected, len(tt.expected))
			}

			// Verify result never exceeds maxLen
			if len(result) > tt.maxLen {
				t.Errorf("TruncateString() result length %d exceeds maxLen %d", len(result), tt.maxLen)
			}
		})
	}
}

func TestTruncateString_PreservesEllipsis(t *testing.T) {
	// When truncating, should add ellipsis
	input := strings.Repeat("a", 100)
	result := TruncateString(input, 50)

	if !strings.HasSuffix(result, "...") {
		t.Errorf("TruncateString() missing ellipsis: %v", result)
	}

	// Length should be exactly maxLen
	if len(result) != 50 {
		t.Errorf("TruncateString() length = %d, want 50", len(result))
	}
}

func TestTruncateString_NoTruncationNeeded(t *testing.T) {
	// When string is shorter than maxLen, should return unchanged
	input := "short string"
	result := TruncateString(input, 100)

	if result != input {
		t.Errorf("TruncateString() = %v, want %v (unchanged)", result, input)
	}

	// Should not add ellipsis
	if strings.HasSuffix(result, "...") {
		t.Errorf("TruncateString() added unnecessary ellipsis: %v", result)
	}
}

// BenchmarkRedactAPIKey benchmarks API key redaction
func BenchmarkRedactAPIKey(b *testing.B) {
	apiKey := "sk-1234567890abcdefghijklmnopqrstuvwxyz"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = RedactAPIKey(apiKey)
	}
}

// BenchmarkRedactAPIKeyTruncated benchmarks truncated redaction
func BenchmarkRedactAPIKeyTruncated(b *testing.B) {
	apiKey := "sk-1234567890abcdefghijklmnopqrstuvwxyz"
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = RedactAPIKeyTruncated(apiKey)
	}
}

// BenchmarkTruncateString benchmarks string truncation
func BenchmarkTruncateString(b *testing.B) {
	sizes := []int{100, 500, 1000, 5000}

	for _, size := range sizes {
		input := strings.Repeat("a", size)
		b.Run(formatBenchSize(size), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = TruncateString(input, 500)
			}
		})
	}
}

// Helper function to hash a string for testing
func hashString(s string) string {
	hash := sha256.Sum256([]byte(s))
	return hex.EncodeToString(hash[:])
}

// Helper function to format benchmark size
func formatBenchSize(size int) string {
	if size < 1000 {
		return string(rune(size)) + "chars"
	}
	return string(rune(size/1000)) + "kchars"
}
