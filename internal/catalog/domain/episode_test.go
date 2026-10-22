package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

const episodeID = domain.EpisodeID("018f0000-0000-7000-8000-000000000002")

func newDraftEpisode(t *testing.T) *domain.Episode {
	t.Helper()
	e, err := domain.NewEpisode(episodeID, seriesID, domain.EpisodeAttributes{Number: 1, Title: "Episode"}, fixedTime())
	if err != nil {
		t.Fatalf("NewEpisode() error = %v", err)
	}
	return e
}

func validAudio() domain.AudioFile {
	return domain.AudioFile{
		Asset:           domain.Asset{Key: "episodes/1/audio-1.mp3", ContentType: "audio/mpeg", SizeBytes: 1024},
		DurationSeconds: 100,
		OriginalName:    "episode.mp3",
	}
}

func validChapters() []domain.Chapter {
	return []domain.Chapter{
		{StartSeconds: 0, Title: "Intro"},
		{StartSeconds: 30, Title: "Main", EndSeconds: floatPtr(90)},
	}
}

func TestNewEpisode(t *testing.T) {
	now := fixedTime()

	t.Run("creates a draft", func(t *testing.T) {
		e, err := domain.NewEpisode(episodeID, seriesID, domain.EpisodeAttributes{Number: 3, Title: "  Hello  "}, now)
		if err != nil {
			t.Fatalf("NewEpisode() error = %v", err)
		}
		if e.Status() != domain.EpisodeStatusDraft {
			t.Fatalf("Status() = %q", e.Status())
		}
		if e.Number() != 3 || e.Title() != "Hello" {
			t.Fatalf("Number()/Title() = %d/%q", e.Number(), e.Title())
		}
		if e.HasAudio() || e.HasChapters() || e.PublishedAt() != nil {
			t.Fatal("new episode must have no audio, chapters or publication date")
		}
	})

	invalid := map[string]domain.EpisodeAttributes{
		"zero number":          {Number: 0, Title: "Episode"},
		"negative number":      {Number: -1, Title: "Episode"},
		"empty title":          {Number: 1, Title: "  "},
		"title too long":       {Number: 1, Title: strings.Repeat("a", domain.MaxEpisodeTitleLength+1)},
		"description too long": {Number: 1, Title: "Episode", Description: strings.Repeat("a", domain.MaxEpisodeDescriptionLength+1)},
	}
	for name, attrs := range invalid {
		t.Run("rejects "+name, func(t *testing.T) {
			if _, err := domain.NewEpisode(episodeID, seriesID, attrs, now); !errors.Is(err, domain.ErrInvalidEpisode) {
				t.Fatalf("NewEpisode() error = %v, want ErrInvalidEpisode", err)
			}
		})
	}
}

func TestEpisodeUpdate(t *testing.T) {
	e := newDraftEpisode(t)
	later := fixedTime().Add(hourSpan)
	if err := e.Update(domain.EpisodeAttributes{Number: 5, Title: "New", Description: "Desc"}, later); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if e.Number() != 5 || e.Title() != "New" || e.Description() != "Desc" {
		t.Fatalf("attributes = %+v", e.Attributes())
	}
	if !e.UpdatedAt().Equal(later) {
		t.Fatalf("UpdatedAt() = %v", e.UpdatedAt())
	}
}

