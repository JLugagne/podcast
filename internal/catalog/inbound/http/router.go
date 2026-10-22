// Package http is the catalog context's HTTP inbound adapter. It mounts the
// public /api/v1 routes and delegates every use case to the app.
package http

import (
	"log/slog"
	"net/http"

	"github.com/JLugagne/podcast/internal/catalog/inbound/http/commands"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/queries"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/logger"
	"github.com/gorilla/mux"
)

// App is the catalog use-case surface the HTTP adapter needs. It is satisfied
// by *app.App.
type App interface {
	commands.App
	queries.App
}

// Options configures the catalog HTTP adapter. Non-positive upload limits fall
// back to the adapter defaults and a nil Logger to slog.Default.
type Options struct {
	MaxAudioBytes int64
	MaxImageBytes int64
	Logger        *slog.Logger
}

// SetupRouter mounts every catalog route on the /api/v1 subrouter of r.
func SetupRouter(r *mux.Router, app App, opts Options) {
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(shared.InjectActor)
	if opts.Logger != nil {
		api.Use(withLogger(opts.Logger))
	}

	commands.SetupRouter(api, app, commands.Options{
		MaxAudioBytes: opts.MaxAudioBytes,
		MaxImageBytes: opts.MaxImageBytes,
	})
	queries.SetupRouter(api, app)
}

// withLogger attaches l to every request context so handlers and the error
// mapper log through the configured logger.
func withLogger(l *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(logger.WithLogger(r.Context(), l)))
		})
	}
}
