package e2eapi_test

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"

	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

// TestEpisodeCreateAndList covers automatic and explicit numbering and the
// ordered, series-scoped listing.
func TestEpisodeCreateAndList(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Numbered", Author: "Ada"})

	first := e.createEpisode(series.ID, v1.EpisodeInput{Title: "First"})
	if first.Number != 1 {
		t.Fatalf("first episode number = %d, want 1", first.Number)
	}
	second := e.createEpisode(series.ID, v1.EpisodeInput{Title: "Second"})
	if second.Number != 2 {
		t.Fatalf("second episode number = %d, want 2", second.Number)
	}
	explicit := 5
	fifth := e.createEpisode(series.ID, v1.EpisodeInput{Number: &explicit, Title: "Fifth"})
	if fifth.Number != 5 {
		t.Fatalf("explicit episode number = %d, want 5", fifth.Number)
	}
	if fifth.Status != "draft" {
		t.Fatalf("new episode status = %q, want draft", fifth.Status)
	}
	if fifth.Audio != nil {
		t.Fatalf("new episode audio = %+v, want nil", *fifth.Audio)
	}

	var list v1.EpisodeList
	e.get("/api/v1/series/" + series.ID + "/episodes").wantStatus(http.StatusOK).decode(&list)
	if len(list.Episodes) != 3 {
		t.Fatalf("listed %d episodes, want 3", len(list.Episodes))
	}
	for i, want := range []int{1, 2, 5} {
		if got := list.Episodes[i].Number; got != want {
			t.Fatalf("episode %d number = %d, want %d", i, got, want)
		}
	}

	conflict := 2
	e.sendJSON(http.MethodPost, "/api/v1/series/"+series.ID+"/episodes", v1.EpisodeInput{
		Number: &conflict,
		Title:  "Duplicate",
	}, nil).wantError(http.StatusConflict, "EPISODE_NUMBER_TAKEN")

	e.sendJSON(http.MethodPost, "/api/v1/series/"+uuid.NewString()+"/episodes", v1.EpisodeInput{
		Title: "Orphan",
	}, nil).wantError(http.StatusNotFound, "SERIES_NOT_FOUND")

	e.sendJSON(http.MethodPost, "/api/v1/series/not-a-uuid/episodes", v1.EpisodeInput{
		Title: "Bad series id",
	}, nil).wantError(http.StatusUnprocessableEntity, "INVALID_ID")

	e.sendJSON(http.MethodPost, "/api/v1/series/"+series.ID+"/episodes", v1.EpisodeInput{
		Title: "  ",
	}, nil).wantError(http.StatusUnprocessableEntity, "INVALID_EPISODE")

	var fetched v1.Episode
	e.get("/api/v1/episodes/" + first.ID).wantStatus(http.StatusOK).decode(&fetched)
	if fetched.ID != first.ID || fetched.Title != "First" {
		t.Fatalf("fetched episode = %+v, want the created one", fetched)
	}
	e.get("/api/v1/episodes/"+uuid.NewString()).wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")
	e.get("/api/v1/episodes/not-a-uuid").wantError(http.StatusUnprocessableEntity, "INVALID_ID")
}

// TestEpisodeUpdateAndDelete covers the optimistic lock and the deletion of an
// episode together with its audio blob.
func TestEpisodeUpdateAndDelete(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Mutable", Author: "Ada"})
	episode := e.createEpisode(series.ID, v1.EpisodeInput{Title: "Original"})

	version := episode.Version
	var updated v1.Episode
	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episode.ID, v1.EpisodeInput{
		Number:      intPtr(1),
		Title:       "Updated",
		Description: "New description",
		Version:     &version,
	}, &updated).wantStatus(http.StatusOK)
	if updated.Title != "Updated" || updated.Version != 1 {
		t.Fatalf("updated episode = %+v, want title Updated and version 1", updated)
	}

	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episode.ID, v1.EpisodeInput{
		Number:  intPtr(1),
		Title:   "Stale",
		Version: &version,
	}, nil).wantError(http.StatusConflict, "VERSION_CONFLICT")

	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episode.ID, v1.EpisodeInput{
		Number: intPtr(1),
		Title:  "No version",
	}, nil).wantError(http.StatusUnprocessableEntity, "VERSION_REQUIRED")

	e.sendJSON(http.MethodPut, "/api/v1/episodes/not-a-uuid", v1.EpisodeInput{
		Number:  intPtr(1),
		Title:   "Bad id",
		Version: intPtr(1),
	}, nil).wantError(http.StatusUnprocessableEntity, "INVALID_ID")

	e.uploadAudio(episode.ID, []byte("ID3-fake-audio"), "episode.mp3", floatPtr(30))
	if count := countBlobFiles(t, e.blobsDir); count == 0 {
		t.Fatal("no blob stored after the audio upload")
	}

	e.request(http.MethodDelete, "/api/v1/episodes/"+episode.ID, nil, "").wantStatus(http.StatusNoContent)
	e.get("/api/v1/episodes/"+episode.ID).wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")
	e.request(http.MethodDelete, "/api/v1/episodes/"+episode.ID, nil, "").wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")
	if count := countBlobFiles(t, e.blobsDir); count != 0 {
		t.Fatalf("%d blob files left after deleting the episode, want 0", count)
	}
}

