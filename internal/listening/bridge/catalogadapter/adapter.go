// Package catalogadapter is the anti-corruption layer between the listening
// context and catalog: it implements listening's catalog port on top of the
// catalog read-side use cases, translating types and errors in both directions.
package catalogadapter

import (
	"context"
	"errors"

	catalogdomain "github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport"
)

// Adapter reads published catalog episodes for the listening context. It is
// safe for concurrent use.
type Adapter struct {
	episodes episodeuc.EpisodeQueries
	series   seriesuc.SeriesQueries
}

var _ catalogport.Catalog = (*Adapter)(nil)

// New builds the adapter from the catalog read-side use cases.
func New(episodes episodeuc.EpisodeQueries, series seriesuc.SeriesQueries) *Adapter {
	return &Adapter{episodes: episodes, series: series}
}

// PublishedEpisode returns the snapshot of a published episode and of its
// series. A malformed reference, a missing episode, an unpublished episode and
// a missing series all become domain.ErrEpisodeNotFound.
func (a *Adapter) PublishedEpisode(ctx context.Context, ref domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
	id, err := catalogdomain.ParseEpisodeID(string(ref))
	if err != nil {
		return catalogport.EpisodeInfo{}, domain.ErrEpisodeNotFound
	}

	actor := catalogdomain.LocalAdmin()
	episode, err := a.episodes.GetEpisode(ctx, actor, id)
	if err != nil {
		return catalogport.EpisodeInfo{}, translate(err)
	}
	if episode.Status() != catalogdomain.EpisodeStatusPublished {
		return catalogport.EpisodeInfo{}, domain.ErrEpisodeNotFound
	}

	view, err := a.series.GetSeries(ctx, actor, episode.SeriesID())
	if err != nil {
		return catalogport.EpisodeInfo{}, translate(err)
	}

	info := catalogport.EpisodeInfo{
		Ref:               domain.EpisodeRef(episode.ID()),
		SeriesID:          string(view.Series.ID()),
		SeriesTitle:       view.Series.Title(),
		SeriesSlug:        view.Series.Slug(),
		SeriesAccentColor: view.Series.AccentColor(),
		SeriesHasCover:    view.Series.HasCover(),
		Number:            episode.Number(),
		Title:             episode.Title(),
		ChapterCount:      len(episode.Chapters()),
	}
	if audio := episode.Audio(); audio != nil {
		info.DurationSeconds = audio.DurationSeconds
	}
	return info, nil
}

// translate maps a catalog error onto listening's own errors: what the catalog
// reports as not found is, for a listener, an episode that cannot be played.
func translate(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, catalogdomain.ErrNotFound) {
		return domain.ErrEpisodeNotFound
	}
	return err
}
