package tls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
)

func TestValidateCertificate(t *testing.T) {
	// Load valid certificate
	cert, err := tls.LoadX509KeyPair("testdata/server-cert.pem", "testdata/server-key.pem")
	if err != nil {
		t.Fatalf("failed to load test certificate: %v", err)
	}

	tests := []struct {
		name        string
		cert        *tls.Certificate
		expectError bool
	}{
		{
			name:        "valid certificate",
			cert:        &cert,
			expectError: false,
		},
		{
			name:        "nil certificate",
			cert:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCertificate(tt.cert)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateX509Certificate(t *testing.T) {
	// Load valid certificate
	certPEM, err := os.ReadFile("testdata/server-cert.pem")
	if err != nil {
		t.Fatalf("failed to read test certificate: %v", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	tests := []struct {
		name        string
		cert        *x509.Certificate
		expectError bool
	}{
		{
			name:        "valid certificate",
			cert:        cert,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateX509Certificate(tt.cert)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCheckCertificateExpiration(t *testing.T) {
	// Load test certificate
	certPEM, err := os.ReadFile("testdata/server-cert.pem")
	if err != nil {
		t.Fatalf("failed to read test certificate: %v", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	daysUntilExpiry, warning := CheckCertificateExpiration(cert)

	if daysUntilExpiry < 0 {
		t.Errorf("certificate already expired")
	}

	// Test certificate is valid for 365 days, so it should have > 300 days left
	if daysUntilExpiry < 300 {
		t.Errorf("expected > 300 days until expiry, got %d", daysUntilExpiry)
	}

	// Should not have warning since certificate was just created
	if warning != "" {
		t.Errorf("unexpected warning: %s", warning)
	}
}

func TestExtractCertificateInfo(t *testing.T) {
	// Load test certificate
	certPEM, err := os.ReadFile("testdata/server-cert.pem")
	if err != nil {
		t.Fatalf("failed to read test certificate: %v", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatalf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	info := ExtractCertificateInfo(cert)

	if info == nil {
		t.Fatal("expected non-nil certificate info")
	}

	if info.Subject == "" {
		t.Error("expected subject to be set")
	}

	if info.Issuer == "" {
		t.Error("expected issuer to be set")
	}

	if info.SerialNumber == "" {
		t.Error("expected serial number to be set")
	}

	if info.NotBefore.IsZero() {
		t.Error("expected NotBefore to be set")
	}

	if info.NotAfter.IsZero() {
		t.Error("expected NotAfter to be set")
	}

	if info.NotAfter.Before(info.NotBefore) {
		t.Error("NotAfter should be after NotBefore")
	}

	if info.SignatureAlgorithm == "" {
		t.Error("expected signature algorithm to be set")
	}

	if info.PublicKeyAlgorithm == "" {
		t.Error("expected public key algorithm to be set")
	}
}

func TestValidateCertificateChain(t *testing.T) {
	// Load client certificate
	clientCertPEM, err := os.ReadFile("testdata/client-cert.pem")
	if err != nil {
		t.Fatalf("failed to read client certificate: %v", err)
	}

	block, _ := pem.Decode(clientCertPEM)
	if block == nil {
		t.Fatalf("failed to parse certificate PEM")
	}

	clientCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	// Load CA certificate
	caCertPEM, err := os.ReadFile("testdata/ca-cert.pem")
	if err != nil {
		t.Fatalf("failed to read CA certificate: %v", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCertPEM) {
		t.Fatalf("failed to parse CA certificate")
	}

	tests := []struct {
		name        string
		cert        *x509.Certificate
		caPool      *x509.CertPool
		expectError bool
	}{
		{
			name:        "valid chain",
			cert:        clientCert,
			caPool:      caPool,
			expectError: false,
		},
		{
			name:        "invalid chain - wrong CA",
			cert:        clientCert,
			caPool:      x509.NewCertPool(), // Empty pool
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCertificateChain(tt.cert, tt.caPool)
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
