package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

// runWorker periodically logs runtime, configuration and build provenance information until the
// supplied context is cancelled.
//
// Mirrors Services/WorkerService.cs in the sibling .NET repository and worker.rs in the sibling
// Rust repository.
func runWorker(ctx context.Context, settings Settings) error {
	app := settings.App
	meter := otel.Meter(instrumentationName)
	iterations, err := meter.Int64Counter("worker.iterations",
		metric.WithDescription("Number of completed worker iterations"),
		metric.WithUnit("{iteration}"))
	if err != nil {
		return fmt.Errorf("creating worker iteration counter: %w", err)
	}
	tracer := otel.Tracer(instrumentationName)

	slog.InfoContext(ctx, "worker started",
		"greeting", app.Greeting,
		"interval_seconds", app.IntervalSeconds,
		"log_format", app.LogFormat)

	ticker := time.NewTicker(time.Duration(app.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		iterationCtx, span := tracer.Start(ctx, "worker.iteration")

		slog.InfoContext(iterationCtx, app.Greeting,
			"app_name", appName(),
			"process_architecture", runtime.GOARCH,
			"os_description", runtime.GOOS,
			"go_version", runtime.Version())

		slog.InfoContext(iterationCtx, "git provenance",
			"git_repository", settings.GitRepository,
			"git_branch", settings.GitBranch,
			"git_commit", settings.GitCommit,
			"git_tag", settings.GitTag)

		slog.InfoContext(iterationCtx, "github provenance",
			"github_workflow", settings.GitHubWorkflow,
			"github_run_id", settings.GitHubRunID,
			"github_run_number", settings.GitHubRunNumber)

		iterations.Add(iterationCtx, 1)
		span.End()

		select {
		case <-ctx.Done():
			slog.InfoContext(ctx, "worker stopping")
			return nil
		case <-ticker.C:
		}
	}
}

// appName returns the file name of the running executable.
func appName() string {
	exe, err := os.Executable()
	if err != nil {
		return unknown
	}
	return filepath.Base(exe)
}
