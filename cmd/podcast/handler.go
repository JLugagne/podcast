package podcast

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JLugagne/podcast/internal/catalog"
	"github.com/JLugagne/podcast/internal/listening"
	"github.com/JLugagne/podcast/internal/pkg/httpjson"
	"github.com/JLugagne/podcast/internal/pkg/spa"
	"github.com/JLugagne/podcast/web/admin"
	"github.com/JLugagne/podcast/web/player"
)

// healthCheckTimeout bounds the database ping of GET /healthz.
const healthCheckTimeout = 2 * time.Second

// HandlerOptions configures the complete HTTP handler served by the service. Now
// is an optional clock override, which lets tests inject a fixed time.
type HandlerOptions struct {
	Ctx           context.Context
	Pool          *pgxpool.Pool
	StorageDir    string
	MaxAudioBytes int64
	MaxImageBytes int64
	// PublicURL is the externally reachable base URL used for absolute feed links.
	PublicURL string
	Logger    *slog.Logger
	Now       func() time.Time
}

// NewHandler builds the complete handler of the serve command: the catalog API
// under /api/v1, the admin single-page application under /admin, the listener
// single-page application at / and the /healthz probe. Tests use it so the e2e
// suite exercises the real serve wiring.
func NewHandler(opts HandlerOptions) (http.Handler, error) {
	router := mux.NewRouter()
	catalogModule, err := catalog.New(opts.Ctx, catalog.Deps{
		Router:        router,
		PublicURL:     opts.PublicURL,
		Pool:          opts.Pool,
		StorageDir:    opts.StorageDir,
		MaxAudioBytes: opts.MaxAudioBytes,
		MaxImageBytes: opts.MaxImageBytes,
		Logger:        opts.Logger,
		Now:           opts.Now,
	})
	if err != nil {
		return nil, err
	}
	if _, err := listening.New(opts.Ctx, listening.Deps{
		Router:         router,
		Pool:           opts.Pool,
		EpisodeQueries: catalogModule.EpisodeQueries(),
		SeriesQueries:  catalogModule.SeriesQueries(),
		Logger:         opts.Logger,
		Now:            opts.Now,
	}); err != nil {
		return nil, err
	}

	router.HandleFunc("/healthz", healthHandler(opts.Pool)).Methods(http.MethodGet, http.MethodHead)
	router.PathPrefix("/admin").Handler(spa.Handler(admin.Dist(), "/admin"))
	// Unknown API and feed paths answer with the regular JSON error instead of
	// falling through to the player single-page application.
	router.PathPrefix("/api/").HandlerFunc(notFoundJSON)
	router.PathPrefix("/feeds/").HandlerFunc(notFoundJSON)
	router.PathPrefix("/").Handler(spa.Handler(player.Dist(), "/"))

	return router, nil
}

// healthHandler reports whether the database is reachable.
func healthHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()
		if pool == nil || pool.Ping(ctx) != nil {
			httpjson.Error(w, r, http.StatusServiceUnavailable, "UNHEALTHY", "database unreachable")
			return
		}
		_ = httpjson.Respond(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

// notFoundJSON answers unknown API and feed paths with the regular JSON error
// body instead of letting the player single-page application serve them.
func notFoundJSON(w http.ResponseWriter, r *http.Request) {
	httpjson.Error(w, r, http.StatusNotFound, "NOT_FOUND", "resource not found")
}
