package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/listening/app"
	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport/catalogporttest"
	"github.com/JLugagne/podcast/internal/listening/domain/repositories/progressrepo"
	"github.com/JLugagne/podcast/internal/listening/domain/repositories/progressrepo/progressrepotest"
)

var fixedNow = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

const (
	testListener = domain.ListenerID("local")
	testEpisode  = domain.EpisodeRef("018f0000-0000-7000-8000-000000000001")
	testOther    = domain.EpisodeRef("018f0000-0000-7000-8000-000000000002")
	testThird    = domain.EpisodeRef("018f0000-0000-7000-8000-000000000003")
)

// newApp builds the listening app with a fixed clock.
func newApp(progress progressrepo.ProgressRepository, catalog catalogport.Catalog) *app.App {
	return app.New(app.Deps{
		Progress: progress,
		Catalog:  catalog,
		Now:      func() time.Time { return fixedNow },
	})
}

// publishedInfo builds the catalog snapshot of a published episode.
func publishedInfo(ref domain.EpisodeRef, duration float64) catalogport.EpisodeInfo {
	return catalogport.EpisodeInfo{
		Ref:               ref,
		SeriesID:          "018f0000-0000-7000-8000-0000000000aa",
		SeriesTitle:       "Show",
		SeriesSlug:        "show",
		SeriesAccentColor: "#F2A541",
		SeriesHasCover:    true,
		Number:            3,
		Title:             "Episode three",
		DurationSeconds:   duration,
		ChapterCount:      4,
	}
}

// catalogFor returns a catalog double answering one published episode.
func catalogFor(duration float64) *catalogporttest.MockCatalog {
	return &catalogporttest.MockCatalog{
		PublishedEpisodeFn: func(_ context.Context, ref domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
			return publishedInfo(ref, duration), nil
		},
	}
}

// emptyProgress returns a repository double that reports "no row" on Get and
// captures the saved aggregate.
func emptyProgress(saved **domain.Progress) *progressrepotest.MockProgressRepository {
	return &progressrepotest.MockProgressRepository{
		GetFn: func(context.Context, domain.ListenerID, domain.EpisodeRef) (*domain.Progress, error) {
			return nil, domain.ErrProgressNotFound
		},
		SaveFn: func(_ context.Context, progress *domain.Progress) error {
			*saved = progress
			return nil
		},
	}
}

func TestRecordPositionClampsWithCatalogDuration(t *testing.T) {
	var saved *domain.Progress
	application := newApp(emptyProgress(&saved), catalogFor(60))

	progress, err := application.RecordPosition(context.Background(), domain.LocalListener(), testEpisode, 120, nil)
	if err != nil {
		t.Fatalf("RecordPosition error = %v, want nil", err)
	}

	if progress != saved {
		t.Error("RecordPosition did not save the returned aggregate")
	}
	if progress.PositionSeconds() != 60 {
		t.Errorf("PositionSeconds = %v, want 60 (clamped)", progress.PositionSeconds())
	}
	if !progress.Completed() {
		t.Error("Completed = false, want true (position reached the duration)")
	}
	if !progress.CreatedAt().Equal(fixedNow) || !progress.UpdatedAt().Equal(fixedNow) {
		t.Errorf("timestamps = %v/%v, want %v", progress.CreatedAt(), progress.UpdatedAt(), fixedNow)
	}
	if progress.ListenerID() != testListener {
		t.Errorf("ListenerID = %q, want %q", progress.ListenerID(), testListener)
	}
}

func TestRecordPositionKeepsExistingProgress(t *testing.T) {
	created := fixedNow.Add(-2 * time.Hour)
	existing := domain.ReconstituteProgress(testListener, testEpisode, 30, false, created, created)
	var saved *domain.Progress
	repository := &progressrepotest.MockProgressRepository{
		GetFn: func(context.Context, domain.ListenerID, domain.EpisodeRef) (*domain.Progress, error) {
			return existing, nil
		},
		SaveFn: func(_ context.Context, progress *domain.Progress) error {
			saved = progress
			return nil
		},
	}

	progress, err := newApp(repository, catalogFor(60)).RecordPosition(context.Background(), domain.LocalListener(), testEpisode, 42, nil)
	if err != nil {
		t.Fatalf("RecordPosition error = %v, want nil", err)
	}

	if progress.PositionSeconds() != 42 {
		t.Errorf("PositionSeconds = %v, want 42", progress.PositionSeconds())
	}
	if progress.Completed() {
		t.Error("Completed = true, want false")
	}
	if !progress.CreatedAt().Equal(created) {
		t.Errorf("CreatedAt = %v, want the stored creation time %v", progress.CreatedAt(), created)
	}
	if saved != progress {
		t.Error("RecordPosition did not save the stored aggregate")
	}
}

