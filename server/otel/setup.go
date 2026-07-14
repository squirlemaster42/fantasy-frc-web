package otel

import (
	"context"
	"server/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func InitTracer(serviceName string) func(context.Context) error {
	ctx := context.Background()

	// Create OTLP HTTP exporter
	exporter, err := otlptracehttp.New(ctx)
	if err != nil {
		log.Error(ctx, "Failed to create OTLP exporter", "error", err)
		return func(ctx context.Context) error { return nil }
	}

	// Create resource with service name, merging any OTEL_RESOURCE_ATTRIBUTES
	defaultRes := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceName(serviceName),
	)
	res, err := resource.Merge(resource.Environment(), defaultRes)
	if err != nil {
		log.Warn(ctx, "Failed to merge OTEL resource attributes", "error", err)
		res = defaultRes
	}

	// Create tracer provider
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	// Set global tracer provider and propagator
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Return shutdown function
	return tp.Shutdown
}
