package http_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/usecases/progressuc"
	listeninghttp "github.com/JLugagne/podcast/internal/listening/inbound/http"
	v1 "github.com/JLugagne/podcast/pkg/listening/http/v1"
)

// fakeApp is a hand-written App double: every method panics unless its function
// field is set, which asserts that a handler calls only what the test expects.
type fakeApp struct {
	recordPositionFn func(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef, position float64, completed *bool) (*domain.Progress, error)
	markPlayedFn     func(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) (*domain.Progress, error)
	markUnplayedFn   func(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) error
	listProgressFn   func(ctx context.Context, actor *domain.Actor) ([]*domain.Progress, error)
	continueFn       func(ctx context.Context, actor *domain.Actor, limit int) ([]progressuc.ContinueItem, error)
}

var _ listeninghttp.App = (*fakeApp)(nil)

func (f *fakeApp) RecordPosition(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef, position float64, completed *bool) (*domain.Progress, error) {
	if f.recordPositionFn == nil {
		panic("fakeApp.RecordPositionFn not set")
	}
	return f.recordPositionFn(ctx, actor, episode, position, completed)
}

func (f *fakeApp) MarkPlayed(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) (*domain.Progress, error) {
	if f.markPlayedFn == nil {
		panic("fakeApp.MarkPlayedFn not set")
	}
	return f.markPlayedFn(ctx, actor, episode)
}

func (f *fakeApp) MarkUnplayed(ctx context.Context, actor *domain.Actor, episode domain.EpisodeRef) error {
	if f.markUnplayedFn == nil {
		panic("fakeApp.MarkUnplayedFn not set")
	}
	return f.markUnplayedFn(ctx, actor, episode)
}

func (f *fakeApp) ListProgress(ctx context.Context, actor *domain.Actor) ([]*domain.Progress, error) {
	if f.listProgressFn == nil {
		panic("fakeApp.ListProgressFn not set")
	}
	return f.listProgressFn(ctx, actor)
}

func (f *fakeApp) ContinueListening(ctx context.Context, actor *domain.Actor, limit int) ([]progressuc.ContinueItem, error) {
	if f.continueFn == nil {
		panic("fakeApp.ContinueListeningFn not set")
	}
	return f.continueFn(ctx, actor, limit)
}

const (
	testEpisodeID = "018f0000-0000-7000-8000-000000000001"
	testSeriesID  = "018f0000-0000-7000-8000-0000000000aa"
)

var testNow = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

// storedProgress builds a progress fixture.
func storedProgress() *domain.Progress {
	return domain.ReconstituteProgress(domain.LocalListenerID, domain.EpisodeRef(testEpisodeID), 12.5, false, testNow.Add(-time.Hour), testNow)
}

// episodeInfo builds a catalog snapshot fixture.
func episodeInfo() progressuc.ContinueItem {
	return progressuc.ContinueItem{
		Progress: storedProgress(),
		Episode:  catalogEpisodeInfo(),
	}
}

func catalogEpisodeInfo() (info catalogportEpisodeInfo) {
	info.Ref = domain.EpisodeRef(testEpisodeID)
	info.SeriesID = testSeriesID
	info.SeriesTitle = "Show"
	info.SeriesSlug = "show"
	info.SeriesAccentColor = "#F2A541"
	info.SeriesHasCover = true
	info.Number = 3
	info.Title = "Episode three"
	info.DurationSeconds = 300
	info.ChapterCount = 4
	return info
}

// newTestRouter mounts the listening routes.
func newTestRouter(app listeninghttp.App) http.Handler {
	router := mux.NewRouter()
	listeninghttp.SetupRouter(router, app, listeninghttp.Options{})
	return router
}

// request builds a JSON or bodyless request against path.
func request(t *testing.T, method, path, body string) *http.Request {
	t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequestWithContext(context.Background(), method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	return req
}

// serve runs req through h and returns the recorder.
func serve(t *testing.T, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// decodeJSON decodes the response body into T.
func decodeJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var out T
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode response %q: %v", rec.Body.String(), err)
	}
	return out
}

// requireStatus asserts the status code.
func requireStatus(t *testing.T, rec *httptest.ResponseRecorder, want int) {
	t.Helper()
	if rec.Code != want {
		t.Fatalf("status = %d, want %d (body %s)", rec.Code, want, rec.Body.String())
	}
}

// requireError asserts the status code and the error code of the body.
func requireError(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	requireStatus(t, rec, status)
	body := decodeJSON[v1.Error](t, rec)
	if body.Code != code {
		t.Fatalf("error code = %q, want %q (body %s)", body.Code, code, rec.Body.String())
	}
	if body.Message == "" {
		t.Fatalf("error message is empty (body %s)", rec.Body.String())
	}
}
