package log

import (
	"context"
	"server/middleware"
)

func Info(ctx context.Context, msg string, args ...any) {
	middleware.LogWithContext(ctx).Info(msg, args...)
}

func Warn(ctx context.Context, msg string, args ...any) {
	middleware.LogWithContext(ctx).Warn(msg, args...)
}

func Error(ctx context.Context, msg string, args ...any) {
	middleware.LogWithContext(ctx).Error(msg, args...)
}

func Debug(ctx context.Context, msg string, args ...any) {
	middleware.LogWithContext(ctx).Debug(msg, args...)
}
