package logger

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	otellog "go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// newResource builds the OpenTelemetry resource shared by the tracer and
// logger providers. It reads OTEL_SERVICE_NAME from the environment and adds
// host, OS and process attributes.
func newResource(ctx context.Context) (*resource.Resource, error) {
	return resource.New(ctx,
		resource.WithFromEnv(),
		resource.WithHost(),
		resource.WithOS(),
		resource.WithProcess(),
	)
}

// InitTracer configures the global OpenTelemetry TracerProvider and propagator.
// It returns a shutdown func that flushes pending spans; call it before the
// process exits. Endpoint, headers and service name are read from the standard
// OTEL_* environment variables.
func InitTracer(ctx context.Context) (func(context.Context) error, error) {
	// Reads OTEL_EXPORTER_OTLP_ENDPOINT and OTEL_EXPORTER_OTLP_HEADERS from environment
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		return nil, err
	}

	res, err := newResource(ctx)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Makes the tracer available to instrumentation libraries
	otel.SetTracerProvider(tp)

	// Propagates trace context across service boundaries using W3C standards
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp.Shutdown, nil
}

// InitLoggerProvider configures the global OpenTelemetry LoggerProvider so that
// logs emitted through the otelslog bridge are exported over OTLP (e.g. to
// SigNoz). It returns a shutdown func that flushes buffered records; call it
// before the process exits. Endpoint and headers are read from the standard
// OTEL_EXPORTER_OTLP_* environment variables.
func InitLoggerProvider(ctx context.Context) (func(context.Context) error, error) {
	exporter, err := otlploghttp.New(ctx)
	if err != nil {
		return nil, err
	}

	res, err := newResource(ctx)
	if err != nil {
		return nil, err
	}

	lp := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	// Makes the LoggerProvider available to the otelslog bridge via the global.
	otellog.SetLoggerProvider(lp)

	return lp.Shutdown, nil
}
