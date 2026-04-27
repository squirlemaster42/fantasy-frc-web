package log

import (
	"context"
	"log/slog"
	"os"
	"server/middleware"
)

var level = new(slog.LevelVar)

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

func InfoNoContext(msg string, args ...any) {
	slog.Default().Info(msg, args...)
}

func WarnNoContext(msg string, args ...any) {
	slog.Default().Warn(msg, args...)
}

func ErrorNoContext(msg string, args ...any) {
	slog.Default().Error(msg, args...)
}

func DebugNoContext(msg string, args ...any) {
	slog.Default().Debug(msg, args...)
}

func Fatal(msg string, args ...any) {
	slog.Default().Error(msg, args...)
	os.Exit(1)
}

func SetLevel(level slog.Level) {
	slog.SetLogLoggerLevel(level)
}

func getLogger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	return middleware.LogWithContext(ctx)
}
