package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type ctxKey struct{}

// New creates a structured JSON (or text) logger based on level and format.
func New(level, format string) *slog.Logger {
	var handler slog.Handler
	opts := &slog.HandlerOptions{Level: parseLevel(level)}

	w := io.Writer(os.Stdout)
	switch strings.ToLower(format) {
	case "text":
		handler = slog.NewTextHandler(w, opts)
	default:
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithRequestID returns a child logger enriched with request_id.
func WithRequestID(log *slog.Logger, requestID string) *slog.Logger {
	if requestID == "" {
		return log
	}
	return log.With("request_id", requestID)
}

// IntoContext stores the logger in context.
func IntoContext(ctx context.Context, log *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, log)
}

// FromContext retrieves the logger from context, or returns a default JSON logger.
func FromContext(ctx context.Context) *slog.Logger {
	if v := ctx.Value(ctxKey{}); v != nil {
		if log, ok := v.(*slog.Logger); ok && log != nil {
			return log
		}
	}
	return slog.Default()
}
