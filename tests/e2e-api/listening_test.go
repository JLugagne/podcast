package e2eapi_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
	listeningv1 "github.com/JLugagne/podcast/pkg/listening/http/v1"
)

// listeningFixture is a published episode with its series, ready to be played.
type listeningFixture struct {
	series  v1.Series
	episode v1.Episode
}

// seedPublishedEpisode creates a series and a published episode (120 seconds,
// two chapters).
func seedPublishedEpisode(t *testing.T, e *env, title string) listeningFixture {
	t.Helper()
	duration := 120.0
	series := e.createSeries(v1.SeriesInput{
		Title:       title,
		Author:      "Host",
		Description: "About the show",
		AccentColor: "#F2A541",
	})
	episode := e.createEpisode(series.ID, v1.EpisodeInput{Title: title + " episode", Description: "Details"})
	episode = e.uploadAudio(episode.ID, []byte("fake-audio-bytes"), "episode.mp3", &duration)
	episode = e.setChapters(episode.ID, validChapters())
	episode = e.publish(episode.ID)
	return listeningFixture{series: series, episode: episode}
}

// recordPosition sends a position update and returns the public progress.
func (e *env) recordPosition(episodeID string, position float64, completed *bool) listeningv1.Progress {
	e.t.Helper()
	var out listeningv1.Progress
	e.sendJSON(http.MethodPut, "/api/v1/listening/progress/"+episodeID, listeningv1.ProgressInput{
		PositionSeconds: position,
		Completed:       completed,
	}, &out).wantStatus(http.StatusOK)
	return out
}

// listProgress returns the listener's progress rows.
func (e *env) listProgress() listeningv1.ProgressList {
	e.t.Helper()
	var out listeningv1.ProgressList
	e.get("/api/v1/listening/progress").wantStatus(http.StatusOK).decode(&out)
	return out
}

// continueListening returns the continue-listening items.
func (e *env) continueListening(query string) listeningv1.ContinueList {
	e.t.Helper()
	var out listeningv1.ContinueList
	e.get("/api/v1/listening/continue" + query).wantStatus(http.StatusOK).decode(&out)
	return out
}

func TestListeningProgressLifecycle(t *testing.T) {
	e := newEnv(t)
	fixture := seedPublishedEpisode(t, e, "Deep Dive")

	if got := e.listProgress(); len(got.Progress) != 0 {
		t.Fatalf("initial progress = %d rows, want 0", len(got.Progress))
	}

	progress := e.recordPosition(fixture.episode.ID, 12.5, nil)
	if progress.EpisodeID != fixture.episode.ID {
		t.Errorf("episode_id = %q, want %q", progress.EpisodeID, fixture.episode.ID)
	}
	if progress.PositionSeconds != 12.5 {
		t.Errorf("position_seconds = %v, want 12.5", progress.PositionSeconds)
	}
	if progress.Completed {
		t.Error("completed = true, want false")
	}
	mustRFC3339(t, "updated_at", progress.UpdatedAt)

	if got := e.listProgress(); len(got.Progress) != 1 {
		t.Fatalf("progress = %d rows, want 1", len(got.Progress))
	}

	// A position beyond the duration is clamped to it and completes the episode.
	clamped := e.recordPosition(fixture.episode.ID, 500, nil)
	if clamped.PositionSeconds != 120 {
		t.Errorf("clamped position = %v, want 120", clamped.PositionSeconds)
	}
	if !clamped.Completed {
		t.Error("completed = false, want true after reaching the end")
	}
	if items := e.continueListening(""); len(items.Items) != 0 {
		t.Errorf("continue items = %d, want 0 for a completed episode", len(items.Items))
	}

	// Recording an earlier position re-opens the episode.
	reopened := e.recordPosition(fixture.episode.ID, 30, nil)
	if reopened.Completed {
		t.Error("completed = true, want false after reopening")
	}
	if items := e.continueListening(""); len(items.Items) != 1 {
		t.Fatalf("continue items = %d, want 1", len(items.Items))
	} else {
		item := items.Items[0]
		if item.Progress.EpisodeID != fixture.episode.ID || item.Progress.PositionSeconds != 30 {
			t.Errorf("continue progress = %+v", item.Progress)
		}
		episode := item.Episode
		if episode.ID != fixture.episode.ID || episode.Title != fixture.episode.Title {
			t.Errorf("episode = %+v, want the catalog episode", episode)
		}
		if episode.DurationSeconds != 120 || episode.ChapterCount != 2 || episode.Number != fixture.episode.Number {
			t.Errorf("episode = %+v, want duration 120 and 2 chapters", episode)
		}
		if episode.SeriesID != fixture.series.ID || episode.SeriesTitle != fixture.series.Title ||
			episode.SeriesSlug != fixture.series.Slug || episode.AccentColor != "#F2A541" {
			t.Errorf("episode series = %+v, want the catalog series", episode)
		}
		if episode.CoverURL != nil {
			t.Errorf("cover_url = %v, want null without a cover", episode.CoverURL)
		}
	}

	// Reaching the two-second completion window completes the episode.
	nearEnd := e.recordPosition(fixture.episode.ID, 118.5, nil)
	if !nearEnd.Completed {
		t.Error("completed = false, want true inside the completion window")
	}

	// The explicit completed flag marks the episode played whatever the position.
	explicit := e.recordPosition(fixture.episode.ID, 1, boolPtr(true))
	if !explicit.Completed || explicit.PositionSeconds != 120 {
		t.Errorf("progress = %+v, want a completed episode at its end", explicit)
	}
	explicit = e.recordPosition(fixture.episode.ID, 15, boolPtr(false))
	if explicit.Completed || explicit.PositionSeconds != 15 {
		t.Errorf("progress = %+v, want the reopened position 15", explicit)
	}
}

