package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"time"

	securityTLS "mercator-hq/jupiter/pkg/security/tls"

	"github.com/spf13/cobra"
)

var infoFlags struct {
	format string
}

var certsInfoCmd = &cobra.Command{
	Use:   "info [cert-file]",
	Short: "Display certificate details",
	Long: `Display detailed information about a TLS certificate.

This command extracts and displays comprehensive information from
a certificate file including:
  - Subject (CN, Organization, Country)
  - Issuer details
  - Validity period (NotBefore, NotAfter)
  - Subject Alternative Names (DNS, IP)
  - Key usage and extended key usage
  - Signature and public key algorithms
  - Serial number

Output formats:
  - text (default): Human-readable formatted output
  - json: JSON-formatted output for scripting

Examples:
  # Display certificate info in text format
  mercator certs info server.crt

  # Display in JSON format
  mercator certs info --format json server.crt

  # Save JSON output to file
  mercator certs info --format json server.crt > cert-info.json`,
	Args: cobra.ExactArgs(1),
	RunE: displayCertInfo,
}

func init() {
	certsCmd.AddCommand(certsInfoCmd)

	certsInfoCmd.Flags().StringVar(&infoFlags.format, "format", "text", "output format: text, json")
}

func displayCertInfo(cmd *cobra.Command, args []string) error {
	certFile := args[0]

	// Load certificate
	certPEM, err := os.ReadFile(certFile)
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

	// Format output
	if infoFlags.format == "json" {
		return printCertJSON(cert)
	}
	return printCertText(cert, certFile)
}

func printCertText(cert *x509.Certificate, file string) error {
	fmt.Printf("Certificate: %s\n\n", file)

	// Subject
	fmt.Println("Subject:")
	fmt.Printf("  Common Name (CN): %s\n", cert.Subject.CommonName)
	if len(cert.Subject.Organization) > 0 {
		fmt.Printf("  Organization (O): %s\n", cert.Subject.Organization[0])
	}
	if len(cert.Subject.OrganizationalUnit) > 0 {
		fmt.Printf("  Organizational Unit (OU): %s\n", cert.Subject.OrganizationalUnit[0])
	}
	if len(cert.Subject.Country) > 0 {
		fmt.Printf("  Country (C): %s\n", cert.Subject.Country[0])
	}
	if len(cert.Subject.Province) > 0 {
		fmt.Printf("  State/Province (ST): %s\n", cert.Subject.Province[0])
	}
	if len(cert.Subject.Locality) > 0 {
		fmt.Printf("  Locality (L): %s\n", cert.Subject.Locality[0])
	}

	// Issuer
	fmt.Println("\nIssuer:")
	fmt.Printf("  Common Name (CN): %s\n", cert.Issuer.CommonName)
	if len(cert.Issuer.Organization) > 0 {
		fmt.Printf("  Organization (O): %s\n", cert.Issuer.Organization[0])
	}
	if len(cert.Issuer.Country) > 0 {
		fmt.Printf("  Country (C): %s\n", cert.Issuer.Country[0])
	}

	// Validity
	fmt.Println("\nValidity:")
	fmt.Printf("  Not Before: %s\n", cert.NotBefore.Format(time.RFC3339))
	fmt.Printf("  Not After: %s\n", cert.NotAfter.Format(time.RFC3339))

	duration := cert.NotAfter.Sub(cert.NotBefore)
	fmt.Printf("  Duration: %d days\n", int(duration.Hours()/24))

	// Check expiration status
	now := time.Now()
	if now.After(cert.NotAfter) {
		fmt.Printf("  Status: ✗ EXPIRED on %s\n", cert.NotAfter.Format("2006-01-02"))
	} else {
		daysRemaining := int(time.Until(cert.NotAfter).Hours() / 24)
		fmt.Printf("  Status: ✓ Valid (%d days remaining)\n", daysRemaining)
		if daysRemaining < 30 {
			fmt.Printf("  Warning: ⚠  Certificate expires in %d days\n", daysRemaining)
		}
	}

	// Subject Alternative Names
	if len(cert.DNSNames) > 0 || len(cert.IPAddresses) > 0 {
		fmt.Println("\nSubject Alternative Names:")
		for _, san := range cert.DNSNames {
			fmt.Printf("  - DNS: %s\n", san)
		}
		for _, ip := range cert.IPAddresses {
			fmt.Printf("  - IP: %s\n", ip.String())
		}
	}

	// Key Usage
	if cert.KeyUsage != 0 {
		fmt.Println("\nKey Usage:")
		keyUsages := getKeyUsages(cert.KeyUsage)
		for _, usage := range keyUsages {
			fmt.Printf("  - %s\n", usage)
		}
	}

	// Extended Key Usage
	if len(cert.ExtKeyUsage) > 0 {
		fmt.Println("\nExtended Key Usage:")
		for _, usage := range cert.ExtKeyUsage {
			fmt.Printf("  - %s\n", getExtKeyUsage(usage))
		}
	}

	// Algorithms
	fmt.Println("\nAlgorithms:")
	fmt.Printf("  Signature Algorithm: %s\n", cert.SignatureAlgorithm)
	fmt.Printf("  Public Key Algorithm: %s\n", cert.PublicKeyAlgorithm)

	// Additional info
	fmt.Println("\nAdditional Information:")
	fmt.Printf("  Serial Number: %x\n", cert.SerialNumber)
	fmt.Printf("  Version: %d\n", cert.Version)
	fmt.Printf("  Is CA: %v\n", cert.IsCA)

	return nil
}

