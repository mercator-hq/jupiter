package tls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"log/slog"
	"os"
	"sync"
	"time"
)

// CertificateReloader watches certificate files and reloads them automatically.
// This allows certificate renewal (e.g., Let's Encrypt) without server restart.
type CertificateReloader struct {
	certFile string
	keyFile  string
	interval time.Duration

	mu       sync.RWMutex
	cert     *tls.Certificate
	certTime time.Time
	keyTime  time.Time
}

// NewCertificateReloader creates a new certificate reloader.
// interval specifies how often to check for certificate changes.
func NewCertificateReloader(certFile, keyFile string, interval time.Duration) *CertificateReloader {
	return &CertificateReloader{
		certFile: certFile,
		keyFile:  keyFile,
		interval: interval,
	}
}

// Start begins watching certificate files and reloading them when they change.
// It loads the initial certificate and starts a background goroutine to check for updates.
func (r *CertificateReloader) Start(ctx context.Context) error {
	// Load initial certificate
	if err := r.reload(); err != nil {
		return err
	}

	// Log initial certificate info
	r.logCertificateInfo()

	// Start reload loop in background
	go r.reloadLoop(ctx)

	return nil
}

// reloadLoop periodically checks for certificate changes and reloads if needed.
func (r *CertificateReloader) reloadLoop(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if r.needsReload() {
				if err := r.reload(); err != nil {
					slog.Error("failed to reload certificate",
						"error", err,
						"cert_file", r.certFile,
						"key_file", r.keyFile,
					)
				} else {
					slog.Info("certificate reloaded",
						"cert_file", r.certFile,
					)
					r.logCertificateInfo()
				}
			}

		case <-ctx.Done():
			return
		}
	}
}

// needsReload checks if certificate files have been modified since last load.
func (r *CertificateReloader) needsReload() bool {
	certInfo, err := os.Stat(r.certFile)
	if err != nil {
		return false
	}

	keyInfo, err := os.Stat(r.keyFile)
	if err != nil {
		return false
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return certInfo.ModTime().After(r.certTime) || keyInfo.ModTime().After(r.keyTime)
}

// reload loads the certificate and key from disk.
func (r *CertificateReloader) reload() error {
	// Get file modification times
	certInfo, err := os.Stat(r.certFile)
	if err != nil {
		return err
	}

	keyInfo, err := os.Stat(r.keyFile)
	if err != nil {
		return err
	}

	// Load certificate
	cert, err := tls.LoadX509KeyPair(r.certFile, r.keyFile)
	if err != nil {
		return err
	}

	// Validate certificate
	if err := ValidateCertificate(&cert); err != nil {
		return err
	}

	// Update certificate atomically
	r.mu.Lock()
	r.cert = &cert
	r.certTime = certInfo.ModTime()
	r.keyTime = keyInfo.ModTime()
	r.mu.Unlock()

	return nil
}

// GetCertificate returns the current certificate.
func (r *CertificateReloader) GetCertificate() *tls.Certificate {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.cert
}

// GetCertificateFunc returns a function compatible with tls.Config.GetCertificate.
// This allows automatic certificate rotation without server restart.
func (r *CertificateReloader) GetCertificateFunc() func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	return func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		return r.GetCertificate(), nil
	}
}

// logCertificateInfo logs information about the currently loaded certificate.
func (r *CertificateReloader) logCertificateInfo() {
	cert := r.GetCertificate()
	if cert == nil || len(cert.Certificate) == 0 {
		return
	}

	// Parse certificate for logging
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return
	}

	// Check expiration
	daysUntilExpiry, warning := CheckCertificateExpiration(x509Cert)

	if warning != "" {
		slog.Warn("certificate expiring soon",
			"subject", x509Cert.Subject.CommonName,
			"expires_in_days", daysUntilExpiry,
			"expires_at", x509Cert.NotAfter.Format(time.RFC3339),
		)
	} else {
		slog.Info("certificate loaded",
			"subject", x509Cert.Subject.CommonName,
			"issuer", x509Cert.Issuer.CommonName,
			"expires_in_days", daysUntilExpiry,
			"expires_at", x509Cert.NotAfter.Format(time.RFC3339),
		)
	}
}
