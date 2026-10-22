package http_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/catalogport"
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
	v1 "github.com/JLugagne/podcast/pkg/listening/http/v1"
)

// catalogportEpisodeInfo is a local alias so fixtures stay readable.
type catalogportEpisodeInfo = catalogport.EpisodeInfo

const progressPath = "/api/v1/listening/progress/" + testEpisodeID

func TestRecordPosition(t *testing.T) {
	var (
		gotEpisode   domain.EpisodeRef
		gotPosition  float64
		gotCompleted *bool
	)
	app := &fakeApp{recordPositionFn: func(_ context.Context, actor *domain.Actor, episode domain.EpisodeRef, position float64, completed *bool) (*domain.Progress, error) {
		if actor == nil || actor.ID() != "local" {
			t.Errorf("actor = %v, want the local listener", actor)
		}
		gotEpisode, gotPosition, gotCompleted = episode, position, completed
		return storedProgress(), nil
	}}

	rec := serve(t, newTestRouter(app), request(t, http.MethodPut, progressPath, `{"position_seconds":12.5}`))
	requireStatus(t, rec, http.StatusOK)

	body := decodeJSON[v1.Progress](t, rec)
	if body.EpisodeID != testEpisodeID || body.PositionSeconds != 12.5 || body.Completed {
		t.Errorf("body = %+v, want the stored progress", body)
	}
	if body.UpdatedAt != "2026-10-22T12:00:00Z" {
		t.Errorf("updated_at = %q, want the RFC3339 timestamp", body.UpdatedAt)
	}
	if gotEpisode != domain.EpisodeRef(testEpisodeID) || gotPosition != 12.5 {
		t.Errorf("handler passed %q/%v", gotEpisode, gotPosition)
	}
	if gotCompleted != nil {
		t.Errorf("completed = %v, want nil", *gotCompleted)
	}
}

func TestRecordPositionExplicitCompleted(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{name: "completed true", body: `{"position_seconds":0,"completed":true}`, want: true},
		{name: "completed false", body: `{"position_seconds":0,"completed":false}`, want: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got *bool
			app := &fakeApp{recordPositionFn: func(_ context.Context, _ *domain.Actor, _ domain.EpisodeRef, _ float64, completed *bool) (*domain.Progress, error) {
				got = completed
				return storedProgress(), nil
			}}

			serve(t, newTestRouter(app), request(t, http.MethodPut, progressPath, tc.body))

			if got == nil {
				t.Fatal("completed = nil, want a pointer")
			}
			if *got != tc.want {
				t.Errorf("completed = %v, want %v", *got, tc.want)
			}
		})
	}
}

