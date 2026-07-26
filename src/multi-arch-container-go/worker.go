package main

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// runWorker periodically logs runtime, configuration and build provenance information until the
// supplied context is cancelled.
//
// Mirrors Services/WorkerService.cs in the sibling .NET repository and worker.rs in the sibling
// Rust repository.
func runWorker(ctx context.Context, settings Settings) {
	app := settings.App

	slog.Info("worker started",
		"greeting", app.Greeting,
		"interval_seconds", app.IntervalSeconds,
		"log_format", app.LogFormat)

	ticker := time.NewTicker(time.Duration(app.IntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		slog.Info(app.Greeting,
			"app_name", appName(),
			"process_architecture", runtime.GOARCH,
			"os_description", runtime.GOOS,
			"go_version", runtime.Version())

		slog.Info("git provenance",
			"git_repository", settings.GitRepository,
			"git_branch", settings.GitBranch,
			"git_commit", settings.GitCommit,
			"git_tag", settings.GitTag)

		slog.Info("github provenance",
			"github_workflow", settings.GitHubWorkflow,
			"github_run_id", settings.GitHubRunID,
			"github_run_number", settings.GitHubRunNumber)

		select {
		case <-ctx.Done():
			slog.Info("worker stopping")
			return
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
