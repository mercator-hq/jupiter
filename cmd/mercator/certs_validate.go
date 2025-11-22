package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	securityTLS "mercator-hq/jupiter/pkg/security/tls"

	"github.com/spf13/cobra"
)

var certsValidateFlags struct {
	certFile string
	keyFile  string
	caFile   string
}

var certsValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate certificate and key",
	Long: `Validate a TLS certificate and private key.

This command validates:
  - Certificate and key pair match
  - Certificate is not expired
  - Certificate chain validation (if --ca provided)
  - Certificate expiration warnings (<30 days)

The validation checks ensure that the certificate is valid for use
in production and will warn about potential issues.

Examples:
  # Validate certificate and key match
  mercator certs validate --cert server.crt --key server.key

  # Validate certificate chain against CA
  mercator certs validate --cert server.crt --ca ca.pem

  # Validate both key and chain
  mercator certs validate --cert server.crt --key server.key --ca ca.pem`,
	RunE: validateCertificate,
}

func init() {
	certsCmd.AddCommand(certsValidateCmd)

	certsValidateCmd.Flags().StringVar(&certsValidateFlags.certFile, "cert", "", "certificate file (required)")
	certsValidateCmd.Flags().StringVar(&certsValidateFlags.keyFile, "key", "", "private key file")
	certsValidateCmd.Flags().StringVar(&certsValidateFlags.caFile, "ca", "", "CA certificate file")

	_ = certsValidateCmd.MarkFlagRequired("cert")
}

func validateCertificate(cmd *cobra.Command, args []string) error {
	fmt.Printf("Validating certificate: %s\n\n", certsValidateFlags.certFile)

	// Load certificate
	certPEM, err := os.ReadFile(certsValidateFlags.certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to parse certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Validate certificate and key match (if key provided)
	if certsValidateFlags.keyFile != "" {
		if _, err := tls.LoadX509KeyPair(certsValidateFlags.certFile, certsValidateFlags.keyFile); err != nil {
			fmt.Println("✗ Certificate and key do NOT match")
			return err
		}
		fmt.Println("✓ Certificate and key match")
	}

	// Validate chain (if CA provided)
	if certsValidateFlags.caFile != "" {
		if err := validateChain(cert, certsValidateFlags.caFile); err != nil {
			fmt.Println("✗ Certificate chain invalid")
			return err
		}
		fmt.Println("✓ Certificate chain valid")
	}

	// Check expiration
	now := time.Now()
	if now.After(cert.NotAfter) {
		fmt.Printf("✗ Certificate EXPIRED on %s\n", cert.NotAfter.Format("2006-01-02"))
		return fmt.Errorf("certificate expired")
	}
	fmt.Printf("✓ Certificate not expired (valid until %s)\n", cert.NotAfter.Format("2006-01-02"))

	// Warning if expires soon
	daysUntilExpiry, warning := securityTLS.CheckCertificateExpiration(cert)
	if warning != "" {
		fmt.Printf("⚠  Certificate expires in %d days\n", daysUntilExpiry)
	}

	// Print certificate details
	fmt.Println("\nCertificate Details:")
	fmt.Printf("  Subject: %s\n", cert.Subject.CommonName)
	if len(cert.Subject.Organization) > 0 {
		fmt.Printf("  Organization: %s\n", cert.Subject.Organization[0])
	}
	fmt.Printf("  Issuer: %s\n", cert.Issuer.CommonName)
	fmt.Printf("  Serial: %x\n", cert.SerialNumber)
	fmt.Printf("  Valid From: %s\n", cert.NotBefore.Format(time.RFC3339))
	fmt.Printf("  Valid Until: %s\n", cert.NotAfter.Format(time.RFC3339))

	if len(cert.DNSNames) > 0 {
		fmt.Printf("  SANs (DNS): %v\n", cert.DNSNames)
	}

	if len(cert.IPAddresses) > 0 {
		fmt.Printf("  SANs (IP): %v\n", cert.IPAddresses)
	}

	return nil
}

func validateChain(cert *x509.Certificate, caFile string) error {
	// Load CA cert
	caPEM, err := os.ReadFile(caFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return fmt.Errorf("failed to parse CA certificate")
	}

	// Use the ValidateCertificateChain function from pkg/security/tls
	return securityTLS.ValidateCertificateChain(cert, caPool)
}
