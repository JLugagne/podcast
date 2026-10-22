package http_test

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/blobstore"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/episodeuc"
	cataloghttp "github.com/JLugagne/podcast/internal/catalog/inbound/http"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

func TestListEpisodes(t *testing.T) {
	app := &fakeApp{listEpisodesFn: func(_ context.Context, _ *domain.Actor, seriesID domain.SeriesID, q episodeuc.ListQuery) ([]*domain.Episode, error) {
		if seriesID != domain.SeriesID(testSeriesID) {
			t.Errorf("series id = %q", seriesID)
		}
		if q.Status == nil || *q.Status != domain.EpisodeStatusPublished {
			t.Errorf("status filter = %v, want published", q.Status)
		}
		return []*domain.Episode{newPublishedEpisode(), newDraftEpisode()}, nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodGet, "/api/v1/series/"+testSeriesID+"/episodes?status=published", ""))

	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.EpisodeList](t, rec)
	if len(body.Episodes) != 2 {
		t.Fatalf("episodes length = %d, want 2", len(body.Episodes))
	}
	published := body.Episodes[0]
	if published.Status != "published" || published.PublishedAt == nil {
		t.Fatalf("published episode = %+v", published)
	}
	if published.Audio == nil || published.Audio.URL != "/api/v1/episodes/"+testEpisodeID+"/audio" {
		t.Fatalf("audio = %+v", published.Audio)
	}
	if published.SeriesID != testSeriesID {
		t.Fatalf("series_id = %q", published.SeriesID)
	}
	if len(published.Chapters) != 1 || published.Chapters[0].Title != "Intro" {
		t.Fatalf("chapters = %+v", published.Chapters)
	}
	draft := body.Episodes[1]
	if draft.Status != "draft" || draft.PublishedAt != nil {
		t.Fatalf("draft episode = %+v", draft)
	}
}

func TestListEpisodesInvalidStatus(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}),
		request(t, http.MethodGet, "/api/v1/series/"+testSeriesID+"/episodes?status=archived", ""))

	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_EPISODE")
}

func TestCreateEpisode(t *testing.T) {
	app := &fakeApp{createEpisodeFn: func(_ context.Context, _ *domain.Actor, seriesID domain.SeriesID, in episodeuc.CreateInput) (*domain.Episode, error) {
		if seriesID != domain.SeriesID(testSeriesID) {
			t.Errorf("series id = %q", seriesID)
		}
		if in.Number != nil {
			t.Errorf("number = %d, want nil (next available)", *in.Number)
		}
		if in.Title != "Episode" || in.Description != "Details" {
			t.Errorf("input = %+v", in)
		}
		return newDraftEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPost, "/api/v1/series/"+testSeriesID+"/episodes", `{"title":"Episode","description":"Details"}`))

	requireStatus(t, rec, http.StatusCreated)
	body := decodeJSON[v1.Episode](t, rec)
	if body.ID != testEpisodeID || body.Number != 1 || body.Version != 4 {
		t.Fatalf("unexpected episode body: %+v", body)
	}
	if len(body.Chapters) != 0 {
		t.Fatalf("chapters = %+v, want []", body.Chapters)
	}
}

func TestCreateEpisodeWithNumber(t *testing.T) {
	app := &fakeApp{createEpisodeFn: func(_ context.Context, _ *domain.Actor, _ domain.SeriesID, in episodeuc.CreateInput) (*domain.Episode, error) {
		if in.Number == nil || *in.Number != 12 {
			t.Errorf("number = %v, want 12", in.Number)
		}
		return newDraftEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPost, "/api/v1/series/"+testSeriesID+"/episodes", `{"number":12,"title":"Episode"}`))

	requireStatus(t, rec, http.StatusCreated)
}

func TestCreateEpisodeSeriesNotFound(t *testing.T) {
	app := &fakeApp{createEpisodeFn: func(context.Context, *domain.Actor, domain.SeriesID, episodeuc.CreateInput) (*domain.Episode, error) {
		return nil, domain.ErrSeriesNotFound
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPost, "/api/v1/series/"+testSeriesID+"/episodes", `{"title":"Episode"}`))

	requireError(t, rec, http.StatusNotFound, "SERIES_NOT_FOUND")
}

