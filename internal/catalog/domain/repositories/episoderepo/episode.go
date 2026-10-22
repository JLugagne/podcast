// Package episoderepo declares the outbound persistence port for episodes.
package episoderepo

import (
	"context"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

// Filter narrows ListBySeries. A nil Status returns every episode.
type Filter struct {
	Status *domain.EpisodeStatus
}

// EpisodeRepository persists episode aggregates. Update applies optimistic
// locking: it fails with domain.ErrVersionConflict when e.Version() no longer
// matches the stored version, and increments the stored version on success.
//
// Errors: Create returns domain.ErrEpisodeNumberTaken when the number is
// already used in the series. Update returns domain.ErrEpisodeNotFound,
// domain.ErrVersionConflict or domain.ErrEpisodeNumberTaken. Get and Delete
// return domain.ErrEpisodeNotFound. ListBySeries, CountBySeries and NextNumber
// return no error for an empty result.
type EpisodeRepository interface {
	Create(ctx context.Context, e *domain.Episode) error
	Update(ctx context.Context, e *domain.Episode) error
	Get(ctx context.Context, id domain.EpisodeID) (*domain.Episode, error)
	Delete(ctx context.Context, id domain.EpisodeID) error
	ListBySeries(ctx context.Context, seriesID domain.SeriesID, f Filter) ([]*domain.Episode, error)
	CountBySeries(ctx context.Context, seriesID domain.SeriesID) (int, error)
	NextNumber(ctx context.Context, seriesID domain.SeriesID) (int, error)
}
