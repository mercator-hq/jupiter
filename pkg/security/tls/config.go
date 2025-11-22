package tls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"
)

// Config represents TLS configuration for the Mercator proxy server.
// It supports TLS 1.2 and 1.3, configurable cipher suites, and optional mTLS.
type Config struct {
	// Enabled indicates whether TLS should be used
	Enabled bool `yaml:"enabled"`

	// CertFile is the path to the PEM-encoded certificate file
	CertFile string `yaml:"cert_file"`

	// KeyFile is the path to the PEM-encoded private key file
	KeyFile string `yaml:"key_file"`

	// MinVersion is the minimum TLS version to accept ("1.2" or "1.3")
	// Default: "1.3"
	MinVersion string `yaml:"min_version"`

	// CipherSuites is a list of enabled cipher suites
	// If empty, Go's default secure cipher suites are used
	CipherSuites []string `yaml:"cipher_suites"`

	// ReloadInterval is how often to check for certificate changes
	// Format: "5m", "1h", etc.
	// Default: "5m"
	ReloadInterval string `yaml:"cert_reload_interval"`

	// MTLS contains mutual TLS configuration
	MTLS MTLSConfig `yaml:"mtls"`
}

// MTLSConfig represents mutual TLS (client certificate authentication) configuration.
type MTLSConfig struct {
	// Enabled indicates whether mTLS should be used
	Enabled bool `yaml:"enabled"`

	// ClientCAFile is the path to the PEM-encoded CA certificate file
	// used to verify client certificates
	ClientCAFile string `yaml:"client_ca_file"`

	// ClientAuthType specifies how to handle client certificates:
	// - "require": client certificate required, reject if missing
	// - "request": request client certificate, but allow if missing
	// - "verify_if_given": verify client cert if provided, allow if not
	// Default: "require"
	ClientAuthType string `yaml:"client_auth_type"`

	// VerifyClientCert indicates whether to verify client certificates
	// against the CA
	VerifyClientCert bool `yaml:"verify_client_cert"`

	// IdentitySource specifies how to extract client identity from certificate:
	// - "subject.CN": Common Name from Subject
	// - "subject.OU": Organizational Unit from Subject
	// - "subject.O": Organization from Subject
	// - "SAN": First DNS name from Subject Alternative Names
	// Default: "subject.CN"
	IdentitySource string `yaml:"identity_source"`
}

// ToTLSConfig converts Config to crypto/tls.Config.
// It loads certificates, configures TLS versions and cipher suites,
// and sets up mTLS if enabled.
func (c *Config) ToTLSConfig() (*tls.Config, error) {
	if !c.Enabled {
		return nil, nil
	}

	// Validate required fields
	if c.CertFile == "" {
		return nil, fmt.Errorf("cert_file is required when TLS is enabled")
	}
	if c.KeyFile == "" {
		return nil, fmt.Errorf("key_file is required when TLS is enabled")
	}

	// Validate certificate files exist
	if _, err := os.Stat(c.CertFile); err != nil {
		return nil, fmt.Errorf("certificate file not found: %s: %w", c.CertFile, err)
	}
	if _, err := os.Stat(c.KeyFile); err != nil {
		return nil, fmt.Errorf("key file not found: %s: %w", c.KeyFile, err)
	}

	// Load certificate and key
	cert, err := tls.LoadX509KeyPair(c.CertFile, c.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	// Validate certificate
	if err := ValidateCertificate(&cert); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %w", err)
	}

	// Create TLS config
	// #nosec G402 - MinVersion is configurable and validated (TLS 1.0/1.1 rejected)
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   c.parseTLSVersion(),
		CipherSuites: c.parseCipherSuites(),
	}

	// Configure mTLS if enabled
	if c.MTLS.Enabled {
		if err := c.configureMTLS(tlsConfig); err != nil {
			return nil, fmt.Errorf("failed to configure mTLS: %w", err)
		}
	}

	return tlsConfig, nil
}

// parseTLSVersion converts the MinVersion string to a tls.Version constant.
// Supported versions: "1.3" (default), "1.2"
// TLS 1.0 and 1.1 are not supported due to security concerns.
func (c *Config) parseTLSVersion() uint16 {
	switch c.MinVersion {
	case "1.2":
		return tls.VersionTLS12
	case "1.3", "":
		return tls.VersionTLS13
	default:
		// Default to TLS 1.3 for unknown versions
		return tls.VersionTLS13
	}
}

// parseCipherSuites converts cipher suite names to tls.CipherSuite constants.
// If no cipher suites are specified, returns nil to use Go's secure defaults.
func (c *Config) parseCipherSuites() []uint16 {
	if len(c.CipherSuites) == 0 {
		return nil // Use Go's secure defaults
	}

	var suites []uint16
	for _, suite := range c.CipherSuites {
		if id, ok := cipherSuiteMap[suite]; ok {
			suites = append(suites, id)
		}
	}

	return suites
}

// ParseReloadInterval parses the ReloadInterval string into a time.Duration.
// Returns 5 minutes as default if not specified or invalid.
func (c *Config) ParseReloadInterval() time.Duration {
	if c.ReloadInterval == "" {
		return 5 * time.Minute
	}

	duration, err := time.ParseDuration(c.ReloadInterval)
	if err != nil {
		return 5 * time.Minute
	}

	return duration
}

// cipherSuiteMap maps cipher suite names to their tls package constants.
// Only secure cipher suites are included.
var cipherSuiteMap = map[string]uint16{
	// TLS 1.3 cipher suites (always enabled, cannot be disabled)
	"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
	"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
	"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,

	// TLS 1.2 cipher suites (secure options only)
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305":    tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305":  tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
}

// configureMTLS sets up mutual TLS on the provided tls.Config.
// It loads the client CA certificate and configures client authentication.
func (c *Config) configureMTLS(tlsConfig *tls.Config) error {
	if c.MTLS.ClientCAFile == "" {
		return fmt.Errorf("client_ca_file is required when mTLS is enabled")
	}

	// Load client CA certificate
	caCert, err := os.ReadFile(c.MTLS.ClientCAFile)
	if err != nil {
		return fmt.Errorf("failed to read client CA: %w", err)
	}

	// Create CA pool
	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse client CA certificate")
	}

	// Configure client certificate verification
	tlsConfig.ClientCAs = caCertPool
	tlsConfig.ClientAuth = c.parseClientAuthType()

	return nil
}

// parseClientAuthType converts the ClientAuthType string to a tls.ClientAuthType constant.
func (c *Config) parseClientAuthType() tls.ClientAuthType {
	switch c.MTLS.ClientAuthType {
	case "require":
		return tls.RequireAndVerifyClientCert
	case "request":
		return tls.RequestClientCert
	case "verify_if_given":
		return tls.VerifyClientCertIfGiven
	default:
		// Default to requiring and verifying client certificates
		return tls.RequireAndVerifyClientCert
	}
}
