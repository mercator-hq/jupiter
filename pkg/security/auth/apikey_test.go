package auth

import (
	"testing"
	"time"
)

func TestNewAPIKeyValidator(t *testing.T) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-test-1",
			UserID:    "user-1",
			TeamID:    "team-1",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
		{
			Key:       "sk-test-2",
			UserID:    "user-2",
			TeamID:    "team-2",
			Enabled:   true,
			RateLimit: "100/hour",
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)

	if validator == nil {
		t.Fatal("NewAPIKeyValidator returned nil")
	}

	if len(validator.keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(validator.keys))
	}
}

func TestAPIKeyValidator_Validate(t *testing.T) {
	tests := []struct {
		name      string
		keys      []*APIKeyInfo
		testKey   string
		wantError bool
		wantUser  string
	}{
		{
			name: "valid enabled key",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-valid-key",
					UserID:    "user-123",
					TeamID:    "team-eng",
					Enabled:   true,
					RateLimit: "1000/hour",
					CreatedAt: time.Now(),
				},
			},
			testKey:   "sk-valid-key",
			wantError: false,
			wantUser:  "user-123",
		},
		{
			name: "disabled key",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-disabled-key",
					UserID:    "user-456",
					TeamID:    "team-sales",
					Enabled:   false,
					RateLimit: "100/hour",
					CreatedAt: time.Now(),
				},
			},
			testKey:   "sk-disabled-key",
			wantError: true,
			wantUser:  "",
		},
		{
			name: "invalid key",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-valid-key",
					UserID:    "user-123",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			testKey:   "sk-invalid-key",
			wantError: true,
			wantUser:  "",
		},
		{
			name:      "empty key",
			keys:      []*APIKeyInfo{},
			testKey:   "",
			wantError: true,
			wantUser:  "",
		},
		{
			name: "key not found in multiple keys",
			keys: []*APIKeyInfo{
				{
					Key:       "sk-key-1",
					UserID:    "user-1",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
				{
					Key:       "sk-key-2",
					UserID:    "user-2",
					Enabled:   true,
					CreatedAt: time.Now(),
				},
			},
			testKey:   "sk-key-3",
			wantError: true,
			wantUser:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := NewAPIKeyValidator(tt.keys)

			info, err := validator.Validate(tt.testKey)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				if info != nil {
					t.Error("Expected nil info on error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if info == nil {
					t.Fatal("Expected non-nil info")
				}
				if info.UserID != tt.wantUser {
					t.Errorf("Expected user %s, got %s", tt.wantUser, info.UserID)
				}
			}
		})
	}
}

func TestAPIKeyValidator_List(t *testing.T) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-test-1",
			UserID:    "user-1",
			Enabled:   true,
			CreatedAt: time.Now(),
		},
		{
			Key:       "sk-test-2",
			UserID:    "user-2",
			Enabled:   true,
			CreatedAt: time.Now(),
		},
		{
			Key:       "sk-test-3",
			UserID:    "user-3",
			Enabled:   false,
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)
	list := validator.List()

	if len(list) != 3 {
		t.Errorf("Expected 3 keys, got %d", len(list))
	}

	// Verify all keys are present
	keyMap := make(map[string]bool)
	for _, info := range list {
		keyMap[info.Key] = true
	}

	for _, key := range keys {
		if !keyMap[key.Key] {
			t.Errorf("Key %s not found in list", key.Key)
		}
	}
}

func TestAPIKeyValidator_Add(t *testing.T) {
	validator := NewAPIKeyValidator([]*APIKeyInfo{})

	newKey := &APIKeyInfo{
		Key:       "sk-new-key",
		UserID:    "user-new",
		TeamID:    "team-new",
		Enabled:   true,
		RateLimit: "500/hour",
		CreatedAt: time.Now(),
	}

	validator.Add(newKey)

	// Verify key was added
	info, err := validator.Validate("sk-new-key")
	if err != nil {
		t.Errorf("Failed to validate newly added key: %v", err)
	}
	if info.UserID != "user-new" {
		t.Errorf("Expected user user-new, got %s", info.UserID)
	}

	// Verify list includes new key
	list := validator.List()
	if len(list) != 1 {
		t.Errorf("Expected 1 key, got %d", len(list))
	}
}

func TestAPIKeyValidator_Remove(t *testing.T) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-test-1",
			UserID:    "user-1",
			Enabled:   true,
			CreatedAt: time.Now(),
		},
		{
			Key:       "sk-test-2",
			UserID:    "user-2",
			Enabled:   true,
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)

	// Remove first key
	validator.Remove("sk-test-1")

	// Verify key was removed
	_, err := validator.Validate("sk-test-1")
	if err == nil {
		t.Error("Expected error for removed key, got none")
	}

	// Verify second key still exists
	info, err := validator.Validate("sk-test-2")
	if err != nil {
		t.Errorf("Unexpected error for remaining key: %v", err)
	}
	if info.UserID != "user-2" {
		t.Errorf("Expected user user-2, got %s", info.UserID)
	}

	// Verify list count
	list := validator.List()
	if len(list) != 1 {
		t.Errorf("Expected 1 key after removal, got %d", len(list))
	}
}

func TestAPIKeyValidator_Update(t *testing.T) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-test-key",
			UserID:    "user-old",
			TeamID:    "team-old",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)

	// Update the key
	updatedKey := &APIKeyInfo{
		Key:       "sk-test-key",
		UserID:    "user-new",
		TeamID:    "team-new",
		Enabled:   false,
		RateLimit: "500/hour",
		CreatedAt: time.Now(),
	}

	err := validator.Update(updatedKey)
	if err != nil {
		t.Errorf("Failed to update key: %v", err)
	}

	// Verify key was updated (should fail because Enabled is false)
	_, err = validator.Validate("sk-test-key")
	if err == nil {
		t.Error("Expected error for disabled key, got none")
	}

	// Re-enable and verify changes
	updatedKey.Enabled = true
	validator.Update(updatedKey)

	info, err := validator.Validate("sk-test-key")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if info.UserID != "user-new" {
		t.Errorf("Expected user user-new, got %s", info.UserID)
	}
	if info.TeamID != "team-new" {
		t.Errorf("Expected team team-new, got %s", info.TeamID)
	}
	if info.RateLimit != "500/hour" {
		t.Errorf("Expected rate limit 500/hour, got %s", info.RateLimit)
	}
}

func TestAPIKeyValidator_UpdateNonExistent(t *testing.T) {
	validator := NewAPIKeyValidator([]*APIKeyInfo{})

	nonExistentKey := &APIKeyInfo{
		Key:       "sk-nonexistent",
		UserID:    "user-test",
		Enabled:   true,
		CreatedAt: time.Now(),
	}

	err := validator.Update(nonExistentKey)
	if err == nil {
		t.Error("Expected error when updating non-existent key, got none")
	}
}

func TestAPIKeyValidator_ConcurrentAccess(t *testing.T) {
	keys := []*APIKeyInfo{
		{
			Key:       "sk-test-key",
			UserID:    "user-test",
			Enabled:   true,
			CreatedAt: time.Now(),
		},
	}

	validator := NewAPIKeyValidator(keys)

	// Test concurrent reads
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := validator.Validate("sk-test-key")
			if err != nil {
				t.Errorf("Concurrent validation failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
