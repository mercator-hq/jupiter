package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var generateFlags struct {
	hosts    string
	org      string
	validity int
	keySize  int
	output   string
}

var certsGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate self-signed certificate",
	Long: `Generate a self-signed TLS certificate for testing.

This command generates a self-signed certificate and private key
that can be used for development and testing purposes. The generated
certificate should NOT be used in production.

Features:
  - RSA key generation (2048, 3072, or 4096 bits)
  - Multiple Subject Alternative Names (DNS and IP)
  - Configurable validity period
  - Configurable organization name
  - Secure file permissions (0600 for private key)

⚠️  WARNING: Self-signed certificates are for TESTING ONLY!
   Do not use in production. For production, use certificates
   from a trusted Certificate Authority (e.g., Let's Encrypt).

Examples:
  # Generate certificate for localhost
  mercator certs generate --host localhost

  # Generate with multiple hosts
  mercator certs generate --host "localhost,127.0.0.1,app.local"

  # Generate with custom parameters
  mercator certs generate \
    --host "localhost,127.0.0.1" \
    --org "My Company" \
    --validity 365 \
    --key-size 2048 \
    --output certs/`,
	RunE: generateCertificate,
}

func init() {
	certsCmd.AddCommand(certsGenerateCmd)

	certsGenerateCmd.Flags().StringVar(&generateFlags.hosts, "host", "localhost", "comma-separated hostnames and IPs")
	certsGenerateCmd.Flags().StringVar(&generateFlags.org, "org", "Mercator", "organization name")
	certsGenerateCmd.Flags().IntVar(&generateFlags.validity, "validity", 365, "validity in days")
	certsGenerateCmd.Flags().IntVar(&generateFlags.keySize, "key-size", 2048, "RSA key size (2048, 3072, 4096)")
	certsGenerateCmd.Flags().StringVarP(&generateFlags.output, "output", "o", "certs", "output directory")
}

func generateCertificate(cmd *cobra.Command, args []string) error {
	fmt.Println("Generating self-signed certificate...")

	// Validate key size
	if generateFlags.keySize != 2048 && generateFlags.keySize != 3072 && generateFlags.keySize != 4096 {
		return fmt.Errorf("invalid key size: %d (must be 2048, 3072, or 4096)", generateFlags.keySize)
	}

	// Parse hosts
	hosts := strings.Split(generateFlags.hosts, ",")
	var dnsNames []string
	var ipAddresses []net.IP

	for _, host := range hosts {
		host = strings.TrimSpace(host)
		if ip := net.ParseIP(host); ip != nil {
			ipAddresses = append(ipAddresses, ip)
		} else {
			dnsNames = append(dnsNames, host)
		}
	}

	// Generate private key
	fmt.Printf("Generating %d-bit RSA private key...\n", generateFlags.keySize)
	privateKey, err := rsa.GenerateKey(rand.Reader, generateFlags.keySize)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Create certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, generateFlags.validity)

	template := x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{generateFlags.org},
			CommonName:   hosts[0],
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		DNSNames:              dnsNames,
		IPAddresses:           ipAddresses,
	}

	// Create self-signed certificate
	fmt.Println("Creating self-signed certificate...")
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Create output directory with restricted permissions (0750)
	if err := os.MkdirAll(generateFlags.output, 0750); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Write certificate
	certPath := filepath.Join(generateFlags.output, "cert.pem")
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Write private key with restricted permissions (0600)
	keyPath := filepath.Join(generateFlags.output, "key.pem")
	keyFile, err := os.OpenFile(keyPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create key file: %w", err)
	}
	defer keyFile.Close()

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privateKeyBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Print summary
	fmt.Println()
	fmt.Println("Certificate Generation Summary:")
	fmt.Println("================================")
	fmt.Printf("Hosts: %s\n", generateFlags.hosts)
	if len(dnsNames) > 0 {
		fmt.Printf("  DNS Names: %v\n", dnsNames)
	}
	if len(ipAddresses) > 0 {
		fmt.Printf("  IP Addresses: %v\n", ipAddresses)
	}
	fmt.Printf("Organization: %s\n", generateFlags.org)
	fmt.Printf("Validity: %d days\n", generateFlags.validity)
	fmt.Printf("Key Size: %d bits\n", generateFlags.keySize)
	fmt.Printf("Not Before: %s\n", notBefore.Format("2006-01-02 15:04:05 MST"))
	fmt.Printf("Not After: %s\n", notAfter.Format("2006-01-02 15:04:05 MST"))
	fmt.Println()

	fmt.Printf("✓ Certificate generated: %s\n", certPath)
	fmt.Printf("✓ Private key generated: %s\n", keyPath)
	fmt.Println()

	fmt.Println("⚠️  WARNING: Self-signed certificates are for TESTING ONLY")
	fmt.Println("    Do not use in production!")
	fmt.Println()

	fmt.Println("To use with Mercator, add to your config.yaml:")
	fmt.Println("---")
	fmt.Println("security:")
	fmt.Println("  tls:")
	fmt.Println("    enabled: true")
	fmt.Printf("    cert_file: \"%s\"\n", certPath)
	fmt.Printf("    key_file: \"%s\"\n", keyPath)
	fmt.Println("    min_version: \"1.3\"")
	fmt.Println()

	fmt.Println("For production, use certificates from a trusted CA:")
	fmt.Println("  - Let's Encrypt (free, automated): https://letsencrypt.org/")
	fmt.Println("  - Commercial CAs: DigiCert, GlobalSign, Sectigo, etc.")

	return nil
}
