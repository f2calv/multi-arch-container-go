package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAppEnvKeyMapsToDottedPath(t *testing.T) {
	got := appEnvKey("APP__INTERVAL_SECONDS")

	if want := "app.interval_seconds"; got != want {
		t.Errorf("appEnvKey() = %q, want %q", got, want)
	}
}

func TestDefaultSettingsArePopulated(t *testing.T) {
	settings := defaultSettings()

	if settings.App.IntervalSeconds != 3 {
		t.Errorf("IntervalSeconds = %d, want 3", settings.App.IntervalSeconds)
	}
	if settings.App.LogFormat != LogFormatText {
		t.Errorf("LogFormat = %q, want %q", settings.App.LogFormat, LogFormatText)
	}
	if settings.GitTag != unknown {
		t.Errorf("GitTag = %q, want %q", settings.GitTag, unknown)
	}
}

func TestLoadConfigurationEnvironmentOverridesDefaults(t *testing.T) {
	t.Setenv("APP__GREETING", "hello from a test")
	t.Setenv("APP__INTERVAL_SECONDS", "17")
	t.Setenv("GIT_TAG", "1.2.3")

	settings, err := loadConfiguration("does-not-exist.json")
	if err != nil {
		t.Fatalf("loadConfiguration() returned an error: %v", err)
	}

	if settings.App.Greeting != "hello from a test" {
		t.Errorf("Greeting = %q", settings.App.Greeting)
	}
	if settings.App.IntervalSeconds != 17 {
		t.Errorf("IntervalSeconds = %d, want 17", settings.App.IntervalSeconds)
	}
	if settings.GitTag != "1.2.3" {
		t.Errorf("GitTag = %q, want %q", settings.GitTag, "1.2.3")
	}
	if settings.GitBranch != unknown {
		t.Errorf("GitBranch = %q, want %q (unset variables keep their default)", settings.GitBranch, unknown)
	}
}

func TestLoadConfigurationRejectsInvalidAppSettings(t *testing.T) {
	tests := map[string]string{
		"blank greeting":     `{"app":{"greeting":" "}}`,
		"interval too small": `{"app":{"interval_seconds":0}}`,
		"interval too large": `{"app":{"interval_seconds":3601}}`,
		"unknown log format": `{"app":{"log_format":"xml"}}`,
	}

	for name, contents := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "appsettings.json")
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatalf("writing configuration: %v", err)
			}

			_, err := loadConfiguration(path)

			if err == nil || !strings.HasPrefix(err.Error(), "app.") {
				t.Errorf("loadConfiguration() error = %v, want app validation error", err)
			}
		})
	}
}

func TestAppNameIsNotEmpty(t *testing.T) {
	if got := appName(); got == "" {
		t.Error("appName() returned an empty string")
	}
}
