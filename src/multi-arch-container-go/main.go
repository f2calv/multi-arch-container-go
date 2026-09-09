// Package main is a multi-architecture container demonstrator.
//
// A trivial worker process, implemented identically in four languages:
//   - https://github.com/f2calv/multi-arch-container-dotnet
//   - https://github.com/f2calv/multi-arch-container-go (this one)
//   - https://github.com/f2calv/multi-arch-container-rust
//   - https://github.com/f2calv/multi-arch-container-python
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// configFile is the optional configuration file, resolved relative to the working directory
// (/app inside the container).
const configFile = "appsettings.json"

func main() {
	if err := run(); err != nil {
		slog.Error("application error", "error", err)
		os.Exit(1)
	}
}

func run() (runError error) {
	// 1) Configuration: appsettings.json -> environment variables (environment wins).
	settings, err := loadConfiguration(configFile)
	if err != nil {
		return fmt.Errorf("loading configuration: %w", err)
	}

	// 2) Shutdown signalling. NotifyContext traps SIGINT and the SIGTERM that `docker stop` and
	//    `kubectl delete pod` send, cancelling the context the worker selects on.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 3) Structured logging and optional OpenTelemetry export. Application code only ever calls
	//    slog, so exporters can be swapped without touching worker code.
	shutdownTelemetry, err := initTelemetry(ctx, settings.App, settings.GitTag)
	if err != nil {
		return fmt.Errorf("initializing telemetry: %w", err)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		runError = errors.Join(runError, shutdownTelemetry(shutdownCtx))
	}()

	slog.InfoContext(ctx, "Hit Ctrl-C to exit....")

	// 4) The worker itself.
	return runWorker(ctx, settings)
}
