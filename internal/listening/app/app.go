// Package app implements the listening use-case ports by orchestrating the
// progress repository and the catalog port. Dependencies are injected through
// New.
package app

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport"
	"github.com/JLugagne/podcast/internal/listening/domain/repositories/progressrepo"
)

// Deps are the app's collaborators. Now defaults to time.Now and Logger to
// slog.Default.
type Deps struct {
	Progress progressrepo.ProgressRepository
	Catalog  catalogport.Catalog
	Logger   *slog.Logger
	Now      func() time.Time
}

// App implements every listening use-case port.
type App struct {
	progress progressrepo.ProgressRepository
	catalog  catalogport.Catalog
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
	return &App{progress: d.Progress, catalog: d.Catalog, log: log, now: now}
}

// requireActor rejects a request without a principal.
func requireActor(actor *domain.Actor) error {
	if actor == nil {
		return domain.ErrUnauthorized
	}
	return nil
}

// listenerOf derives the listener identity from the acting principal.
func listenerOf(actor *domain.Actor) domain.ListenerID { return domain.ListenerID(actor.ID()) }

// loadProgress returns the stored progress, or a fresh one when the listener
// never started the episode.
func (a *App) loadProgress(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef, now time.Time) (*domain.Progress, error) {
	progress, err := a.progress.Get(ctx, listener, episode)
	if errors.Is(err, domain.ErrProgressNotFound) {
		return domain.NewProgress(listener, episode, now), nil
	}
	if err != nil {
		return nil, err
	}
	return progress, nil
}