func TestRecordPositionExplicitCompleted(t *testing.T) {
	tests := []struct {
		name          string
		position      float64
		completed     bool
		startPosition float64
		startComplete bool
		wantPosition  float64
		wantCompleted bool
	}{
		{
			name:          "completed true marks played whatever the position",
			position:      5,
			completed:     true,
			wantPosition:  60,
			wantCompleted: true,
		},
		{
			name:          "completed false records the position",
			position:      30,
			completed:     false,
			wantPosition:  30,
			wantCompleted: false,
		},
		{
			name:          "completed false reopens a played episode",
			position:      30,
			completed:     false,
			startPosition: 60,
			startComplete: true,
			wantPosition:  30,
			wantCompleted: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := domain.ReconstituteProgress(testListener, testEpisode, tt.startPosition, tt.startComplete, fixedNow.Add(-time.Hour), fixedNow.Add(-time.Hour))
			var saved *domain.Progress
			repository := &progressrepotest.MockProgressRepository{
				GetFn: func(context.Context, domain.ListenerID, domain.EpisodeRef) (*domain.Progress, error) {
					return existing, nil
				},
				SaveFn: func(_ context.Context, progress *domain.Progress) error {
					saved = progress
					return nil
				},
			}
			completed := tt.completed

			progress, err := newApp(repository, catalogFor(60)).RecordPosition(context.Background(), domain.LocalListener(), testEpisode, tt.position, &completed)
			if err != nil {
				t.Fatalf("RecordPosition error = %v, want nil", err)
			}
			if progress.PositionSeconds() != tt.wantPosition {
				t.Errorf("PositionSeconds = %v, want %v", progress.PositionSeconds(), tt.wantPosition)
			}
			if progress.Completed() != tt.wantCompleted {
				t.Errorf("Completed = %v, want %v", progress.Completed(), tt.wantCompleted)
			}
			if saved == nil {
				t.Error("RecordPosition did not save the aggregate")
			}
		})
	}
}

func TestRecordPositionErrors(t *testing.T) {
	tests := []struct {
		name    string
		catalog catalogport.Catalog
		actor   *domain.Actor
		pos     float64
		wantErr error
	}{
		{
			name: "unknown or draft episode",
			catalog: &catalogporttest.MockCatalog{
				PublishedEpisodeFn: func(context.Context, domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
					return catalogport.EpisodeInfo{}, domain.ErrEpisodeNotFound
				},
			},
			actor:   domain.LocalListener(),
			pos:     10,
			wantErr: domain.ErrEpisodeNotFound,
		},
		{
			name:    "negative position",
			catalog: catalogFor(60),
			actor:   domain.LocalListener(),
			pos:     -1,
			wantErr: domain.ErrInvalidPosition,
		},
		{
			name:    "nil actor",
			catalog: catalogFor(60),
			actor:   nil,
			pos:     10,
			wantErr: domain.ErrUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var saved *domain.Progress
			application := newApp(emptyProgress(&saved), tt.catalog)

			_, err := application.RecordPosition(context.Background(), tt.actor, testEpisode, tt.pos, nil)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("RecordPosition error = %v, want %v", err, tt.wantErr)
			}
			if saved != nil {
				t.Error("RecordPosition saved an aggregate on error")
			}
		})
	}
}

func TestRecordPositionNilActorDoesNotTouchTheCatalog(t *testing.T) {
	catalog := &catalogporttest.MockCatalog{}
	application := newApp(&progressrepotest.MockProgressRepository{}, catalog)

	if _, err := application.RecordPosition(context.Background(), nil, testEpisode, 10, nil); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("RecordPosition error = %v, want %v", err, domain.ErrUnauthorized)
	}
}

func TestMarkPlayed(t *testing.T) {
	var saved *domain.Progress
	application := newApp(emptyProgress(&saved), catalogFor(95.5))

	progress, err := application.MarkPlayed(context.Background(), domain.LocalListener(), testEpisode)
	if err != nil {
		t.Fatalf("MarkPlayed error = %v, want nil", err)
	}

	if progress.PositionSeconds() != 95.5 || !progress.Completed() {
		t.Errorf("progress = %v/%v, want 95.5/true", progress.PositionSeconds(), progress.Completed())
	}
	if saved != progress {
		t.Error("MarkPlayed did not save the aggregate")
	}
}