// TestEpisodeListStatusFilter covers the status query parameter.
func TestEpisodeListStatusFilter(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Filtered", Author: "Ada"})
	published := e.createEpisode(series.ID, v1.EpisodeInput{Title: "Published"})
	draft := e.createEpisode(series.ID, v1.EpisodeInput{Title: "Draft"})

	e.uploadAudio(published.ID, []byte("audio-bytes"), "a.mp3", floatPtr(120))
	e.setChapters(published.ID, validChapters())
	e.publish(published.ID)

	var onlyPublished v1.EpisodeList
	e.get("/api/v1/series/" + series.ID + "/episodes?status=published").
		wantStatus(http.StatusOK).decode(&onlyPublished)
	if len(onlyPublished.Episodes) != 1 || onlyPublished.Episodes[0].ID != published.ID {
		t.Fatalf("published filter = %+v, want only %s", onlyPublished.Episodes, published.ID)
	}

	var onlyDraft v1.EpisodeList
	e.get("/api/v1/series/" + series.ID + "/episodes?status=draft").
		wantStatus(http.StatusOK).decode(&onlyDraft)
	if len(onlyDraft.Episodes) != 1 || onlyDraft.Episodes[0].ID != draft.ID {
		t.Fatalf("draft filter = %+v, want only %s", onlyDraft.Episodes, draft.ID)
	}

	e.get("/api/v1/series/"+series.ID+"/episodes?status=archived").
		wantError(http.StatusUnprocessableEntity, "INVALID_EPISODE")
}

