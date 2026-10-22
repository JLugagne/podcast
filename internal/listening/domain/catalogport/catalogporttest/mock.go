// Package catalogporttest provides function-based test doubles for the
// catalogport port. An unset func panics so unexpected calls surface.
package catalogporttest

import (
	"context"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport"
)

// MockCatalog is a function-field double of catalogport.Catalog.
type MockCatalog struct {
	PublishedEpisodeFn func(ctx context.Context, ref domain.EpisodeRef) (catalogport.EpisodeInfo, error)
}

var _ catalogport.Catalog = (*MockCatalog)(nil)

// PublishedEpisode calls PublishedEpisodeFn.
func (m *MockCatalog) PublishedEpisode(ctx context.Context, ref domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
	if m.PublishedEpisodeFn == nil {
		panic("MockCatalog.PublishedEpisodeFn not set")
	}
	return m.PublishedEpisodeFn(ctx, ref)
}
