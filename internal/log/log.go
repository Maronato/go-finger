package log

import (
	"context"
	"io"
	"log/slog"

	"git.maronato.dev/maronato/finger/internal/config"
)

type loggerCtxKey struct{}

// NewLogger creates a new logger with the given debug level.
func NewLogger(w io.Writer, cfg *config.Config) *slog.Logger {
	level := slog.LevelInfo
	addSource := false

	if cfg.Debug {
		level = slog.LevelDebug
		addSource = true
	}

	return slog.New(
		slog.NewJSONHandler(w, &slog.HandlerOptions{
			Level:     level,
			AddSource: addSource,
		}),
	)
}

func FromContext(ctx context.Context) *slog.Logger {
	l, ok := ctx.Value(loggerCtxKey{}).(*slog.Logger)
	if !ok {
		panic("logger not found in context")
	}

	return l
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey{}, l)
}
