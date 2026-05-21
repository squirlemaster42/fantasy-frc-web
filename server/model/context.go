package model

import (
	"context"
	"time"
)

// WithQueryTimeout wraps a context with a timeout for database operations.
// Callers should defer the returned cancel function.
func WithQueryTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(ctx, timeout)
}
