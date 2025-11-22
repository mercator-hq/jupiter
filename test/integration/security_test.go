//go:build integration

package integration

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	securityAuth "mercator-hq/jupiter/pkg/security/auth"
	"mercator-hq/jupiter/pkg/security/secrets"
	securityTLS "mercator-hq/jupiter/pkg/security/tls"
)

// TestTLSServerIntegration tests end-to-end TLS server functionality.
func TestTLSServerIntegration(t *testing.T) {
	// Load test certificates
	certFile := "testdata/certs/server-cert.pem"
	keyFile := "testdata/certs/server-key.pem"

	// Create TLS config
	tlsConfig := &securityTLS.Config{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "1.3",
	}

	goTLSConfig, err := tlsConfig.ToTLSConfig()
	if err != nil {
		t.Fatalf("failed to create TLS config: %v", err)
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Hello, TLS!"))
	})

	// Create HTTPS server
	server := httptest.NewUnstartedServer(handler)
	server.TLS = goTLSConfig
	server.StartTLS()
	defer server.Close()

	// Create client with InsecureSkipVerify for test certificates
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}

	// Make request
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("HTTPS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "Hello, TLS!" {
		t.Errorf("unexpected response body: %q", body)
	}

	// Verify TLS was used
	if resp.TLS == nil {
		t.Error("response.TLS is nil, TLS not used")
	} else {
		if resp.TLS.Version < tls.VersionTLS13 {
			t.Errorf("TLS version too low: 0x%x", resp.TLS.Version)
		}
	}
}

// TestMTLSIntegration tests end-to-end mTLS (mutual TLS) functionality.
func TestMTLSIntegration(t *testing.T) {
	certFile := "testdata/certs/server-cert.pem"
	keyFile := "testdata/certs/server-key.pem"
	clientCAFile := "testdata/certs/ca-cert.pem"
	clientCertFile := "testdata/certs/client-cert.pem"
	clientKeyFile := "testdata/certs/client-key.pem"

	// Create server with mTLS
	tlsConfig := &securityTLS.Config{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "1.3",
		MTLS: securityTLS.MTLSConfig{
			Enabled:          true,
			ClientCAFile:     clientCAFile,
			ClientAuthType:   "require",
			VerifyClientCert: true,
			IdentitySource:   "subject.CN",
		},
	}

	goTLSConfig, err := tlsConfig.ToTLSConfig()
	if err != nil {
		t.Fatalf("failed to create TLS config: %v", err)
	}

	// Create handler that extracts client identity
	var clientIdentity string
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIdentity = securityTLS.GetClientIdentity(r, "subject.CN")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated: " + clientIdentity))
	})

	server := httptest.NewUnstartedServer(handler)
	server.TLS = goTLSConfig
	server.StartTLS()
	defer server.Close()

	// Load client certificate
	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		t.Fatalf("failed to load client certificate: %v", err)
	}

	// Create client with client certificate
	caCert, _ := os.ReadFile(clientCAFile)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				Certificates:       []tls.Certificate{clientCert},
				RootCAs:            caCertPool,
				InsecureSkipVerify: true, // For test server
			},
		},
	}

	// Make request
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("mTLS request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if clientIdentity == "" {
		t.Error("client identity not extracted")
	}

	t.Logf("Client identity extracted: %s", clientIdentity)
	t.Logf("Response body: %s", string(body))
}

