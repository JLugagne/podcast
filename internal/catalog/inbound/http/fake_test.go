package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	cataloghttp "github.com/JLugagne/podcast/internal/catalog/inbound/http"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
	"github.com/gorilla/mux"
)

// fakeApp is a hand-written App double: every method panics unless its
// function field is set, which asserts that a handler calls only what the test
// expects.
type fakeApp struct {
	createSeriesFn func(ctx context.Context, actor *domain.Actor, attrs domain.SeriesAttributes) (*seriesuc.SeriesView, error)
	updateSeriesFn func(ctx context.Context, actor *domain.Actor, id domain.SeriesID, attrs domain.SeriesAttributes, version int) (*seriesuc.SeriesView, error)
	deleteSeriesFn func(ctx context.Context, actor *domain.Actor, id domain.SeriesID) error
	uploadCoverFn  func(ctx context.Context, actor *domain.Actor, id domain.SeriesID, in seriesuc.UploadInput) (*seriesuc.SeriesView, error)
	removeCoverFn  func(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error)

	getSeriesFn       func(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error)
	getSeriesBySlugFn func(ctx context.Context, actor *domain.Actor, slug string) (*seriesuc.SeriesView, error)
	listSeriesFn      func(ctx context.Context, actor *domain.Actor, q seriesuc.ListQuery) ([]*seriesuc.SeriesView, error)
	openCoverFn       func(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (blobstore.Blob, error)

	createEpisodeFn func(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, in episodeuc.CreateInput) (*domain.Episode, error)
	updateEpisodeFn func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, attrs domain.EpisodeAttributes, version int) (*domain.Episode, error)
	deleteEpisodeFn func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) error
	uploadAudioFn   func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, in episodeuc.AudioUploadInput) (*domain.Episode, error)
	setChaptersFn   func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, chapters []domain.Chapter) (*domain.Episode, error)
	publishFn       func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error)
	unpublishFn     func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error)

	getEpisodeFn   func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error)
	listEpisodesFn func(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, q episodeuc.ListQuery) ([]*domain.Episode, error)
	openAudioFn    func(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (blobstore.Blob, error)
}

func (f *fakeApp) CreateSeries(ctx context.Context, actor *domain.Actor, attrs domain.SeriesAttributes) (*seriesuc.SeriesView, error) {
	if f.createSeriesFn == nil {
		panic("fakeApp.CreateSeriesFn not set")
	}
	return f.createSeriesFn(ctx, actor, attrs)
}

func (f *fakeApp) UpdateSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID, attrs domain.SeriesAttributes, version int) (*seriesuc.SeriesView, error) {
	if f.updateSeriesFn == nil {
		panic("fakeApp.UpdateSeriesFn not set")
	}
	return f.updateSeriesFn(ctx, actor, id, attrs, version)
}

func (f *fakeApp) DeleteSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID) error {
	if f.deleteSeriesFn == nil {
		panic("fakeApp.DeleteSeriesFn not set")
	}
	return f.deleteSeriesFn(ctx, actor, id)
}

func (f *fakeApp) UploadCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID, in seriesuc.UploadInput) (*seriesuc.SeriesView, error) {
	if f.uploadCoverFn == nil {
		panic("fakeApp.UploadCoverFn not set")
	}
	return f.uploadCoverFn(ctx, actor, id, in)
}

func (f *fakeApp) RemoveCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error) {
	if f.removeCoverFn == nil {
		panic("fakeApp.RemoveCoverFn not set")
	}
	return f.removeCoverFn(ctx, actor, id)
}

func (f *fakeApp) GetSeries(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error) {
	if f.getSeriesFn == nil {
		panic("fakeApp.GetSeriesFn not set")
	}
	return f.getSeriesFn(ctx, actor, id)
}

