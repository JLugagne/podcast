package app_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
)

func intPtr(v int) *int { return &v }

func TestCreateEpisodeUsesNextNumber(t *testing.T) {
	e := newEnv(t)
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(0, nil), nil
	}
	e.episodes.NextNumberFn = func(_ context.Context, seriesID domain.SeriesID) (int, error) {
		if seriesID != testSeriesID {
			t.Fatalf("NextNumber() series = %q", seriesID)
		}
		return 7, nil
	}
	var created *domain.Episode
	e.episodes.CreateFn = func(_ context.Context, ep *domain.Episode) error {
		created = ep
		return nil
	}

	ep, err := e.app.CreateEpisode(context.Background(), domain.LocalAdmin(), testSeriesID, episodeuc.CreateInput{Title: "Pilot"})
	if err != nil {
		t.Fatalf("CreateEpisode() error = %v", err)
	}
	if created == nil || created.Number() != 7 {
		t.Fatalf("created episode = %+v", created)
	}
	if created.Status() != domain.EpisodeStatusDraft || created.SeriesID() != testSeriesID || created.Title() != "Pilot" {
		t.Fatalf("created episode = %+v", created.Attributes())
	}
	if ep != created {
		t.Fatal("CreateEpisode must return the created aggregate")
	}
}

func TestCreateEpisodeExplicitNumber(t *testing.T) {
	e := newEnv(t)
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(0, nil), nil
	}
	e.episodes.CreateFn = func(_ context.Context, ep *domain.Episode) error {
		if ep.Number() != 3 {
			t.Fatalf("Number() = %d, want 3", ep.Number())
		}
		return nil
	}

	if _, err := e.app.CreateEpisode(context.Background(), domain.LocalAdmin(), testSeriesID, episodeuc.CreateInput{Number: intPtr(3), Title: "Pilot"}); err != nil {
		t.Fatalf("CreateEpisode() error = %v", err)
	}
}

func TestCreateEpisodeMissingSeries(t *testing.T) {
	e := newEnv(t)
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return nil, domain.ErrSeriesNotFound
	}

	_, err := e.app.CreateEpisode(context.Background(), domain.LocalAdmin(), testSeriesID, episodeuc.CreateInput{Title: "Pilot"})
	if !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Fatalf("CreateEpisode() error = %v, want ErrSeriesNotFound", err)
	}
	if len(e.events) != 0 {
		t.Fatalf("nothing must be written, events = %v", e.events)
	}
}

func TestUpdateEpisodeVersionConflict(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, nil, nil, 5), nil
	}

	_, err := e.app.UpdateEpisode(context.Background(), domain.LocalAdmin(), testEpisodeID, domain.EpisodeAttributes{Number: 1, Title: "Renamed"}, 4)
	if !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("UpdateEpisode() error = %v, want ErrVersionConflict", err)
	}
	if len(e.events) != 0 {
		t.Fatalf("nothing must be written, events = %v", e.events)
	}
}

func TestUploadAudio(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, nil, nil, 1), nil
	}
	var newKey string
	e.blobs.PutFn = func(_ context.Context, key string, _ io.Reader, contentType string) (int64, error) {
		newKey = key
		if contentType != "audio/mpeg" {
			t.Fatalf("Put() contentType = %q", contentType)
		}
		if !strings.HasPrefix(key, "episodes/"+string(testEpisodeID)+"/audio-") || !strings.HasSuffix(key, ".mp3") {
			t.Fatalf("Put() key = %q", key)
		}
		return 4096, nil
	}
	e.episodes.UpdateFn = func(_ context.Context, ep *domain.Episode) error {
		audio := ep.Audio()
		if audio == nil {
			t.Fatal("audio must be attached before Update")
		}
		if audio.Key != newKey || audio.SizeBytes != 4096 || audio.ContentType != "audio/mpeg" {
			t.Fatalf("audio asset = %+v", audio)
		}
		if audio.DurationSeconds != 90 || audio.OriginalName != "episode.mp3" {
			t.Fatalf("audio metadata = %+v", audio)
		}
		return nil
	}

	ep, err := e.app.UploadAudio(context.Background(), domain.LocalAdmin(), testEpisodeID, episodeuc.AudioUploadInput{
		Reader:          strings.NewReader("audio"),
		ContentType:     "audio/mpeg",
		OriginalName:    "episode.mp3",
		DurationSeconds: 90,
	})
	if err != nil {
		t.Fatalf("UploadAudio() error = %v", err)
	}
	if ep.Audio() == nil || ep.Audio().Key != newKey {
		t.Fatalf("returned audio = %+v", ep.Audio())
	}
}

