package pgadapter_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/outbound/pgadapter"
)

func createSeriesForTest(ctx context.Context, t *testing.T, s *domain.Series) *domain.Series {
	t.Helper()
	if err := pgadapter.NewSeriesRepository(testPool).Create(ctx, s); err != nil {
		t.Fatalf("Create series: %v", err)
	}
	return s
}

func TestEpisodeRoundTrip(t *testing.T) {
	requireDB(t)
	seriesRepo := pgadapter.NewSeriesRepository(testPool)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)
	audio := domain.AudioFile{
		Asset:           domain.Asset{Key: "episodes/x/audio-1.mp3", ContentType: "audio/mpeg", SizeBytes: 1024},
		DurationSeconds: 300,
		OriginalName:    "pilot.mp3",
	}
	if err := e.AttachAudio(audio, testNow); err != nil {
		t.Fatalf("AttachAudio: %v", err)
	}
	chapters := []domain.Chapter{
		{StartSeconds: 0, Title: "Intro", URL: "https://example.com/intro", ImageURL: "https://example.com/intro.png", TOC: true},
		{StartSeconds: 120.5, Title: "Main", EndSeconds: float64Ptr(290), TOC: false},
	}
	if err := e.SetChapters(chapters, testNow); err != nil {
		t.Fatalf("SetChapters: %v", err)
	}
	publishedAt := testNow.Add(30 * time.Minute)
	e.SetVersion(0)
	if err := e.Publish(publishedAt); err != nil {
		t.Fatalf("Publish: %v", err)
	}
	if err := episodeRepo.Create(ctx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := episodeRepo.Get(ctx, e.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.ID() != e.ID() || got.SeriesID() != s.ID() {
		t.Errorf("Get returned id/series (%s, %s), want (%s, %s)", got.ID(), got.SeriesID(), e.ID(), s.ID())
	}
	if got.Number() != 1 || got.Title() != "Pilot" || got.Description() != "Episode description" {
		t.Errorf("attributes = %+v, want the stored ones", got.Attributes())
	}
	if got.Status() != domain.EpisodeStatusPublished {
		t.Errorf("Status = %q, want published", got.Status())
	}
	if got.Version() != 0 {
		t.Errorf("Version = %d, want 0 after Create", got.Version())
	}
	if got.PublishedAt() == nil || !got.PublishedAt().Equal(publishedAt) {
		t.Errorf("PublishedAt = %v, want %s", got.PublishedAt(), publishedAt)
	}
	gotAudio := got.Audio()
	if gotAudio == nil || !reflect.DeepEqual(*gotAudio, audio) {
		t.Errorf("Audio = %+v, want %+v", gotAudio, audio)
	}
	if gotChapters := got.Chapters(); !reflect.DeepEqual(gotChapters, chapters) {
		t.Errorf("Chapters = %+v, want %+v", gotChapters, chapters)
	}

	bySlug, err := seriesRepo.GetBySlug(ctx, "my-show")
	if err != nil || bySlug.ID() != s.ID() {
		t.Fatalf("GetBySlug after episode create = %v, %v", bySlug, err)
	}
}

func TestEpisodeDefaultsAndNotFound(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)
	if err := episodeRepo.Create(ctx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := episodeRepo.Get(ctx, e.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status() != domain.EpisodeStatusDraft {
		t.Errorf("Status = %q, want draft", got.Status())
	}
	if got.HasAudio() || got.Audio() != nil {
		t.Errorf("Audio = %+v, want nil", got.Audio())
	}
	if got.HasChapters() || got.Chapters() != nil {
		t.Errorf("Chapters = %+v, want empty", got.Chapters())
	}
	if got.PublishedAt() != nil {
		t.Errorf("PublishedAt = %v, want nil", got.PublishedAt())
	}

	missing := domain.EpisodeID(domain.NewID())
	if _, err := episodeRepo.Get(ctx, missing); !errors.Is(err, domain.ErrEpisodeNotFound) {
		t.Errorf("Get missing = %v, want ErrEpisodeNotFound", err)
	}
	if err := episodeRepo.Delete(ctx, missing); !errors.Is(err, domain.ErrEpisodeNotFound) {
		t.Errorf("Delete missing = %v, want ErrEpisodeNotFound", err)
	}
}

func TestEpisodeNumberIsUniquePerSeries(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	other := createSeriesForTest(ctx, t, mustSeriesAt(t, "other-show", "Other Show", testNow))

	if err := episodeRepo.Create(ctx, mustEpisodeAt(t, s.ID(), 1, "First", testNow)); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	if err := episodeRepo.Create(ctx, mustEpisodeAt(t, s.ID(), 1, "Duplicate", testNow)); !errors.Is(err, domain.ErrEpisodeNumberTaken) {
		t.Fatalf("Create duplicate number = %v, want ErrEpisodeNumberTaken", err)
	}
	if err := episodeRepo.Create(ctx, mustEpisodeAt(t, other.ID(), 1, "Other first", testNow)); err != nil {
		t.Fatalf("Create same number in another series: %v", err)
	}
}

func TestEpisodeCreateWithUnknownSeries(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	e := mustEpisodeAt(t, domain.SeriesID(domain.NewID()), 1, "Orphan", testNow)
	if err := episodeRepo.Create(ctx, e); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Fatalf("Create with unknown series = %v, want ErrSeriesNotFound", err)
	}
}

func TestEpisodeUpdateIncrementsVersion(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)
	if err := episodeRepo.Create(ctx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	loaded, err := episodeRepo.Get(ctx, e.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := loaded.Update(domain.EpisodeAttributes{Number: 2, Title: "Renamed", Description: "New description"}, testNow); err != nil {
		t.Fatalf("Update entity: %v", err)
	}
	if err := loaded.AttachAudio(domain.AudioFile{
		Asset:           domain.Asset{Key: "episodes/x/audio-1.mp3", ContentType: "audio/mpeg", SizeBytes: 10},
		DurationSeconds: 0,
	}, testNow); err != nil {
		t.Fatalf("AttachAudio: %v", err)
	}
	if err := episodeRepo.Update(ctx, loaded); err != nil {
		t.Fatalf("Update: %v", err)
	}
	if loaded.Version() != 1 {
		t.Errorf("aggregate Version = %d, want 1", loaded.Version())
	}

	got, err := episodeRepo.Get(ctx, e.ID())
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if got.Number() != 2 || got.Title() != "Renamed" || got.Description() != "New description" {
		t.Errorf("attributes = %+v, want the updated ones", got.Attributes())
	}
	if got.Version() != 1 {
		t.Errorf("stored Version = %d, want 1", got.Version())
	}
	if got.Audio() == nil || got.Audio().OriginalName != "" {
		t.Errorf("Audio = %+v, want the attached asset", got.Audio())
	}
}

func TestEpisodeUpdateConflict(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)
	if err := episodeRepo.Create(ctx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}

	first, err := episodeRepo.Get(ctx, e.ID())
	if err != nil {
		t.Fatalf("Get first: %v", err)
	}
	second, err := episodeRepo.Get(ctx, e.ID())
	if err != nil {
		t.Fatalf("Get second: %v", err)
	}

	if err := first.Update(domain.EpisodeAttributes{Number: 1, Title: "First writer", Description: ""}, testNow); err != nil {
		t.Fatalf("Update first entity: %v", err)
	}
	if err := episodeRepo.Update(ctx, first); err != nil {
		t.Fatalf("Update first: %v", err)
	}
	if err := second.Update(domain.EpisodeAttributes{Number: 1, Title: "Second writer", Description: ""}, testNow); err != nil {
		t.Fatalf("Update second entity: %v", err)
	}
	if err := episodeRepo.Update(ctx, second); !errors.Is(err, domain.ErrVersionConflict) {
		t.Fatalf("stale Update = %v, want ErrVersionConflict", err)
	}
}

func TestEpisodeUpdateNumberConflict(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	if err := episodeRepo.Create(ctx, mustEpisodeAt(t, s.ID(), 1, "First", testNow)); err != nil {
		t.Fatalf("Create first: %v", err)
	}
	second := mustEpisodeAt(t, s.ID(), 2, "Second", testNow)
	if err := episodeRepo.Create(ctx, second); err != nil {
		t.Fatalf("Create second: %v", err)
	}

	loaded, err := episodeRepo.Get(ctx, second.ID())
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if err := loaded.Update(domain.EpisodeAttributes{Number: 1, Title: "Second", Description: ""}, testNow); err != nil {
		t.Fatalf("Update entity: %v", err)
	}
	if err := episodeRepo.Update(ctx, loaded); !errors.Is(err, domain.ErrEpisodeNumberTaken) {
		t.Fatalf("Update to a taken number = %v, want ErrEpisodeNumberTaken", err)
	}
}

func TestEpisodeUpdateMissing(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	ghost := domain.ReconstituteEpisode(
		domain.EpisodeID(domain.NewID()), domain.SeriesID(domain.NewID()),
		domain.EpisodeAttributes{Number: 1, Title: "Ghost", Description: ""},
		domain.EpisodeStatusDraft, nil, nil, nil, testNow, testNow, 0,
	)
	if err := episodeRepo.Update(ctx, ghost); !errors.Is(err, domain.ErrEpisodeNotFound) {
		t.Fatalf("Update missing = %v, want ErrEpisodeNotFound", err)
	}
}

func TestEpisodeDelete(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)
	if err := episodeRepo.Create(ctx, e); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := episodeRepo.Delete(ctx, e.ID()); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := episodeRepo.Get(ctx, e.ID()); !errors.Is(err, domain.ErrEpisodeNotFound) {
		t.Errorf("Get after Delete = %v, want ErrEpisodeNotFound", err)
	}
}

