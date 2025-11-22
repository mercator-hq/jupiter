// Package server provides the main HTTP proxy server for LLM traffic.
package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/proxy/handlers"
	"mercator-hq/jupiter/pkg/proxy/middleware"
)

// Server is the main HTTP proxy server for LLM traffic.
type Server struct {
	config          *config.ProxyConfig
	securityConfig  *config.SecurityConfig
	httpServer      *http.Server
	providerManager ProviderManager
	shutdownChan    chan struct{}
	shutdownOnce    sync.Once
	mu              sync.RWMutex
	isRunning       bool
}

// ProviderManager is the interface for managing LLM providers.
type ProviderManager interface {
	GetProvider(name string) (providers.Provider, error)
	GetHealthyProviders() map[string]providers.Provider
	Close() error
}

// NewServer creates a new proxy server.
func NewServer(cfg *config.ProxyConfig, securityCfg *config.SecurityConfig, pm ProviderManager) *Server {
	return &Server{
		config:          cfg,
		securityConfig:  securityCfg,
		providerManager: pm,
		shutdownChan:    make(chan struct{}),
		isRunning:       false,
	}
}

// Start starts the HTTP server and blocks until shutdown.
func (s *Server) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		return fmt.Errorf("server is already running")
	}
	s.isRunning = true
	s.mu.Unlock()

	// Create router with middleware chain
	handler := s.setupRoutes()

	// Create HTTP server
	s.httpServer = &http.Server{
		Addr:           s.config.ListenAddress,
		Handler:        handler,
		ReadTimeout:    s.config.ReadTimeout,
		WriteTimeout:   s.config.WriteTimeout,
		IdleTimeout:    s.config.IdleTimeout,
		MaxHeaderBytes: s.config.MaxHeaderBytes,
	}

	// Configure TLS if enabled
	if s.securityConfig.TLS.Enabled {
		tlsConfig, err := s.configureTLS()
		if err != nil {
			return fmt.Errorf("failed to configure TLS: %w", err)
		}
		s.httpServer.TLSConfig = tlsConfig
	}

	// Start server in goroutine
	errChan := make(chan error, 1)
	go func() {
		slog.Info("starting proxy server",
			"address", s.config.ListenAddress,
			"tls_enabled", s.securityConfig.TLS.Enabled,
		)

		var err error
		if s.securityConfig.TLS.Enabled {
			err = s.httpServer.ListenAndServeTLS(
				s.securityConfig.TLS.CertFile,
				s.securityConfig.TLS.KeyFile,
			)
		} else {
			err = s.httpServer.ListenAndServe()
		}

		if err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Set up signal handlers
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for shutdown signal or error
	select {
	case <-ctx.Done():
		slog.Info("context cancelled, initiating shutdown")
		return s.Shutdown(context.Background())
	case sig := <-sigChan:
		slog.Info("received shutdown signal", "signal", sig.String())
		return s.Shutdown(context.Background())
	case err := <-errChan:
		return err
	case <-s.shutdownChan:
		slog.Info("shutdown requested")
		return s.Shutdown(context.Background())
	}
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	var shutdownErr error

	s.shutdownOnce.Do(func() {
		s.mu.Lock()
		if !s.isRunning {
			s.mu.Unlock()
			return
		}
		s.mu.Unlock()

		slog.Info("initiating graceful shutdown", "timeout", s.config.ShutdownTimeout.String())

		// Create shutdown context with timeout
		shutdownCtx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
		defer cancel()

		// Shutdown HTTP server
		if s.httpServer != nil {
			if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
				slog.Error("error during server shutdown", "error", err)
				shutdownErr = fmt.Errorf("server shutdown error: %w", err)
			}
		}

		s.mu.Lock()
		s.isRunning = false
		s.mu.Unlock()

		slog.Info("proxy server stopped")
	})

	return shutdownErr
}

// setupRoutes configures HTTP routes and middleware chain.
func (s *Server) setupRoutes() http.Handler {
	mux := http.NewServeMux()

	// Create handlers
	chatHandler := handlers.NewChatHandler(s.providerManager)
	healthHandler := handlers.NewHealthHandler()
	readyHandler := handlers.NewReadyHandler(s.providerManager)
	wsHandler := handlers.NewWebSocketHandler(s.providerManager)
	providerHealthHandler := handlers.NewProviderHealthHandler(s.providerManager)

	// Register routes
	mux.Handle("/v1/chat/completions", chatHandler)
	mux.Handle("/health", healthHandler)
	mux.Handle("/ready", readyHandler)
	mux.Handle("/health/providers", providerHealthHandler)
	mux.Handle("/v1/chat/completions/ws", wsHandler)

	// Apply middleware chain
	var handler http.Handler = mux

	// Timeout middleware
	handler = middleware.TimeoutMiddleware(s.config.WriteTimeout)(handler)

	// CORS middleware
	corsConfig := s.convertCORSConfig()
	handler = middleware.CORSMiddleware(corsConfig)(handler)

	// Request ID middleware
	handler = middleware.RequestIDMiddleware(handler)

	// Logging middleware
	handler = middleware.LoggingMiddleware(handler)

	// Recovery middleware (outermost)
	handler = middleware.RecoveryMiddleware(handler)

	return handler
}

// configureTLS configures TLS settings.
func (s *Server) configureTLS() (*tls.Config, error) {
	if s.securityConfig.TLS.CertFile == "" {
		return nil, fmt.Errorf("TLS cert file not specified")
	}

	if s.securityConfig.TLS.KeyFile == "" {
		return nil, fmt.Errorf("TLS key file not specified")
	}

	// Check if files exist
	if _, err := os.Stat(s.securityConfig.TLS.CertFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("TLS cert file not found: %s", s.securityConfig.TLS.CertFile)
	}

	if _, err := os.Stat(s.securityConfig.TLS.KeyFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("TLS key file not found: %s", s.securityConfig.TLS.KeyFile)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS13,
		CipherSuites: []uint16{
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	return tlsConfig, nil
}

// IsRunning returns true if the server is running.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// Handler returns the configured HTTP handler.
func (s *Server) Handler() http.Handler {
	return s.setupRoutes()
}

// Health performs a health check on the server.
func (s *Server) Health() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isRunning {
		return fmt.Errorf("server is not running")
	}

	// Check if at least one provider is healthy
	healthyProviders := s.providerManager.GetHealthyProviders()
	if len(healthyProviders) == 0 {
		return fmt.Errorf("no healthy providers available")
	}

	return nil
}

// convertCORSConfig converts config.CORSConfig to middleware.CORSConfig.
func (s *Server) convertCORSConfig() *middleware.CORSConfig {
	return &middleware.CORSConfig{
		Enabled:          s.config.CORS.Enabled,
		AllowedOrigins:   s.config.CORS.AllowedOrigins,
		AllowedMethods:   s.config.CORS.AllowedMethods,
		AllowedHeaders:   s.config.CORS.AllowedHeaders,
		ExposedHeaders:   s.config.CORS.ExposedHeaders,
		MaxAge:           s.config.CORS.MaxAge,
		AllowCredentials: s.config.CORS.AllowCredentials,
	}
}