// TestSecretManagementIntegration tests end-to-end secret management.
func TestSecretManagementIntegration(t *testing.T) {
	// Create temporary secrets directory
	tmpDir := t.TempDir()

	// Create secret files
	secretsData := map[string]string{
		"api-key-1": "sk-test-key-1",
		"api-key-2": "sk-test-key-2",
		"password":  "supersecret",
	}

	for name, value := range secretsData {
		path := filepath.Join(tmpDir, name)
		err := os.WriteFile(path, []byte(value), 0600)
		if err != nil {
			t.Fatalf("failed to create secret file: %v", err)
		}
	}

	// Create file provider
	fileProvider, err := secrets.NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create file provider: %v", err)
	}
	defer fileProvider.Close()

	// Create env provider
	os.Setenv("MERCATOR_SECRET_ENV_KEY", "env-secret-value")
	envProvider := secrets.NewEnvProvider("MERCATOR_SECRET_")

	// Create manager with both providers
	cacheConfig := secrets.CacheConfig{
		Enabled: true,
		TTL:     5 * time.Minute,
		MaxSize: 100,
	}

	manager := secrets.NewManager(
		[]secrets.SecretProvider{fileProvider, envProvider},
		cacheConfig,
	)

	// Test file-based secret
	ctx := context.Background()
	value, err := manager.GetSecret(ctx, "api-key-1")
	if err != nil {
		t.Fatalf("failed to get file secret: %v", err)
	}
	if value != "sk-test-key-1" {
		t.Errorf("wrong value: got %q, want %q", value, "sk-test-key-1")
	}

	// Test env secret
	value, err = manager.GetSecret(ctx, "env-key")
	if err != nil {
		t.Fatalf("failed to get env secret: %v", err)
	}
	if value != "env-secret-value" {
		t.Errorf("wrong value: got %q, want %q", value, "env-secret-value")
	}

	// Test secret reference resolution
	input := "api_key: ${secret:api-key-1}, password: ${secret:password}"
	resolved, err := manager.ResolveReferences(ctx, input)
	if err != nil {
		t.Fatalf("failed to resolve references: %v", err)
	}

	expected := "api_key: sk-test-key-1, password: supersecret"
	if resolved != expected {
		t.Errorf("resolution failed:\ngot:  %q\nwant: %q", resolved, expected)
	}

	// Test cache hit (should be faster)
	start := time.Now()
	_, err = manager.GetSecret(ctx, "api-key-1")
	duration := time.Since(start)
	if err != nil {
		t.Fatalf("cached get failed: %v", err)
	}
	if duration > 10*time.Millisecond {
		t.Errorf("cache hit too slow: %v", duration)
	}

	t.Logf("Secret management integration test passed")
}

// TestAPIKeyAuthenticationIntegration tests end-to-end API key authentication.
func TestAPIKeyAuthenticationIntegration(t *testing.T) {
	// Create validator with test keys
	validator := securityAuth.NewAPIKeyValidator([]*securityAuth.APIKeyInfo{
		{
			Key:       "sk-valid-key-123",
			UserID:    "user-1",
			TeamID:    "team-eng",
			Enabled:   true,
			RateLimit: "1000/hour",
			CreatedAt: time.Now(),
		},
		{
			Key:       "sk-disabled-key",
			UserID:    "user-2",
			TeamID:    "team-sales",
			Enabled:   false,
			RateLimit: "100/hour",
			CreatedAt: time.Now(),
		},
	})

	// Configure sources
	sources := []securityAuth.APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
		{Type: "header", Name: "X-API-Key", Scheme: ""},
	}

	middleware := securityAuth.NewAPIKeyMiddleware(validator, sources)

	// Create test handler
	var capturedKeyInfo *securityAuth.APIKeyInfo
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyInfo, ok := securityAuth.GetAPIKeyInfo(r.Context())
		if !ok {
			t.Error("key info not in context")
		}
		capturedKeyInfo = keyInfo
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Authenticated"))
	})

	// Wrap with middleware
	server := httptest.NewServer(middleware.Handle(handler))
	defer server.Close()

	tests := []struct {
		name         string
		header       string
		headerValue  string
		expectStatus int
		expectUserID string
	}{
		{
			name:         "valid bearer token",
			header:       "Authorization",
			headerValue:  "Bearer sk-valid-key-123",
			expectStatus: http.StatusOK,
			expectUserID: "user-1",
		},
		{
			name:         "valid custom header",
			header:       "X-API-Key",
			headerValue:  "sk-valid-key-123",
			expectStatus: http.StatusOK,
			expectUserID: "user-1",
		},
		{
			name:         "disabled key",
			header:       "Authorization",
			headerValue:  "Bearer sk-disabled-key",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "invalid key",
			header:       "Authorization",
			headerValue:  "Bearer sk-invalid-key",
			expectStatus: http.StatusUnauthorized,
		},
		{
			name:         "missing key",
			header:       "",
			headerValue:  "",
			expectStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capturedKeyInfo = nil

			req, _ := http.NewRequest("GET", server.URL, nil)
			if tt.header != "" {
				req.Header.Set(tt.header, tt.headerValue)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectStatus {
				t.Errorf("status = %d, want %d", resp.StatusCode, tt.expectStatus)
			}

			if tt.expectStatus == http.StatusOK {
				if capturedKeyInfo == nil {
					t.Error("key info not captured")
				} else if capturedKeyInfo.UserID != tt.expectUserID {
					t.Errorf("userID = %q, want %q", capturedKeyInfo.UserID, tt.expectUserID)
				}
			}
		})
	}
}