func (f *fakeApp) ListSeries(ctx context.Context, actor *domain.Actor, q seriesuc.ListQuery) ([]*seriesuc.SeriesView, error) {
	if f.listSeriesFn == nil {
		panic("fakeApp.ListSeriesFn not set")
	}
	return f.listSeriesFn(ctx, actor, q)
}

func (f *fakeApp) OpenCover(ctx context.Context, actor *domain.Actor, id domain.SeriesID) (blobstore.Blob, error) {
	if f.openCoverFn == nil {
		panic("fakeApp.OpenCoverFn not set")
	}
	return f.openCoverFn(ctx, actor, id)
}

func (f *fakeApp) CreateEpisode(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, in episodeuc.CreateInput) (*domain.Episode, error) {
	if f.createEpisodeFn == nil {
		panic("fakeApp.CreateEpisodeFn not set")
	}
	return f.createEpisodeFn(ctx, actor, seriesID, in)
}

func (f *fakeApp) UpdateEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, attrs domain.EpisodeAttributes, version int) (*domain.Episode, error) {
	if f.updateEpisodeFn == nil {
		panic("fakeApp.UpdateEpisodeFn not set")
	}
	return f.updateEpisodeFn(ctx, actor, id, attrs, version)
}

func (f *fakeApp) DeleteEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) error {
	if f.deleteEpisodeFn == nil {
		panic("fakeApp.DeleteEpisodeFn not set")
	}
	return f.deleteEpisodeFn(ctx, actor, id)
}

func (f *fakeApp) UploadAudio(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, in episodeuc.AudioUploadInput) (*domain.Episode, error) {
	if f.uploadAudioFn == nil {
		panic("fakeApp.UploadAudioFn not set")
	}
	return f.uploadAudioFn(ctx, actor, id, in)
}

func (f *fakeApp) SetChapters(ctx context.Context, actor *domain.Actor, id domain.EpisodeID, chapters []domain.Chapter) (*domain.Episode, error) {
	if f.setChaptersFn == nil {
		panic("fakeApp.SetChaptersFn not set")
	}
	return f.setChaptersFn(ctx, actor, id, chapters)
}

func (f *fakeApp) Publish(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
	if f.publishFn == nil {
		panic("fakeApp.PublishFn not set")
	}
	return f.publishFn(ctx, actor, id)
}

func (f *fakeApp) Unpublish(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
	if f.unpublishFn == nil {
		panic("fakeApp.UnpublishFn not set")
	}
	return f.unpublishFn(ctx, actor, id)
}

func (f *fakeApp) GetEpisode(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
	if f.getEpisodeFn == nil {
		panic("fakeApp.GetEpisodeFn not set")
	}
	return f.getEpisodeFn(ctx, actor, id)
}

func (f *fakeApp) ListEpisodes(ctx context.Context, actor *domain.Actor, seriesID domain.SeriesID, q episodeuc.ListQuery) ([]*domain.Episode, error) {
	if f.listEpisodesFn == nil {
		panic("fakeApp.ListEpisodesFn not set")
	}
	return f.listEpisodesFn(ctx, actor, seriesID, q)
}

func (f *fakeApp) OpenAudio(ctx context.Context, actor *domain.Actor, id domain.EpisodeID) (blobstore.Blob, error) {
	if f.openAudioFn == nil {
		panic("fakeApp.OpenAudioFn not set")
	}
	return f.openAudioFn(ctx, actor, id)
}

// Test constants: valid UUIDs and a fixed clock shared by every fixture.
const (
	testSeriesID  = "018f0000-0000-7000-8000-000000000001"
	testEpisodeID = "018f0000-0000-7000-8000-000000000002"
	testOtherID   = "018f0000-0000-7000-8000-000000000003"
)

var testNow = time.Date(2026, 10, 21, 12, 30, 0, 0, time.UTC)

