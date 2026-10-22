package http_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	cataloghttp "github.com/JLugagne/podcast/internal/catalog/inbound/http"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

func TestListSeries(t *testing.T) {
	var gotQuery string
	app := &fakeApp{listSeriesFn: func(_ context.Context, actor *domain.Actor, q seriesuc.ListQuery) ([]*seriesuc.SeriesView, error) {
		if actor.ID() != domain.LocalAdminID {
			t.Errorf("actor = %q, want the local admin", actor.ID())
		}
		gotQuery = q.Query
		return []*seriesuc.SeriesView{{
			Series:               newSeries(),
			EpisodeCount:         4,
			PublishedCount:       2,
			TotalDurationSeconds: 123.5,
		}}, nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/series?q=show", ""))

	requireStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if gotQuery != "show" {
		t.Fatalf("query = %q, want %q", gotQuery, "show")
	}
	body := decodeJSON[v1.SeriesList](t, rec)
	if len(body.Series) != 1 {
		t.Fatalf("series length = %d, want 1", len(body.Series))
	}
	s := body.Series[0]
	if s.ID != testSeriesID || s.Slug != "show" || s.AccentColor != "#7A9CFF" {
		t.Fatalf("unexpected series body: %+v", s)
	}
	if s.FeedURL != "/feeds/show.xml" {
		t.Fatalf("feed_url = %q, want /feeds/show.xml", s.FeedURL)
	}
	if s.CoverURL == nil || *s.CoverURL != "/api/v1/series/"+testSeriesID+"/cover" {
		t.Fatalf("cover_url = %v", s.CoverURL)
	}
	if s.EpisodeCount != 4 || s.PublishedCount != 2 || s.TotalDurationSeconds != 123.5 {
		t.Fatalf("unexpected counters: %+v", s)
	}
	if s.Version != 3 {
		t.Fatalf("version = %d, want 3", s.Version)
	}
	if s.CreatedAt != "2026-10-21T12:30:00Z" {
		t.Fatalf("created_at = %q", s.CreatedAt)
	}
}

func TestListSeriesEmpty(t *testing.T) {
	app := &fakeApp{listSeriesFn: func(context.Context, *domain.Actor, seriesuc.ListQuery) ([]*seriesuc.SeriesView, error) {
		return nil, nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/series", ""))

	requireStatus(t, rec, http.StatusOK)
	if got := strings.TrimSpace(rec.Body.String()); got != `{"series":[]}` {
		t.Fatalf("body = %s, want an empty series array", got)
	}
}

func TestCreateSeries(t *testing.T) {
	app := &fakeApp{createSeriesFn: func(_ context.Context, actor *domain.Actor, attrs domain.SeriesAttributes) (*seriesuc.SeriesView, error) {
		if actor.ID() != domain.LocalAdminID {
			t.Errorf("actor = %q, want the local admin", actor.ID())
		}
		want := domain.SeriesAttributes{Title: "Show", Author: "Host", Description: "About", AccentColor: "#7A9CFF", Slug: "new-show"}
		if attrs != want {
			t.Errorf("attrs = %+v, want %+v", attrs, want)
		}
		return &seriesuc.SeriesView{Series: newSeries()}, nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodPost, "/api/v1/series",
		`{"slug":"new-show","title":"Show","author":"Host","description":"About","accent_color":"#7A9CFF"}`))

	requireStatus(t, rec, http.StatusCreated)
	body := decodeJSON[v1.Series](t, rec)
	if body.ID != testSeriesID || body.Version != 3 {
		t.Fatalf("unexpected series body: %+v", body)
	}
}

func TestCreateSeriesRejectsUnknownField(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), request(t, http.MethodPost, "/api/v1/series",
		`{"title":"Show","unexpected":true}`))

	requireError(t, rec, http.StatusBadRequest, "BAD_REQUEST")
}

func TestGetSeries(t *testing.T) {
	app := &fakeApp{getSeriesFn: func(_ context.Context, _ *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error) {
		if id != domain.SeriesID(testSeriesID) {
			t.Errorf("id = %q", id)
		}
		return &seriesuc.SeriesView{Series: newSeries()}, nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/series/"+testSeriesID, ""))

	requireStatus(t, rec, http.StatusOK)
	if body := decodeJSON[v1.Series](t, rec); body.ID != testSeriesID {
		t.Fatalf("id = %q", body.ID)
	}
}

func TestGetSeriesNotFound(t *testing.T) {
	app := &fakeApp{getSeriesFn: func(context.Context, *domain.Actor, domain.SeriesID) (*seriesuc.SeriesView, error) {
		return nil, domain.ErrSeriesNotFound
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/series/"+testOtherID, ""))

	requireError(t, rec, http.StatusNotFound, "SERIES_NOT_FOUND")
}

func TestInvalidSeriesID(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/series/not-a-uuid", ""))

	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_ID")
}

func TestUpdateSeries(t *testing.T) {
	app := &fakeApp{updateSeriesFn: func(_ context.Context, _ *domain.Actor, id domain.SeriesID, attrs domain.SeriesAttributes, version int) (*seriesuc.SeriesView, error) {
		if id != domain.SeriesID(testSeriesID) {
			t.Errorf("id = %q", id)
		}
		if version != 7 {
			t.Errorf("version = %d, want 7", version)
		}
		if attrs.Title != "Updated" {
			t.Errorf("title = %q", attrs.Title)
		}
		return &seriesuc.SeriesView{Series: newSeries()}, nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodPut, "/api/v1/series/"+testSeriesID,
		`{"title":"Updated","author":"Host","description":"About","version":7}`))

	requireStatus(t, rec, http.StatusOK)
}

func TestUpdateSeriesRequiresVersion(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), request(t, http.MethodPut, "/api/v1/series/"+testSeriesID,
		`{"title":"Updated","author":"Host","description":"About"}`))

	requireError(t, rec, http.StatusUnprocessableEntity, "VERSION_REQUIRED")
}

func TestUpdateSeriesVersionConflict(t *testing.T) {
	app := &fakeApp{updateSeriesFn: func(context.Context, *domain.Actor, domain.SeriesID, domain.SeriesAttributes, int) (*seriesuc.SeriesView, error) {
		return nil, domain.ErrVersionConflict
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodPut, "/api/v1/series/"+testSeriesID,
		`{"title":"Updated","author":"Host","description":"About","version":1}`))

	requireError(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestDeleteSeries(t *testing.T) {
	app := &fakeApp{deleteSeriesFn: func(_ context.Context, _ *domain.Actor, id domain.SeriesID) error {
		if id != domain.SeriesID(testSeriesID) {
			t.Errorf("id = %q", id)
		}
		return nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodDelete, "/api/v1/series/"+testSeriesID, ""))

	requireStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestDeleteSeriesNotEmpty(t *testing.T) {
	app := &fakeApp{deleteSeriesFn: func(context.Context, *domain.Actor, domain.SeriesID) error {
		return domain.ErrSeriesNotEmpty
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodDelete, "/api/v1/series/"+testSeriesID, ""))

	requireError(t, rec, http.StatusConflict, "SERIES_NOT_EMPTY")
}

func TestUploadCover(t *testing.T) {
	data := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	app := &fakeApp{uploadCoverFn: func(_ context.Context, _ *domain.Actor, id domain.SeriesID, in seriesuc.UploadInput) (*seriesuc.SeriesView, error) {
		if id != domain.SeriesID(testSeriesID) {
			t.Errorf("id = %q", id)
		}
		read, err := io.ReadAll(in.Reader)
		if err != nil {
			return nil, err
		}
		if string(read) != string(data) {
			t.Errorf("body = %q, want %q", read, data)
		}
		if in.ContentType != "image/png" {
			t.Errorf("content type = %q, want the sniffed image/png", in.ContentType)
		}
		return &seriesuc.SeriesView{Series: newSeries()}, nil
	}}

	req := uploadRequest(t, "/api/v1/series/"+testSeriesID+"/cover", func(writer *multipart.Writer) {
		addFile(t, writer, "cover.png", "", data)
	})
	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), req)

	requireStatus(t, rec, http.StatusOK)
	if body := decodeJSON[v1.Series](t, rec); body.CoverURL == nil {
		t.Fatal("cover_url is null after upload")
	}
}

func TestUploadCoverHonoursDeclaredContentType(t *testing.T) {
	app := &fakeApp{uploadCoverFn: func(_ context.Context, _ *domain.Actor, _ domain.SeriesID, in seriesuc.UploadInput) (*seriesuc.SeriesView, error) {
		if in.ContentType != "image/webp" {
			t.Errorf("content type = %q, want image/webp", in.ContentType)
		}
		if _, err := io.Copy(io.Discard, in.Reader); err != nil {
			return nil, err
		}
		return &seriesuc.SeriesView{Series: newSeries()}, nil
	}}

	req := uploadRequest(t, "/api/v1/series/"+testSeriesID+"/cover", func(writer *multipart.Writer) {
		addFile(t, writer, "cover.webp", "image/webp", []byte("RIFF....WEBPVP8 "))
	})
	requireStatus(t, serve(t, newTestRouter(t, app, cataloghttp.Options{}), req), http.StatusOK)
}

func TestUploadCoverMissingFile(t *testing.T) {
	req := uploadRequest(t, "/api/v1/series/"+testSeriesID+"/cover", func(writer *multipart.Writer) {
		if err := writer.WriteField("note", "no file here"); err != nil {
			t.Fatalf("write field: %v", err)
		}
	})

	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), req)

	requireError(t, rec, http.StatusUnprocessableEntity, "MISSING_FILE")
}

func TestUploadCoverTooLarge(t *testing.T) {
	app := &fakeApp{uploadCoverFn: func(_ context.Context, _ *domain.Actor, _ domain.SeriesID, in seriesuc.UploadInput) (*seriesuc.SeriesView, error) {
		_, err := io.ReadAll(in.Reader)
		return nil, err
	}}

	req := uploadRequest(t, "/api/v1/series/"+testSeriesID+"/cover", func(writer *multipart.Writer) {
		addFile(t, writer, "cover.png", "image/png", make([]byte, 1024))
	})
	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{MaxImageBytes: 4}), req)

	requireError(t, rec, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
}

func TestUploadCoverUnsupportedType(t *testing.T) {
	app := &fakeApp{uploadCoverFn: func(context.Context, *domain.Actor, domain.SeriesID, seriesuc.UploadInput) (*seriesuc.SeriesView, error) {
		return nil, domain.ErrUnsupportedImageType
	}}

	req := uploadRequest(t, "/api/v1/series/"+testSeriesID+"/cover", func(writer *multipart.Writer) {
		addFile(t, writer, "cover.gif", "image/gif", []byte("GIF89a"))
	})
	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), req)

	requireError(t, rec, http.StatusUnprocessableEntity, "UNSUPPORTED_IMAGE_TYPE")
}

