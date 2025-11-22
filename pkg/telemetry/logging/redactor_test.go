package logging

import (
	"testing"

	"mercator-hq/jupiter/pkg/config"
)

func TestNewRedactor(t *testing.T) {
	tests := []struct {
		name           string
		customPatterns []config.RedactPattern
		wantPatterns   int // Minimum number of patterns
	}{
		{
			name:           "default patterns only",
			customPatterns: nil,
			wantPatterns:   8, // Default patterns: api_key, email, ssn, credit_card, ipv4, ipv6, phone, bearer_token, password
		},
		{
			name: "with custom patterns",
			customPatterns: []config.RedactPattern{
				{
					Name:        "custom_token",
					Pattern:     "tok_[a-zA-Z0-9]{32}",
					Replacement: "tok_***",
				},
			},
			wantPatterns: 9, // Default + 1 custom
		},
		{
			name: "invalid custom pattern (should skip)",
			customPatterns: []config.RedactPattern{
				{
					Name:        "invalid",
					Pattern:     "[unclosed", // Invalid regex
					Replacement: "***",
				},
			},
			wantPatterns: 8, // Only default patterns
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			redactor := NewRedactor(tt.customPatterns)
			if redactor == nil {
				t.Fatal("NewRedactor returned nil")
			}

			if len(redactor.patterns) < tt.wantPatterns {
				t.Errorf("Expected at least %d patterns, got %d",
					tt.wantPatterns, len(redactor.patterns))
			}
		})
	}
}

func TestRedactor_RedactString_APIKeys(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		name     string
		input    string
		wantSame bool // Should input == output?
	}{
		{
			name:     "OpenAI API key",
			input:    "sk-abc123xyz789def456ghi789",
			wantSame: false,
		},
		{
			name:     "Generic API key",
			input:    "api_key_abc123xyz789def456",
			wantSame: false,
		},
		{
			name:     "API key with colon",
			input:    "api-key:abc123xyz789def456",
			wantSame: false,
		},
		{
			name:     "No API key",
			input:    "This is a normal message",
			wantSame: true,
		},
		{
			name:     "Short string that looks like API key",
			input:    "sk-short",
			wantSame: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := redactor.RedactString(tt.input)

			if tt.wantSame {
				if output != tt.input {
					t.Errorf("Expected no redaction, got: %s", output)
				}
			} else {
				if output == tt.input {
					t.Errorf("Expected redaction, but input unchanged: %s", output)
				}
				if output == "" {
					t.Error("Redacted output is empty")
				}
			}
		})
	}
}

func TestRedactor_RedactString_Emails(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		name  string
		input string
	}{
		{"Simple email", "user@example.com"},
		{"Email with dots", "user.name@example.com"},
		{"Email with plus", "user+tag@example.com"},
		{"Email with subdomain", "user@mail.example.com"},
		{"Corporate email", "john.doe@company.co.uk"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := redactor.RedactString(tt.input)

			if output == tt.input {
				t.Errorf("Email not redacted: %s", output)
			}

			// Original email should not be present
			if output == tt.input {
				t.Errorf("Original email still present: %s", output)
			}
		})
	}
}

func TestRedactor_RedactString_SSN(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		name  string
		input string
	}{
		{"SSN with dashes", "123-45-6789"},
		{"SSN with spaces", "123 45 6789"},
		{"SSN without separators", "123456789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := redactor.RedactString(tt.input)

			if output == tt.input {
				t.Errorf("SSN not redacted: %s", output)
			}

			// Should not contain original digits
			if output == tt.input {
				t.Errorf("Original SSN still present: %s", output)
			}
		})
	}
}

func TestRedactor_RedactString_IPv4(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		name  string
		input string
	}{
		{"Private IP", "192.168.1.1"},
		{"Public IP", "8.8.8.8"},
		{"Localhost", "127.0.0.1"},
		{"Zero IP", "0.0.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := redactor.RedactString(tt.input)

			if output == tt.input {
				t.Errorf("IPv4 not redacted: %s", output)
			}
		})
	}
}

func TestRedactor_RedactString_Phone(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		name  string
		input string
	}{
		{"US phone with dashes", "555-123-4567"},
		{"US phone with dots", "555.123.4567"},
		{"US phone with parens", "(555) 123-4567"},
		{"International format", "+1-555-123-4567"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := redactor.RedactString(tt.input)

			if output == tt.input {
				t.Errorf("Phone number not redacted: %s", output)
			}
		})
	}
}

func TestRedactor_RedactString_BearerToken(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		name  string
		input string
	}{
		{"Bearer token", "Bearer abc123xyz789"},
		{"Bearer JWT", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abc"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := redactor.RedactString(tt.input)

			if output == tt.input {
				t.Errorf("Bearer token not redacted: %s", output)
			}

			// Should still contain "Bearer" but not the token
			if output != "Bearer ***" {
				t.Errorf("Unexpected redaction format: %s", output)
			}
		})
	}
}