func TestCreateEpisodeNumberTaken(t *testing.T) {
	app := &fakeApp{createEpisodeFn: func(context.Context, *domain.Actor, domain.SeriesID, episodeuc.CreateInput) (*domain.Episode, error) {
		return nil, domain.ErrEpisodeNumberTaken
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPost, "/api/v1/series/"+testSeriesID+"/episodes", `{"number":1,"title":"Episode"}`))

	requireError(t, rec, http.StatusConflict, "EPISODE_NUMBER_TAKEN")
}

func TestGetEpisode(t *testing.T) {
	app := &fakeApp{getEpisodeFn: func(_ context.Context, _ *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
		if id != domain.EpisodeID(testEpisodeID) {
			t.Errorf("id = %q", id)
		}
		return newPublishedEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/episodes/"+testEpisodeID, ""))

	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.Episode](t, rec)
	if body.Status != "published" || body.Audio == nil || body.PublishedAt == nil {
		t.Fatalf("unexpected episode body: %+v", body)
	}
}

func TestGetEpisodeInvalidID(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/episodes/nope", ""))

	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_ID")
}

func TestGetEpisodeNotFound(t *testing.T) {
	app := &fakeApp{getEpisodeFn: func(context.Context, *domain.Actor, domain.EpisodeID) (*domain.Episode, error) {
		return nil, domain.ErrEpisodeNotFound
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodGet, "/api/v1/episodes/"+testEpisodeID, ""))

	requireError(t, rec, http.StatusNotFound, "EPISODE_NOT_FOUND")
}

func TestUpdateEpisode(t *testing.T) {
	app := &fakeApp{updateEpisodeFn: func(_ context.Context, _ *domain.Actor, id domain.EpisodeID, attrs domain.EpisodeAttributes, version int) (*domain.Episode, error) {
		if id != domain.EpisodeID(testEpisodeID) {
			t.Errorf("id = %q", id)
		}
		if version != 4 || attrs.Number != 2 || attrs.Title != "Renamed" {
			t.Errorf("version = %d, attrs = %+v", version, attrs)
		}
		return newDraftEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPut, "/api/v1/episodes/"+testEpisodeID, `{"number":2,"title":"Renamed","description":"D","version":4}`))

	requireStatus(t, rec, http.StatusOK)
}

func TestUpdateEpisodeRequiresVersion(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}),
		request(t, http.MethodPut, "/api/v1/episodes/"+testEpisodeID, `{"number":2,"title":"Renamed"}`))

	requireError(t, rec, http.StatusUnprocessableEntity, "VERSION_REQUIRED")
}

func TestUpdateEpisodeVersionConflict(t *testing.T) {
	app := &fakeApp{updateEpisodeFn: func(context.Context, *domain.Actor, domain.EpisodeID, domain.EpisodeAttributes, int) (*domain.Episode, error) {
		return nil, domain.ErrVersionConflict
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPut, "/api/v1/episodes/"+testEpisodeID, `{"number":2,"title":"Renamed","version":1}`))

	requireError(t, rec, http.StatusConflict, "VERSION_CONFLICT")
}

func TestDeleteEpisode(t *testing.T) {
	app := &fakeApp{deleteEpisodeFn: func(_ context.Context, _ *domain.Actor, id domain.EpisodeID) error {
		if id != domain.EpisodeID(testEpisodeID) {
			t.Errorf("id = %q", id)
		}
		return nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodDelete, "/api/v1/episodes/"+testEpisodeID, ""))

	requireStatus(t, rec, http.StatusNoContent)
	if rec.Body.Len() != 0 {
		t.Fatalf("body = %q, want empty", rec.Body.String())
	}
}