func TestListeningContinueIncludesCover(t *testing.T) {
	e := newEnv(t)
	fixture := seedPublishedEpisode(t, e, "Art Show")
	e.uploadCover(fixture.series.ID, []byte("fake-png-bytes"), "image/png", "cover.png").wantStatus(http.StatusOK)
	e.recordPosition(fixture.episode.ID, 10, nil)

	items := e.continueListening("")
	if len(items.Items) != 1 {
		t.Fatalf("continue items = %d, want 1", len(items.Items))
	}
	cover := items.Items[0].Episode.CoverURL
	want := "/api/v1/series/" + fixture.series.ID + "/cover"
	if cover == nil || *cover != want {
		t.Errorf("cover_url = %v, want %q", cover, want)
	}
}

func TestListeningPlayedAndUnplayed(t *testing.T) {
	e := newEnv(t)
	fixture := seedPublishedEpisode(t, e, "Played Show")

	var played listeningv1.Progress
	e.sendJSON(http.MethodPost, "/api/v1/listening/progress/"+fixture.episode.ID+"/played", nil, &played).wantStatus(http.StatusOK)
	if !played.Completed || played.PositionSeconds != 120 {
		t.Errorf("played = %+v, want a completed episode at its end", played)
	}
	if got := e.listProgress(); len(got.Progress) != 1 || !got.Progress[0].Completed {
		t.Errorf("progress = %+v, want one completed row", got.Progress)
	}

	e.sendJSON(http.MethodDelete, "/api/v1/listening/progress/"+fixture.episode.ID, nil, nil).wantStatus(http.StatusNoContent)
	if got := e.listProgress(); len(got.Progress) != 0 {
		t.Errorf("progress = %+v, want no row after unplaying", got.Progress)
	}
	// Unplaying an episode the listener never played is idempotent.
	e.sendJSON(http.MethodDelete, "/api/v1/listening/progress/"+fixture.episode.ID, nil, nil).wantStatus(http.StatusNoContent)
}

func TestListeningRejectsDraftAndUnknownEpisodes(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Draft Show", Author: "Host"})
	draft := e.createEpisode(series.ID, v1.EpisodeInput{Title: "Draft episode"})

	e.sendJSON(http.MethodPut, "/api/v1/listening/progress/"+draft.ID, listeningv1.ProgressInput{PositionSeconds: 10}, nil).
		wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")
	e.sendJSON(http.MethodPost, "/api/v1/listening/progress/"+draft.ID+"/played", nil, nil).
		wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")

	unknown := "018f0000-0000-7000-8000-0000000000ff"
	e.sendJSON(http.MethodPut, "/api/v1/listening/progress/"+unknown, listeningv1.ProgressInput{PositionSeconds: 10}, nil).
		wantError(http.StatusNotFound, "EPISODE_NOT_FOUND")
}

