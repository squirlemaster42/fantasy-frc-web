package otel

import (
	"context"
	"server/log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

const asyncTracerName = "server"

// StartAsyncSpan starts a new OpenTelemetry span for work that outlives the
// incoming HTTP request. It detaches from the request context (so cancellation
// does not abort background work) while preserving the correlation ID and
// linking the new span back to the request span.
func StartAsyncSpan(ctx context.Context, name string) (context.Context, trace.Span) {
	detachedCtx := context.Background()
	if corrID := log.GetCorrelationID(ctx); corrID != "" {
		detachedCtx = log.WithCorrelationID(detachedCtx, corrID)
	}

	return otel.Tracer(asyncTracerName).Start(
		detachedCtx,
		name,
		trace.WithLinks(trace.LinkFromContext(ctx)),
	)
}