func TestUploadAudio(t *testing.T) {
	data := []byte("ID3\x04\x00\x00\x00\x00\x00\x00")
	app := &fakeApp{uploadAudioFn: func(_ context.Context, _ *domain.Actor, id domain.EpisodeID, in episodeuc.AudioUploadInput) (*domain.Episode, error) {
		if id != domain.EpisodeID(testEpisodeID) {
			t.Errorf("id = %q", id)
		}
		read, err := io.ReadAll(in.Reader)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(read, data) {
			t.Errorf("body = %q, want %q", read, data)
		}
		if in.ContentType != "audio/mpeg" {
			t.Errorf("content type = %q", in.ContentType)
		}
		if in.OriginalName != "episode.mp3" {
			t.Errorf("original name = %q", in.OriginalName)
		}
		if in.DurationSeconds != 95.5 {
			t.Errorf("duration = %v, want 95.5", in.DurationSeconds)
		}
		return newDraftEpisode(), nil
	}}

	req := uploadRequest(t, "/api/v1/episodes/"+testEpisodeID+"/audio", func(writer *multipart.Writer) {
		if err := writer.WriteField("duration_seconds", "95.5"); err != nil {
			t.Fatalf("write duration: %v", err)
		}
		addFile(t, writer, "episode.mp3", "audio/mpeg", data)
	})
	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), req)

	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.Episode](t, rec)
	if body.Audio == nil || body.Audio.URL != "/api/v1/episodes/"+testEpisodeID+"/audio" {
		t.Fatalf("audio = %+v", body.Audio)
	}
	if body.Audio.DurationSeconds != 95.5 || body.Audio.OriginalName != "episode.mp3" {
		t.Fatalf("audio = %+v", body.Audio)
	}
}

func TestUploadAudioDurationAfterFileIsIgnored(t *testing.T) {
	app := &fakeApp{uploadAudioFn: func(_ context.Context, _ *domain.Actor, _ domain.EpisodeID, in episodeuc.AudioUploadInput) (*domain.Episode, error) {
		if in.DurationSeconds != 0 {
			t.Errorf("duration = %v, want 0: fields after the file part are never read", in.DurationSeconds)
		}
		if _, err := io.Copy(io.Discard, in.Reader); err != nil {
			return nil, err
		}
		return newDraftEpisode(), nil
	}}

	req := uploadRequest(t, "/api/v1/episodes/"+testEpisodeID+"/audio", func(writer *multipart.Writer) {
		addFile(t, writer, "episode.mp3", "audio/mpeg", []byte("ID3"))
		if err := writer.WriteField("duration_seconds", "95.5"); err != nil {
			t.Fatalf("write duration: %v", err)
		}
	})
	requireStatus(t, serve(t, newTestRouter(t, app, cataloghttp.Options{}), req), http.StatusOK)
}

func TestUploadAudioMissingFile(t *testing.T) {
	req := uploadRequest(t, "/api/v1/episodes/"+testEpisodeID+"/audio", func(writer *multipart.Writer) {
		if err := writer.WriteField("duration_seconds", "12"); err != nil {
			t.Fatalf("write duration: %v", err)
		}
	})

	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), req)

	requireError(t, rec, http.StatusUnprocessableEntity, "MISSING_FILE")
}

func TestUploadAudioInvalidDuration(t *testing.T) {
	req := uploadRequest(t, "/api/v1/episodes/"+testEpisodeID+"/audio", func(writer *multipart.Writer) {
		if err := writer.WriteField("duration_seconds", "half an hour"); err != nil {
			t.Fatalf("write duration: %v", err)
		}
		addFile(t, writer, "episode.mp3", "audio/mpeg", []byte("ID3"))
	})
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), req)

	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_DURATION")
}

func TestUploadAudioTooLarge(t *testing.T) {
	app := &fakeApp{uploadAudioFn: func(_ context.Context, _ *domain.Actor, _ domain.EpisodeID, in episodeuc.AudioUploadInput) (*domain.Episode, error) {
		_, err := io.ReadAll(in.Reader)
		return nil, err
	}}

	req := uploadRequest(t, "/api/v1/episodes/"+testEpisodeID+"/audio", func(writer *multipart.Writer) {
		addFile(t, writer, "episode.mp3", "audio/mpeg", make([]byte, 1024))
	})
	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{MaxAudioBytes: 8}), req)

	requireError(t, rec, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
}

