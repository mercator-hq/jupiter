package auth

import (
	"fmt"
	"sync"
)

// APIKeyValidator validates API keys against a configured set of keys
type APIKeyValidator struct {
	mu   sync.RWMutex
	keys map[string]*APIKeyInfo
}

// NewAPIKeyValidator creates a new API key validator with the given keys
func NewAPIKeyValidator(keys []*APIKeyInfo) *APIKeyValidator {
	keyMap := make(map[string]*APIKeyInfo)
	for _, key := range keys {
		keyMap[key.Key] = key
	}

	return &APIKeyValidator{
		keys: keyMap,
	}
}

// Validate checks if the given API key is valid and returns its info
func (v *APIKeyValidator) Validate(key string) (*APIKeyInfo, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	info, ok := v.keys[key]
	if !ok {
		return nil, fmt.Errorf("invalid API key")
	}

	if !info.Enabled {
		return nil, fmt.Errorf("API key disabled")
	}

	return info, nil
}

// List returns all configured API keys
func (v *APIKeyValidator) List() []*APIKeyInfo {
	v.mu.RLock()
	defer v.mu.RUnlock()

	keys := make([]*APIKeyInfo, 0, len(v.keys))
	for _, key := range v.keys {
		keys = append(keys, key)
	}
	return keys
}

// Add adds a new API key to the validator
func (v *APIKeyValidator) Add(info *APIKeyInfo) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys[info.Key] = info
}

// Remove removes an API key from the validator
func (v *APIKeyValidator) Remove(key string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	delete(v.keys, key)
}

// Update updates an existing API key's information
func (v *APIKeyValidator) Update(info *APIKeyInfo) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if _, ok := v.keys[info.Key]; !ok {
		return fmt.Errorf("API key not found")
	}

	v.keys[info.Key] = info
	return nil
}
