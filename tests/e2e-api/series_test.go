package e2eapi_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

// TestSeriesCRUD covers creation, reading, update and deletion of a series,
// including the optimistic-lock version.
func TestSeriesCRUD(t *testing.T) {
	e := newEnv(t)

	created := e.createSeries(v1.SeriesInput{
		Title:       "My Great Show",
		Author:      "Ada Lovelace",
		Description: "A show about numbers",
	})
	if created.ID == "" {
		t.Fatal("created series has no id")
	}
	if created.Slug != "my-great-show" {
		t.Fatalf("slug = %q, want %q", created.Slug, "my-great-show")
	}
	if created.AccentColor != "#7A9CFF" {
		t.Fatalf("accent_color = %q, want the default #7A9CFF", created.AccentColor)
	}
	if created.CoverURL != nil {
		t.Fatalf("cover_url = %v, want nil", *created.CoverURL)
	}
	if created.Version != 0 {
		t.Fatalf("version = %d, want 0", created.Version)
	}
	if created.EpisodeCount != 0 || created.PublishedCount != 0 || created.TotalDurationSeconds != 0 {
		t.Fatalf("counters = (%d, %d, %v), want zero", created.EpisodeCount, created.PublishedCount, created.TotalDurationSeconds)
	}
	mustRFC3339(t, "created_at", created.CreatedAt)
	mustRFC3339(t, "updated_at", created.UpdatedAt)

	var fetched v1.Series
	e.get("/api/v1/series/" + created.ID).wantStatus(http.StatusOK).decode(&fetched)
	if fetched.ID != created.ID || fetched.Title != "My Great Show" {
		t.Fatalf("fetched series = %+v, want id %q and the created title", fetched, created.ID)
	}

	version := created.Version
	var updated v1.Series
	e.sendJSON(http.MethodPut, "/api/v1/series/"+created.ID, v1.SeriesInput{
		Title:   "My Renamed Show",
		Author:  "Ada Lovelace",
		Version: &version,
	}, &updated).wantStatus(http.StatusOK)
	if updated.Title != "My Renamed Show" {
		t.Fatalf("updated title = %q", updated.Title)
	}
	if updated.Slug != "my-great-show" {
		t.Fatalf("slug after update = %q, want the previous slug", updated.Slug)
	}
	if updated.Version != 1 {
		t.Fatalf("version after update = %d, want 1", updated.Version)
	}

	e.sendJSON(http.MethodPut, "/api/v1/series/"+created.ID, v1.SeriesInput{
		Title:   "Stale update",
		Version: &version,
	}, nil).wantError(http.StatusConflict, "VERSION_CONFLICT")

	e.sendJSON(http.MethodPut, "/api/v1/series/"+created.ID, v1.SeriesInput{
		Title: "No version",
	}, nil).wantError(http.StatusUnprocessableEntity, "VERSION_REQUIRED")

	e.request(http.MethodDelete, "/api/v1/series/"+created.ID, nil, "").wantStatus(http.StatusNoContent)
	e.get("/api/v1/series/"+created.ID).wantError(http.StatusNotFound, "SERIES_NOT_FOUND")
	e.request(http.MethodDelete, "/api/v1/series/"+created.ID, nil, "").wantError(http.StatusNotFound, "SERIES_NOT_FOUND")
}

// TestSeriesListAndQuery covers the list endpoint with and without a search
// query.
func TestSeriesListAndQuery(t *testing.T) {
	e := newEnv(t)

	e.createSeries(v1.SeriesInput{Title: "Alpha Signals", Author: "Ada"})
	e.createSeries(v1.SeriesInput{Title: "Beta Waves", Author: "Grace"})

	var all v1.SeriesList
	e.get("/api/v1/series").wantStatus(http.StatusOK).decode(&all)
	if len(all.Series) != 2 {
		t.Fatalf("listed %d series, want 2", len(all.Series))
	}

	var filtered v1.SeriesList
	e.get("/api/v1/series?q=alph").wantStatus(http.StatusOK).decode(&filtered)
	if len(filtered.Series) != 1 || filtered.Series[0].Title != "Alpha Signals" {
		t.Fatalf("query result = %+v, want only Alpha Signals", filtered.Series)
	}

	var empty v1.SeriesList
	resp := e.get("/api/v1/series?q=no-such-series").wantStatus(http.StatusOK)
	if string(resp.body) == "" {
		t.Fatal("empty list response has no body")
	}
	resp.decode(&empty)
	if empty.Series == nil {
		t.Fatal("series field is null, want an empty array")
	}
	if len(empty.Series) != 0 {
		t.Fatalf("query result has %d series, want none", len(empty.Series))
	}
}

