package log

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

var level = new(slog.LevelVar)

type contextKey string

const correlationIDKey contextKey = "correlation_id"
const CorrelationIDHeader = "X-Correlation-ID"

func SetupLogger(format string) {
	var handler slog.Handler
	if format == "text" {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}
	slog.SetDefault(slog.New(handler))
}

const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

func Info(ctx context.Context, msg string, args ...any) {
	getLogger(ctx).Info(msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	getLogger(ctx).Warn(msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	getLogger(ctx).Error(msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	getLogger(ctx).Debug(msg, args...)
}

func DebugNoContext(msg string, args ...any) {
	slog.Default().Debug(msg, args...)
}

func Fatal(ctx context.Context, msg string, args ...any) {
	getLogger(ctx).Error(msg, args...)
	os.Exit(1)
}

func SetLevel(l slog.Level) {
	level.Set(l)
}

func GetCorrelationID(ctx context.Context) string {
	if corrID, ok := ctx.Value(correlationIDKey).(string); ok {
		return corrID
	}
	return ""
}

func WithCorrelationID(ctx context.Context, corrID string) context.Context {
	return context.WithValue(ctx, correlationIDKey, corrID)
}

func NewCorrelationContext(ctx context.Context) context.Context {
	return WithCorrelationID(ctx, uuid.New().String())
}

func PropagateCorrelationID(ctx context.Context, req *http.Request) {
	if req == nil {
		return
	}
	if corrID := GetCorrelationID(ctx); corrID != "" {
		req.Header.Set(CorrelationIDHeader, corrID)
	}
}

func DetachCorrelationID(ctx context.Context) context.Context {
	detached := context.Background()
	if corrID := GetCorrelationID(ctx); corrID != "" {
		detached = WithCorrelationID(detached, corrID)
	}
	return detached
}

func LogWithContext(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	if ctx == nil {
		return logger
	}

	if corrID := GetCorrelationID(ctx); corrID != "" {
		logger = logger.With("correlationId", corrID)
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		logger = logger.With(
			"traceId", span.SpanContext().TraceID().String(),
			"spanId", span.SpanContext().SpanID().String(),
		)
	}

	return logger
}

func getLogger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	return LogWithContext(ctx)
}
