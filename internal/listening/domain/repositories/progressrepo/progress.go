// Package progressrepo declares the outbound persistence port for listening
// progress.
package progressrepo

import (
	"context"

	"github.com/JLugagne/podcast/internal/listening/domain"
)

// Filter narrows List. InProgressOnly keeps the rows that are started but not
// completed; a non-positive Limit means "every matching row".
type Filter struct {
	InProgressOnly bool
	Limit          int
}

// ProgressRepository persists listening progress, one row per (listener,
// episode) pair.
//
// Get returns domain.ErrProgressNotFound when no row matches. Save upserts, so
// it never conflicts on an existing row. Delete is idempotent: removing a row
// that does not exist is not an error. List returns the rows of one listener,
// most recently updated first.
type ProgressRepository interface {
	Get(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) (*domain.Progress, error)
	Save(ctx context.Context, progress *domain.Progress) error
	Delete(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) error
	List(ctx context.Context, listener domain.ListenerID, filter Filter) ([]*domain.Progress, error)
}
