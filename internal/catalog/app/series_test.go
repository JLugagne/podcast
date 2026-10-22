package app_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
)

func TestCreateSeries(t *testing.T) {
	e := newEnv(t)
	var created *domain.Series
	e.series.CreateFn = func(_ context.Context, s *domain.Series) error {
		created = s
		return nil
	}

	view, err := e.app.CreateSeries(context.Background(), domain.LocalAdmin(), domain.SeriesAttributes{Title: "My Show"})
	if err != nil {
		t.Fatalf("CreateSeries() error = %v", err)
	}
	if created == nil || created.Slug() != "my-show" {
		t.Fatalf("created series = %+v", created)
	}
	if view.Series != created {
		t.Fatal("view must wrap the created series")
	}
	if view.EpisodeCount != 0 || view.PublishedCount != 0 || view.TotalDurationSeconds != 0 {
		t.Fatalf("view counters = %+v", view)
	}
	if !created.CreatedAt().Equal(e.now) {
		t.Fatalf("CreatedAt() = %v, want %v", created.CreatedAt(), e.now)
	}
}

func TestCreateSeriesRequiresActor(t *testing.T) {
	e := newEnv(t)
	if _, err := e.app.CreateSeries(context.Background(), nil, domain.SeriesAttributes{Title: "Show"}); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("CreateSeries() error = %v, want ErrUnauthorized", err)
	}
	if len(e.events) != 0 {
		t.Fatalf("no transaction must run, events = %v", e.events)
	}
}

func TestUpdateSeriesVersionConflict(t *testing.T) {
	e := newEnv(t)
	e.series.GetFn = func(_ context.Context, id domain.SeriesID) (*domain.Series, error) {
		if id != testSeriesID {
			t.Fatalf("Get() id = %q", id)
		}
		return testSeries(3, nil), nil
	}

	_, err := e.app.UpdateSeries(context.Background(), domain.LocalAdmin(), testSeriesID, domain.SeriesAttributes{Title: "Renamed"}, 2)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("UpdateSeries() error = %v, want ErrVersionConflict", err)
	}
	if len(e.events) != 0 {
		t.Fatalf("nothing must be written, events = %v", e.events)
	}
}

func TestUpdateSeries(t *testing.T) {
	e := newEnv(t)
	e.now = baseTime
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(4, nil), nil
	}
	var updated *domain.Series
	e.series.UpdateFn = func(_ context.Context, s *domain.Series) error {
		updated = s
		return nil
	}
	e.episodes.ListBySeriesFn = func(context.Context, domain.SeriesID, episoderepo.Filter) ([]*domain.Episode, error) {
		return []*domain.Episode{
			testEpisode(domain.EpisodeStatusDraft, 1, testAudio("a", 120), nil, 1),
			testEpisode(domain.EpisodeStatusPublished, 2, testAudio("b", 100), nil, 1),
		}, nil
	}

	view, err := e.app.UpdateSeries(context.Background(), domain.LocalAdmin(), testSeriesID, domain.SeriesAttributes{Title: "Renamed"}, 4)
	if err != nil {
		t.Fatalf("UpdateSeries() error = %v", err)
	}
	if updated == nil || updated.Title() != "Renamed" {
		t.Fatalf("updated series = %+v", updated)
	}
	if updated.Version() != 4 {
		t.Fatalf("entity version must be untouched, got %d", updated.Version())
	}
	if view.EpisodeCount != 2 || view.PublishedCount != 1 || view.TotalDurationSeconds != 220 {
		t.Fatalf("view = %+v", view)
	}
}

func TestDeleteSeriesNotEmpty(t *testing.T) {
	e := newEnv(t)
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(1, nil), nil
	}
	e.episodes.CountBySeriesFn = func(context.Context, domain.SeriesID) (int, error) { return 1, nil }

	err := e.app.DeleteSeries(context.Background(), domain.LocalAdmin(), testSeriesID)
	if !errors.Is(err, domain.ErrSeriesNotEmpty) {
		t.Fatalf("DeleteSeries() error = %v, want ErrSeriesNotEmpty", err)
	}
	if len(e.events) != 0 {
		t.Fatalf("nothing must be written, events = %v", e.events)
	}
}

