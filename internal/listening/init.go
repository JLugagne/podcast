// Package listening is the composition root of the listening bounded context:
// it runs the migrations, builds the adapters, injects them into the app and
// mounts the HTTP routes under /api/v1/listening.
package listening

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/listening/app"
	"github.com/JLugagne/podcast/internal/listening/bridge/catalogadapter"
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
	httpinbound "github.com/JLugagne/podcast/internal/listening/inbound/http"
	"github.com/JLugagne/podcast/internal/listening/outbound/pgadapter"
	"github.com/JLugagne/podcast/internal/pkg/postgres"
)

// Deps are the shared resources of the listening context plus the catalog
// read-side use cases consumed by the bridge.
type Deps struct {
	Router         *mux.Router
	Pool           *pgxpool.Pool
	EpisodeQueries episodeuc.EpisodeQueries
	SeriesQueries  seriesuc.SeriesQueries
	Logger         *slog.Logger
	Now            func() time.Time
}

// Module is what other contexts may use from listening, exposed as ports only.
type Module struct {
	app *app.App
}

// ProgressCommands returns the write-side listening use cases.
func (m *Module) ProgressCommands() progressuc.ProgressCommands { return m.app }

// ProgressQueries returns the read-side listening use cases.
func (m *Module) ProgressQueries() progressuc.ProgressQueries { return m.app }

// Migrate applies the listening migrations; it is idempotent.
func Migrate(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	return postgres.Migrate(ctx, pool, pgadapter.Migrations(), "listening", log)
}

// New wires the listening context: it migrates the database, builds the
// repository and the catalog bridge, injects them into the app and mounts the
// HTTP routes on the router.
func New(ctx context.Context, d Deps) (*Module, error) {
	if d.Pool == nil {
		return nil, errors.New("listening: pool is required")
	}
	if d.Router == nil {
		return nil, errors.New("listening: router is required")
	}
	if d.EpisodeQueries == nil {
		return nil, errors.New("listening: episode queries are required")
	}
	if d.SeriesQueries == nil {
		return nil, errors.New("listening: series queries are required")
	}
	if err := Migrate(ctx, d.Pool, d.Logger); err != nil {
		return nil, err
	}

	application := app.New(app.Deps{
		Progress: pgadapter.NewProgressRepository(d.Pool),
		Catalog:  catalogadapter.New(d.EpisodeQueries, d.SeriesQueries),
		Logger:   d.Logger,
		Now:      d.Now,
	})

	httpinbound.SetupRouter(d.Router, application, httpinbound.Options{Logger: d.Logger})

	return &Module{app: application}, nil
}