func TestAttachAudio(t *testing.T) {
	t.Run("accepts supported audio", func(t *testing.T) {
		e := newDraftEpisode(t)
		now := fixedTime().Add(time.Hour)
		if err := e.AttachAudio(validAudio(), now); err != nil {
			t.Fatalf("AttachAudio() error = %v", err)
		}
		audio := e.Audio()
		if audio == nil || audio.Key != "episodes/1/audio-1.mp3" || audio.DurationSeconds != 100 {
			t.Fatalf("Audio() = %+v", audio)
		}
		if !e.UpdatedAt().Equal(now) {
			t.Fatalf("UpdatedAt() = %v", e.UpdatedAt())
		}
	})

	t.Run("rejects unsupported type", func(t *testing.T) {
		e := newDraftEpisode(t)
		audio := validAudio()
		audio.ContentType = "application/octet-stream"
		if err := e.AttachAudio(audio, fixedTime()); !errors.Is(err, domain.ErrUnsupportedAudioType) {
			t.Fatalf("AttachAudio() error = %v, want ErrUnsupportedAudioType", err)
		}
		if e.HasAudio() {
			t.Fatal("audio must not be attached on error")
		}
	})

	t.Run("rejects negative duration", func(t *testing.T) {
		e := newDraftEpisode(t)
		audio := validAudio()
		audio.DurationSeconds = -1
		if err := e.AttachAudio(audio, fixedTime()); !errors.Is(err, domain.ErrInvalidEpisode) {
			t.Fatalf("AttachAudio() error = %v, want ErrInvalidEpisode", err)
		}
	})

	t.Run("rejects chapters past the duration", func(t *testing.T) {
		e := newDraftEpisode(t)
		if err := e.SetChapters(validChapters(), fixedTime()); err != nil {
			t.Fatalf("SetChapters() error = %v", err)
		}
		audio := validAudio()
		audio.DurationSeconds = 30
		if err := e.AttachAudio(audio, fixedTime()); !errors.Is(err, domain.ErrChapterOutOfRange) {
			t.Fatalf("AttachAudio() error = %v, want ErrChapterOutOfRange", err)
		}
	})
}

func TestSetChapters(t *testing.T) {
	t.Run("stores validated chapters", func(t *testing.T) {
		e := newDraftEpisode(t)
		now := fixedTime().Add(time.Hour)
		if err := e.SetChapters(validChapters(), now); err != nil {
			t.Fatalf("SetChapters() error = %v", err)
		}
		if got := e.Chapters(); len(got) != 2 || got[1].Title != "Main" {
			t.Fatalf("Chapters() = %+v", got)
		}
		if !e.UpdatedAt().Equal(now) {
			t.Fatalf("UpdatedAt() = %v", e.UpdatedAt())
		}
	})

	t.Run("propagates validation errors", func(t *testing.T) {
		e := newDraftEpisode(t)
		if err := e.SetChapters([]domain.Chapter{{StartSeconds: 5, Title: "A"}}, fixedTime()); !errors.Is(err, domain.ErrInvalidChapters) {
			t.Fatalf("SetChapters() error = %v, want ErrInvalidChapters", err)
		}
	})

	t.Run("rejects chapters past the audio duration", func(t *testing.T) {
		e := newDraftEpisode(t)
		audio := validAudio()
		audio.DurationSeconds = 100
		if err := e.AttachAudio(audio, fixedTime()); err != nil {
			t.Fatalf("AttachAudio() error = %v", err)
		}
		chapters := []domain.Chapter{{StartSeconds: 0, Title: "A"}, {StartSeconds: 100, Title: "B"}}
		if err := e.SetChapters(chapters, fixedTime()); !errors.Is(err, domain.ErrChapterOutOfRange) {
			t.Fatalf("SetChapters() error = %v, want ErrChapterOutOfRange", err)
		}
	})
}

func TestPublish(t *testing.T) {
	t.Run("requires audio", func(t *testing.T) {
		e := newDraftEpisode(t)
		if err := e.Publish(fixedTime()); !errors.Is(err, domain.ErrEpisodeAudioMissing) {
			t.Fatalf("Publish() error = %v, want ErrEpisodeAudioMissing", err)
		}
	})

	t.Run("requires chapters", func(t *testing.T) {
		e := newDraftEpisode(t)
		if err := e.AttachAudio(validAudio(), fixedTime()); err != nil {
			t.Fatalf("AttachAudio() error = %v", err)
		}
		if err := e.Publish(fixedTime()); !errors.Is(err, domain.ErrEpisodeChaptersMissing) {
			t.Fatalf("Publish() error = %v, want ErrEpisodeChaptersMissing", err)
		}
	})

	t.Run("publishes and is idempotent", func(t *testing.T) {
		e := newDraftEpisode(t)
		if err := e.AttachAudio(validAudio(), fixedTime()); err != nil {
			t.Fatalf("AttachAudio() error = %v", err)
		}
		if err := e.SetChapters(validChapters(), fixedTime()); err != nil {
			t.Fatalf("SetChapters() error = %v", err)
		}
		published := fixedTime().Add(time.Hour)
		if err := e.Publish(published); err != nil {
			t.Fatalf("Publish() error = %v", err)
		}
		if e.Status() != domain.EpisodeStatusPublished {
			t.Fatalf("Status() = %q", e.Status())
		}
		if at := e.PublishedAt(); at == nil || !at.Equal(published) {
			t.Fatalf("PublishedAt() = %v", at)
		}

		later := published.Add(time.Hour)
		if err := e.Publish(later); err != nil {
			t.Fatalf("second Publish() error = %v", err)
		}
		if !e.UpdatedAt().Equal(published) {
			t.Fatalf("idempotent Publish changed UpdatedAt to %v", e.UpdatedAt())
		}
	})
}

