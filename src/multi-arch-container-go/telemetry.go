package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

const instrumentationName = "github.com/f2calv/multi-arch-container-go"

type multiHandler struct {
	handlers []slog.Handler
}

// initTelemetry installs console logging and optional OTLP/HTTP exporters.
//
// log/slog is the Go standard library's structured logger - the analogue of Serilog in the
// sibling .NET repository and `tracing` in the sibling Rust repository.
//
// Verbosity is controlled by the conventional LOG_LEVEL environment variable
// (debug|info|warn|error) and defaults to info.
func initTelemetry(ctx context.Context, cfg AppConfig, version string) (func(context.Context) error, error) {
	level := slog.LevelInfo
	if value, ok := os.LookupEnv("LOG_LEVEL"); ok && value != "" {
		if err := level.UnmarshalText([]byte(value)); err != nil {
			level = slog.LevelInfo
		}
	}

	options := &slog.HandlerOptions{Level: level}

	var consoleHandler slog.Handler
	if cfg.LogFormat == LogFormatJSON {
		consoleHandler = slog.NewJSONHandler(os.Stdout, options)
	} else {
		consoleHandler = slog.NewTextHandler(os.Stdout, options)
	}

	if strings.TrimSpace(os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT")) == "" {
		slog.SetDefault(slog.New(consoleHandler))
		return func(context.Context) error { return nil }, nil
	}

	serviceName := os.Getenv("OTEL_SERVICE_NAME")
	if serviceName == "" {
		serviceName = "multi-arch-container-go"
	}

	telemetryResource, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", version),
		),
		resource.WithFromEnv(),
	)
	if err != nil {
		return nil, fmt.Errorf("creating OpenTelemetry resource: %w", err)
	}

	traceExporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("creating OTLP trace exporter: %w", err)
	}
	metricExporter, err := otlpmetrichttp.New(ctx)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("creating OTLP metric exporter: %w", err),
			traceExporter.Shutdown(ctx),
		)
	}
	logExporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, errors.Join(
			fmt.Errorf("creating OTLP log exporter: %w", err),
			metricExporter.Shutdown(ctx),
			traceExporter.Shutdown(ctx),
		)
	}

	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(telemetryResource),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(telemetryResource),
	)
	loggerProvider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(logExporter)),
		sdklog.WithResource(telemetryResource),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)
	slog.SetDefault(slog.New(multiHandler{handlers: []slog.Handler{
		consoleHandler,
		otelslog.NewHandler(instrumentationName, otelslog.WithLoggerProvider(loggerProvider)),
	}}))

	return func(shutdownCtx context.Context) error {
		return errors.Join(
			loggerProvider.Shutdown(shutdownCtx),
			meterProvider.Shutdown(shutdownCtx),
			tracerProvider.Shutdown(shutdownCtx),
		)
	}, nil
}

func (handler multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	for _, child := range handler.handlers {
		if child.Enabled(ctx, level) {
			return true
		}
	}

	return false
}

func (handler multiHandler) Handle(ctx context.Context, record slog.Record) error {
	var handleError error
	for _, child := range handler.handlers {
		if child.Enabled(ctx, record.Level) {
			handleError = errors.Join(handleError, child.Handle(ctx, record.Clone()))
		}
	}

	return handleError
}

func (handler multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	handlers := make([]slog.Handler, 0, len(handler.handlers))
	for _, child := range handler.handlers {
		handlers = append(handlers, child.WithAttrs(attrs))
	}

	return multiHandler{handlers: handlers}
}

func (handler multiHandler) WithGroup(name string) slog.Handler {
	handlers := make([]slog.Handler, 0, len(handler.handlers))
	for _, child := range handler.handlers {
		handlers = append(handlers, child.WithGroup(name))
	}

	return multiHandler{handlers: handlers}
}
