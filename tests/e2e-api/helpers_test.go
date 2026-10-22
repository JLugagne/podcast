package e2eapi_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JLugagne/podcast/cmd/podcast"
	"github.com/JLugagne/podcast/internal/pkg/httpserver"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

const (
	// maxAudioBytes is the small audio limit used by the e2e environment so a
	// single oversized upload is enough to trigger a 413.
	maxAudioBytes = 1 << 20
	// maxImageBytes is the image limit of the e2e environment.
	maxImageBytes = 64 << 10
	// publicURL is the externally reachable base URL used for absolute feed
	// links.
	publicURL = "https://podcast.example"
)

// env is one isolated API test environment: a fresh router, blob directory and
// truncated catalog tables.
type env struct {
	t        *testing.T
	server   *httptest.Server
	client   *http.Client
	pool     *pgxpool.Pool
	blobsDir string
}

// requireDB skips the test when the shared database is unavailable and clears
// the catalog and listening tables so every test starts from a known state.
func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
	if _, err := testPool.Exec(context.Background(), "TRUNCATE listening_progress, catalog_episodes, catalog_series"); err != nil {
		t.Fatalf("truncate catalog and listening tables: %v", err)
	}
	return testPool
}

// newEnv builds the full serve handler — catalog API, admin and player SPAs and
// health probe — over the shared database and a per-test blob directory.
func newEnv(t *testing.T) *env {
	t.Helper()
	pool := requireDB(t)
	dir := t.TempDir()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	handler, err := podcast.NewHandler(podcast.HandlerOptions{
		Ctx:           context.Background(),
		Pool:          pool,
		StorageDir:    dir,
		MaxAudioBytes: maxAudioBytes,
		MaxImageBytes: maxImageBytes,
		PublicURL:     publicURL,
		Logger:        log,
	})
	if err != nil {
		t.Fatalf("build serve handler: %v", err)
	}

	server := httptest.NewServer(httpserver.New("", handler, log).Handler())
	t.Cleanup(server.Close)

	return &env{t: t, server: server, client: server.Client(), pool: pool, blobsDir: dir}
}

// apiResponse captures a complete HTTP response.
type apiResponse struct {
	t      *testing.T
	status int
	header http.Header
	body   []byte
}

// request performs one HTTP request and reads the whole response body.
func (e *env) request(method, path string, body io.Reader, contentType string) *apiResponse {
	e.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), method, e.server.URL+path, body)
	if err != nil {
		e.t.Fatalf("%s %s: build request: %v", method, path, err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("%s %s: send request: %v", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("%s %s: read response: %v", method, path, err)
	}
	return &apiResponse{t: e.t, status: resp.StatusCode, header: resp.Header, body: data}
}

// get performs a GET request.
func (e *env) get(path string) *apiResponse {
	e.t.Helper()
	return e.request(http.MethodGet, path, nil, "")
}

// getWithHeaders performs a GET request with extra request headers, such as a
// Range header.
func (e *env) getWithHeaders(path string, headers map[string]string) *apiResponse {
	e.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, e.server.URL+path, nil)
	if err != nil {
		e.t.Fatalf("GET %s: build request: %v", path, err)
	}
	for name, value := range headers {
		req.Header.Set(name, value)
	}
	resp, err := e.client.Do(req)
	if err != nil {
		e.t.Fatalf("GET %s: send request: %v", path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		e.t.Fatalf("GET %s: read response: %v", path, err)
	}
	return &apiResponse{t: e.t, status: resp.StatusCode, header: resp.Header, body: data}
}

// sendJSON sends payload as a JSON body, decoding the response into out when it
// is not nil.
func (e *env) sendJSON(method, path string, payload, out any) *apiResponse {
	e.t.Helper()
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			e.t.Fatalf("%s %s: encode payload: %v", method, path, err)
		}
		body = bytes.NewReader(raw)
	}
	resp := e.request(method, path, body, "application/json")
	if out != nil {
		resp.decode(out)
	}
	return resp
}

// multipartPart is one ordered part of a multipart request. A filename makes
// the part a file part; otherwise it is a plain form field.
type multipartPart struct {
	name        string
	filename    string
	contentType string
	data        []byte
}

// upload sends an ordered multipart body. Part order is preserved, which matters
// for the audio endpoint: duration_seconds must precede the file part.
func (e *env) upload(path string, parts []multipartPart) *apiResponse {
	e.t.Helper()
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	for _, part := range parts {
		if part.filename == "" {
			if err := writer.WriteField(part.name, string(part.data)); err != nil {
				e.t.Fatalf("%s %s: write field %q: %v", http.MethodPut, path, part.name, err)
			}
			continue
		}
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, part.name, part.filename))
		header.Set("Content-Type", part.contentType)
		filePart, err := writer.CreatePart(header)
		if err != nil {
			e.t.Fatalf("%s %s: create file part %q: %v", http.MethodPut, path, part.name, err)
		}
		if _, err := filePart.Write(part.data); err != nil {
			e.t.Fatalf("%s %s: write file part %q: %v", http.MethodPut, path, part.name, err)
		}
	}
	if err := writer.Close(); err != nil {
		e.t.Fatalf("%s %s: close multipart body: %v", http.MethodPut, path, err)
	}
	return e.request(http.MethodPut, path, bytes.NewReader(buf.Bytes()), writer.FormDataContentType())
}