// TestSeriesValidationAndConflicts covers the validation and conflict error
// paths of the series endpoints.
func TestSeriesValidationAndConflicts(t *testing.T) {
	e := newEnv(t)

	e.createSeries(v1.SeriesInput{Title: "Taken", Slug: "taken-slug", Author: "Ada"})
	e.sendJSON(http.MethodPost, "/api/v1/series", v1.SeriesInput{
		Title: "Another",
		Slug:  "taken-slug",
	}, nil).wantError(http.StatusConflict, "SLUG_TAKEN")

	e.sendJSON(http.MethodPost, "/api/v1/series", v1.SeriesInput{Title: "   "}, nil).
		wantError(http.StatusUnprocessableEntity, "INVALID_SERIES")
	e.sendJSON(http.MethodPost, "/api/v1/series", v1.SeriesInput{Title: "Bad color", AccentColor: "red"}, nil).
		wantError(http.StatusUnprocessableEntity, "INVALID_SERIES")

	e.request(http.MethodPost, "/api/v1/series", strings.NewReader("{"), "application/json").
		wantError(http.StatusBadRequest, "BAD_REQUEST")
	e.request(http.MethodPost, "/api/v1/series", strings.NewReader(`{"title":"x","unknown":1}`), "application/json").
		wantError(http.StatusBadRequest, "BAD_REQUEST")

	e.get("/api/v1/series/not-a-uuid").wantError(http.StatusUnprocessableEntity, "INVALID_ID")
	e.get("/api/v1/series/"+uuid.NewString()).wantError(http.StatusNotFound, "SERIES_NOT_FOUND")
}

// TestSeriesCover covers cover upload, download and removal.
func TestSeriesCover(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Covered", Author: "Ada"})

	coverBytes := []byte("\x89PNG\r\n\x1a\nfake-cover")
	var withCover v1.Series
	e.uploadCover(series.ID, coverBytes, "image/png", "cover.png").wantStatus(http.StatusOK).decode(&withCover)
	if withCover.CoverURL == nil {
		t.Fatal("cover_url is null after the upload")
	}
	if want := "/api/v1/series/" + series.ID + "/cover"; *withCover.CoverURL != want {
		t.Fatalf("cover_url = %q, want %q", *withCover.CoverURL, want)
	}

	downloaded := e.get("/api/v1/series/" + series.ID + "/cover").wantStatus(http.StatusOK)
	if got := downloaded.header.Get("Content-Type"); got != "image/png" {
		t.Fatalf("cover content type = %q, want image/png", got)
	}
	if string(downloaded.body) != string(coverBytes) {
		t.Fatalf("cover bytes = %q, want %q", string(downloaded.body), string(coverBytes))
	}

	var removed v1.Series
	e.request(http.MethodDelete, "/api/v1/series/"+series.ID+"/cover", nil, "").
		wantStatus(http.StatusOK).decode(&removed)
	if removed.CoverURL != nil {
		t.Fatalf("cover_url = %v after removal, want nil", *removed.CoverURL)
	}
	e.get("/api/v1/series/"+series.ID+"/cover").wantError(http.StatusNotFound, "ASSET_NOT_FOUND")

	e.uploadCover(series.ID, []byte("GIF89a"), "image/gif", "cover.gif").
		wantError(http.StatusUnprocessableEntity, "UNSUPPORTED_IMAGE_TYPE")

	e.upload("/api/v1/series/"+series.ID+"/cover", []multipartPart{
		{name: "note", data: []byte("no file here")},
	}).wantError(http.StatusUnprocessableEntity, "MISSING_FILE")

	e.uploadCover(uuid.NewString(), coverBytes, "image/png", "cover.png").
		wantError(http.StatusNotFound, "SERIES_NOT_FOUND")
}
