/*
Package security provides transport security (TLS/mTLS), secret management,
and authentication for Mercator Jupiter.

# TLS Configuration

Configure TLS for the proxy server:

	cfg := &tls.Config{
		Enabled:  true,
		CertFile: "/etc/mercator/certs/server.crt",
		KeyFile:  "/etc/mercator/certs/server.key",
		MinVersion: "1.3",
	}

	tlsConfig, err := cfg.ToTLSConfig()
	if err != nil {
		log.Fatal(err)
	}

# Secret Management

Load secrets from multiple providers:

	manager := secrets.NewManager([]secrets.SecretProvider{
		secrets.NewEnvProvider("MERCATOR_SECRET_"),
		secrets.NewFileProvider("/var/secrets", true),
	}, cacheConfig)

	apiKey, err := manager.GetSecret(ctx, "openai-api-key")
	if err != nil {
		log.Fatal(err)
	}

# API Key Authentication

Validate API keys in HTTP middleware:

	validator := auth.NewAPIKeyValidator(apiKeys)
	middleware := auth.NewAPIKeyMiddleware(validator, sources)

	http.Handle("/", middleware.Handle(handler))
*/
package security
