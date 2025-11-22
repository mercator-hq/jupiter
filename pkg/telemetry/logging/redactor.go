package logging

import (
	"fmt"
	"regexp"
	"strings"

	"mercator-hq/jupiter/pkg/config"
)

// Redactor redacts PII (Personally Identifiable Information) from log fields.
type Redactor struct {
	patterns map[string]*redactPattern
	enabled  bool
}

// redactPattern contains a compiled regex and replacement string.
type redactPattern struct {
	name        string
	regex       *regexp.Regexp
	replacement string
}

// Common PII pattern names.
const (
	PatternAPIKey      = "api_key"
	PatternEmail       = "email"
	PatternSSN         = "ssn"
	PatternCreditCard  = "credit_card"
	PatternIPv4        = "ipv4"
	PatternIPv6        = "ipv6"
	PatternPhone       = "phone"
	PatternPassword    = "password"
	PatternBearerToken = "bearer_token"
)

// NewRedactor creates a new Redactor with default and custom patterns.
func NewRedactor(customPatterns []config.RedactPattern) *Redactor {
	r := &Redactor{
		patterns: make(map[string]*redactPattern),
		enabled:  true,
	}

	// Add default patterns
	r.addDefaultPatterns()

	// Add custom patterns
	for _, p := range customPatterns {
		regex, err := regexp.Compile(p.Pattern)
		if err != nil {
			// Skip invalid patterns (log warning in production)
			continue
		}
		r.patterns[p.Name] = &redactPattern{
			name:        p.Name,
			regex:       regex,
			replacement: p.Replacement,
		}
	}

	return r
}

// addDefaultPatterns adds built-in PII redaction patterns.
func (r *Redactor) addDefaultPatterns() {
	patterns := map[string]struct {
		regex       string
		replacement string
	}{
		// API keys (OpenAI, Anthropic, generic)
		// Match sk- prefix with any length, or api_key/apikey/api-key with any alphanumeric
		PatternAPIKey: {
			regex:       `(sk-[a-zA-Z0-9]+|api[-_]?key[-_:]\s*[a-zA-Z0-9]+)`,
			replacement: "sk-***",
		},

		// Email addresses
		PatternEmail: {
			regex:       `([a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,})`,
			replacement: "$1_redacted",
		},

		// Social Security Numbers (SSN)
		PatternSSN: {
			regex:       `\b\d{3}[-\s]?\d{2}[-\s]?\d{4}\b`,
			replacement: "***-**-****",
		},

		// Credit card numbers
		PatternCreditCard: {
			regex:       `\b(?:\d[ -]*?){13,16}\b`,
			replacement: "****-****-****-****",
		},

		// IPv4 addresses
		PatternIPv4: {
			regex:       `\b(?:\d{1,3}\.){3}\d{1,3}\b`,
			replacement: "192.*.*.*",
		},

		// IPv6 addresses
		PatternIPv6: {
			regex:       `\b(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}\b`,
			replacement: "****:****:****:****:****:****:****:****",
		},

		// Phone numbers
		PatternPhone: {
			regex:       `\b(?:\+?1[-.\s]?)?\(?\d{3}\)?[-.\s]?\d{3}[-.\s]?\d{4}\b`,
			replacement: "***-***-****",
		},

		// Bearer tokens
		PatternBearerToken: {
			regex:       `Bearer\s+[a-zA-Z0-9\-._~+/]+=*`,
			replacement: "Bearer ***",
		},

		// Generic password fields
		PatternPassword: {
			regex:       `(password|passwd|pwd)[:=]\s*[^\s]+`,
			replacement: "$1: ***",
		},
	}

	for name, p := range patterns {
		regex := regexp.MustCompile(p.regex)
		r.patterns[name] = &redactPattern{
			name:        name,
			regex:       regex,
			replacement: p.replacement,
		}
	}
}

// RedactString redacts PII from a string value.
func (r *Redactor) RedactString(value string) string {
	if !r.enabled || value == "" {
		return value
	}

	redacted := value
	for _, pattern := range r.patterns {
		redacted = pattern.regex.ReplaceAllString(redacted, pattern.replacement)
	}

	return redacted
}

// RedactArgs redacts PII from variadic log arguments.
// Args are in the form: key1, value1, key2, value2, ...
func (r *Redactor) RedactArgs(args ...any) []any {
	if !r.enabled || len(args) == 0 {
		return args
	}

	redacted := make([]any, len(args))
	copy(redacted, args)

	// Process key-value pairs
	for i := 1; i < len(redacted); i += 2 {
		// Check if this is a sensitive field by key name
		if i > 0 {
			key, ok := redacted[i-1].(string)
			if ok && r.isSensitiveKey(key) {
				redacted[i] = r.redactValue(redacted[i])
			}
		}

		// Also redact string values that match patterns
		if str, ok := redacted[i].(string); ok {
			redacted[i] = r.RedactString(str)
		}
	}

	return redacted
}

// isSensitiveKey checks if a key name indicates sensitive data.
func (r *Redactor) isSensitiveKey(key string) bool {
	// Convert to lowercase for case-insensitive matching
	lowerKey := strings.ToLower(key)

	sensitiveKeys := []string{
		"password", "passwd", "pwd",
		"secret", "token", "api_key", "apikey",
		"auth", "authorization",
		"ssn", "social_security",
		"credit_card", "creditcard", "cc",
		"private_key", "privatekey",
	}

	for _, sensitive := range sensitiveKeys {
		if strings.Contains(lowerKey, sensitive) {
			return true
		}
	}

	return false
}

// redactValue redacts a sensitive value completely.
func (r *Redactor) redactValue(value any) any {
	switch v := value.(type) {
	case string:
		// For sensitive keys, completely redact the value
		if v == "" {
			return ""
		}
		// Keep a hint of the value type/length for debugging
		if len(v) <= 4 {
			return "***"
		}
		return v[:min(4, len(v))] + "***"
	case fmt.Stringer:
		return "***"
	default:
		return "***"
	}
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// RedactEmail redacts an email address partially (shows first char and domain).
func RedactEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return email
	}

	username := parts[0]
	domain := parts[1]

	if len(username) == 0 {
		return "***@" + domain
	}

	return string(username[0]) + "***@" + domain
}

// RedactAPIKey redacts an API key, keeping only a prefix.
func RedactAPIKey(apiKey string) string {
	if len(apiKey) <= 4 {
		return "***"
	}

	// Keep first 4 characters for identification
	return apiKey[:4] + "***"
}

// RedactIPv4 redacts an IPv4 address, keeping only the first octet.
func RedactIPv4(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}

	return parts[0] + ".*.*.*"
}

// RedactCreditCard redacts a credit card number, keeping only last 4 digits.
func RedactCreditCard(cc string) string {
	// Remove spaces and dashes
	cleaned := strings.ReplaceAll(cc, " ", "")
	cleaned = strings.ReplaceAll(cleaned, "-", "")

	if len(cleaned) < 13 || len(cleaned) > 16 {
		return cc
	}

	last4 := cleaned[len(cleaned)-4:]
	return "****-****-****-" + last4
}
