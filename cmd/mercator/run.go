package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/spf13/cobra"
	"mercator-hq/jupiter/pkg/cli"
	"mercator-hq/jupiter/pkg/config"
	"mercator-hq/jupiter/pkg/evidence"
	"mercator-hq/jupiter/pkg/evidence/recorder"
	"mercator-hq/jupiter/pkg/evidence/retention"
	"mercator-hq/jupiter/pkg/evidence/storage"
	"mercator-hq/jupiter/pkg/policy/engine"
	"mercator-hq/jupiter/pkg/policy/engine/source"
	"mercator-hq/jupiter/pkg/providerfactory"
	"mercator-hq/jupiter/pkg/providers"
	"mercator-hq/jupiter/pkg/server"
)

var runFlags struct {
	listenAddress string
	logLevel      string
	dryRun        bool
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start the Mercator proxy server",
	Long: `Start the Mercator proxy server with the specified configuration.

The server listens on the configured address and proxies LLM API requests through
the policy engine, evidence recorder, and routing system.

Examples:
  # Start with default config
  mercator run

  # Start with custom config
  mercator run --config /etc/mercator/config.yaml

  # Override listen address
  mercator run --listen 0.0.0.0:8080

  # Validate config without starting server
  mercator run --dry-run`,
	RunE: runServer,
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().StringVarP(&runFlags.listenAddress, "listen", "l", "", "override listen address")
	runCmd.Flags().StringVar(&runFlags.logLevel, "log-level", "", "override log level (debug, info, warn, error)")
	runCmd.Flags().BoolVar(&runFlags.dryRun, "dry-run", false, "validate config without starting server")
}

