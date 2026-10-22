// Package seriesuc declares the inbound use-case ports for series.
package seriesuc

import (
	"context"
	"io"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
)

// UploadInput is a streamed cover upload. Size limits are enforced by the
// inbound adapter.
type UploadInput struct {
	Reader      io.Reader
	ContentType string
}

// ListQuery narrows ListSeries. Query matches slug, title or author.
type ListQuery struct {
	Query  string
	Limit  int
	Offset int
}

// SeriesView is a series with derived read data.
type SeriesView struct {
	Series               *domain.Series
	EpisodeCount         int
	PublishedCount       int
	TotalDurationSeconds float64
}

// SeriesCommands mutate series.
type SeriesCommands interface {
	CreateSeries(ctx context.Context, actor *domain.Actor, attrs domain.SeriesAttributes) (*SeriesView, error)
	UpdateSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID, attrs domain.SeriesAttributes, expectedVersion int) (*SeriesView, error)
	DeleteSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID) error
	UploadCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID, in UploadInput) (*SeriesView, error)
	RemoveCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*SeriesView, error)
}

// SeriesQueries never mutate state.
type SeriesQueries interface {
	GetSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*SeriesView, error)
	// GetSeriesBySlug returns the series owning slug with its derived counters.
	GetSeriesBySlug(ctx context.Context, actor *domain.Actor, slug string) (*SeriesView, error)
	ListSeries(ctx context.Context, actor *domain.Actor, q ListQuery) ([]*SeriesView, error)
	OpenCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (blobstore.Blob, error)
}
