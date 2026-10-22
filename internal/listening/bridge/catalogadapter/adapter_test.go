package catalogadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	catalogdomain "github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/listening/bridge/catalogadapter"
	"github.com/JLugagne/podcast/internal/listening/domain"
)

const (
	episodeID = "018f0000-0000-7000-8000-000000000001"
	seriesID  = "018f0000-0000-7000-8000-0000000000aa"
)

var now = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

// fakeEpisodes is a function-field double of episodeuc.EpisodeQueries.
type fakeEpisodes struct {
	get func(ctx context.Context, actor *catalogdomain.Actor, id catalogdomain.EpisodeID) (*catalogdomain.Episode, error)
}

var _ episodeuc.EpisodeQueries = fakeEpisodes{}

func (f fakeEpisodes) GetEpisode(ctx context.Context, actor *catalogdomain.Actor, id catalogdomain.EpisodeID) (*catalogdomain.Episode, error) {
	if f.get == nil {
		panic("fakeEpisodes.GetEpisode called but not expected")
	}
	return f.get(ctx, actor, id)
}

func (fakeEpisodes) ListEpisodes(context.Context, *catalogdomain.Actor, catalogdomain.SeriesID, episodeuc.ListQuery) ([]*catalogdomain.Episode, error) {
	panic("fakeEpisodes.ListEpisodes called but not expected")
}

func (fakeEpisodes) OpenAudio(context.Context, *catalogdomain.Actor, catalogdomain.EpisodeID) (blobstore.Blob, error) {
	panic("fakeEpisodes.OpenAudio called but not expected")
}

// fakeSeries is a function-field double of seriesuc.SeriesQueries.
type fakeSeries struct {
	get func(ctx context.Context, actor *catalogdomain.Actor, id catalogdomain.SeriesID) (*seriesuc.SeriesView, error)
}

var _ seriesuc.SeriesQueries = fakeSeries{}

func (f fakeSeries) GetSeries(ctx context.Context, actor *catalogdomain.Actor, id catalogdomain.SeriesID) (*seriesuc.SeriesView, error) {
	if f.get == nil {
		panic("fakeSeries.GetSeries called but not expected")
	}
	return f.get(ctx, actor, id)
}

func (fakeSeries) GetSeriesBySlug(context.Context, *catalogdomain.Actor, string) (*seriesuc.SeriesView, error) {
	panic("fakeSeries.GetSeriesBySlug called but not expected")
}

func (fakeSeries) ListSeries(context.Context, *catalogdomain.Actor, seriesuc.ListQuery) ([]*seriesuc.SeriesView, error) {
	panic("fakeSeries.ListSeries called but not expected")
}

func (fakeSeries) OpenCover(context.Context, *catalogdomain.Actor, catalogdomain.SeriesID) (blobstore.Blob, error) {
	panic("fakeSeries.OpenCover called but not expected")
}

// publishedEpisode builds a stored published episode with audio and chapters.
func publishedEpisode(status catalogdomain.EpisodeStatus, chapters int) *catalogdomain.Episode {
	list := make([]catalogdomain.Chapter, 0, chapters)
	for i := 0; i < chapters; i++ {
		list = append(list, catalogdomain.Chapter{StartSeconds: float64(i * 10), Title: "Chapter", TOC: true})
	}
	audio := &catalogdomain.AudioFile{
		Asset:           catalogdomain.Asset{Key: "episodes/" + episodeID + "/audio.mp3", ContentType: "audio/mpeg", SizeBytes: 128},
		DurationSeconds: 300,
		OriginalName:    "episode.mp3",
	}
	published := now
	return catalogdomain.ReconstituteEpisode(
		catalogdomain.EpisodeID(episodeID),
		catalogdomain.SeriesID(seriesID),
		catalogdomain.EpisodeAttributes{Number: 2, Title: "Episode two", Description: "Details"},
		status, audio, list, &published,
		now, now, 1,
	)
}

// storedSeries builds a stored series with a cover.
func storedSeries(cover bool) *seriesuc.SeriesView {
	var asset *catalogdomain.Asset
	if cover {
		asset = &catalogdomain.Asset{Key: "series/" + seriesID + "/cover.png", ContentType: "image/png", SizeBytes: 8}
	}
	return &seriesuc.SeriesView{
		Series: catalogdomain.ReconstituteSeries(
			catalogdomain.SeriesID(seriesID),
			catalogdomain.SeriesAttributes{Title: "Show", Author: "Host", Description: "About", AccentColor: "#F2A541", Slug: "show"},
			asset, now, now, 1,
		),
		EpisodeCount: 3,
	}
}