func TestRemoveCover(t *testing.T) {
	bare := domain.ReconstituteSeries(domain.SeriesID(testSeriesID),
		domain.SeriesAttributes{Title: "Show", Author: "Host", Description: "About", AccentColor: "#7A9CFF", Slug: "show"},
		nil, testNow, testNow, 4)
	app := &fakeApp{removeCoverFn: func(_ context.Context, _ *domain.Actor, id domain.SeriesID) (*seriesuc.SeriesView, error) {
		if id != domain.SeriesID(testSeriesID) {
			t.Errorf("id = %q", id)
		}
		return &seriesuc.SeriesView{Series: bare}, nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodDelete, "/api/v1/series/"+testSeriesID+"/cover", ""))

	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.Series](t, rec)
	if body.CoverURL != nil {
		t.Fatalf("cover_url = %v, want null", *body.CoverURL)
	}
}

func TestOpenCover(t *testing.T) {
	data := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")
	app := &fakeApp{openCoverFn: func(_ context.Context, _ *domain.Actor, id domain.SeriesID) (blobstore.Blob, error) {
		if id != domain.SeriesID(testSeriesID) {
			t.Errorf("id = %q", id)
		}
		return newBlob(data, "image/png"), nil
	}}
	router := newTestRouter(t, app, cataloghttp.Options{})

	rec := serve(t, router, request(t, http.MethodGet, "/api/v1/series/"+testSeriesID+"/cover", ""))
	requireStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("Content-Type = %q", got)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-cache" {
		t.Fatalf("Cache-Control = %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Fatalf("body = %q, want %q", rec.Body.Bytes(), data)
	}

	req := request(t, http.MethodGet, "/api/v1/series/"+testSeriesID+"/cover", "")
	req.Header.Set("Range", "bytes=0-3")
	rec = serve(t, router, req)
	requireStatus(t, rec, http.StatusPartialContent)
	if !bytes.Equal(rec.Body.Bytes(), data[:4]) {
		t.Fatalf("partial body = %q, want %q", rec.Body.Bytes(), data[:4])
	}
}

func TestOpenCoverNotFound(t *testing.T) {
	app := &fakeApp{openCoverFn: func(context.Context, *domain.Actor, domain.SeriesID) (blobstore.Blob, error) {
		return blobstore.Blob{}, domain.ErrAssetNotFound
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/series/"+testSeriesID+"/cover", ""))

	requireError(t, rec, http.StatusNotFound, "ASSET_NOT_FOUND")
}

func TestCreateSeriesBodyTooLarge(t *testing.T) {
	body := `{"title":"` + strings.Repeat("a", 2<<20) + `"}`

	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), request(t, http.MethodPost, "/api/v1/series", body))

	requireError(t, rec, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
}

func TestUnexpectedErrorIsInternal(t *testing.T) {
	app := &fakeApp{getSeriesFn: func(context.Context, *domain.Actor, domain.SeriesID) (*seriesuc.SeriesView, error) {
		return nil, errors.New("database exploded")
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/series/"+testSeriesID, ""))

	requireError(t, rec, http.StatusInternalServerError, "INTERNAL")
}