// decode unmarshals the response body into out.
func (r *apiResponse) decode(out any) {
	r.t.Helper()
	if len(r.body) == 0 {
		r.t.Fatalf("decode: empty body (status %d)", r.status)
	}
	if err := json.Unmarshal(r.body, out); err != nil {
		r.t.Fatalf("decode: %v (body %q)", err, string(r.body))
	}
}

// wantStatus asserts the response status and returns the response.
func (r *apiResponse) wantStatus(status int) *apiResponse {
	r.t.Helper()
	if r.status != status {
		r.t.Fatalf("status = %d, want %d (body %q)", r.status, status, string(r.body))
	}
	return r
}

// wantError asserts the status and the {code} of an error response.
func (r *apiResponse) wantError(status int, code string) {
	r.t.Helper()
	r.wantStatus(status)
	var body v1.Error
	r.decode(&body)
	if body.Code != code {
		r.t.Fatalf("error code = %q, want %q (body %q)", body.Code, code, string(r.body))
	}
	if body.Message == "" {
		r.t.Fatalf("error %q has an empty message", code)
	}
}

// createSeries creates a series and returns its public representation.
func (e *env) createSeries(in v1.SeriesInput) v1.Series {
	e.t.Helper()
	var out v1.Series
	e.sendJSON(http.MethodPost, "/api/v1/series", in, &out).wantStatus(http.StatusCreated)
	return out
}

// createEpisode creates an episode and returns its public representation.
func (e *env) createEpisode(seriesID string, in v1.EpisodeInput) v1.Episode {
	e.t.Helper()
	var out v1.Episode
	e.sendJSON(http.MethodPost, "/api/v1/series/"+seriesID+"/episodes", in, &out).wantStatus(http.StatusCreated)
	return out
}

// uploadAudio uploads audio bytes with an optional duration and returns the
// updated episode.
func (e *env) uploadAudio(episodeID string, data []byte, fileName string, duration *float64) v1.Episode {
	e.t.Helper()
	parts := make([]multipartPart, 0, 2)
	if duration != nil {
		parts = append(parts, multipartPart{name: "duration_seconds", data: []byte(formatFloat(*duration))})
	}
	parts = append(parts, multipartPart{name: "file", filename: fileName, contentType: "audio/mpeg", data: data})
	var out v1.Episode
	e.upload("/api/v1/episodes/"+episodeID+"/audio", parts).wantStatus(http.StatusOK).decode(&out)
	return out
}

// setChapters replaces the chapters of an episode with a valid document.
func (e *env) setChapters(episodeID string, doc v1.ChaptersDocument) v1.Episode {
	e.t.Helper()
	var out v1.Episode
	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episodeID+"/chapters", doc, &out).wantStatus(http.StatusOK)
	return out
}

// publish marks an episode published and returns its public representation.
func (e *env) publish(episodeID string) v1.Episode {
	e.t.Helper()
	var out v1.Episode
	e.sendJSON(http.MethodPost, "/api/v1/episodes/"+episodeID+"/publish", nil, &out).wantStatus(http.StatusOK)
	return out
}

// uploadCover uploads an image as the series cover.
func (e *env) uploadCover(seriesID string, data []byte, contentType, fileName string) *apiResponse {
	e.t.Helper()
	return e.upload("/api/v1/series/"+seriesID+"/cover", []multipartPart{
		{name: "file", filename: fileName, contentType: contentType, data: data},
	})
}

// validChapters returns a two-chapter document that fits in the default test
// duration.
func validChapters() v1.ChaptersDocument {
	end := 60.0
	return v1.ChaptersDocument{
		Version: "1.2.0",
		Chapters: []v1.ChapterItem{
			{StartTime: 0, Title: "Intro"},
			{StartTime: 30, Title: "Body", EndTime: &end},
		},
	}
}

// countBlobFiles counts the files stored under the environment blob directory.
func countBlobFiles(t *testing.T, dir string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(dir, func(_ string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			count++
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("walk blob directory: %v", err)
	}
	return count
}

// formatFloat renders a duration the way the multipart field expects.
func formatFloat(value float64) string {
	return strings.TrimSuffix(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
}

// mustRFC3339 asserts value is a valid RFC3339 timestamp and returns it.
func mustRFC3339(t *testing.T, label, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("%s = %q is not RFC3339: %v", label, value, err)
	}
	return parsed
}
