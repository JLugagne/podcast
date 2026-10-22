// Package catalog is the composition root of the catalog bounded context: it
// runs the migrations, builds the adapters, injects them into the app and
// mounts the HTTP routes under /api/v1.
package catalog

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/gorilla/mux"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JLugagne/podcast/internal/catalog/app"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/feed"
	httpinbound "github.com/JLugagne/podcast/internal/catalog/inbound/http"
	"github.com/JLugagne/podcast/internal/catalog/outbound/fsblob"
	"github.com/JLugagne/podcast/internal/catalog/outbound/pgadapter"
	"github.com/JLugagne/podcast/internal/pkg/postgres"
)

// Deps are the shared resources of the catalog context. Blobs overrides the
// default file-system store when set (useful for tests); it is also required
// when StorageDir is empty.
type Deps struct {
	Router        *mux.Router
	Pool          *pgxpool.Pool
	Blobs         blobstore.Store
	StorageDir    string
	MaxAudioBytes int64
	MaxImageBytes int64
	Logger        *slog.Logger
	Now           func() time.Time
	// PublicURL is the externally reachable base URL used for absolute feed links.
	PublicURL string
}

// Module is what other contexts may use from catalog, exposed as ports only.
type Module struct {
	app *app.App
}

// SeriesQueries returns the read-side series use cases.
func (m *Module) SeriesQueries() seriesuc.SeriesQueries { return m.app }

// EpisodeQueries returns the read-side episode use cases.
func (m *Module) EpisodeQueries() episodeuc.EpisodeQueries { return m.app }

// Migrate applies the catalog migrations; it is idempotent.
func Migrate(ctx context.Context, pool *pgxpool.Pool, log *slog.Logger) error {
	return postgres.Migrate(ctx, pool, pgadapter.Migrations(), "catalog", log)
}

// New wires the catalog context: it migrates the database, builds the
// repositories and the blob store, injects them into the app and mounts the
// HTTP routes on the router.
func New(ctx context.Context, d Deps) (*Module, error) {
	if d.Pool == nil {
		return nil, errors.New("catalog: pool is required")
	}
	if d.Router == nil {
		return nil, errors.New("catalog: router is required")
	}
	if err := Migrate(ctx, d.Pool, d.Logger); err != nil {
		return nil, err
	}

	blobs := d.Blobs
	if blobs == nil {
		store, err := fsblob.New(d.StorageDir)
		if err != nil {
			return nil, err
		}
		blobs = store
	}

	application := app.New(app.Deps{
		UoW:      pgadapter.NewUnitOfWork(d.Pool),
		Series:   pgadapter.NewSeriesRepository(d.Pool),
		Episodes: pgadapter.NewEpisodeRepository(d.Pool),
		Blobs:    blobs,
		Logger:   d.Logger,
		Now:      d.Now,
	})

	feed.SetupRouter(d.Router, application, feed.Options{
		PublicURL: d.PublicURL,
		Now:       d.Now,
	})
	httpinbound.SetupRouter(d.Router, application, httpinbound.Options{
		MaxAudioBytes: d.MaxAudioBytes,
		MaxImageBytes: d.MaxImageBytes,
		Logger:        d.Logger,
	})

	return &Module{app: application}, nil
}