func TestRecordPositionErrors(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		body       string
		appErr     error
		wantStatus int
		wantCode   string
	}{
		{
			name:       "unknown episode",
			path:       progressPath,
			body:       `{"position_seconds":10}`,
			appErr:     domain.ErrEpisodeNotFound,
			wantStatus: http.StatusNotFound,
			wantCode:   "EPISODE_NOT_FOUND",
		},
		{
			name:       "negative position",
			path:       progressPath,
			body:       `{"position_seconds":-1}`,
			appErr:     domain.ErrInvalidPosition,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "INVALID_POSITION",
		},
		{
			name:       "malformed episode id",
			path:       "/api/v1/listening/progress/not-a-uuid",
			body:       `{"position_seconds":10}`,
			wantStatus: http.StatusUnprocessableEntity,
			wantCode:   "INVALID_ID",
		},
		{
			name:       "malformed json",
			path:       progressPath,
			body:       `{"position_seconds":`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "unknown field",
			path:       progressPath,
			body:       `{"position_seconds":10,"extra":true}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   "BAD_REQUEST",
		},
		{
			name:       "unexpected failure",
			path:       progressPath,
			body:       `{"position_seconds":10}`,
			appErr:     errors.New("boom"),
			wantStatus: http.StatusInternalServerError,
			wantCode:   "INTERNAL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &fakeApp{recordPositionFn: func(context.Context, *domain.Actor, domain.EpisodeRef, float64, *bool) (*domain.Progress, error) {
				return nil, tt.appErr
			}}
			rec := serve(t, newTestRouter(app), request(t, http.MethodPut, tt.path, tt.body))
			requireError(t, rec, tt.wantStatus, tt.wantCode)
		})
	}
}

func TestRecordPositionRejectsMissingBody(t *testing.T) {
	app := &fakeApp{}
	rec := serve(t, newTestRouter(app), request(t, http.MethodPut, progressPath, ""))
	requireError(t, rec, http.StatusBadRequest, "BAD_REQUEST")
}

func TestMarkPlayed(t *testing.T) {
	completed := domain.ReconstituteProgress(domain.LocalListenerID, domain.EpisodeRef(testEpisodeID), 300, true, testNow, testNow)
	app := &fakeApp{markPlayedFn: func(_ context.Context, _ *domain.Actor, episode domain.EpisodeRef) (*domain.Progress, error) {
		if episode != domain.EpisodeRef(testEpisodeID) {
			t.Errorf("episode = %q, want %q", episode, testEpisodeID)
		}
		return completed, nil
	}}

	rec := serve(t, newTestRouter(app), request(t, http.MethodPost, progressPath+"/played", ""))
	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.Progress](t, rec)
	if !body.Completed || body.PositionSeconds != 300 {
		t.Errorf("body = %+v, want a completed progress", body)
	}
}

func TestMarkPlayedErrors(t *testing.T) {
	app := &fakeApp{markPlayedFn: func(context.Context, *domain.Actor, domain.EpisodeRef) (*domain.Progress, error) {
		return nil, domain.ErrEpisodeNotFound
	}}
	rec := serve(t, newTestRouter(app), request(t, http.MethodPost, progressPath+"/played", ""))
	requireError(t, rec, http.StatusNotFound, "EPISODE_NOT_FOUND")

	rec = serve(t, newTestRouter(app), request(t, http.MethodPost, "/api/v1/listening/progress/bad/played", ""))
	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_ID")
}

func TestMarkUnplayed(t *testing.T) {
	var got domain.EpisodeRef
	app := &fakeApp{markUnplayedFn: func(_ context.Context, _ *domain.Actor, episode domain.EpisodeRef) error {
		got = episode
		return nil
	}}

	rec := serve(t, newTestRouter(app), request(t, http.MethodDelete, progressPath, ""))
	requireStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
	if got != domain.EpisodeRef(testEpisodeID) {
		t.Errorf("episode = %q, want %q", got, testEpisodeID)
	}
}

func TestMarkUnplayedErrors(t *testing.T) {
	app := &fakeApp{markUnplayedFn: func(context.Context, *domain.Actor, domain.EpisodeRef) error {
		return domain.ErrProgressNotFound
	}}
	rec := serve(t, newTestRouter(app), request(t, http.MethodDelete, progressPath, ""))
	requireError(t, rec, http.StatusNotFound, "PROGRESS_NOT_FOUND")

	rec = serve(t, newTestRouter(app), request(t, http.MethodDelete, "/api/v1/listening/progress/bad", ""))
	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_ID")
}

func TestListProgress(t *testing.T) {
	complete := domain.ReconstituteProgress(domain.LocalListenerID, domain.EpisodeRef(testEpisodeID), 300, true, testNow, testNow)
	app := &fakeApp{listProgressFn: func(_ context.Context, actor *domain.Actor) ([]*domain.Progress, error) {
		if actor == nil || actor.ID() != "local" {
			t.Errorf("actor = %v, want the local listener", actor)
		}
		return []*domain.Progress{storedProgress(), complete}, nil
	}}

	rec := serve(t, newTestRouter(app), request(t, http.MethodGet, "/api/v1/listening/progress", ""))
	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.ProgressList](t, rec)
	if len(body.Progress) != 2 {
		t.Fatalf("progress = %d rows, want 2", len(body.Progress))
	}
	if body.Progress[1].Completed != true {
		t.Errorf("second row = %+v, want completed", body.Progress[1])
	}
}

func TestListProgressEmptyListIsNotNull(t *testing.T) {
	app := &fakeApp{listProgressFn: func(context.Context, *domain.Actor) ([]*domain.Progress, error) {
		return nil, nil
	}}

	rec := serve(t, newTestRouter(app), request(t, http.MethodGet, "/api/v1/listening/progress", ""))
	requireStatus(t, rec, http.StatusOK)
	if body := rec.Body.String(); body != "{\"progress\":[]}\n" {
		t.Errorf("body = %q, want an empty JSON list", body)
	}
}

func TestListProgressErrors(t *testing.T) {
	app := &fakeApp{listProgressFn: func(context.Context, *domain.Actor) ([]*domain.Progress, error) {
		return nil, errors.New("boom")
	}}
	rec := serve(t, newTestRouter(app), request(t, http.MethodGet, "/api/v1/listening/progress", ""))
	requireError(t, rec, http.StatusInternalServerError, "INTERNAL")
}

func TestContinueListening(t *testing.T) {
	tests := []struct {
		name      string
		query     string
		wantLimit int
	}{
		{name: "default limit", query: "", wantLimit: 0},
		{name: "explicit limit", query: "?limit=3", wantLimit: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got int
			app := &fakeApp{continueFn: func(_ context.Context, actor *domain.Actor, limit int) ([]progressuc.ContinueItem, error) {
				if actor == nil || actor.ID() != "local" {
					t.Errorf("actor = %v, want the local listener", actor)
				}
				got = limit
				return []progressuc.ContinueItem{episodeInfo()}, nil
			}}

			rec := serve(t, newTestRouter(app), request(t, http.MethodGet, "/api/v1/listening/continue"+tt.query, ""))
			requireStatus(t, rec, http.StatusOK)

			body := decodeJSON[v1.ContinueList](t, rec)
			if len(body.Items) != 1 {
				t.Fatalf("items = %d, want 1", len(body.Items))
			}
			item := body.Items[0]
			if item.Episode.ID != testEpisodeID || item.Episode.SeriesID != testSeriesID || item.Episode.SeriesTitle != "Show" ||
				item.Episode.SeriesSlug != "show" || item.Episode.AccentColor != "#F2A541" || item.Episode.ChapterCount != 4 ||
				item.Episode.DurationSeconds != 300 || item.Episode.Number != 3 || item.Episode.Title != "Episode three" {
				t.Errorf("episode = %+v", item.Episode)
			}
			if item.Episode.CoverURL == nil || *item.Episode.CoverURL != "/api/v1/series/"+testSeriesID+"/cover" {
				t.Errorf("cover_url = %v, want the series cover path", item.Episode.CoverURL)
			}
			if item.Progress.EpisodeID != testEpisodeID {
				t.Errorf("progress.episode_id = %q, want %q", item.Progress.EpisodeID, testEpisodeID)
			}
			if got != tt.wantLimit {
				t.Errorf("limit = %d, want %d", got, tt.wantLimit)
			}
		})
	}
}

func TestContinueListeningWithoutCover(t *testing.T) {
	item := episodeInfo()
	item.Episode.SeriesHasCover = false
	app := &fakeApp{continueFn: func(context.Context, *domain.Actor, int) ([]progressuc.ContinueItem, error) {
		return []progressuc.ContinueItem{item}, nil
	}}

	rec := serve(t, newTestRouter(app), request(t, http.MethodGet, "/api/v1/listening/continue", ""))
	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.ContinueList](t, rec)
	if body.Items[0].Episode.CoverURL != nil {
		t.Errorf("cover_url = %v, want null", body.Items[0].Episode.CoverURL)
	}
}

func TestContinueListeningRejectsBadLimit(t *testing.T) {
	for _, query := range []string{"?limit=abc", "?limit=-1"} {
		t.Run(query, func(t *testing.T) {
			app := &fakeApp{}
			rec := serve(t, newTestRouter(app), request(t, http.MethodGet, "/api/v1/listening/continue"+query, ""))
			requireError(t, rec, http.StatusBadRequest, "BAD_REQUEST")
		})
	}
}

func TestContinueListeningErrors(t *testing.T) {
	app := &fakeApp{continueFn: func(context.Context, *domain.Actor, int) ([]progressuc.ContinueItem, error) {
		return nil, errors.New("boom")
	}}
	rec := serve(t, newTestRouter(app), request(t, http.MethodGet, "/api/v1/listening/continue", ""))
	requireError(t, rec, http.StatusInternalServerError, "INTERNAL")
}