// TestEpisodeAudio covers streamed upload, full download, range requests and
// the upload error paths.
func TestEpisodeAudio(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Audio", Author: "Ada"})
	episode := e.createEpisode(series.ID, v1.EpisodeInput{Title: "With audio"})

	audio := []byte("ID3\x04\x00\x00fake-mp3-payload-for-e2e")
	uploaded := e.uploadAudio(episode.ID, audio, "episode.mp3", floatPtr(12.5))
	if uploaded.Audio == nil {
		t.Fatal("episode has no audio after the upload")
	}
	if want := "/api/v1/episodes/" + episode.ID + "/audio"; uploaded.Audio.URL != want {
		t.Fatalf("audio url = %q, want %q", uploaded.Audio.URL, want)
	}
	if uploaded.Audio.ContentType != "audio/mpeg" {
		t.Fatalf("audio content type = %q, want audio/mpeg", uploaded.Audio.ContentType)
	}
	if uploaded.Audio.SizeBytes != int64(len(audio)) {
		t.Fatalf("audio size = %d, want %d", uploaded.Audio.SizeBytes, len(audio))
	}
	if uploaded.Audio.DurationSeconds != 12.5 {
		t.Fatalf("audio duration = %v, want 12.5", uploaded.Audio.DurationSeconds)
	}
	if uploaded.Audio.OriginalName != "episode.mp3" {
		t.Fatalf("audio original name = %q, want episode.mp3", uploaded.Audio.OriginalName)
	}

	full := e.get("/api/v1/episodes/" + episode.ID + "/audio").wantStatus(http.StatusOK)
	if !bytes.Equal(full.body, audio) {
		t.Fatalf("audio body = %q, want %q", string(full.body), string(audio))
	}
	if got := full.header.Get("Content-Type"); got != "audio/mpeg" {
		t.Fatalf("audio content type = %q, want audio/mpeg", got)
	}
	if got := full.header.Get("Accept-Ranges"); got != "bytes" {
		t.Fatalf("accept-ranges = %q, want bytes", got)
	}

	partial := e.getWithHeaders("/api/v1/episodes/"+episode.ID+"/audio", map[string]string{
		"Range": "bytes=4-9",
	}).wantStatus(http.StatusPartialContent)
	if !bytes.Equal(partial.body, audio[4:10]) {
		t.Fatalf("range body = %q, want %q", string(partial.body), string(audio[4:10]))
	}
	wantRange := fmt.Sprintf("bytes 4-9/%d", len(audio))
	if got := partial.header.Get("Content-Range"); got != wantRange {
		t.Fatalf("content-range = %q, want %q", got, wantRange)
	}

	e.upload("/api/v1/episodes/"+episode.ID+"/audio", []multipartPart{
		{name: "file", filename: "movie.mp4", contentType: "video/mp4", data: []byte("not audio")},
	}).wantError(http.StatusUnprocessableEntity, "UNSUPPORTED_AUDIO_TYPE")

	e.upload("/api/v1/episodes/"+episode.ID+"/audio", []multipartPart{
		{name: "duration_seconds", data: []byte("abc")},
		{name: "file", filename: "a.mp3", contentType: "audio/mpeg", data: audio},
	}).wantError(http.StatusUnprocessableEntity, "INVALID_DURATION")

	e.upload("/api/v1/episodes/"+episode.ID+"/audio", []multipartPart{
		{name: "duration_seconds", data: []byte("3")},
	}).wantError(http.StatusUnprocessableEntity, "MISSING_FILE")

	e.upload("/api/v1/episodes/"+episode.ID+"/audio", []multipartPart{
		{name: "file", filename: "big.mp3", contentType: "audio/mpeg", data: make([]byte, maxAudioBytes+512)},
	}).wantError(http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")

	silent := e.createEpisode(series.ID, v1.EpisodeInput{Title: "No audio"})
	e.get("/api/v1/episodes/"+silent.ID+"/audio").wantError(http.StatusNotFound, "ASSET_NOT_FOUND")
	e.get("/api/v1/episodes/not-a-uuid/audio").wantError(http.StatusUnprocessableEntity, "INVALID_ID")
}

// TestEpisodeChapters covers the Podcasting 2.0 chapters round trip and its
// validation rules.
func TestEpisodeChapters(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Chapters", Author: "Ada"})
	episode := e.createEpisode(series.ID, v1.EpisodeInput{Title: "Chaptered"})
	e.uploadAudio(episode.ID, []byte("audio-bytes"), "a.mp3", floatPtr(120))

	updated := e.setChapters(episode.ID, validChapters())
	if len(updated.Chapters) != 2 {
		t.Fatalf("episode has %d chapters, want 2", len(updated.Chapters))
	}
	if updated.Chapters[0].StartSeconds != 0 || updated.Chapters[0].Title != "Intro" {
		t.Fatalf("first chapter = %+v, want Intro at 0", updated.Chapters[0])
	}
	if !updated.Chapters[0].TOC {
		t.Fatal("first chapter toc = false, want the default true")
	}

	doc := e.get("/api/v1/episodes/" + episode.ID + "/chapters").wantStatus(http.StatusOK)
	if got := doc.header.Get("Content-Type"); got != "application/json+chapters" {
		t.Fatalf("chapters content type = %q, want application/json+chapters", got)
	}
	var served v1.ChaptersDocument
	doc.decode(&served)
	if served.Version != "1.2.0" {
		t.Fatalf("chapters document version = %q, want 1.2.0", served.Version)
	}
	if len(served.Chapters) != 2 || served.Chapters[1].StartTime != 30 {
		t.Fatalf("served chapters = %+v, want the two saved chapters", served.Chapters)
	}
	if served.Chapters[1].EndTime == nil || *served.Chapters[1].EndTime != 60 {
		t.Fatalf("served second chapter endTime = %v, want 60", served.Chapters[1].EndTime)
	}

	// Unknown members of the extensible document are ignored.
	raw := `{"version":"1.1","unknown":"ignored","chapters":[{"startTime":0,"title":"Only","extra":1}]}`
	var lenient v1.Episode
	e.request(http.MethodPut, "/api/v1/episodes/"+episode.ID+"/chapters", strings.NewReader(raw), "application/json").
		wantStatus(http.StatusOK).decode(&lenient)
	if len(lenient.Chapters) != 1 {
		t.Fatalf("lenient document produced %d chapters, want 1", len(lenient.Chapters))
	}

	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episode.ID+"/chapters", v1.ChaptersDocument{
		Version:  "2.0.0",
		Chapters: []v1.ChapterItem{{StartTime: 0, Title: "Intro"}},
	}, nil).wantError(http.StatusUnprocessableEntity, "INVALID_CHAPTERS_VERSION")

	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episode.ID+"/chapters", v1.ChaptersDocument{
		Version:  "1.2.0",
		Chapters: []v1.ChapterItem{{StartTime: 5, Title: "Late start"}},
	}, nil).wantError(http.StatusUnprocessableEntity, "INVALID_CHAPTERS")

	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episode.ID+"/chapters", v1.ChaptersDocument{
		Version: "1.2.0",
		Chapters: []v1.ChapterItem{
			{StartTime: 0, Title: "One"},
			{StartTime: 0, Title: "Duplicate"},
		},
	}, nil).wantError(http.StatusUnprocessableEntity, "INVALID_CHAPTERS")

	e.sendJSON(http.MethodPut, "/api/v1/episodes/"+episode.ID+"/chapters", v1.ChaptersDocument{
		Version: "1.2.0",
		Chapters: []v1.ChapterItem{
			{StartTime: 0, Title: "Intro"},
			{StartTime: 130, Title: "Beyond the audio"},
		},
	}, nil).wantError(http.StatusUnprocessableEntity, "CHAPTER_OUT_OF_RANGE")

	e.get("/api/v1/episodes/not-a-uuid/chapters").wantError(http.StatusUnprocessableEntity, "INVALID_ID")
	e.get("/api/v1/episodes/"+uuid.NewString()+"/chapters").wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")
}