func TestListeningValidation(t *testing.T) {
	e := newEnv(t)
	fixture := seedPublishedEpisode(t, e, "Validation Show")
	basePath := "/api/v1/listening/progress/"

	e.request(http.MethodPut, basePath+"not-a-uuid", strings.NewReader(`{"position_seconds":1}`), "application/json").
		wantError(http.StatusUnprocessableEntity, "INVALID_ID")

	e.sendJSON(http.MethodPut, basePath+fixture.episode.ID, listeningv1.ProgressInput{PositionSeconds: -1}, nil).
		wantError(http.StatusUnprocessableEntity, "INVALID_POSITION")

	e.request(http.MethodPut, basePath+fixture.episode.ID, strings.NewReader(`{"position_seconds":`), "application/json").
		wantError(http.StatusBadRequest, "BAD_REQUEST")

	e.request(http.MethodPut, basePath+fixture.episode.ID, strings.NewReader(`{"position_seconds":1,"unknown":true}`), "application/json").
		wantError(http.StatusBadRequest, "BAD_REQUEST")

	e.get("/api/v1/listening/continue?limit=abc").wantError(http.StatusBadRequest, "BAD_REQUEST")
	e.get("/api/v1/listening/continue?limit=-2").wantError(http.StatusBadRequest, "BAD_REQUEST")

	e.request(http.MethodDelete, basePath+"not-a-uuid", nil, "").wantError(http.StatusUnprocessableEntity, "INVALID_ID")
}

func TestListeningContinueSkipsUnavailableEpisodes(t *testing.T) {
	e := newEnv(t)
	series := e.createSeries(v1.SeriesInput{Title: "Serial Show", Author: "Host"})
	duration := 120.0

	seed := func(number int) v1.Episode {
		episode := e.createEpisode(series.ID, v1.EpisodeInput{Number: &number, Title: "Episode"})
		episode = e.uploadAudio(episode.ID, []byte("fake-audio-bytes"), "episode.mp3", &duration)
		episode = e.setChapters(episode.ID, validChapters())
		return e.publish(episode.ID)
	}

	first := seed(1)
	second := seed(2)
	third := seed(3)

	for _, episode := range []v1.Episode{first, second, third} {
		e.recordPosition(episode.ID, 10, nil)
		time.Sleep(2 * time.Millisecond)
	}

	items := e.continueListening("")
	if len(items.Items) != 3 {
		t.Fatalf("continue items = %d, want 3", len(items.Items))
	}
	wantOrder := []string{third.ID, second.ID, first.ID}
	for i, want := range wantOrder {
		if items.Items[i].Episode.ID != want {
			t.Errorf("item %d = %q, want %q (newest first)", i, items.Items[i].Episode.ID, want)
		}
	}

	// The progress list keeps every row, newest first.
	progress := e.listProgress()
	if len(progress.Progress) != 3 || progress.Progress[0].EpisodeID != third.ID {
		t.Errorf("progress = %+v, want 3 rows newest first", progress.Progress)
	}

	// The limit bounds the number of returned items, newest first.
	if limited := e.continueListening("?limit=2"); len(limited.Items) != 2 ||
		limited.Items[0].Episode.ID != third.ID || limited.Items[1].Episode.ID != second.ID {
		t.Errorf("limited items = %+v, want the two newest episodes", limited.Items)
	}

	// An unpublished episode is skipped but keeps its progress row.
	e.sendJSON(http.MethodPost, "/api/v1/episodes/"+second.ID+"/unpublish", nil, nil).wantStatus(http.StatusOK)
	// A deleted episode is skipped too, even though its row still references it.
	e.sendJSON(http.MethodDelete, "/api/v1/episodes/"+third.ID, nil, nil).wantStatus(http.StatusNoContent)

	items = e.continueListening("")
	if len(items.Items) != 1 || items.Items[0].Episode.ID != first.ID {
		t.Fatalf("continue items = %+v, want only the first episode", items.Items)
	}
	if got := e.listProgress(); len(got.Progress) != 3 {
		t.Errorf("progress = %d rows, want the 3 stored rows", len(got.Progress))
	}

	// Skipped episodes do not consume the limit: the only available item is
	// returned even though it is behind two unavailable rows.
	if limited := e.continueListening("?limit=1"); len(limited.Items) != 1 || limited.Items[0].Episode.ID != first.ID {
		t.Errorf("limited items = %+v, want the first available episode", limited.Items)
	}
}

// boolPtr returns a pointer to b.
func boolPtr(b bool) *bool { return &b }
