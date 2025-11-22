package auth

import "time"

// APIKeyInfo represents an API key with metadata
type APIKeyInfo struct {
	Key       string
	UserID    string
	TeamID    string
	Enabled   bool
	RateLimit string
	CreatedAt time.Time
}

// APIKeyStore stores and validates API keys
type APIKeyStore interface {
	Validate(key string) (*APIKeyInfo, error)
	List() []*APIKeyInfo
}
