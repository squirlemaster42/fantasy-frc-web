package handler

import (
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// setSpanAttrs extracts the current span from the Echo request context
// and sets the given attributes. It is a no-op if there is no active span.
func setSpanAttrs(c echo.Context, attrs ...attribute.KeyValue) {
	span := trace.SpanFromContext(c.Request().Context())
	if span.IsRecording() {
		span.SetAttributes(attrs...)
	}
}

// recordSpanError records an error on the current span and marks it as Error.
func recordSpanError(c echo.Context, err error) {
	span := trace.SpanFromContext(c.Request().Context())
	if span.IsRecording() && err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
}
