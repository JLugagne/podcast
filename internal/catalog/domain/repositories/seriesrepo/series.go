// Package seriesrepo declares the outbound persistence port for series.
package seriesrepo

import (
	"context"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

// Filter narrows List.
//
// Query matches the slug, title or author case-insensitively. Limit caps the
// number of returned rows (zero means no limit); Offset skips rows.
type Filter struct {
	Query  string
	Limit  int
	Offset int
}

// SeriesRepository persists series aggregates. Update applies optimistic
// locking: it fails with domain.ErrVersionConflict when s.Version() no longer
// matches the stored version, and increments the stored version on success.
//
// Errors: Create returns domain.ErrSlugTaken when the slug is already used.
// Update returns domain.ErrSeriesNotFound, domain.ErrVersionConflict or
// domain.ErrSlugTaken. Get and GetBySlug return domain.ErrSeriesNotFound.
// Delete returns domain.ErrSeriesNotFound, or domain.ErrSeriesNotEmpty when
// episodes still belong to the series.
type SeriesRepository interface {
	Create(ctx context.Context, s *domain.Series) error
	Update(ctx context.Context, s *domain.Series) error
	Get(ctx context.Context, id domain.SeriesID) (*domain.Series, error)
	GetBySlug(ctx context.Context, slug string) (*domain.Series, error)
	List(ctx context.Context, f Filter) ([]*domain.Series, error)
	Delete(ctx context.Context, id domain.SeriesID) error
}