// newSeries builds a stored series with a cover, version 3.
func newSeries() *domain.Series {
	return domain.ReconstituteSeries(
		domain.SeriesID(testSeriesID),
		domain.SeriesAttributes{Title: "Show", Author: "Host", Description: "About", AccentColor: "#7A9CFF", Slug: "show"},
		&domain.Asset{Key: "series/" + testSeriesID + "/cover.png", ContentType: "image/png", SizeBytes: 8},
		testNow, testNow.Add(time.Hour), 3,
	)
}

// newDraftEpisode builds a stored draft episode with audio.
func newDraftEpisode() *domain.Episode {
	return episodeWith(domain.EpisodeStatusDraft, nil)
}

// newPublishedEpisode builds a published episode with audio and chapters.
func newPublishedEpisode() *domain.Episode {
	published := testNow.Add(2 * time.Hour)
	chapters := []domain.Chapter{{StartSeconds: 0, Title: "Intro", TOC: true}}
	return episodeWith(domain.EpisodeStatusPublished, &published, chapters...)
}

// newEpisodeWithChapters builds a draft episode carrying the given chapters.
func newEpisodeWithChapters(chapters ...domain.Chapter) *domain.Episode {
	return episodeWith(domain.EpisodeStatusDraft, nil, chapters...)
}

func episodeWith(status domain.EpisodeStatus, publishedAt *time.Time, chapters ...domain.Chapter) *domain.Episode {
	audio := &domain.AudioFile{
		Asset:           domain.Asset{Key: "episodes/" + testEpisodeID + "/audio.mp3", ContentType: "audio/mpeg", SizeBytes: 42},
		DurationSeconds: 95.5,
		OriginalName:    "episode.mp3",
	}
	return domain.ReconstituteEpisode(
		domain.EpisodeID(testEpisodeID),
		domain.SeriesID(testSeriesID),
		domain.EpisodeAttributes{Number: 1, Title: "Episode", Description: "Details"},
		status, audio, chapters, publishedAt,
		testNow, testNow.Add(time.Hour), 4,
	)
}

// fakeBlob serves fixed bytes through the blobstore.Blob contract.
type fakeBlob struct{ *bytes.Reader }

func (fakeBlob) Close() error { return nil }

// newBlob wraps data in a readable blob with the given content type.
func newBlob(data []byte, contentType string) blobstore.Blob {
	return blobstore.Blob{
		ReadSeekCloser: fakeBlob{bytes.NewReader(data)},
		Size:           int64(len(data)),
		ContentType:    contentType,
		ModTime:        testNow,
	}
}

// newTestRouter mounts the catalog routes with the given upload limits.
func newTestRouter(t *testing.T, app *fakeApp, opts cataloghttp.Options) http.Handler {
	t.Helper()
	router := mux.NewRouter()
	cataloghttp.SetupRouter(router, app, opts)
	return router
}

// serve runs req through h and returns the recorder.
func serve(t *testing.T, h http.Handler, req *http.Request) *httptest.ResponseRecorder {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
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

// uploadRequest builds a multipart request. write must fill the writer.
func uploadRequest(t *testing.T, path string, write func(*multipart.Writer)) *http.Request {
	t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	write(writer)
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}
	req := httptest.NewRequestWithContext(context.Background(), http.MethodPut, path, &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

// addFile appends a file part with an explicit content type.
func addFile(t *testing.T, writer *multipart.Writer, filename, contentType string, data []byte) {
	t.Helper()
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename=%q`, "file", filename))
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	part, err := writer.CreatePart(header)
	if err != nil {
		t.Fatalf("create file part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write file part: %v", err)
	}
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

// requireStatus asserts the status code and returns the recorder.
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

func (f *fakeApp) GetSeriesBySlug(ctx context.Context, actor *domain.Actor, slug string) (*seriesuc.SeriesView, error) {
	if f.getSeriesBySlugFn == nil {
		panic("fakeApp.GetSeriesBySlugFn not set")
	}
	return f.getSeriesBySlugFn(ctx, actor, slug)
}