func TestUploadAudioUnsupportedType(t *testing.T) {
	app := &fakeApp{uploadAudioFn: func(context.Context, *domain.Actor, domain.EpisodeID, episodeuc.AudioUploadInput) (*domain.Episode, error) {
		return nil, domain.ErrUnsupportedAudioType
	}}

	req := uploadRequest(t, "/api/v1/episodes/"+testEpisodeID+"/audio", func(writer *multipart.Writer) {
		addFile(t, writer, "notes.txt", "text/plain", []byte("hello"))
	})
	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), req)

	requireError(t, rec, http.StatusUnprocessableEntity, "UNSUPPORTED_AUDIO_TYPE")
}

func TestOpenAudio(t *testing.T) {
	data := []byte("ID3audio-bytes")
	app := &fakeApp{openAudioFn: func(_ context.Context, _ *domain.Actor, id domain.EpisodeID) (blobstore.Blob, error) {
		if id != domain.EpisodeID(testEpisodeID) {
			t.Errorf("id = %q", id)
		}
		return newBlob(data, "audio/mpeg"), nil
	}}
	router := newTestRouter(t, app, cataloghttp.Options{})

	rec := serve(t, router, request(t, http.MethodGet, "/api/v1/episodes/"+testEpisodeID+"/audio", ""))
	requireStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "audio/mpeg" {
		t.Fatalf("Content-Type = %q", got)
	}
	if !bytes.Equal(rec.Body.Bytes(), data) {
		t.Fatalf("body = %q", rec.Body.Bytes())
	}

	req := request(t, http.MethodGet, "/api/v1/episodes/"+testEpisodeID+"/audio", "")
	req.Header.Set("Range", "bytes=3-7")
	rec = serve(t, router, req)
	requireStatus(t, rec, http.StatusPartialContent)
	if !bytes.Equal(rec.Body.Bytes(), data[3:8]) {
		t.Fatalf("partial body = %q, want %q", rec.Body.Bytes(), data[3:8])
	}
}

func TestSetChapters(t *testing.T) {
	app := &fakeApp{setChaptersFn: func(_ context.Context, _ *domain.Actor, id domain.EpisodeID, chapters []domain.Chapter) (*domain.Episode, error) {
		if id != domain.EpisodeID(testEpisodeID) {
			t.Errorf("id = %q", id)
		}
		if len(chapters) != 2 {
			t.Fatalf("chapters length = %d, want 2", len(chapters))
		}
		first, second := chapters[0], chapters[1]
		if first.StartSeconds != 0 || first.Title != "Intro" || first.URL != "https://example.com/0" {
			t.Errorf("first chapter = %+v", first)
		}
		if !first.TOC {
			t.Error("first chapter toc = false, want the default true")
		}
		if second.StartSeconds != 15 || second.TOC {
			t.Errorf("second chapter = %+v", second)
		}
		if second.ImageURL != "https://example.com/img.png" {
			t.Errorf("second image = %q", second.ImageURL)
		}
		if second.EndSeconds == nil || *second.EndSeconds != 30.5 {
			t.Errorf("second end = %v", second.EndSeconds)
		}
		return newEpisodeWithChapters(chapters...), nil
	}}

	body := `{"version":"1.2.0","chapters":[` +
		`{"startTime":0,"title":"Intro","url":"https://example.com/0"},` +
		`{"startTime":15,"endTime":30.5,"title":"Second","img":"https://example.com/img.png","toc":false}],` +
		`"unknown_field":"ignored"}`
	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPut, "/api/v1/episodes/"+testEpisodeID+"/chapters", body))

	requireStatus(t, rec, http.StatusOK)
	episode := decodeJSON[v1.Episode](t, rec)
	if len(episode.Chapters) != 2 || episode.Chapters[1].Title != "Second" {
		t.Fatalf("chapters = %+v", episode.Chapters)
	}
}

func TestSetChaptersInvalidVersion(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}),
		request(t, http.MethodPut, "/api/v1/episodes/"+testEpisodeID+"/chapters",
			`{"version":"2.0.0","chapters":[{"startTime":0,"title":"Intro"}]}`))

	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_CHAPTERS_VERSION")
}

func TestSetChaptersBadJSON(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}),
		request(t, http.MethodPut, "/api/v1/episodes/"+testEpisodeID+"/chapters", `{"version":`))

	requireError(t, rec, http.StatusBadRequest, "BAD_REQUEST")
}