func TestDeleteSeriesDeletesCoverAfterCommit(t *testing.T) {
	e := newEnv(t)
	cover := &domain.Asset{Key: "series/x/cover-old.png", ContentType: "image/png", SizeBytes: 10}
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(1, cover), nil
	}
	e.episodes.CountBySeriesFn = func(context.Context, domain.SeriesID) (int, error) { return 0, nil }
	e.series.DeleteFn = func(context.Context, domain.SeriesID) error {
		e.events = append(e.events, "delete-series")
		return nil
	}
	e.blobs.DeleteFn = func(_ context.Context, key string) error {
		e.events = append(e.events, "delete-blob:"+key)
		return nil
	}

	if err := e.app.DeleteSeries(context.Background(), domain.LocalAdmin(), testSeriesID); err != nil {
		t.Fatalf("DeleteSeries() error = %v", err)
	}
	assertOrder(t, e.events, []string{"delete-series", "commit", "delete-blob:" + cover.Key})
}

func TestUploadCoverReplacesOldCover(t *testing.T) {
	e := newEnv(t)
	old := &domain.Asset{Key: "series/x/cover-old.png", ContentType: "image/png", SizeBytes: 10}
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(1, old), nil
	}
	var newKey string
	e.blobs.PutFn = func(_ context.Context, key string, _ io.Reader, contentType string) (int64, error) {
		newKey = key
		if contentType != "image/png" {
			t.Fatalf("Put() contentType = %q", contentType)
		}
		if !strings.HasPrefix(key, "series/"+string(testSeriesID)+"/cover-") || !strings.HasSuffix(key, ".png") {
			t.Fatalf("Put() key = %q", key)
		}
		return 123, nil
	}
	e.series.UpdateFn = func(_ context.Context, s *domain.Series) error {
		c := s.Cover()
		if c == nil || c.Key != newKey || c.SizeBytes != 123 || c.ContentType != "image/png" {
			t.Fatalf("updated cover = %+v", c)
		}
		return nil
	}
	e.episodes.ListBySeriesFn = func(context.Context, domain.SeriesID, episoderepo.Filter) ([]*domain.Episode, error) {
		return nil, nil
	}
	e.blobs.DeleteFn = func(_ context.Context, key string) error {
		e.events = append(e.events, "delete-blob:"+key)
		return nil
	}

	view, err := e.app.UploadCover(context.Background(), domain.LocalAdmin(), testSeriesID, seriesuc.UploadInput{
		Reader:      strings.NewReader("image-bytes"),
		ContentType: "image/png",
	})
	if err != nil {
		t.Fatalf("UploadCover() error = %v", err)
	}
	if got := view.Series.Cover(); got == nil || got.Key != newKey {
		t.Fatalf("view cover = %+v", got)
	}
	assertOrder(t, e.events, []string{"commit", "delete-blob:" + old.Key})
}

func TestUploadCoverDiscardsNewBlobOnFailure(t *testing.T) {
	e := newEnv(t)
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return nil, domain.ErrSeriesNotFound
	}
	var newKey string
	e.blobs.PutFn = func(_ context.Context, key string, _ io.Reader, _ string) (int64, error) {
		newKey = key
		return 5, nil
	}
	e.blobs.DeleteFn = func(_ context.Context, key string) error {
		e.events = append(e.events, "delete-blob:"+key)
		return nil
	}

	_, err := e.app.UploadCover(context.Background(), domain.LocalAdmin(), testSeriesID, seriesuc.UploadInput{
		Reader:      strings.NewReader("image-bytes"),
		ContentType: "image/png",
	})
	if !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Fatalf("UploadCover() error = %v, want ErrSeriesNotFound", err)
	}
	assertOrder(t, e.events, []string{"delete-blob:" + newKey})
}

