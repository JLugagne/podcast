// Package logger builds slog loggers and carries them through request
// contexts.
package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// contextKey is the private type of the context key, so no other package can
// collide with it.
type contextKey struct{}

// New builds a slog logger writing to the first writer given, or os.Stdout
// when none is passed. Format is "json" or "text" (any other value falls back
// to text); level is any slog level name, such as "info" or "debug".
func New(format, level string, w ...io.Writer) (*slog.Logger, error) {
	out := io.Writer(os.Stdout)
	if len(w) > 0 && w[0] != nil {
		out = w[0]
	}
	var lvl slog.Level
	if level == "" {
		level = "info"
	}
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		return nil, fmt.Errorf("logger: parse level %q: %w", level, err)
	}
	opts := &slog.HandlerOptions{Level: lvl}
	var handler slog.Handler
	if format == "json" {
		handler = slog.NewJSONHandler(out, opts)
	} else {
		handler = slog.NewTextHandler(out, opts)
	}
	return slog.New(handler), nil
}

// WithLogger stores l in ctx. A nil logger is ignored.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	if l == nil {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, l)
}

// FromContext returns the logger stored in ctx, or slog.Default when there is
// none.
func FromContext(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(contextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}
