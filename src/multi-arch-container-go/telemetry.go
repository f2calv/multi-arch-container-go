package main

import (
	"log/slog"
	"os"
)

// initLogger installs the process-wide structured logger.
//
// log/slog is the Go standard library's structured logger - the analogue of Serilog in the
// sibling .NET repository and `tracing` in the sibling Rust repository.
//
// Verbosity is controlled by the conventional LOG_LEVEL environment variable
// (debug|info|warn|error) and defaults to info.
func initLogger(cfg AppConfig) *slog.Logger {
	level := slog.LevelInfo
	if value, ok := os.LookupEnv("LOG_LEVEL"); ok && value != "" {
		if err := level.UnmarshalText([]byte(value)); err != nil {
			level = slog.LevelInfo
		}
	}

	options := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.LogFormat == LogFormatJSON {
		handler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		handler = slog.NewTextHandler(os.Stdout, options)
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger
}