func TestUploadCoverRejectsUnsupportedType(t *testing.T) {
	e := newEnv(t)
	_, err := e.app.UploadCover(context.Background(), domain.LocalAdmin(), testSeriesID, seriesuc.UploadInput{
		Reader:      strings.NewReader("gif"),
		ContentType: "image/gif",
	})
	if !errors.Is(err, domain.ErrUnsupportedImageType) {
		t.Fatalf("UploadCover() error = %v, want ErrUnsupportedImageType", err)
	}
}

func TestRemoveCoverDeletesBlobAfterCommit(t *testing.T) {
	e := newEnv(t)
	cover := &domain.Asset{Key: "series/x/cover-old.png", ContentType: "image/png", SizeBytes: 10}
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(1, cover), nil
	}
	e.series.UpdateFn = func(_ context.Context, s *domain.Series) error {
		if s.Cover() != nil {
			t.Fatal("cover must be cleared before Update")
		}
		return nil
	}
	e.episodes.ListBySeriesFn = func(context.Context, domain.SeriesID, episoderepo.Filter) ([]*domain.Episode, error) {
		return nil, nil
	}
	e.blobs.DeleteFn = func(_ context.Context, key string) error {
		e.events = append(e.events, "delete-blob:"+key)
		return nil
	}

	view, err := e.app.RemoveCover(context.Background(), domain.LocalAdmin(), testSeriesID)
	if err != nil {
		t.Fatalf("RemoveCover() error = %v", err)
	}
	if view.Series.Cover() != nil {
		t.Fatal("view cover must be nil")
	}
	assertOrder(t, e.events, []string{"commit", "delete-blob:" + cover.Key})
}

// TestGetSeriesBySlug returns the series looked up by slug with its counters.
func TestGetSeriesBySlug(t *testing.T) {
	e := newEnv(t)
	e.series.GetBySlugFn = func(_ context.Context, slug string) (*domain.Series, error) {
		if slug != "show" {
			t.Errorf("GetBySlug() slug = %q, want %q", slug, "show")
		}
		return testSeries(2, nil), nil
	}
	e.episodes.ListBySeriesFn = func(context.Context, domain.SeriesID, episoderepo.Filter) ([]*domain.Episode, error) {
		return []*domain.Episode{
			testEpisode(domain.EpisodeStatusDraft, 1, testAudio("a", 120), nil, 1),
			testEpisode(domain.EpisodeStatusPublished, 2, testAudio("b", 60), nil, 1),
		}, nil
	}

	view, err := e.app.GetSeriesBySlug(context.Background(), domain.LocalAdmin(), "show")
	if err != nil {
		t.Fatalf("GetSeriesBySlug() error = %v", err)
	}
	if view.Series.Slug() != "show" {
		t.Fatalf("series slug = %q, want show", view.Series.Slug())
	}
	if view.EpisodeCount != 2 || view.PublishedCount != 1 || view.TotalDurationSeconds != 180 {
		t.Fatalf("view counters = %+v", view)
	}
}

// TestGetSeriesBySlugNotFound propagates the repository not-found error.
func TestGetSeriesBySlugNotFound(t *testing.T) {
	e := newEnv(t)
	e.series.GetBySlugFn = func(context.Context, string) (*domain.Series, error) {
		return nil, domain.ErrSeriesNotFound
	}

	if _, err := e.app.GetSeriesBySlug(context.Background(), domain.LocalAdmin(), "missing"); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Fatalf("GetSeriesBySlug() error = %v, want ErrSeriesNotFound", err)
	}
}

// TestGetSeriesBySlugRequiresActor rejects a nil actor.
func TestGetSeriesBySlugRequiresActor(t *testing.T) {
	e := newEnv(t)
	if _, err := e.app.GetSeriesBySlug(context.Background(), nil, "show"); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("GetSeriesBySlug() error = %v, want ErrUnauthorized", err)
	}
}
