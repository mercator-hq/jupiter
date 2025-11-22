package tls

import (
	"crypto/tls"
	"testing"
	"time"
)

func TestConfig_ToTLSConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid TLS 1.3 config",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "testdata/server-key.pem",
				MinVersion: "1.3",
			},
			expectError: false,
		},
		{
			name: "valid TLS 1.2 config",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "testdata/server-key.pem",
				MinVersion: "1.2",
			},
			expectError: false,
		},
		{
			name: "TLS disabled",
			config: Config{
				Enabled: false,
			},
			expectError: false,
		},
		{
			name: "missing cert file",
			config: Config{
				Enabled:  true,
				CertFile: "",
				KeyFile:  "testdata/server-key.pem",
			},
			expectError: true,
			errorMsg:    "cert_file is required",
		},
		{
			name: "missing key file",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "",
			},
			expectError: true,
			errorMsg:    "key_file is required",
		},
		{
			name: "cert file not found",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/nonexistent.pem",
				KeyFile:  "testdata/server-key.pem",
			},
			expectError: true,
			errorMsg:    "certificate file not found",
		},
		{
			name: "key file not found",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "testdata/nonexistent.pem",
			},
			expectError: true,
			errorMsg:    "key file not found",
		},
		{
			name: "with cipher suites",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "testdata/server-key.pem",
				MinVersion: "1.3",
				CipherSuites: []string{
					"TLS_AES_128_GCM_SHA256",
					"TLS_AES_256_GCM_SHA384",
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := tt.config.ToTLSConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !tt.config.Enabled && tlsConfig != nil {
				t.Errorf("expected nil config when TLS disabled, got %v", tlsConfig)
				return
			}

			if tt.config.Enabled {
				if tlsConfig == nil {
					t.Errorf("expected non-nil TLS config")
					return
				}

				// Verify certificates loaded
				if len(tlsConfig.Certificates) == 0 {
					t.Errorf("expected certificates to be loaded")
				}

				// Verify TLS version
				expectedVersion := tt.config.parseTLSVersion()
				if tlsConfig.MinVersion != expectedVersion {
					t.Errorf("expected MinVersion %d, got %d", expectedVersion, tlsConfig.MinVersion)
				}

				// Verify cipher suites if specified
				if len(tt.config.CipherSuites) > 0 {
					if len(tlsConfig.CipherSuites) == 0 {
						t.Errorf("expected cipher suites to be set")
					}
				}
			}
		})
	}
}

func TestConfig_parseTLSVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected uint16
	}{
		{
			name:     "TLS 1.3",
			version:  "1.3",
			expected: tls.VersionTLS13,
		},
		{
			name:     "TLS 1.2",
			version:  "1.2",
			expected: tls.VersionTLS12,
		},
		{
			name:     "empty defaults to 1.3",
			version:  "",
			expected: tls.VersionTLS13,
		},
		{
			name:     "unknown defaults to 1.3",
			version:  "1.1",
			expected: tls.VersionTLS13,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{MinVersion: tt.version}
			result := config.parseTLSVersion()
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestConfig_parseCipherSuites(t *testing.T) {
	tests := []struct {
		name     string
		suites   []string
		expected int
	}{
		{
			name:     "empty returns nil",
			suites:   []string{},
			expected: 0,
		},
		{
			name: "TLS 1.3 suites",
			suites: []string{
				"TLS_AES_128_GCM_SHA256",
				"TLS_AES_256_GCM_SHA384",
			},
			expected: 2,
		},
		{
			name: "TLS 1.2 suites",
			suites: []string{
				"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
				"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			},
			expected: 2,
		},
		{
			name: "unknown suites ignored",
			suites: []string{
				"TLS_AES_128_GCM_SHA256",
				"UNKNOWN_SUITE",
			},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{CipherSuites: tt.suites}
			result := config.parseCipherSuites()

			if tt.expected == 0 && result != nil {
				t.Errorf("expected nil, got %v", result)
			}

			if tt.expected > 0 && len(result) != tt.expected {
				t.Errorf("expected %d cipher suites, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestConfig_ParseReloadInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval string
		expected time.Duration
	}{
		{
			name:     "empty defaults to 5m",
			interval: "",
			expected: 5 * time.Minute,
		},
		{
			name:     "valid duration",
			interval: "10m",
			expected: 10 * time.Minute,
		},
		{
			name:     "invalid defaults to 5m",
			interval: "invalid",
			expected: 5 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{ReloadInterval: tt.interval}
			result := config.ParseReloadInterval()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConfig_MTLSConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid mTLS config",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "testdata/server-key.pem",
				MTLS: MTLSConfig{
					Enabled:          true,
					ClientCAFile:     "testdata/ca-cert.pem",
					ClientAuthType:   "require",
					VerifyClientCert: true,
					IdentitySource:   "subject.CN",
				},
			},
			expectError: false,
		},
		{
			name: "mTLS without CA file",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "testdata/server-key.pem",
				MTLS: MTLSConfig{
					Enabled:      true,
					ClientCAFile: "",
				},
			},
			expectError: true,
			errorMsg:    "client_ca_file is required",
		},
		{
			name: "mTLS with invalid CA file",
			config: Config{
				Enabled:  true,
				CertFile: "testdata/server-cert.pem",
				KeyFile:  "testdata/server-key.pem",
				MTLS: MTLSConfig{
					Enabled:      true,
					ClientCAFile: "testdata/nonexistent.pem",
				},
			},
			expectError: true,
			errorMsg:    "failed to read client CA",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tlsConfig, err := tt.config.ToTLSConfig()

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Verify mTLS configuration
			if tt.config.MTLS.Enabled {
				if tlsConfig.ClientCAs == nil {
					t.Errorf("expected ClientCAs to be set")
				}

				expectedAuthType := tt.config.parseClientAuthType()
				if tlsConfig.ClientAuth != expectedAuthType {
					t.Errorf("expected ClientAuth %v, got %v", expectedAuthType, tlsConfig.ClientAuth)
				}
			}
		})
	}
}

func TestConfig_parseClientAuthType(t *testing.T) {
	tests := []struct {
		name       string
		authType   string
		expected   tls.ClientAuthType
	}{
		{
			name:     "require",
			authType: "require",
			expected: tls.RequireAndVerifyClientCert,
		},
		{
			name:     "request",
			authType: "request",
			expected: tls.RequestClientCert,
		},
		{
			name:     "verify_if_given",
			authType: "verify_if_given",
			expected: tls.VerifyClientCertIfGiven,
		},
		{
			name:     "empty defaults to require",
			authType: "",
			expected: tls.RequireAndVerifyClientCert,
		},
		{
			name:     "unknown defaults to require",
			authType: "unknown",
			expected: tls.RequireAndVerifyClientCert,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := Config{
				MTLS: MTLSConfig{
					ClientAuthType: tt.authType,
				},
			}
			result := config.parseClientAuthType()
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		len(s) > len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