// TestEpisodePublish covers the publishing checklist and unpublishing.
func TestEpisodePublish(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Publishing", Author: "Ada"})
	episode := e.createEpisode(series.ID, v1.EpisodeInput{Title: "To publish"})

	e.sendJSON(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/publish", nil, nil).
		wantError(http.StatusUnprocessableEntity, "EPISODE_AUDIO_MISSING")

	e.uploadAudio(episode.ID, []byte("audio-bytes"), "a.mp3", floatPtr(120))

	e.sendJSON(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/publish", nil, nil).
		wantError(http.StatusUnprocessableEntity, "EPISODE_CHAPTERS_MISSING")

	e.setChapters(episode.ID, validChapters())

	published := e.publish(episode.ID)
	if published.Status != "published" {
		t.Fatalf("status = %q, want published", published.Status)
	}
	if published.PublishedAt == nil {
		t.Fatal("published_at is null after publishing")
	}
	mustRFC3339(t, "published_at", *published.PublishedAt)

	var unpublished v1.Episode
	e.sendJSON(http.MethodPost, "/api/v1/episodes/"+episode.ID+"/unpublish", nil, &unpublished).
		wantStatus(http.StatusOK)
	if unpublished.Status != "draft" {
		t.Fatalf("status after unpublish = %q, want draft", unpublished.Status)
	}
	if unpublished.PublishedAt != nil {
		t.Fatalf("published_at = %v after unpublish, want nil", *unpublished.PublishedAt)
	}

	e.sendJSON(http.MethodPost, "/api/v1/episodes/"+uuid.NewString()+"/publish", nil, nil).
		wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")
	e.sendJSON(http.MethodPost, "/api/v1/episodes/not-a-uuid/publish", nil, nil).
		wantError(http.StatusUnprocessableEntity, "INVALID_ID")
	e.sendJSON(http.MethodPost, "/api/v1/episodes/not-a-uuid/unpublish", nil, nil).
		wantError(http.StatusUnprocessableEntity, "INVALID_ID")
}

// TestDeleteSeriesWithEpisodes covers the SERIES_NOT_EMPTY conflict and the
// empty-series deletion once its episodes are gone.
func TestDeleteSeriesWithEpisodes(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Guarded", Author: "Ada"})
	episode := e.createEpisode(series.ID, v1.EpisodeInput{Title: "Only episode"})

	e.request(http.MethodDelete, "/api/v1/series/"+series.ID, nil, "").
		wantError(http.StatusConflict, "SERIES_NOT_EMPTY")

	e.request(http.MethodDelete, "/api/v1/episodes/"+episode.ID, nil, "").wantStatus(http.StatusNoContent)
	e.request(http.MethodDelete, "/api/v1/series/"+series.ID, nil, "").wantStatus(http.StatusNoContent)
	e.get("/api/v1/series/"+series.ID).wantError(http.StatusNotFound, "SERIES_NOT_FOUND")
}

// intPtr returns a pointer to v.
func intPtr(v int) *int { return &v }

// floatPtr returns a pointer to v.
func floatPtr(v float64) *float64 { return &v }
