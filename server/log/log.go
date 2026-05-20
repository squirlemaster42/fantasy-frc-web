package log

import (
	"context"
	"log/slog"
	"os"
	"server/middleware"
	"time"

	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var level = new(slog.LevelVar)
var otelShutdown func(context.Context) error

func SetupLogger(format, serviceName, serviceVersion string) {
	var stdoutHandler slog.Handler
	if format == "text" {
		stdoutHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	} else {
		stdoutHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: level,
		})
	}

	otelHandler := otelslog.NewHandler(
		serviceName,
		otelslog.WithAttributes(
			attribute.String("service.name", serviceName),
			attribute.String("service.version", serviceVersion),
		),
	)

	handler := &multiHandler{
		stdout: stdoutHandler,
		otel:   otelHandler,
	}
	slog.SetDefault(slog.New(handler))
}

func SetShutdownFunc(fn func(context.Context) error) {
	otelShutdown = fn
}

const LevelDebug = slog.LevelDebug

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
	if otelShutdown != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = otelShutdown(shutdownCtx)
	}
	os.Exit(1)
}

func SetLevel(l slog.Level) {
	slog.SetLogLoggerLevel(l)
	level.Set(l)
}

func getLogger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	return middleware.LogWithContext(ctx)
}

// multiHandler writes log records to both a stdout handler and an OTel handler.
type multiHandler struct {
	stdout slog.Handler
	otel   slog.Handler
}

func (h *multiHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return h.stdout.Enabled(ctx, l) || h.otel.Enabled(ctx, l)
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	var attrs []slog.Attr

	if corrID := middleware.GetCorrelationID(ctx); corrID != "" {
		attrs = append(attrs, slog.String("correlation_id", corrID))
	}

	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		attrs = append(attrs,
			slog.String("trace_id", span.SpanContext().TraceID().String()),
			slog.String("span_id", span.SpanContext().SpanID().String()),
		)
	}

	if len(attrs) > 0 {
		r.AddAttrs(attrs...)
	}

	var errs []error
	if err := h.stdout.Handle(ctx, r); err != nil {
		errs = append(errs, err)
	}
	if err := h.otel.Handle(ctx, r); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &multiHandler{
		stdout: h.stdout.WithAttrs(attrs),
		otel:   h.otel.WithAttrs(attrs),
	}
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	return &multiHandler{
		stdout: h.stdout.WithGroup(name),
		otel:   h.otel.WithGroup(name),
	}
}
