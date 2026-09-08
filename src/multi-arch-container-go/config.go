package main

import (
	"fmt"
	"strings"

	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// unknown is the placeholder used when a build provenance variable is absent,
// i.e. when running outside a container.
const unknown = "n/a"

// LogFormat is the console log output format.
type LogFormat string

const (
	// LogFormatText is human-readable console output. Best for local development.
	LogFormatText LogFormat = "text"
	// LogFormatJSON is newline-delimited JSON. Best for log shipping (Loki, Elasticsearch, OpenTelemetry).
	LogFormatJSON LogFormat = "json"
)

// AppConfig is bound from the "app" section of appsettings.json.
//
// Every value can be overridden by an environment variable using the same double-underscore
// section separator as the sibling repositories, e.g. APP__INTERVAL_SECONDS=10.
//
// Note: keys are snake_case because koanf lower-cases environment keys but preserves file keys
// verbatim - snake_case is the only casing where both sources resolve to the same key.
type AppConfig struct {
	// Greeting is written on every iteration of the worker loop.
	Greeting string `koanf:"greeting"`
	// IntervalSeconds is the delay between worker loop iterations.
	IntervalSeconds int `koanf:"interval_seconds"`
	// LogFormat selects the console log output format.
	LogFormat LogFormat `koanf:"log_format"`
}

// Settings is the root configuration object: application configuration plus the flat build
// provenance variables baked into the container image by the ARG/ENV block of the Dockerfile.
//
// Mirrors Models/_AppConfig.cs + Models/_BuildInfo.cs in the sibling .NET repository and
// config.rs in the sibling Rust repository.
type Settings struct {
	App AppConfig `koanf:"app"`

	GitRepository   string `koanf:"git_repository"`
	GitBranch       string `koanf:"git_branch"`
	GitCommit       string `koanf:"git_commit"`
	GitTag          string `koanf:"git_tag"`
	GitHubWorkflow  string `koanf:"github_workflow"`
	GitHubRunID     string `koanf:"github_run_id"`
	GitHubRunNumber string `koanf:"github_run_number"`
}

// defaultSettings returns the configuration used when neither the file nor the environment
// supplies a value. koanf only overwrites keys it actually finds, so these survive as defaults.
func defaultSettings() Settings {
	return Settings{
		App: AppConfig{
			Greeting:        "Hello from a multi-architecture container",
			IntervalSeconds: 3,
			LogFormat:       LogFormatText,
		},
		GitRepository:   unknown,
		GitBranch:       unknown,
		GitCommit:       unknown,
		GitTag:          unknown,
		GitHubWorkflow:  unknown,
		GitHubRunID:     unknown,
		GitHubRunNumber: unknown,
	}
}

// loadConfiguration layers configuration sources in ascending order of precedence:
// struct defaults -> appsettings.json (optional) -> environment variables.
//
// This mirrors Microsoft.Extensions.Configuration in the sibling .NET repository and the
// `config` crate in the sibling Rust repository.
func loadConfiguration(path string) (Settings, error) {
	settings := defaultSettings()
	k := koanf.New(".")

	// The file is optional so the binary runs unchanged outside a container.
	if err := k.Load(file.Provider(path), json.Parser()); err != nil {
		if !strings.Contains(err.Error(), "no such file") && !strings.Contains(err.Error(), "cannot find the file") {
			return settings, fmt.Errorf("loading %s: %w", path, err)
		}
	}

	// APP__GREETING -> app.greeting
	if err := k.Load(env.Provider(".", env.Opt{
		Prefix: "APP__",
		TransformFunc: func(key, value string) (string, any) {
			return appEnvKey(key), value
		},
	}), nil); err != nil {
		return settings, fmt.Errorf("loading APP__ environment variables: %w", err)
	}

	// GIT_TAG -> git_tag, GITHUB_RUN_ID -> github_run_id
	if err := k.Load(env.Provider(".", env.Opt{
		Prefix: "GIT",
		TransformFunc: func(key, value string) (string, any) {
			return strings.ToLower(key), value
		},
	}), nil); err != nil {
		return settings, fmt.Errorf("loading GIT environment variables: %w", err)
	}

	if err := k.Unmarshal("", &settings); err != nil {
		return settings, fmt.Errorf("unmarshalling configuration: %w", err)
	}

	return settings, nil
}

// appEnvKey maps an APP__ prefixed environment variable name onto a dotted configuration key.
func appEnvKey(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, "__", "."))
}