func TestEpisodeListAndCount(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))
	for _, number := range []int{3, 1, 2} {
		if err := episodeRepo.Create(ctx, mustEpisodeAt(t, s.ID(), number, "Episode", testNow)); err != nil {
			t.Fatalf("Create episode %d: %v", number, err)
		}
	}

	all, err := episodeRepo.ListBySeries(ctx, s.ID(), episoderepo.Filter{})
	if err != nil {
		t.Fatalf("ListBySeries: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("ListBySeries returned %d episodes, want 3", len(all))
	}
	for i, want := range []int{1, 2, 3} {
		if all[i].Number() != want {
			t.Errorf("ListBySeries[%d].Number = %d, want %d", i, all[i].Number(), want)
		}
	}

	count, err := episodeRepo.CountBySeries(ctx, s.ID())
	if err != nil {
		t.Fatalf("CountBySeries: %v", err)
	}
	if count != 3 {
		t.Errorf("CountBySeries = %d, want 3", count)
	}

	publishedStatus := domain.EpisodeStatusPublished
	published := domain.ReconstituteEpisode(
		domain.EpisodeID(domain.NewID()), s.ID(),
		domain.EpisodeAttributes{Number: 4, Title: "Published", Description: ""},
		domain.EpisodeStatusPublished, nil, nil, &testNow, testNow, testNow, 0,
	)
	if err := episodeRepo.Create(ctx, published); err != nil {
		t.Fatalf("Create published episode: %v", err)
	}

	filtered, err := episodeRepo.ListBySeries(ctx, s.ID(), episoderepo.Filter{Status: &publishedStatus})
	if err != nil {
		t.Fatalf("ListBySeries published: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID() != published.ID() {
		t.Errorf("ListBySeries(published) = %d episodes, want only the published one", len(filtered))
	}

	drafts, err := episodeRepo.ListBySeries(ctx, s.ID(), episoderepo.Filter{Status: strPtr(domain.EpisodeStatusDraft)})
	if err != nil {
		t.Fatalf("ListBySeries drafts: %v", err)
	}
	if len(drafts) != 3 {
		t.Errorf("ListBySeries(draft) = %d episodes, want 3", len(drafts))
	}
}

func TestEpisodeNextNumber(t *testing.T) {
	requireDB(t)
	episodeRepo := pgadapter.NewEpisodeRepository(testPool)
	ctx := context.Background()

	s := createSeriesForTest(ctx, t, mustSeriesAt(t, "my-show", "My Show", testNow))

	next, err := episodeRepo.NextNumber(ctx, s.ID())
	if err != nil {
		t.Fatalf("NextNumber on empty series: %v", err)
	}
	if next != 1 {
		t.Errorf("NextNumber on empty series = %d, want 1", next)
	}

	for _, number := range []int{1, 3} {
		if err := episodeRepo.Create(ctx, mustEpisodeAt(t, s.ID(), number, "Episode", testNow)); err != nil {
			t.Fatalf("Create episode %d: %v", number, err)
		}
	}
	next, err = episodeRepo.NextNumber(ctx, s.ID())
	if err != nil {
		t.Fatalf("NextNumber: %v", err)
	}
	if next != 4 {
		t.Errorf("NextNumber = %d, want 4", next)
	}
}

func strPtr(status domain.EpisodeStatus) *domain.EpisodeStatus { return &status }
