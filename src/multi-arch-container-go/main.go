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
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// configFile is the optional configuration file, resolved relative to the working directory
// (/app inside the container).
const configFile = "appsettings.json"

func main() {
	// 1) Configuration: appsettings.json -> environment variables (environment wins).
	settings, err := loadConfiguration(configFile)
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// 2) Structured logging. Application code only ever calls the slog package functions, so the
	//    handler (text vs JSON, filtering, exporters) can be swapped without touching it.
	initLogger(settings.App)

	// 3) Shutdown signalling. NotifyContext traps SIGINT and the SIGTERM that `docker stop` and
	//    `kubectl delete pod` send, cancelling the context the worker selects on.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	slog.Info("Hit Ctrl-C to exit....")

	// 4) The worker itself.
	runWorker(ctx, settings)
}
