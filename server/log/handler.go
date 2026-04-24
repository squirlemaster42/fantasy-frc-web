package log

import (
	"context"
	"log/slog"
	"os"
	"time"
)

type dualHandler struct {
	jsonHandler slog.Handler
	buffer      *RingBuffer
	level       slog.Level
}

func NewDualHandler(buffer *RingBuffer, level slog.Level) *dualHandler {
	jsonHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format(time.RFC3339))
			}
			return a
		},
	})

	return &dualHandler{
		jsonHandler: jsonHandler,
		buffer:      buffer,
		level:       level,
	}
}

func (h *dualHandler) Handle(ctx context.Context, r slog.Record) error {
	if !h.Enabled(ctx, r.Level) {
		return nil
	}

	if err := h.jsonHandler.Handle(ctx, r); err != nil {
		return err
	}

	entry := LogEntry{
		Time:  r.Time,
		Level: r.Level,
		Msg:   r.Message,
		Attrs: make(map[string]any),
	}

	r.Attrs(func(a slog.Attr) bool {
		entry.Attrs[a.Key] = a.Value.Any()
		return true
	})

	h.buffer.Add(entry)
	return nil
}

func (h *dualHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return level >= h.level && h.jsonHandler.Enabled(ctx, level)
}

func (h *dualHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &dualHandler{
		jsonHandler: h.jsonHandler.WithAttrs(attrs),
		buffer:      h.buffer,
		level:       h.level,
	}
}

func (h *dualHandler) WithGroup(name string) slog.Handler {
	return &dualHandler{
		jsonHandler: h.jsonHandler.WithGroup(name),
		buffer:      h.buffer,
		level:       h.level,
	}
}

func (h *dualHandler) SetLevel(level slog.Level) {
	h.level = level
}