func TestMarkPlayedRejectsUnknownEpisodeAndNilActor(t *testing.T) {
	catalog := &catalogporttest.MockCatalog{
		PublishedEpisodeFn: func(context.Context, domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
			return catalogport.EpisodeInfo{}, domain.ErrEpisodeNotFound
		},
	}
	application := newApp(&progressrepotest.MockProgressRepository{}, catalog)

	if _, err := application.MarkPlayed(context.Background(), domain.LocalListener(), testEpisode); !errors.Is(err, domain.ErrEpisodeNotFound) {
		t.Fatalf("MarkPlayed error = %v, want %v", err, domain.ErrEpisodeNotFound)
	}
	if _, err := application.MarkPlayed(context.Background(), nil, testEpisode); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("MarkPlayed error = %v, want %v", err, domain.ErrUnauthorized)
	}
}

func TestMarkUnplayed(t *testing.T) {
	var deleted domain.EpisodeRef
	repository := &progressrepotest.MockProgressRepository{
		DeleteFn: func(_ context.Context, listener domain.ListenerID, episode domain.EpisodeRef) error {
			if listener != testListener {
				t.Errorf("Delete listener = %q, want %q", listener, testListener)
			}
			deleted = episode
			return nil
		},
	}

	if err := newApp(repository, catalogFor(60)).MarkUnplayed(context.Background(), domain.LocalListener(), testEpisode); err != nil {
		t.Fatalf("MarkUnplayed error = %v, want nil", err)
	}
	if deleted != testEpisode {
		t.Errorf("Delete episode = %q, want %q", deleted, testEpisode)
	}
}

func TestMarkUnplayedRejectsNilActor(t *testing.T) {
	application := newApp(&progressrepotest.MockProgressRepository{}, catalogFor(60))

	if err := application.MarkUnplayed(context.Background(), nil, testEpisode); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("MarkUnplayed error = %v, want %v", err, domain.ErrUnauthorized)
	}
}

func TestListProgress(t *testing.T) {
	rows := []*domain.Progress{
		domain.ReconstituteProgress(testListener, testEpisode, 10, false, fixedNow, fixedNow),
	}
	repository := &progressrepotest.MockProgressRepository{
		ListFn: func(_ context.Context, listener domain.ListenerID, filter progressrepo.Filter) ([]*domain.Progress, error) {
			if listener != testListener {
				t.Errorf("List listener = %q, want %q", listener, testListener)
			}
			if filter.InProgressOnly || filter.Limit != 0 {
				t.Errorf("List filter = %+v, want the zero filter", filter)
			}
			return rows, nil
		},
	}

	got, err := newApp(repository, catalogFor(60)).ListProgress(context.Background(), domain.LocalListener())
	if err != nil {
		t.Fatalf("ListProgress error = %v, want nil", err)
	}
	if len(got) != 1 || got[0] != rows[0] {
		t.Errorf("ListProgress = %v, want the stored rows", got)
	}
}

func TestListProgressRejectsNilActor(t *testing.T) {
	application := newApp(&progressrepotest.MockProgressRepository{}, catalogFor(60))

	if _, err := application.ListProgress(context.Background(), nil); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ListProgress error = %v, want %v", err, domain.ErrUnauthorized)
	}
}

func TestContinueListeningSkipsUnavailableEpisodesAndKeepsOrder(t *testing.T) {
	newest := domain.ReconstituteProgress(testListener, testEpisode, 30, false, fixedNow, fixedNow)
	gone := domain.ReconstituteProgress(testListener, testOther, 20, false, fixedNow, fixedNow.Add(-time.Minute))
	oldest := domain.ReconstituteProgress(testListener, domain.EpisodeRef("018f0000-0000-7000-8000-000000000003"), 10, false, fixedNow, fixedNow.Add(-2*time.Minute))

	repository := &progressrepotest.MockProgressRepository{
		ListFn: func(_ context.Context, _ domain.ListenerID, filter progressrepo.Filter) ([]*domain.Progress, error) {
			if !filter.InProgressOnly {
				t.Error("ContinueListening listed completed rows too")
			}
			return []*domain.Progress{newest, gone, oldest}, nil
		},
	}
	catalog := &catalogporttest.MockCatalog{
		PublishedEpisodeFn: func(_ context.Context, ref domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
			if ref == testOther {
				return catalogport.EpisodeInfo{}, domain.ErrEpisodeNotFound
			}
			return publishedInfo(ref, 60), nil
		},
	}

	items, err := newApp(repository, catalog).ContinueListening(context.Background(), domain.LocalListener(), 10)
	if err != nil {
		t.Fatalf("ContinueListening error = %v, want nil", err)
	}
	if len(items) != 2 {
		t.Fatalf("ContinueListening returned %d items, want 2", len(items))
	}
	if items[0].Progress != newest || items[1].Progress != oldest {
		t.Error("ContinueListening did not keep the repository order")
	}
	if items[0].Episode.Ref != testEpisode || items[0].Episode.Title != "Episode three" {
		t.Errorf("ContinueListening episode = %+v, want the catalog snapshot", items[0].Episode)
	}
}

