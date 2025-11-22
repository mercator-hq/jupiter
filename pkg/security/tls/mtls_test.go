package tls

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"os"
	"testing"
)

func TestExtractClientIdentity(t *testing.T) {
	// Load client certificate
	clientCertPEM, err := os.ReadFile("testdata/client-cert.pem")
	if err != nil {
		t.Fatalf("failed to read client certificate: %v", err)
	}

	block, _ := pem.Decode(clientCertPEM)
	if block == nil {
		t.Fatalf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	tests := []struct {
		name     string
		cert     *x509.Certificate
		source   string
		expected string
	}{
		{
			name:     "extract CN",
			cert:     cert,
			source:   "subject.CN",
			expected: "test-client",
		},
		{
			name:     "extract OU",
			cert:     cert,
			source:   "subject.OU",
			expected: "engineering",
		},
		{
			name:     "extract O",
			cert:     cert,
			source:   "subject.O",
			expected: "TestOrg",
		},
		{
			name:     "empty source defaults to CN",
			cert:     cert,
			source:   "",
			expected: "test-client",
		},
		{
			name:     "nil cert returns empty",
			cert:     nil,
			source:   "subject.CN",
			expected: "",
		},
		{
			name:     "unknown source returns empty",
			cert:     cert,
			source:   "unknown",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractClientIdentity(tt.cert, tt.source)
			if result != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestExtractClientCertInfo(t *testing.T) {
	// Load client certificate
	clientCertPEM, err := os.ReadFile("testdata/client-cert.pem")
	if err != nil {
		t.Fatalf("failed to read client certificate: %v", err)
	}

	block, _ := pem.Decode(clientCertPEM)
	if block == nil {
		t.Fatalf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}

	tests := []struct {
		name           string
		cert           *x509.Certificate
		identitySource string
		expectNil      bool
	}{
		{
			name:           "valid certificate with CN source",
			cert:           cert,
			identitySource: "subject.CN",
			expectNil:      false,
		},
		{
			name:           "valid certificate with OU source",
			cert:           cert,
			identitySource: "subject.OU",
			expectNil:      false,
		},
		{
			name:           "nil certificate",
			cert:           nil,
			identitySource: "subject.CN",
			expectNil:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := ExtractClientCertInfo(tt.cert, tt.identitySource)

			if tt.expectNil {
				if info != nil {
					t.Errorf("expected nil, got %v", info)
				}
				return
			}

			if info == nil {
				t.Fatal("expected non-nil info")
			}

			if info.Identity == "" {
				t.Error("expected identity to be set")
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

			if len(info.OrganizationalUnit) == 0 {
				t.Error("expected organizational unit to be set")
			}

			if len(info.Organization) == 0 {
				t.Error("expected organization to be set")
			}
		})
	}
}

func TestGetClientCertificate(t *testing.T) {
	tests := []struct {
		name      string
		request   *http.Request
		expectNil bool
	}{
		{
			name:      "request without TLS",
			request:   &http.Request{},
			expectNil: true,
		},
		{
			name: "request with TLS but no peer certificates",
			request: &http.Request{
				TLS: &tls.ConnectionState{},
			},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cert := GetClientCertificate(tt.request)

			if tt.expectNil && cert != nil {
				t.Errorf("expected nil certificate, got %v", cert)
			}

			if !tt.expectNil && cert == nil {
				t.Error("expected non-nil certificate")
			}
		})
	}
}

func TestGetClientIdentity(t *testing.T) {
	tests := []struct {
		name           string
		request        *http.Request
		identitySource string
		expected       string
	}{
		{
			name:           "request without TLS",
			request:        &http.Request{},
			identitySource: "subject.CN",
			expected:       "",
		},
		{
			name: "request with TLS but no peer certificates",
			request: &http.Request{
				TLS: &tls.ConnectionState{},
			},
			identitySource: "subject.CN",
			expected:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			identity := GetClientIdentity(tt.request, tt.identitySource)
			if identity != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, identity)
			}
		})
	}
}

func TestValidateClientCertificate(t *testing.T) {
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
			name:        "valid client certificate",
			cert:        clientCert,
			caPool:      caPool,
			expectError: false,
		},
		{
			name:        "nil certificate",
			cert:        nil,
			caPool:      caPool,
			expectError: true,
		},
		{
			name:        "wrong CA pool",
			cert:        clientCert,
			caPool:      x509.NewCertPool(),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClientCertificate(tt.cert, tt.caPool)

			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