func runServer(cmd *cobra.Command, args []string) error {
	// Load configuration
	if err := config.Initialize(cfgFile); err != nil {
		return cli.NewConfigError("", fmt.Sprintf("failed to load config: %v", err))
	}
	cfg := config.GetConfig()

	// Apply flag overrides
	if runFlags.listenAddress != "" {
		cfg.Proxy.ListenAddress = runFlags.listenAddress
	}
	if runFlags.logLevel != "" {
		cfg.Telemetry.Logging.Level = runFlags.logLevel
	}

	// Initialize logging based on config
	var logLevel slog.Level
	switch cfg.Telemetry.Logging.Level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	if runFlags.dryRun {
		fmt.Println("✓ Configuration valid")
		return nil
	}

	// Print startup banner
	printBanner(cfg)

	// Create provider manager
	slog.Info("initializing provider manager")
	manager := providerfactory.NewManager()
	defer manager.Close()

	// Convert provider configs to slice for loading
	providerConfigs := make([]providers.ProviderConfig, 0, len(cfg.Providers))
	for name, providerCfg := range cfg.Providers {
		pc := providers.ProviderConfig{
			Name:       name,
			Type:       name,
			BaseURL:    providerCfg.BaseURL,
			APIKey:     providerCfg.APIKey,
			Timeout:    providerCfg.Timeout,
			MaxRetries: providerCfg.MaxRetries,
		}
		providerConfigs = append(providerConfigs, pc)
	}

	if len(providerConfigs) > 0 {
		if err := manager.LoadFromConfig(providerConfigs); err != nil {
			slog.Warn("some providers failed to initialize", "error", err)
		}
	} else {
		slog.Warn("no providers configured")
	}

	fmt.Printf("✓ Providers initialized (%d providers)\n", manager.ProviderCount())

	// Initialize policy engine (if mode is file and file exists)
	var policyEngine *engine.InterpreterEngine
	if cfg.Policy.Mode == "file" && cfg.Policy.FilePath != "" {
		slog.Info("initializing policy engine",
			"mode", cfg.Policy.Mode,
			"policy_path", cfg.Policy.FilePath,
		)

		policySource := source.NewFileSource(cfg.Policy.FilePath, logger)
		engineConfig := engine.DefaultEngineConfig()
		engineConfig.EnableTrace = true
		engineConfig.FailSafeMode = engine.FailOpen
		engineConfig.DefaultAction = engine.ActionAllow

		var err error
		policyEngine, err = engine.NewInterpreterEngine(engineConfig, policySource, logger)
		if err != nil {
			slog.Warn("failed to initialize policy engine", "error", err)
		} else {
			defer policyEngine.Close()
			fmt.Printf("✓ Policy engine loaded (%d policies)\n", len(policyEngine.GetPolicies()))
		}
	}

	// Initialize evidence recording (if enabled)
	var evidenceRecorder *recorder.Recorder
	var pruner *retention.Pruner
	if cfg.Evidence.Enabled {
		slog.Info("initializing evidence recording",
			"backend", cfg.Evidence.Backend,
		)

		var evidenceStorage evidence.Storage
		var err error
		switch cfg.Evidence.Backend {
		case "sqlite":
			sqliteConfig := &storage.SQLiteConfig{
				Path:         cfg.Evidence.SQLite.Path,
				MaxOpenConns: cfg.Evidence.SQLite.MaxOpenConns,
				MaxIdleConns: cfg.Evidence.SQLite.MaxIdleConns,
				WALMode:      cfg.Evidence.SQLite.WALMode,
				BusyTimeout:  cfg.Evidence.SQLite.BusyTimeout,
			}
			evidenceStorage, err = storage.NewSQLiteStorage(sqliteConfig)
			if err != nil {
				return fmt.Errorf("failed to create SQLite storage: %w", err)
			}
		case "memory":
			evidenceStorage = storage.NewMemoryStorage()
		default:
			return fmt.Errorf("unsupported evidence backend: %s", cfg.Evidence.Backend)
		}
		defer evidenceStorage.Close()

		recorderConfig := &recorder.Config{
			Enabled:        true,
			AsyncBuffer:    cfg.Evidence.Recorder.AsyncBuffer,
			WriteTimeout:   cfg.Evidence.Recorder.WriteTimeout,
			HashRequest:    cfg.Evidence.Recorder.HashRequest,
			HashResponse:   cfg.Evidence.Recorder.HashResponse,
			RedactAPIKeys:  cfg.Evidence.Recorder.RedactAPIKeys,
			MaxFieldLength: cfg.Evidence.Recorder.MaxFieldLength,
		}
		evidenceRecorder = recorder.NewRecorder(evidenceStorage, recorderConfig)
		defer evidenceRecorder.Close()

		// Start retention pruner if schedule is configured
		if cfg.Evidence.Retention.PruneSchedule != "" {
			retentionConfig := &retention.Config{
				RetentionDays:       cfg.Evidence.Retention.Days,
				PruneSchedule:       cfg.Evidence.Retention.PruneSchedule,
				ArchiveBeforeDelete: cfg.Evidence.Retention.ArchiveBeforeDelete,
				ArchivePath:         cfg.Evidence.Retention.ArchivePath,
				MaxRecords:          cfg.Evidence.Retention.MaxRecords,
			}
			pruner = retention.NewPruner(evidenceStorage, retentionConfig)
			ctx := context.Background()
			if err := pruner.Start(ctx); err != nil {
				slog.Warn("failed to start retention scheduler", "error", err)
			} else {
				defer pruner.Stop()
				if next := pruner.NextPruning(); next != nil {
					slog.Debug("evidence retention scheduler started", "next_pruning", next)
				}
			}
		}

		fmt.Println("✓ Evidence store initialized")
	}

	// Create HTTP server
	slog.Info("creating HTTP server")
	srv := server.NewServer(&cfg.Proxy, &cfg.Security, manager)

	// Start server in background goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		slog.Info("starting HTTP server",
			"address", cfg.Proxy.ListenAddress,
			"tls_enabled", cfg.Security.TLS.Enabled,
		)
		if err := srv.Start(ctx); err != nil {
			errChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// Wait for server to be ready
	if err := waitForServerReady(cfg.Proxy.ListenAddress, 5*time.Second); err != nil {
		return fmt.Errorf("server failed to start: %w", err)
	}

	fmt.Println()
	fmt.Printf("✓ Server listening on %s\n", cfg.Proxy.ListenAddress)
	fmt.Printf("✓ Health endpoint: http://%s/health\n", cfg.Proxy.ListenAddress)
	fmt.Printf("✓ Metrics endpoint: http://%s/metrics\n", cfg.Proxy.ListenAddress)
	fmt.Println("\nPress Ctrl+C to stop")

	// Wait for shutdown signal or server error
	sigChan := cli.WaitForShutdown()

	select {
	case err := <-errChan:
		return cli.NewCommandError("run", err)
	case sig := <-sigChan:
		fmt.Printf("\nReceived signal %s, shutting down gracefully...\n", sig)
		cancel()

		// Graceful shutdown with timeout
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Proxy.ShutdownTimeout)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown failed", "error", err)
			return cli.NewCommandError("run", err)
		}

		fmt.Println("✓ Server stopped")
		return nil
	}
}

func printBanner(cfg *config.Config) {
	fmt.Printf("Mercator Jupiter v%s\n", Version)
	fmt.Printf("Loading configuration from: %s\n", cfgFile)
	fmt.Println("✓ Configuration loaded")

	// Count providers
	providerCount := len(cfg.Providers)
	if providerCount > 0 {
		slog.Debug("providers configured", "count", providerCount)
	}

	// Policy info
	if cfg.Policy.Mode == "file" {
		slog.Debug("policy mode", "mode", "file", "path", cfg.Policy.FilePath)
	} else if cfg.Policy.Mode == "git" {
		slog.Debug("policy mode", "mode", "git")
	}

	// Evidence info
	if cfg.Evidence.Enabled {
		slog.Debug("evidence enabled", "backend", cfg.Evidence.Backend)
	}
}

func waitForServerReady(address string, timeout time.Duration) error {
	// Simple delay for MVP - in production this should poll the health endpoint
	time.Sleep(100 * time.Millisecond)
	return nil
}
