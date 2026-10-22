package app_test

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/app"
	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore/blobstoretest"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo/episoderepotest"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo/seriesrepotest"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow/uowtest"
)

var (
	testSeriesID  = domain.SeriesID("018f0000-0000-7000-8000-000000000001")
	testEpisodeID = domain.EpisodeID("018f0000-0000-7000-8000-000000000002")
	baseTime      = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
)

// env wires the app to function-field mocks. The unit-of-work mock runs the
// callback on the mock repositories and records a "commit" event so tests can
// assert what happens after the transaction.
type env struct {
	app      *app.App
	series   *seriesrepotest.MockSeriesRepository
	episodes *episoderepotest.MockEpisodeRepository
	blobs    *blobstoretest.MockStore
	uow      *uowtest.MockUnitOfWork
	repos    *uowtest.MockRepositories
	events   []string
	now      time.Time
}

func newEnv(t *testing.T) *env {
	t.Helper()
	e := &env{
		series:   &seriesrepotest.MockSeriesRepository{},
		episodes: &episoderepotest.MockEpisodeRepository{},
		blobs:    &blobstoretest.MockStore{},
		now:      baseTime,
	}
	e.repos = &uowtest.MockRepositories{
		SeriesFn:   func() seriesrepo.SeriesRepository { return e.series },
		EpisodesFn: func() episoderepo.EpisodeRepository { return e.episodes },
	}
	e.uow = &uowtest.MockUnitOfWork{DoFn: func(ctx context.Context, fn func(context.Context, uow.Repositories) error) error {
		if err := fn(ctx, e.repos); err != nil {
			return err
		}
		e.events = append(e.events, "commit")
		return nil
	}}
	e.app = app.New(app.Deps{
		UoW:      e.uow,
		Series:   e.series,
		Episodes: e.episodes,
		Blobs:    e.blobs,
		Logger:   slog.New(slog.NewTextHandler(io.Discard, nil)),
		Now:      func() time.Time { return e.now },
	})
	return e
}

func testSeries(version int, cover *domain.Asset) *domain.Series {
	attrs := domain.SeriesAttributes{
		Title:       "Show",
		Author:      "Me",
		Description: "Desc",
		AccentColor: domain.DefaultAccentColor,
		Slug:        "show",
	}
	return domain.ReconstituteSeries(testSeriesID, attrs, cover, baseTime, baseTime, version)
}

func testEpisode(status domain.EpisodeStatus, number int, audio *domain.AudioFile, chapters []domain.Chapter, version int) *domain.Episode {
	var published *time.Time
	if status == domain.EpisodeStatusPublished {
		at := baseTime
		published = &at
	}
	return domain.ReconstituteEpisode(
		testEpisodeID,
		testSeriesID,
		domain.EpisodeAttributes{Number: number, Title: "Episode", Description: "Desc"},
		status,
		audio,
		chapters,
		published,
		baseTime,
		baseTime,
		version,
	)
}

func testAudio(key string, duration float64) *domain.AudioFile {
	return &domain.AudioFile{
		Asset:           domain.Asset{Key: key, ContentType: "audio/mpeg", SizeBytes: 2048},
		DurationSeconds: duration,
		OriginalName:    "episode.mp3",
	}
}

func testChapters() []domain.Chapter {
	return []domain.Chapter{
		{StartSeconds: 0, Title: "Intro", TOC: true},
		{StartSeconds: 30, Title: "Main", TOC: true},
	}
}

func assertOrder(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}