func TestSetChaptersDomainError(t *testing.T) {
	app := &fakeApp{setChaptersFn: func(context.Context, *domain.Actor, domain.EpisodeID, []domain.Chapter) (*domain.Episode, error) {
		return nil, domain.ErrInvalidChapters
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodPut, "/api/v1/episodes/"+testEpisodeID+"/chapters",
			`{"version":"1.2.0","chapters":[{"startTime":5,"title":"No zero"}]}`))

	requireError(t, rec, http.StatusUnprocessableEntity, "INVALID_CHAPTERS")
}

func TestGetChapters(t *testing.T) {
	app := &fakeApp{getEpisodeFn: func(context.Context, *domain.Actor, domain.EpisodeID) (*domain.Episode, error) {
		return newPublishedEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodGet, "/api/v1/episodes/"+testEpisodeID+"/chapters", ""))

	requireStatus(t, rec, http.StatusOK)
	if got := rec.Header().Get("Content-Type"); got != "application/json+chapters" {
		t.Fatalf("Content-Type = %q", got)
	}
	doc := decodeJSON[v1.ChaptersDocument](t, rec)
	if doc.Version != "1.2.0" {
		t.Fatalf("version = %q", doc.Version)
	}
	if len(doc.Chapters) != 1 || doc.Chapters[0].Title != "Intro" || doc.Chapters[0].StartTime != 0 {
		t.Fatalf("chapters = %+v", doc.Chapters)
	}
	if doc.Chapters[0].TOC == nil || !*doc.Chapters[0].TOC {
		t.Fatalf("toc = %v, want true", doc.Chapters[0].TOC)
	}
}

func TestGetChaptersEmpty(t *testing.T) {
	app := &fakeApp{getEpisodeFn: func(context.Context, *domain.Actor, domain.EpisodeID) (*domain.Episode, error) {
		return newDraftEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}),
		request(t, http.MethodGet, "/api/v1/episodes/"+testEpisodeID+"/chapters", ""))

	requireStatus(t, rec, http.StatusOK)
	if raw := rec.Body.String(); !strings.Contains(raw, `"chapters":[]`) {
		t.Fatalf("body = %s, want an empty chapters array", raw)
	}
}

func TestPublish(t *testing.T) {
	app := &fakeApp{publishFn: func(_ context.Context, _ *domain.Actor, id domain.EpisodeID) (*domain.Episode, error) {
		if id != domain.EpisodeID(testEpisodeID) {
			t.Errorf("id = %q", id)
		}
		return newPublishedEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodPost, "/api/v1/episodes/"+testEpisodeID+"/publish", ""))

	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.Episode](t, rec)
	if body.Status != "published" || body.PublishedAt == nil {
		t.Fatalf("unexpected episode body: %+v", body)
	}
}

func TestPublishMissingAudio(t *testing.T) {
	app := &fakeApp{publishFn: func(context.Context, *domain.Actor, domain.EpisodeID) (*domain.Episode, error) {
		return nil, domain.ErrEpisodeAudioMissing
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodPost, "/api/v1/episodes/"+testEpisodeID+"/publish", ""))

	requireError(t, rec, http.StatusUnprocessableEntity, "EPISODE_AUDIO_MISSING")
}

func TestUnpublish(t *testing.T) {
	app := &fakeApp{unpublishFn: func(context.Context, *domain.Actor, domain.EpisodeID) (*domain.Episode, error) {
		return newDraftEpisode(), nil
	}}

	rec := serve(t, newTestRouter(t, app, cataloghttp.Options{}), request(t, http.MethodPost, "/api/v1/episodes/"+testEpisodeID+"/unpublish", ""))

	requireStatus(t, rec, http.StatusOK)
	body := decodeJSON[v1.Episode](t, rec)
	if body.Status != "draft" || body.PublishedAt != nil {
		t.Fatalf("unexpected episode body: %+v", body)
	}
}

func TestUnknownRouteIsNotFound(t *testing.T) {
	rec := serve(t, newTestRouter(t, &fakeApp{}, cataloghttp.Options{}), request(t, http.MethodGet, "/series", ""))

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404: routes are mounted under /api/v1", rec.Code)
	}
}
