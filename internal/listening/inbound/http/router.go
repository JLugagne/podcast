// Package http is the listening context's HTTP inbound adapter. It mounts the
// /api/v1/listening routes and delegates every use case to the app.
package http

import (
	"log/slog"
	"net/http"

	"github.com/gorilla/mux"

	"github.com/JLugagne/podcast/internal/listening/inbound/http/commands"
	"github.com/JLugagne/podcast/internal/listening/inbound/http/queries"
	"github.com/JLugagne/podcast/internal/listening/inbound/http/shared"
	"github.com/JLugagne/podcast/internal/pkg/logger"
)

// App is the listening use-case surface the HTTP adapter needs. It is
// satisfied by *app.App.
type App interface {
	commands.App
	queries.App
}

// Options configures the listening HTTP adapter. A nil Logger disables request
// logging injection.
type Options struct {
	Logger *slog.Logger
}

// SetupRouter mounts every listening route on the /api/v1 subrouter of r.
func SetupRouter(r *mux.Router, app App, opts Options) {
	api := r.PathPrefix("/api/v1").Subrouter()
	api.Use(shared.InjectActor)
	if opts.Logger != nil {
		api.Use(withLogger(opts.Logger))
	}

	commands.SetupRouter(api, app)
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