// TestFullSecurityStackIntegration tests the complete security stack: TLS + Auth + Secrets.
func TestFullSecurityStackIntegration(t *testing.T) {
	// 1. Set up secrets
	tmpDir := t.TempDir()
	apiKeyFile := filepath.Join(tmpDir, "api-key")
	err := os.WriteFile(apiKeyFile, []byte("sk-secret-key-789"), 0600)
	if err != nil {
		t.Fatalf("failed to create secret file: %v", err)
	}

	fileProvider, err := secrets.NewFileProvider(tmpDir, false)
	if err != nil {
		t.Fatalf("failed to create file provider: %v", err)
	}
	defer fileProvider.Close()

	manager := secrets.NewManager(
		[]secrets.SecretProvider{fileProvider},
		secrets.CacheConfig{Enabled: true, TTL: 5 * time.Minute, MaxSize: 100},
	)

	// 2. Load API key from secrets
	ctx := context.Background()
	apiKey, err := manager.GetSecret(ctx, "api-key")
	if err != nil {
		t.Fatalf("failed to load API key: %v", err)
	}

	// 3. Set up API key auth
	validator := securityAuth.NewAPIKeyValidator([]*securityAuth.APIKeyInfo{
		{Key: apiKey, UserID: "user-1", TeamID: "team-eng", Enabled: true},
	})

	sources := []securityAuth.APIKeySource{
		{Type: "header", Name: "Authorization", Scheme: "Bearer"},
	}

	authMiddleware := securityAuth.NewAPIKeyMiddleware(validator, sources)

	// 4. Set up TLS
	certFile := "testdata/certs/server-cert.pem"
	keyFile := "testdata/certs/server-key.pem"

	tlsConfig := &securityTLS.Config{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "1.3",
	}

	goTLSConfig, err := tlsConfig.ToTLSConfig()
	if err != nil {
		t.Fatalf("failed to create TLS config: %v", err)
	}

	// 5. Create handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		keyInfo, _ := securityAuth.GetAPIKeyInfo(r.Context())
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Secure request from %s", keyInfo.UserID)
	})

	// 6. Create server with TLS + Auth
	server := httptest.NewUnstartedServer(authMiddleware.Handle(handler))
	server.TLS = goTLSConfig
	server.StartTLS()
	defer server.Close()

	// 7. Make authenticated request over TLS
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	req, _ := http.NewRequest("GET", server.URL, nil)
	req.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}

	if resp.TLS == nil {
		t.Error("TLS not used")
	}

	body, _ := io.ReadAll(resp.Body)
	expected := "Secure request from user-1"
	if string(body) != expected {
		t.Errorf("body = %q, want %q", body, expected)
	}

	t.Log("Full security stack working: TLS + Auth + Secrets ✓")
}