func TestPublishedEpisode(t *testing.T) {
	episodes := fakeEpisodes{get: func(_ context.Context, actor *catalogdomain.Actor, id catalogdomain.EpisodeID) (*catalogdomain.Episode, error) {
		if actor == nil || actor.ID() != "local-admin" {
			t.Errorf("actor = %v, want the local catalog admin", actor)
		}
		if id != catalogdomain.EpisodeID(episodeID) {
			t.Errorf("id = %q, want %q", id, episodeID)
		}
		return publishedEpisode(catalogdomain.EpisodeStatusPublished, 4), nil
	}}
	series := fakeSeries{get: func(_ context.Context, _ *catalogdomain.Actor, id catalogdomain.SeriesID) (*seriesuc.SeriesView, error) {
		if id != catalogdomain.SeriesID(seriesID) {
			t.Errorf("series id = %q, want %q", id, seriesID)
		}
		return storedSeries(true), nil
	}}

	info, err := catalogadapter.New(episodes, series).PublishedEpisode(context.Background(), domain.EpisodeRef(episodeID))
	if err != nil {
		t.Fatalf("PublishedEpisode error = %v, want nil", err)
	}

	want := struct {
		ref        domain.EpisodeRef
		seriesID   string
		title      string
		slug       string
		accent     string
		hasCover   bool
		number     int
		epTitle    string
		duration   float64
		chapterCnt int
	}{domain.EpisodeRef(episodeID), seriesID, "Show", "show", "#F2A541", true, 2, "Episode two", 300, 4}

	if info.Ref != want.ref || info.SeriesID != want.seriesID || info.SeriesTitle != want.title ||
		info.SeriesSlug != want.slug || info.SeriesAccentColor != want.accent || info.SeriesHasCover != want.hasCover ||
		info.Number != want.number || info.Title != want.epTitle || info.DurationSeconds != want.duration ||
		info.ChapterCount != want.chapterCnt {
		t.Errorf("PublishedEpisode = %+v, want %+v", info, want)
	}
}

func TestPublishedEpisodeWithoutCoverOrChapters(t *testing.T) {
	episodes := fakeEpisodes{get: func(context.Context, *catalogdomain.Actor, catalogdomain.EpisodeID) (*catalogdomain.Episode, error) {
		return publishedEpisode(catalogdomain.EpisodeStatusPublished, 0), nil
	}}
	series := fakeSeries{get: func(context.Context, *catalogdomain.Actor, catalogdomain.SeriesID) (*seriesuc.SeriesView, error) {
		return storedSeries(false), nil
	}}

	info, err := catalogadapter.New(episodes, series).PublishedEpisode(context.Background(), domain.EpisodeRef(episodeID))
	if err != nil {
		t.Fatalf("PublishedEpisode error = %v, want nil", err)
	}
	if info.SeriesHasCover {
		t.Error("SeriesHasCover = true, want false")
	}
	if info.ChapterCount != 0 {
		t.Errorf("ChapterCount = %d, want 0", info.ChapterCount)
	}
}

func TestPublishedEpisodeErrors(t *testing.T) {
	tests := []struct {
		name     string
		episodes fakeEpisodes
		series   fakeSeries
		ref      domain.EpisodeRef
	}{
		{
			name: "draft episode",
			episodes: fakeEpisodes{get: func(context.Context, *catalogdomain.Actor, catalogdomain.EpisodeID) (*catalogdomain.Episode, error) {
				return publishedEpisode(catalogdomain.EpisodeStatusDraft, 1), nil
			}},
			ref: domain.EpisodeRef(episodeID),
		},
		{
			name: "unknown episode",
			episodes: fakeEpisodes{get: func(context.Context, *catalogdomain.Actor, catalogdomain.EpisodeID) (*catalogdomain.Episode, error) {
				return nil, catalogdomain.ErrEpisodeNotFound
			}},
			ref: domain.EpisodeRef(episodeID),
		},
		{
			name: "unknown series",
			episodes: fakeEpisodes{get: func(context.Context, *catalogdomain.Actor, catalogdomain.EpisodeID) (*catalogdomain.Episode, error) {
				return publishedEpisode(catalogdomain.EpisodeStatusPublished, 1), nil
			}},
			series: fakeSeries{get: func(context.Context, *catalogdomain.Actor, catalogdomain.SeriesID) (*seriesuc.SeriesView, error) {
				return nil, catalogdomain.ErrSeriesNotFound
			}},
			ref: domain.EpisodeRef(episodeID),
		},
		{
			name: "malformed reference",
			ref:  domain.EpisodeRef("not-a-uuid"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := catalogadapter.New(tt.episodes, tt.series).PublishedEpisode(context.Background(), tt.ref)
			if !errors.Is(err, domain.ErrEpisodeNotFound) {
				t.Fatalf("PublishedEpisode error = %v, want %v", err, domain.ErrEpisodeNotFound)
			}
		})
	}
}

func TestPublishedEpisodePropagatesUnexpectedErrors(t *testing.T) {
	wantErr := errors.New("catalog database is down")
	episodes := fakeEpisodes{get: func(context.Context, *catalogdomain.Actor, catalogdomain.EpisodeID) (*catalogdomain.Episode, error) {
		return nil, wantErr
	}}

	_, err := catalogadapter.New(episodes, fakeSeries{}).PublishedEpisode(context.Background(), domain.EpisodeRef(episodeID))
	if !errors.Is(err, wantErr) {
		t.Fatalf("PublishedEpisode error = %v, want %v", err, wantErr)
	}
}