func TestUploadAudioRejectsUnsupportedType(t *testing.T) {
	e := newEnv(t)
	_, err := e.app.UploadAudio(context.Background(), domain.LocalAdmin(), testEpisodeID, episodeuc.AudioUploadInput{
		Reader:      strings.NewReader("audio"),
		ContentType: "application/octet-stream",
	})
	if !errors.Is(err, domain.ErrUnsupportedAudioType) {
		t.Fatalf("UploadAudio() error = %v, want ErrUnsupportedAudioType", err)
	}
}

func TestUploadAudioDiscardsNewBlobOnFailure(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return nil, domain.ErrEpisodeNotFound
	}
	var newKey string
	e.blobs.PutFn = func(_ context.Context, key string, _ io.Reader, _ string) (int64, error) {
		newKey = key
		e.events = append(e.events, "put")
		return 10, nil
	}
	e.blobs.DeleteFn = func(_ context.Context, key string) error {
		e.events = append(e.events, "delete-blob:"+key)
		return nil
	}

	_, err := e.app.UploadAudio(context.Background(), domain.LocalAdmin(), testEpisodeID, episodeuc.AudioUploadInput{
		Reader:      strings.NewReader("audio"),
		ContentType: "audio/mpeg",
	})
	if !errors.Is(err, domain.ErrEpisodeNotFound) {
		t.Fatalf("UploadAudio() error = %v, want ErrEpisodeNotFound", err)
	}
	assertOrder(t, e.events, []string{"put", "delete-blob:" + newKey})
}

func TestUploadAudioReplacesOldBlobAfterCommit(t *testing.T) {
	e := newEnv(t)
	const oldKey = "episodes/x/audio-old.mp3"
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, testAudio(oldKey, 60), nil, 1), nil
	}
	e.blobs.PutFn = func(_ context.Context, _ string, _ io.Reader, _ string) (int64, error) {
		e.events = append(e.events, "put")
		return 10, nil
	}
	e.episodes.UpdateFn = func(context.Context, *domain.Episode) error { return nil }
	e.blobs.DeleteFn = func(_ context.Context, key string) error {
		e.events = append(e.events, "delete-blob:"+key)
		return nil
	}

	if _, err := e.app.UploadAudio(context.Background(), domain.LocalAdmin(), testEpisodeID, episodeuc.AudioUploadInput{
		Reader:          strings.NewReader("audio"),
		ContentType:     "audio/mpeg",
		DurationSeconds: 60,
	}); err != nil {
		t.Fatalf("UploadAudio() error = %v", err)
	}
	assertOrder(t, e.events, []string{"put", "commit", "delete-blob:" + oldKey})
}

func TestSetChapters(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, nil, nil, 1), nil
	}
	e.episodes.UpdateFn = func(_ context.Context, ep *domain.Episode) error {
		chapters := ep.Chapters()
		if len(chapters) != 2 || chapters[0].StartSeconds != 0 || chapters[1].Title != "Main" {
			t.Fatalf("chapters = %+v", chapters)
		}
		return nil
	}

	if _, err := e.app.SetChapters(context.Background(), domain.LocalAdmin(), testEpisodeID, testChapters()); err != nil {
		t.Fatalf("SetChapters() error = %v", err)
	}
}

func TestSetChaptersOutOfRange(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, testAudio("a", 60), nil, 1), nil
	}

	chapters := []domain.Chapter{{StartSeconds: 0, Title: "Intro"}, {StartSeconds: 120, Title: "Far"}}
	_, err := e.app.SetChapters(context.Background(), domain.LocalAdmin(), testEpisodeID, chapters)
	if !errors.Is(err, domain.ErrChapterOutOfRange) {
		t.Fatalf("SetChapters() error = %v, want ErrChapterOutOfRange", err)
	}
	if len(e.events) != 0 {
		t.Fatalf("nothing must be written, events = %v", e.events)
	}
}

