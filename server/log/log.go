package log

import (
	"context"
	"log/slog"
	"os"
	"server/middleware"
)

var defaultLogger = slog.Default()

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
	defaultLogger.Info(msg, args...)
}

func WarnNoContext(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

func ErrorNoContext(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

func DebugNoContext(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

func Fatal(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
	os.Exit(1)
}

func SetLevel(level slog.Level) {
	slog.SetLogLoggerLevel(level)
}

func getLogger(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return defaultLogger
	}
	return middleware.LogWithContext(ctx)
}