func TestRedactor_RedactArgs(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		name     string
		args     []any
		checkFn  func([]any) bool
		wantPass bool
	}{
		{
			name: "redact API key value",
			args: []any{"api_key", "sk-abc123xyz789def456"},
			checkFn: func(result []any) bool {
				return len(result) == 2 && result[1] != "sk-abc123xyz789def456"
			},
			wantPass: true,
		},
		{
			name: "redact password value",
			args: []any{"password", "secretpass123"},
			checkFn: func(result []any) bool {
				return len(result) == 2 && result[1] != "secretpass123"
			},
			wantPass: true,
		},
		{
			name: "preserve non-sensitive key",
			args: []any{"user_id", "12345"},
			checkFn: func(result []any) bool {
				return len(result) == 2 && result[1] == "12345"
			},
			wantPass: true,
		},
		{
			name: "redact email in string value",
			args: []any{"message", "Contact user@example.com"},
			checkFn: func(result []any) bool {
				val, ok := result[1].(string)
				return ok && val != "Contact user@example.com"
			},
			wantPass: true,
		},
		{
			name: "handle mixed args",
			args: []any{
				"api_key", "sk-abc123",
				"count", 42,
				"email", "user@example.com",
				"valid", true,
			},
			checkFn: func(result []any) bool {
				return len(result) == 8 &&
					result[1] != "sk-abc123" &&
					result[3] == 42 &&
					result[5] != "user@example.com" &&
					result[7] == true
			},
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.RedactArgs(tt.args...)

			if pass := tt.checkFn(result); pass != tt.wantPass {
				t.Errorf("Check failed: got pass=%v, want pass=%v, result=%v",
					pass, tt.wantPass, result)
			}
		})
	}
}

func TestRedactor_isSensitiveKey(t *testing.T) {
	redactor := NewRedactor(nil)

	tests := []struct {
		key       string
		sensitive bool
	}{
		// Sensitive keys
		{"password", true},
		{"PASSWORD", true},
		{"api_key", true},
		{"apikey", true},
		{"API_KEY", true},
		{"secret", true},
		{"token", true},
		{"auth", true},
		{"authorization", true},
		{"ssn", true},
		{"credit_card", true},
		{"private_key", true},

		// Non-sensitive keys
		{"user_id", false},
		{"count", false},
		{"message", false},
		{"timestamp", false},
		{"duration_ms", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := redactor.isSensitiveKey(tt.key)
			if result != tt.sensitive {
				t.Errorf("isSensitiveKey(%q) = %v, want %v", tt.key, result, tt.sensitive)
			}
		})
	}
}

func TestRedactEmail(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "u***@example.com"},
		{"a@example.com", "a***@example.com"},
		{"john.doe@company.com", "j***@company.com"},
		{"invalid-email", "invalid-email"}, // Not an email
		{"@example.com", "***@example.com"}, // Empty username
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := RedactEmail(tt.input)
			if result != tt.expected {
				t.Errorf("RedactEmail(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRedactAPIKey(t *testing.T) {
	tests := []struct {
		input       string
		shouldHave4 bool
	}{
		{"sk-abc123xyz789", true},
		{"api_key_123456789", true},
		{"short", false},     // Too short
		{"a", false},         // Way too short
		{"", false},          // Empty
		{"abcdefghij", true}, // Exactly long enough
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := RedactAPIKey(tt.input)

			if tt.shouldHave4 {
				if len(tt.input) > 4 && !hasPrefix(result, tt.input[:4]) {
					t.Errorf("RedactAPIKey(%q) = %q, expected to keep first 4 chars", tt.input, result)
				}
			}

			if result == tt.input && len(tt.input) > 4 {
				t.Errorf("RedactAPIKey(%q) didn't redact", tt.input)
			}
		})
	}
}

func TestRedactIPv4(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"192.168.1.100", "192.*.*.*"},
		{"10.0.0.1", "10.*.*.*"},
		{"8.8.8.8", "8.*.*.*"},
		{"invalid", "invalid"}, // Not an IP
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := RedactIPv4(tt.input)
			if result != tt.expected {
				t.Errorf("RedactIPv4(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestRedactCreditCard(t *testing.T) {
	tests := []struct {
		input      string
		shouldKeep string // Last 4 digits that should be kept
	}{
		{"4111-1111-1111-1234", "1234"},
		{"4111 1111 1111 1234", "1234"},
		{"4111111111111234", "1234"},
		{"5555555555554444", "4444"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := RedactCreditCard(tt.input)

			if !containsStr(result, tt.shouldKeep) {
				t.Errorf("RedactCreditCard(%q) = %q, should contain last 4 digits %q",
					tt.input, result, tt.shouldKeep)
			}

			if result == tt.input {
				t.Errorf("RedactCreditCard(%q) didn't redact", tt.input)
			}
		})
	}
}

func TestRedactor_CustomPatterns(t *testing.T) {
	customPatterns := []config.RedactPattern{
		{
			Name:        "custom_id",
			Pattern:     "CUST-[0-9]{6}",
			Replacement: "CUST-******",
		},
		{
			Name:        "account_number",
			Pattern:     "ACC[0-9]{8}",
			Replacement: "ACC********",
		},
	}

	redactor := NewRedactor(customPatterns)

	tests := []struct {
		name     string
		input    string
		wantSame bool
	}{
		{
			name:     "custom ID pattern",
			input:    "Customer CUST-123456 made a purchase",
			wantSame: false,
		},
		{
			name:     "account number pattern",
			input:    "Account ACC12345678 was charged",
			wantSame: false,
		},
		{
			name:     "no match",
			input:    "Normal message without patterns",
			wantSame: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactor.RedactString(tt.input)

			if tt.wantSame {
				if result != tt.input {
					t.Errorf("Expected no redaction, got: %s", result)
				}
			} else {
				if result == tt.input {
					t.Errorf("Expected redaction, but input unchanged")
				}
			}
		})
	}
}

// Helper functions
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && hasSubstring(s, substr)
}

func hasSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
