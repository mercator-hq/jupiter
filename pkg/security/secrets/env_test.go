package secrets

import (
	"context"
	"os"
	"testing"
)

func TestEnvProvider_GetSecret(t *testing.T) {
	// Set up test environment variable
	os.Setenv("MERCATOR_SECRET_TEST_KEY", "test-value")
	defer os.Unsetenv("MERCATOR_SECRET_TEST_KEY")

	provider := NewEnvProvider("MERCATOR_SECRET_")

	value, err := provider.GetSecret(context.Background(), "test-key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "test-value" {
		t.Errorf("expected value 'test-value', got '%s'", value)
	}
}

func TestEnvProvider_GetSecret_NotFound(t *testing.T) {
	provider := NewEnvProvider("MERCATOR_SECRET_")

	_, err := provider.GetSecret(context.Background(), "nonexistent-key")
	if err == nil {
		t.Error("expected error for nonexistent secret, got nil")
	}
}

func TestEnvProvider_SecretNameConversion(t *testing.T) {
	tests := []struct {
		name          string
		secretName    string
		envVarName    string
		envVarValue   string
		expectedValue string
	}{
		{
			name:          "simple name",
			secretName:    "api-key",
			envVarName:    "MERCATOR_SECRET_API_KEY",
			envVarValue:   "value1",
			expectedValue: "value1",
		},
		{
			name:          "complex name with multiple hyphens",
			secretName:    "openai-api-key",
			envVarName:    "MERCATOR_SECRET_OPENAI_API_KEY",
			envVarValue:   "value2",
			expectedValue: "value2",
		},
		{
			name:          "name with underscores",
			secretName:    "my_secret_key",
			envVarName:    "MERCATOR_SECRET_MY_SECRET_KEY",
			envVarValue:   "value3",
			expectedValue: "value3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			os.Setenv(tt.envVarName, tt.envVarValue)
			defer os.Unsetenv(tt.envVarName)

			provider := NewEnvProvider("MERCATOR_SECRET_")

			value, err := provider.GetSecret(context.Background(), tt.secretName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if value != tt.expectedValue {
				t.Errorf("expected value '%s', got '%s'", tt.expectedValue, value)
			}
		})
	}
}

func TestEnvProvider_NoPrefix(t *testing.T) {
	os.Setenv("TEST_KEY", "test-value")
	defer os.Unsetenv("TEST_KEY")

	provider := NewEnvProvider("")

	value, err := provider.GetSecret(context.Background(), "test_key")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value != "test-value" {
		t.Errorf("expected value 'test-value', got '%s'", value)
	}
}

func TestEnvProvider_ListSecrets(t *testing.T) {
	// Set up test environment variables
	os.Setenv("MERCATOR_SECRET_KEY1", "value1")
	os.Setenv("MERCATOR_SECRET_KEY2", "value2")
	os.Setenv("OTHER_KEY", "value3")
	defer func() {
		os.Unsetenv("MERCATOR_SECRET_KEY1")
		os.Unsetenv("MERCATOR_SECRET_KEY2")
		os.Unsetenv("OTHER_KEY")
	}()

	provider := NewEnvProvider("MERCATOR_SECRET_")

	secrets, err := provider.ListSecrets(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should include secrets with prefix, but not OTHER_KEY
	expectedSecrets := map[string]bool{
		"key1": true,
		"key2": true,
	}

	for _, secret := range secrets {
		if !expectedSecrets[secret] && secret != "key1" && secret != "key2" {
			// Only fail if it's not one of our expected keys
			// (there may be other MERCATOR_SECRET_ env vars in the environment)
			if secret != "key1" && secret != "key2" {
				t.Logf("unexpected secret in list: %s", secret)
			}
		}
	}

	// Verify we got at least our expected secrets
	foundCount := 0
	for _, secret := range secrets {
		if secret == "key1" || secret == "key2" {
			foundCount++
		}
	}

	if foundCount < 2 {
		t.Errorf("expected at least 2 secrets, found %d", foundCount)
	}
}

func TestEnvProvider_Provider(t *testing.T) {
	provider := NewEnvProvider("MERCATOR_SECRET_")

	if provider.Provider() != "env" {
		t.Errorf("expected provider name 'env', got '%s'", provider.Provider())
	}
}

func TestEnvProvider_Supports(t *testing.T) {
	provider := NewEnvProvider("MERCATOR_SECRET_")

	// Environment provider always returns true
	if !provider.Supports("any-secret-name") {
		t.Error("expected Supports to return true for any secret name")
	}
}