func TestPublishMissingAudio(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, nil, testChapters(), 1), nil
	}

	_, err := e.app.Publish(context.Background(), domain.LocalAdmin(), testEpisodeID)
	if !errors.Is(err, domain.ErrEpisodeAudioMissing) {
		t.Fatalf("Publish() error = %v, want ErrEpisodeAudioMissing", err)
	}
}

func TestPublishMissingChapters(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, testAudio("a", 60), nil, 1), nil
	}

	_, err := e.app.Publish(context.Background(), domain.LocalAdmin(), testEpisodeID)
	if !errors.Is(err, domain.ErrEpisodeChaptersMissing) {
		t.Fatalf("Publish() error = %v, want ErrEpisodeChaptersMissing", err)
	}
}

func TestPublish(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, testAudio("a", 60), testChapters(), 1), nil
	}
	e.episodes.UpdateFn = func(_ context.Context, ep *domain.Episode) error {
		if ep.Status() != domain.EpisodeStatusPublished {
			t.Fatalf("Status() = %q", ep.Status())
		}
		if at := ep.PublishedAt(); at == nil || !at.Equal(e.now) {
			t.Fatalf("PublishedAt() = %v, want %v", at, e.now)
		}
		return nil
	}

	ep, err := e.app.Publish(context.Background(), domain.LocalAdmin(), testEpisodeID)
	if err != nil {
		t.Fatalf("Publish() error = %v", err)
	}
	if ep.Status() != domain.EpisodeStatusPublished {
		t.Fatalf("Status() = %q", ep.Status())
	}
}

func TestUnpublish(t *testing.T) {
	e := newEnv(t)
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusPublished, 1, testAudio("a", 60), testChapters(), 1), nil
	}
	e.episodes.UpdateFn = func(_ context.Context, ep *domain.Episode) error {
		if ep.Status() != domain.EpisodeStatusDraft || ep.PublishedAt() != nil {
			t.Fatalf("unexpected episode = %q / %v", ep.Status(), ep.PublishedAt())
		}
		return nil
	}

	ep, err := e.app.Unpublish(context.Background(), domain.LocalAdmin(), testEpisodeID)
	if err != nil {
		t.Fatalf("Unpublish() error = %v", err)
	}
	if ep.Status() != domain.EpisodeStatusDraft || ep.PublishedAt() != nil {
		t.Fatalf("unexpected episode = %q / %v", ep.Status(), ep.PublishedAt())
	}
}

func TestDeleteEpisodeDeletesAudioAfterCommit(t *testing.T) {
	e := newEnv(t)
	const audioKey = "episodes/x/audio.mp3"
	e.episodes.GetFn = func(context.Context, domain.EpisodeID) (*domain.Episode, error) {
		return testEpisode(domain.EpisodeStatusDraft, 1, testAudio(audioKey, 60), nil, 1), nil
	}
	e.episodes.DeleteFn = func(context.Context, domain.EpisodeID) error {
		e.events = append(e.events, "delete-episode")
		return nil
	}
	e.blobs.DeleteFn = func(_ context.Context, key string) error {
		e.events = append(e.events, "delete-blob:"+key)
		return nil
	}

	if err := e.app.DeleteEpisode(context.Background(), domain.LocalAdmin(), testEpisodeID); err != nil {
		t.Fatalf("DeleteEpisode() error = %v", err)
	}
	assertOrder(t, e.events, []string{"delete-episode", "commit", "delete-blob:" + audioKey})
}