func TestUnpublish(t *testing.T) {
	e := newDraftEpisode(t)
	if err := e.AttachAudio(validAudio(), fixedTime()); err != nil {
		t.Fatalf("AttachAudio() error = %v", err)
	}
	if err := e.SetChapters(validChapters(), fixedTime()); err != nil {
		t.Fatalf("SetChapters() error = %v", err)
	}
	published := fixedTime().Add(time.Hour)
	if err := e.Publish(published); err != nil {
		t.Fatalf("Publish() error = %v", err)
	}

	later := published.Add(time.Hour)
	e.Unpublish(later)
	if e.Status() != domain.EpisodeStatusDraft {
		t.Fatalf("Status() = %q", e.Status())
	}
	if e.PublishedAt() != nil {
		t.Fatalf("PublishedAt() = %v, want nil", e.PublishedAt())
	}
	if !e.UpdatedAt().Equal(later) {
		t.Fatalf("UpdatedAt() = %v", e.UpdatedAt())
	}

	e.Unpublish(later.Add(time.Hour))
	if !e.UpdatedAt().Equal(later) {
		t.Fatalf("Unpublish on a draft changed UpdatedAt to %v", e.UpdatedAt())
	}
}

func TestParseEpisodeStatus(t *testing.T) {
	for raw, want := range map[string]domain.EpisodeStatus{
		"draft":       domain.EpisodeStatusDraft,
		"published":   domain.EpisodeStatusPublished,
		" PUBLISHED ": domain.EpisodeStatusPublished,
	} {
		got, err := domain.ParseEpisodeStatus(raw)
		if err != nil || got != want {
			t.Fatalf("ParseEpisodeStatus(%q) = %q, %v", raw, got, err)
		}
	}
	if _, err := domain.ParseEpisodeStatus("archived"); !errors.Is(err, domain.ErrInvalidEpisode) {
		t.Fatalf("ParseEpisodeStatus(archived) error = %v, want ErrInvalidEpisode", err)
	}
}

func TestReconstituteEpisode(t *testing.T) {
	created := fixedTime()
	updated := created.Add(time.Hour)
	published := updated.Add(time.Hour)
	audio := validAudio()

	e := domain.ReconstituteEpisode(
		episodeID,
		seriesID,
		domain.EpisodeAttributes{Number: 4, Title: "Ep", Description: "Desc"},
		domain.EpisodeStatusPublished,
		&audio,
		validChapters(),
		&published,
		created,
		updated,
		9,
	)
	if e.Number() != 4 || e.Title() != "Ep" || e.Status() != domain.EpisodeStatusPublished || e.Version() != 9 {
		t.Fatalf("unexpected episode: %+v, status %q, version %d", e.Attributes(), e.Status(), e.Version())
	}
	if e.Audio() == nil || e.Audio().OriginalName != "episode.mp3" {
		t.Fatalf("Audio() = %+v", e.Audio())
	}
	if len(e.Chapters()) != 2 {
		t.Fatalf("Chapters() = %+v", e.Chapters())
	}
	if e.PublishedAt() == nil || !e.PublishedAt().Equal(published) {
		t.Fatalf("PublishedAt() = %v", e.PublishedAt())
	}
}