func TestContinueListeningSkipsDoNotConsumeTheLimit(t *testing.T) {
	gone := domain.ReconstituteProgress(testListener, testOther, 20, false, fixedNow, fixedNow)
	first := domain.ReconstituteProgress(testListener, testEpisode, 10, false, fixedNow, fixedNow.Add(-time.Minute))
	second := domain.ReconstituteProgress(testListener, testThird, 5, false, fixedNow, fixedNow.Add(-2*time.Minute))

	repository := &progressrepotest.MockProgressRepository{
		ListFn: func(_ context.Context, _ domain.ListenerID, filter progressrepo.Filter) ([]*domain.Progress, error) {
			rows := []*domain.Progress{gone, first, second}
			if filter.Limit > 0 && len(rows) > filter.Limit {
				rows = rows[:filter.Limit]
			}
			return rows, nil
		},
	}
	catalog := &catalogporttest.MockCatalog{
		PublishedEpisodeFn: func(_ context.Context, ref domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
			if ref == testOther {
				return catalogport.EpisodeInfo{}, domain.ErrEpisodeNotFound
			}
			return publishedInfo(ref, 60), nil
		},
	}

	items, err := newApp(repository, catalog).ContinueListening(context.Background(), domain.LocalListener(), 2)
	if err != nil {
		t.Fatalf("ContinueListening error = %v, want nil", err)
	}
	if len(items) != 2 {
		t.Fatalf("ContinueListening returned %d items, want 2 available items", len(items))
	}
	if items[0].Progress != first || items[1].Progress != second {
		t.Error("a skipped episode consumed a slot: want the next two available episodes")
	}
}

func TestContinueListeningLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit int
		want  int
	}{
		{name: "zero falls back to the default", limit: 0, want: 5},
		{name: "negative falls back to the default", limit: -3, want: 5},
		{name: "keeps a custom limit", limit: 3, want: 3},
		{name: "caps the limit", limit: 500, want: 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rows := make([]*domain.Progress, 0, 60)
			for range 60 {
				rows = append(rows, domain.ReconstituteProgress(testListener, testEpisode, 10, false, fixedNow, fixedNow))
			}
			repository := &progressrepotest.MockProgressRepository{
				ListFn: func(_ context.Context, _ domain.ListenerID, filter progressrepo.Filter) ([]*domain.Progress, error) {
					if !filter.InProgressOnly {
						t.Error("ContinueListening listed completed rows too")
					}
					return rows, nil
				},
			}

			items, err := newApp(repository, catalogFor(60)).ContinueListening(context.Background(), domain.LocalListener(), tt.limit)
			if err != nil {
				t.Fatalf("ContinueListening error = %v, want nil", err)
			}
			if len(items) != tt.want {
				t.Errorf("ContinueListening returned %d items, want %d", len(items), tt.want)
			}
		})
	}
}

func TestContinueListeningPropagatesCatalogFailures(t *testing.T) {
	wantErr := errors.New("catalog is down")
	repository := &progressrepotest.MockProgressRepository{
		ListFn: func(context.Context, domain.ListenerID, progressrepo.Filter) ([]*domain.Progress, error) {
			return []*domain.Progress{domain.ReconstituteProgress(testListener, testEpisode, 10, false, fixedNow, fixedNow)}, nil
		},
	}
	catalog := &catalogporttest.MockCatalog{
		PublishedEpisodeFn: func(context.Context, domain.EpisodeRef) (catalogport.EpisodeInfo, error) {
			return catalogport.EpisodeInfo{}, wantErr
		},
	}

	if _, err := newApp(repository, catalog).ContinueListening(context.Background(), domain.LocalListener(), 5); !errors.Is(err, wantErr) {
		t.Fatalf("ContinueListening error = %v, want %v", err, wantErr)
	}
}

func TestContinueListeningRejectsNilActor(t *testing.T) {
	application := newApp(&progressrepotest.MockProgressRepository{}, catalogFor(60))

	if _, err := application.ContinueListening(context.Background(), nil, 5); !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("ContinueListening error = %v, want %v", err, domain.ErrUnauthorized)
	}
}
