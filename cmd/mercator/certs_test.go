package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Helper function to create a test certificate and key
func createTestCertificate(t *testing.T, outputDir string) (certPath, keyPath string) {
	t.Helper()

	// Generate private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	// Create certificate template
	serialNumber, _ := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 365),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	// Create self-signed certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatalf("failed to create certificate: %v", err)
	}

	// Ensure output directory exists
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		t.Fatalf("failed to create output directory: %v", err)
	}

	// Write certificate
	certPath = filepath.Join(outputDir, "test-cert.pem")
	certFile, err := os.Create(certPath)
	if err != nil {
		t.Fatalf("failed to create cert file: %v", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("failed to write cert: %v", err)
	}

	// Write private key
	keyPath = filepath.Join(outputDir, "test-key.pem")
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		t.Fatalf("failed to create key file: %v", err)
	}
	defer keyFile.Close()

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		t.Fatalf("failed to write key: %v", err)
	}

	return certPath, keyPath
}

func TestCertsGenerate(t *testing.T) {
	outputDir := t.TempDir()

	tests := []struct {
		name     string
		hosts    string
		org      string
		validity int
		keySize  int
		wantErr  bool
	}{
		{
			name:     "valid certificate generation",
			hosts:    "localhost",
			org:      "Test Company",
			validity: 365,
			keySize:  2048,
			wantErr:  false,
		},
		{
			name:     "multiple hosts",
			hosts:    "localhost,127.0.0.1,example.com",
			org:      "Test Company",
			validity: 365,
			keySize:  2048,
			wantErr:  false,
		},
		{
			name:     "invalid key size",
			hosts:    "localhost",
			org:      "Test Company",
			validity: 365,
			keySize:  1024, // Invalid key size
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set flags
			generateFlags.hosts = tt.hosts
			generateFlags.org = tt.org
			generateFlags.validity = tt.validity
			generateFlags.keySize = tt.keySize
			generateFlags.output = filepath.Join(outputDir, tt.name)

			// Run generate
			err := generateCertificate(nil, nil)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.wantErr {
				// Verify files were created
				certPath := filepath.Join(generateFlags.output, "cert.pem")
				keyPath := filepath.Join(generateFlags.output, "key.pem")

				if _, err := os.Stat(certPath); os.IsNotExist(err) {
					t.Errorf("certificate file not created: %s", certPath)
				}

				if _, err := os.Stat(keyPath); os.IsNotExist(err) {
					t.Errorf("key file not created: %s", keyPath)
				}

				// Verify key file permissions
				info, err := os.Stat(keyPath)
				if err != nil {
					t.Errorf("failed to stat key file: %v", err)
				} else {
					mode := info.Mode().Perm()
					if mode != 0600 {
						t.Errorf("incorrect key file permissions: got %o, want 0600", mode)
					}
				}
			}
		})
	}
}

func TestCertsValidate(t *testing.T) {
	outputDir := t.TempDir()
	certPath, keyPath := createTestCertificate(t, outputDir)

	tests := []struct {
		name     string
		certFile string
		keyFile  string
		wantErr  bool
	}{
		{
			name:     "valid certificate and key",
			certFile: certPath,
			keyFile:  keyPath,
			wantErr:  false,
		},
		{
			name:     "certificate only",
			certFile: certPath,
			keyFile:  "",
			wantErr:  false,
		},
		{
			name:     "nonexistent certificate",
			certFile: filepath.Join(outputDir, "nonexistent.pem"),
			keyFile:  "",
			wantErr:  true,
		},
		{
			name:     "mismatched certificate and key",
			certFile: certPath,
			keyFile:  certPath, // Using cert file as key file (mismatch)
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set flags
			certsValidateFlags.certFile = tt.certFile
			certsValidateFlags.keyFile = tt.keyFile
			certsValidateFlags.caFile = ""

			// Run validate
			err := validateCertificate(nil, nil)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestCertsInfo(t *testing.T) {
	outputDir := t.TempDir()
	certPath, _ := createTestCertificate(t, outputDir)

	tests := []struct {
		name     string
		certFile string
		format   string
		wantErr  bool
	}{
		{
			name:     "text format",
			certFile: certPath,
			format:   "text",
			wantErr:  false,
		},
		{
			name:     "json format",
			certFile: certPath,
			format:   "json",
			wantErr:  false,
		},
		{
			name:     "nonexistent certificate",
			certFile: filepath.Join(outputDir, "nonexistent.pem"),
			format:   "text",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set flags
			infoFlags.format = tt.format

			// Run info
			err := displayCertInfo(nil, []string{tt.certFile})

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateChain(t *testing.T) {
	outputDir := t.TempDir()

	// Create a simple CA certificate
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	caTemplate := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test CA"},
			CommonName:   "Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caDerBytes, err := x509.CreateCertificate(rand.Reader, &caTemplate, &caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	caCert, err := x509.ParseCertificate(caDerBytes)
	if err != nil {
		t.Fatalf("failed to parse CA certificate: %v", err)
	}

	// Write CA cert to file
	caPath := filepath.Join(outputDir, "ca.pem")
	caFile, err := os.Create(caPath)
	if err != nil {
		t.Fatalf("failed to create CA file: %v", err)
	}
	defer caFile.Close()

	if err := pem.Encode(caFile, &pem.Block{Type: "CERTIFICATE", Bytes: caDerBytes}); err != nil {
		t.Fatalf("failed to write CA cert: %v", err)
	}

	// Create a leaf certificate signed by the CA
	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate leaf key: %v", err)
	}

	leafTemplate := x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"Test Leaf"},
			CommonName:   "localhost",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, 365),
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}

	leafDerBytes, err := x509.CreateCertificate(rand.Reader, &leafTemplate, &caTemplate, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create leaf certificate: %v", err)
	}

	leafCert, err := x509.ParseCertificate(leafDerBytes)
	if err != nil {
		t.Fatalf("failed to parse leaf certificate: %v", err)
	}

	tests := []struct {
		name    string
		cert    *x509.Certificate
		caFile  string
		wantErr bool
	}{
		{
			name:    "valid chain",
			cert:    leafCert,
			caFile:  caPath,
			wantErr: false,
		},
		{
			name:    "invalid CA file",
			cert:    caCert,
			caFile:  filepath.Join(outputDir, "nonexistent.pem"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateChain(tt.cert, tt.caFile)

			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
