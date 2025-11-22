package cli

import (
	"context"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler creates a context that is canceled on SIGINT or SIGTERM.
func SetupSignalHandler() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		cancel()
	}()

	return ctx
}

// WaitForShutdown blocks until a shutdown signal is received.
func WaitForShutdown() <-chan os.Signal {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	return sigChan
}
