package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"time"
)

// ValidateCertificate checks if a certificate is valid and not expired.
// It returns an error if the certificate is expired or will expire soon.
func ValidateCertificate(cert *tls.Certificate) error {
	if cert == nil {
		return fmt.Errorf("certificate is nil")
	}

	if len(cert.Certificate) == 0 {
		return fmt.Errorf("certificate chain is empty")
	}

	// Parse the leaf certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	return ValidateX509Certificate(x509Cert)
}

// ValidateX509Certificate validates an x509 certificate for expiration.
func ValidateX509Certificate(cert *x509.Certificate) error {
	now := time.Now()

	// Check if certificate is not yet valid
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (valid from %s)", cert.NotBefore.Format(time.RFC3339))
	}

	// Check if certificate is expired
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate expired on %s", cert.NotAfter.Format(time.RFC3339))
	}

	return nil
}

// CheckCertificateExpiration checks if a certificate is expiring soon.
// Returns the number of days until expiration and a warning if < 30 days.
func CheckCertificateExpiration(cert *x509.Certificate) (daysUntilExpiry int, warning string) {
	now := time.Now()
	duration := cert.NotAfter.Sub(now)
	daysUntilExpiry = int(duration.Hours() / 24)

	if daysUntilExpiry < 30 {
		warning = fmt.Sprintf("certificate expires in %d days (on %s)",
			daysUntilExpiry, cert.NotAfter.Format("2006-01-02"))
	}

	return daysUntilExpiry, warning
}

// ValidateCertificateChain validates a certificate chain against a CA pool.
func ValidateCertificateChain(cert *x509.Certificate, caPool *x509.CertPool) error {
	opts := x509.VerifyOptions{
		Roots:     caPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate chain validation failed: %w", err)
	}

	return nil
}

// GetCertificateInfo extracts human-readable information from a certificate.
type CertificateInfo struct {
	Subject            string
	Issuer             string
	SerialNumber       string
	NotBefore          time.Time
	NotAfter           time.Time
	DNSNames           []string
	IPAddresses        []string
	SignatureAlgorithm string
	PublicKeyAlgorithm string
}

// ExtractCertificateInfo extracts information from an x509 certificate.
func ExtractCertificateInfo(cert *x509.Certificate) *CertificateInfo {
	info := &CertificateInfo{
		Subject:            cert.Subject.String(),
		Issuer:             cert.Issuer.String(),
		SerialNumber:       fmt.Sprintf("%x", cert.SerialNumber),
		NotBefore:          cert.NotBefore,
		NotAfter:           cert.NotAfter,
		DNSNames:           cert.DNSNames,
		SignatureAlgorithm: cert.SignatureAlgorithm.String(),
		PublicKeyAlgorithm: cert.PublicKeyAlgorithm.String(),
	}

	// Extract IP addresses
	for _, ip := range cert.IPAddresses {
		info.IPAddresses = append(info.IPAddresses, ip.String())
	}

	return info
}