// TestCertificateReloadIntegration tests certificate auto-reload functionality.
func TestCertificateReloadIntegration(t *testing.T) {
	// Create temporary directory for certificates
	tmpDir := t.TempDir()
	certFile := filepath.Join(tmpDir, "cert.pem")
	keyFile := filepath.Join(tmpDir, "key.pem")

	// Copy initial certificates
	copyFile(t, "testdata/certs/server-cert.pem", certFile)
	copyFile(t, "testdata/certs/server-key.pem", keyFile)

	// Create reloader with short interval
	reloader := securityTLS.NewCertificateReloader(certFile, keyFile, 100*time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := reloader.Start(ctx)
	if err != nil {
		t.Fatalf("Start() failed: %v", err)
	}

	// Create HTTPS server with reloader
	tlsConfig := &tls.Config{
		GetCertificate: reloader.GetCertificateFunc(),
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	})

	server := httptest.NewUnstartedServer(handler)
	server.TLS = tlsConfig
	server.StartTLS()
	defer server.Close()

	// Make first request
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		},
	}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	resp.Body.Close()

	// Update certificate file (touch to change mtime)
	time.Sleep(200 * time.Millisecond)
	now := time.Now().Add(2 * time.Second)
	err = os.Chtimes(certFile, now, now)
	if err != nil {
		t.Fatalf("failed to update cert mtime: %v", err)
	}

	// Wait for reload
	time.Sleep(300 * time.Millisecond)

	// Make second request (should use reloaded cert)
	resp, err = client.Get(server.URL)
	if err != nil {
		t.Fatalf("second request failed: %v", err)
	}
	resp.Body.Close()

	t.Log("Certificate reload integration test passed")
}

// TestSecretRotationIntegration tests zero-downtime secret rotation.
func TestSecretRotationIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	secretFile := filepath.Join(tmpDir, "rotating-secret")

	// Write initial secret
	err := os.WriteFile(secretFile, []byte("secret-v1"), 0600)
	if err != nil {
		t.Fatalf("failed to write initial secret: %v", err)
	}

	// Create file provider with watch enabled
	fileProvider, err := secrets.NewFileProvider(tmpDir, true)
	if err != nil {
		t.Fatalf("failed to create file provider: %v", err)
	}
	defer fileProvider.Close()

	manager := secrets.NewManager(
		[]secrets.SecretProvider{fileProvider},
		secrets.CacheConfig{Enabled: true, TTL: 1 * time.Second, MaxSize: 100},
	)

	ctx := context.Background()

	// Read initial secret
	value, err := manager.GetSecret(ctx, "rotating-secret")
	if err != nil {
		t.Fatalf("failed to get initial secret: %v", err)
	}
	if value != "secret-v1" {
		t.Errorf("initial secret wrong: got %q, want %q", value, "secret-v1")
	}

	// Rotate secret
	err = os.WriteFile(secretFile, []byte("secret-v2"), 0600)
	if err != nil {
		t.Fatalf("failed to rotate secret: %v", err)
	}

	// Wait for file watcher to detect change and cache to expire
	time.Sleep(1500 * time.Millisecond)

	// Refresh provider (FileProvider implements RefreshableProvider)
	err = fileProvider.Refresh(ctx)
	if err != nil {
		t.Fatalf("failed to refresh provider: %v", err)
	}

	// Read rotated secret
	value, err = manager.GetSecret(ctx, "rotating-secret")
	if err != nil {
		t.Fatalf("failed to get rotated secret: %v", err)
	}
	if value != "secret-v2" {
		t.Errorf("rotated secret wrong: got %q, want %q", value, "secret-v2")
	}

	t.Log("Secret rotation integration test passed")
}

// Helper function to copy files.
func copyFile(t *testing.T, src, dst string) {
	t.Helper()

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("failed to read %s: %v", src, err)
	}

	err = os.WriteFile(dst, data, 0600)
	if err != nil {
		t.Fatalf("failed to write %s: %v", dst, err)
	}
}