func printCertJSON(cert *x509.Certificate) error {
	info := securityTLS.ExtractCertificateInfo(cert)

	// Calculate days remaining
	daysRemaining := int(time.Until(cert.NotAfter).Hours() / 24)
	isExpired := time.Now().After(cert.NotAfter)

	data := map[string]interface{}{
		"subject": map[string]interface{}{
			"common_name":         cert.Subject.CommonName,
			"organization":        cert.Subject.Organization,
			"organizational_unit": cert.Subject.OrganizationalUnit,
			"country":             cert.Subject.Country,
			"province":            cert.Subject.Province,
			"locality":            cert.Subject.Locality,
		},
		"issuer": map[string]interface{}{
			"common_name":  cert.Issuer.CommonName,
			"organization": cert.Issuer.Organization,
			"country":      cert.Issuer.Country,
		},
		"validity": map[string]interface{}{
			"not_before":     cert.NotBefore.Format(time.RFC3339),
			"not_after":      cert.NotAfter.Format(time.RFC3339),
			"duration_days":  int(cert.NotAfter.Sub(cert.NotBefore).Hours() / 24),
			"days_remaining": daysRemaining,
			"is_expired":     isExpired,
		},
		"sans": map[string]interface{}{
			"dns": info.DNSNames,
			"ip":  info.IPAddresses,
		},
		"key_usage":            getKeyUsages(cert.KeyUsage),
		"ext_key_usage":        getExtKeyUsages(cert.ExtKeyUsage),
		"signature_algorithm":  info.SignatureAlgorithm,
		"public_key_algorithm": info.PublicKeyAlgorithm,
		"serial_number":        info.SerialNumber,
		"version":              cert.Version,
		"is_ca":                cert.IsCA,
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func getKeyUsages(usage x509.KeyUsage) []string {
	var usages []string
	if usage&x509.KeyUsageDigitalSignature != 0 {
		usages = append(usages, "Digital Signature")
	}
	if usage&x509.KeyUsageContentCommitment != 0 {
		usages = append(usages, "Content Commitment")
	}
	if usage&x509.KeyUsageKeyEncipherment != 0 {
		usages = append(usages, "Key Encipherment")
	}
	if usage&x509.KeyUsageDataEncipherment != 0 {
		usages = append(usages, "Data Encipherment")
	}
	if usage&x509.KeyUsageKeyAgreement != 0 {
		usages = append(usages, "Key Agreement")
	}
	if usage&x509.KeyUsageCertSign != 0 {
		usages = append(usages, "Certificate Sign")
	}
	if usage&x509.KeyUsageCRLSign != 0 {
		usages = append(usages, "CRL Sign")
	}
	if usage&x509.KeyUsageEncipherOnly != 0 {
		usages = append(usages, "Encipher Only")
	}
	if usage&x509.KeyUsageDecipherOnly != 0 {
		usages = append(usages, "Decipher Only")
	}
	return usages
}

func getExtKeyUsages(usages []x509.ExtKeyUsage) []string {
	var result []string
	for _, usage := range usages {
		result = append(result, getExtKeyUsage(usage))
	}
	return result
}

func getExtKeyUsage(usage x509.ExtKeyUsage) string {
	switch usage {
	case x509.ExtKeyUsageAny:
		return "Any"
	case x509.ExtKeyUsageServerAuth:
		return "Server Authentication"
	case x509.ExtKeyUsageClientAuth:
		return "Client Authentication"
	case x509.ExtKeyUsageCodeSigning:
		return "Code Signing"
	case x509.ExtKeyUsageEmailProtection:
		return "Email Protection"
	case x509.ExtKeyUsageIPSECEndSystem:
		return "IPSEC End System"
	case x509.ExtKeyUsageIPSECTunnel:
		return "IPSEC Tunnel"
	case x509.ExtKeyUsageIPSECUser:
		return "IPSEC User"
	case x509.ExtKeyUsageTimeStamping:
		return "Time Stamping"
	case x509.ExtKeyUsageOCSPSigning:
		return "OCSP Signing"
	case x509.ExtKeyUsageMicrosoftServerGatedCrypto:
		return "Microsoft Server Gated Crypto"
	case x509.ExtKeyUsageNetscapeServerGatedCrypto:
		return "Netscape Server Gated Crypto"
	case x509.ExtKeyUsageMicrosoftCommercialCodeSigning:
		return "Microsoft Commercial Code Signing"
	case x509.ExtKeyUsageMicrosoftKernelCodeSigning:
		return "Microsoft Kernel Code Signing"
	default:
		return fmt.Sprintf("Unknown (%d)", usage)
	}
}
