package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func BenchmarkToTLSConfig(b *testing.B) {
	certFile := filepath.Join(testDataDir, "server-cert.pem")
	keyFile := filepath.Join(testDataDir, "server-key.pem")

	cfg := &Config{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "1.3",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cfg.ToTLSConfig()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateCertificate(b *testing.B) {
	certFile := filepath.Join(testDataDir, "server-cert.pem")
	keyFile := filepath.Join(testDataDir, "server-key.pem")

	cert, _ := tls.LoadX509KeyPair(certFile, keyFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidateCertificate(&cert)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkExtractClientIdentity(b *testing.B) {
	certFile := filepath.Join(testDataDir, "client-cert.pem")
	cert := loadX509Cert(b, certFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		identity := ExtractClientIdentity(cert, "subject.CN")
		_ = identity
	}
}

func BenchmarkCertificateReloaderGetCertificate(b *testing.B) {
	certFile := filepath.Join(testDataDir, "server-cert.pem")
	keyFile := filepath.Join(testDataDir, "server-key.pem")

	reloader := NewCertificateReloader(certFile, keyFile, 5*time.Minute)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = reloader.Start(ctx)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cert := reloader.GetCertificate()
		_ = cert
	}
}

func BenchmarkCheckCertificateExpiration(b *testing.B) {
	certFile := filepath.Join(testDataDir, "server-cert.pem")
	cert := loadX509Cert(b, certFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		days, warning := CheckCertificateExpiration(cert)
		_, _ = days, warning
	}
}

func BenchmarkExtractCertificateInfo(b *testing.B) {
	certFile := filepath.Join(testDataDir, "server-cert.pem")
	cert := loadX509Cert(b, certFile)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		info := ExtractCertificateInfo(cert)
		_ = info
	}
}

func BenchmarkValidateCertificateChain(b *testing.B) {
	certFile := filepath.Join(testDataDir, "client-cert.pem")
	caFile := filepath.Join(testDataDir, "ca-cert.pem")

	cert := loadX509Cert(b, certFile)

	// Load CA into cert pool
	caCertPEM, _ := os.ReadFile(caFile)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCertPEM)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := ValidateCertificateChain(cert, caCertPool)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkToTLSConfig_WithMTLS(b *testing.B) {
	certFile := filepath.Join(testDataDir, "server-cert.pem")
	keyFile := filepath.Join(testDataDir, "server-key.pem")
	caFile := filepath.Join(testDataDir, "ca-cert.pem")

	cfg := &Config{
		Enabled:    true,
		CertFile:   certFile,
		KeyFile:    keyFile,
		MinVersion: "1.3",
		MTLS: MTLSConfig{
			Enabled:          true,
			ClientCAFile:     caFile,
			ClientAuthType:   "require",
			VerifyClientCert: true,
			IdentitySource:   "subject.CN",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cfg.ToTLSConfig()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// Helper function to load X.509 certificate from file
func loadX509Cert(b *testing.B, path string) *x509.Certificate {
	b.Helper()

	certPEM, err := os.ReadFile(path)
	if err != nil {
		b.Fatalf("failed to read cert file: %v", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		b.Fatal("failed to decode PEM block")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		b.Fatalf("failed to parse certificate: %v", err)
	}

	return cert
}