func TestQueries(t *testing.T) {
	e := newEnv(t)
	e.series.GetFn = func(context.Context, domain.SeriesID) (*domain.Series, error) {
		return testSeries(1, nil), nil
	}
	var filters []episoderepo.Filter
	e.episodes.ListBySeriesFn = func(_ context.Context, seriesID domain.SeriesID, f episoderepo.Filter) ([]*domain.Episode, error) {
		if seriesID != testSeriesID {
			t.Fatalf("ListBySeries() series = %q", seriesID)
		}
		filters = append(filters, f)
		return []*domain.Episode{testEpisode(domain.EpisodeStatusPublished, 1, testAudio("a", 60), testChapters(), 1)}, nil
	}

	view, err := e.app.GetSeries(context.Background(), domain.LocalAdmin(), testSeriesID)
	if err != nil {
		t.Fatalf("GetSeries() error = %v", err)
	}
	if view.EpisodeCount != 1 || view.PublishedCount != 1 || view.TotalDurationSeconds != 60 {
		t.Fatalf("view = %+v", view)
	}

	status := domain.EpisodeStatusPublished
	list, err := e.app.ListEpisodes(context.Background(), domain.LocalAdmin(), testSeriesID, episodeuc.ListQuery{Status: &status})
	if err != nil {
		t.Fatalf("ListEpisodes() error = %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("ListEpisodes() = %+v", list)
	}
	if len(filters) != 2 || filters[0].Status != nil {
		t.Fatalf("filters = %+v", filters)
	}
	if filters[1].Status == nil || *filters[1].Status != domain.EpisodeStatusPublished {
		t.Fatalf("ListEpisodes filter = %+v", filters[1])
	}
}

func TestNilActorIsRejected(t *testing.T) {
	e := newEnv(t)
	ctx := context.Background()

	tests := []struct {
		name string
		call func() error
	}{
		{"CreateSeries", func() error {
			_, err := e.app.CreateSeries(ctx, nil, domain.SeriesAttributes{Title: "Show"})
			return err
		}},
		{"UpdateSeries", func() error {
			_, err := e.app.UpdateSeries(ctx, nil, testSeriesID, domain.SeriesAttributes{Title: "Show"}, 0)
			return err
		}},
		{"DeleteSeries", func() error { return e.app.DeleteSeries(ctx, nil, testSeriesID) }},
		{"UploadCover", func() error {
			_, err := e.app.UploadCover(ctx, nil, testSeriesID, seriesuc.UploadInput{Reader: strings.NewReader("x"), ContentType: "image/png"})
			return err
		}},
		{"RemoveCover", func() error {
			_, err := e.app.RemoveCover(ctx, nil, testSeriesID)
			return err
		}},
		{"GetSeries", func() error {
			_, err := e.app.GetSeries(ctx, nil, testSeriesID)
			return err
		}},
		{"ListSeries", func() error {
			_, err := e.app.ListSeries(ctx, nil, seriesuc.ListQuery{})
			return err
		}},
		{"OpenCover", func() error {
			_, err := e.app.OpenCover(ctx, nil, testSeriesID)
			return err
		}},
		{"CreateEpisode", func() error {
			_, err := e.app.CreateEpisode(ctx, nil, testSeriesID, episodeuc.CreateInput{Title: "Ep"})
			return err
		}},
		{"UpdateEpisode", func() error {
			_, err := e.app.UpdateEpisode(ctx, nil, testEpisodeID, domain.EpisodeAttributes{Number: 1, Title: "Ep"}, 0)
			return err
		}},
		{"DeleteEpisode", func() error { return e.app.DeleteEpisode(ctx, nil, testEpisodeID) }},
		{"UploadAudio", func() error {
			_, err := e.app.UploadAudio(ctx, nil, testEpisodeID, episodeuc.AudioUploadInput{Reader: strings.NewReader("x"), ContentType: "audio/mpeg"})
			return err
		}},
		{"SetChapters", func() error {
			_, err := e.app.SetChapters(ctx, nil, testEpisodeID, testChapters())
			return err
		}},
		{"Publish", func() error {
			_, err := e.app.Publish(ctx, nil, testEpisodeID)
			return err
		}},
		{"Unpublish", func() error {
			_, err := e.app.Unpublish(ctx, nil, testEpisodeID)
			return err
		}},
		{"GetEpisode", func() error {
			_, err := e.app.GetEpisode(ctx, nil, testEpisodeID)
			return err
		}},
		{"ListEpisodes", func() error {
			_, err := e.app.ListEpisodes(ctx, nil, testSeriesID, episodeuc.ListQuery{})
			return err
		}},
		{"OpenAudio", func() error {
			_, err := e.app.OpenAudio(ctx, nil, testEpisodeID)
			return err
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.call(); !errors.Is(err, domain.ErrUnauthorized) {
				t.Fatalf("%s(nil actor) error = %v, want ErrUnauthorized", tc.name, err)
			}
		})
	}
	if len(e.events) != 0 {
		t.Fatalf("no transaction must run, events = %v", e.events)
	}
}
