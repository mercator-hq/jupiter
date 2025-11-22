/*
Package tls provides TLS and mTLS configuration for Mercator Jupiter.

# TLS Server Configuration

Enable TLS 1.3 for the HTTP proxy server:

	cfg := &tls.Config{
		Enabled:    true,
		CertFile:   "/etc/mercator/certs/server.crt",
		KeyFile:    "/etc/mercator/certs/server.key",
		MinVersion: "1.3",
		CipherSuites: []string{
			"TLS_AES_128_GCM_SHA256",
			"TLS_AES_256_GCM_SHA384",
		},
	}

	tlsConfig, err := cfg.ToTLSConfig()
	if err != nil {
		log.Fatal(err)
	}

# Mutual TLS (mTLS)

Enable client certificate authentication:

	cfg := &tls.Config{
		Enabled:  true,
		CertFile: "/etc/mercator/certs/server.crt",
		KeyFile:  "/etc/mercator/certs/server.key",
		MTLS: MTLSConfig{
			Enabled:          true,
			ClientCAFile:     "/etc/mercator/certs/client-ca.pem",
			ClientAuthType:   "require",
			VerifyClientCert: true,
			IdentitySource:   "subject.CN",
		},
	}

# Certificate Auto-Reload

Automatically reload certificates without server restart:

	reloader := NewCertificateReloader(certFile, keyFile, 5*time.Minute)
	if err := reloader.Start(ctx); err != nil {
		log.Fatal(err)
	}

	tlsConfig.GetCertificate = reloader.GetCertificateFunc()
*/
package tls
