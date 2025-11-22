package tls

import (
	"crypto/x509"
	"fmt"
	"net/http"
)

// ExtractClientIdentity extracts identity from a client certificate
// based on the configured identity source.
//
// Supported identity sources:
//   - "subject.CN": Common Name from Subject
//   - "subject.OU": Organizational Unit from Subject
//   - "subject.O": Organization from Subject
//   - "SAN": First DNS name from Subject Alternative Names
//
// Returns an empty string if the identity cannot be extracted.
func ExtractClientIdentity(cert *x509.Certificate, source string) string {
	if cert == nil {
		return ""
	}

	switch source {
	case "subject.CN", "":
		return cert.Subject.CommonName

	case "subject.OU":
		if len(cert.Subject.OrganizationalUnit) > 0 {
			return cert.Subject.OrganizationalUnit[0]
		}

	case "subject.O":
		if len(cert.Subject.Organization) > 0 {
			return cert.Subject.Organization[0]
		}

	case "SAN":
		if len(cert.DNSNames) > 0 {
			return cert.DNSNames[0]
		}
	}

	return ""
}

// ClientCertInfo represents information extracted from a client certificate.
type ClientCertInfo struct {
	Identity           string
	Subject            string
	Issuer             string
	SerialNumber       string
	OrganizationalUnit []string
	Organization       []string
	DNSNames           []string
}

// ExtractClientCertInfo extracts detailed information from a client certificate.
func ExtractClientCertInfo(cert *x509.Certificate, identitySource string) *ClientCertInfo {
	if cert == nil {
		return nil
	}

	return &ClientCertInfo{
		Identity:           ExtractClientIdentity(cert, identitySource),
		Subject:            cert.Subject.String(),
		Issuer:             cert.Issuer.String(),
		SerialNumber:       fmt.Sprintf("%x", cert.SerialNumber),
		OrganizationalUnit: cert.Subject.OrganizationalUnit,
		Organization:       cert.Subject.Organization,
		DNSNames:           cert.DNSNames,
	}
}

// GetClientCertificate extracts the client certificate from an HTTP request.
// Returns nil if no client certificate is present or TLS is not used.
func GetClientCertificate(r *http.Request) *x509.Certificate {
	if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
		return nil
	}

	// Return the first (leaf) certificate
	return r.TLS.PeerCertificates[0]
}

// GetClientIdentity extracts client identity from an HTTP request
// based on the configured identity source.
func GetClientIdentity(r *http.Request, identitySource string) string {
	cert := GetClientCertificate(r)
	if cert == nil {
		return ""
	}

	return ExtractClientIdentity(cert, identitySource)
}

// ValidateClientCertificate validates a client certificate against a CA pool.
func ValidateClientCertificate(cert *x509.Certificate, caPool *x509.CertPool) error {
	if cert == nil {
		return fmt.Errorf("client certificate is nil")
	}

	// Verify certificate against CA pool
	opts := x509.VerifyOptions{
		Roots:     caPool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("client certificate validation failed: %w", err)
	}

	// Check expiration
	if err := ValidateX509Certificate(cert); err != nil {
		return err
	}

	return nil
}
