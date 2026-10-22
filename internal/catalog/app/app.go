// Package app implements the catalog use-case ports by orchestrating the
// repositories, the unit of work and the blob store. Dependencies are injected
// through New.
package app

import (
	"context"
	"log/slog"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
)

// Deps are the app's collaborators. Now defaults to time.Now and Logger to
// slog.Default.
type Deps struct {
	UoW      uow.UnitOfWork
	Series   seriesrepo.SeriesRepository
	Episodes episoderepo.EpisodeRepository
	Blobs    blobstore.Store
	Logger   *slog.Logger
	Now      func() time.Time
}

// App implements every catalog use-case port.
type App struct {
	uow      uow.UnitOfWork
	series   seriesrepo.SeriesRepository
	episodes episoderepo.EpisodeRepository
	blobs    blobstore.Store
	log      *slog.Logger
	now      func() time.Time
}

// New builds the application with the given collaborators.
func New(d Deps) *App {
	now := d.Now
	if now == nil {
		now = time.Now
	}
	log := d.Logger
	if log == nil {
		log = slog.Default()
	}
	return &App{
		uow:      d.UoW,
		series:   d.Series,
		episodes: d.Episodes,
		blobs:    d.Blobs,
		log:      log,
		now:      now,
	}
}

func requireActor(actor *domain.Actor) error {
	if actor == nil {
		return domain.ErrUnauthorized
	}
	return nil
}

// deleteBlob removes a blob after its transaction committed (or after a failed
// upload before the transaction). Failures are logged, never returned.
func (a *App) deleteBlob(ctx context.Context, reason, key string) {
	if key == "" {
		return
	}
	if err := a.blobs.Delete(ctx, key); err != nil {
		a.log.WarnContext(ctx, reason, "key", key, "error", err)
	}
}
