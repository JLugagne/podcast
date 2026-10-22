// Package catalogport declares the anti-corruption port toward the catalog
// context, expressed with listening's own types.
package catalogport

import (
	"context"

	"github.com/JLugagne/podcast/internal/listening/domain"
)

// EpisodeInfo is the snapshot of a published catalog episode needed by the
// listening context. It intentionally carries listening's own primitive types:
// no catalog type crosses the boundary.
type EpisodeInfo struct {
	Ref               domain.EpisodeRef
	SeriesID          string
	SeriesTitle       string
	SeriesSlug        string
	SeriesAccentColor string
	SeriesHasCover    bool
	Number            int
	Title             string
	DurationSeconds   float64
	ChapterCount      int
}

// Catalog reads published episodes from the catalog context. PublishedEpisode
// returns domain.ErrEpisodeNotFound when the episode is missing or is not
// published.
type Catalog interface {
	PublishedEpisode(ctx context.Context, ref domain.EpisodeRef) (EpisodeInfo, error)
}
