package domain

import (
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

// EpisodeStatus is the publication state of an episode.
type EpisodeStatus string

const (
	// EpisodeStatusDraft means the episode is not publicly available yet.
	EpisodeStatusDraft EpisodeStatus = "draft"
	// EpisodeStatusPublished means the episode is publicly available.
	EpisodeStatusPublished EpisodeStatus = "published"

	// MaxEpisodeTitleLength is the maximum episode title length in runes.
	MaxEpisodeTitleLength = 200
	// MaxEpisodeDescriptionLength is the maximum episode description length in runes.
	MaxEpisodeDescriptionLength = 10000
)

// ParseEpisodeStatus parses a stored or queried episode status.
func ParseEpisodeStatus(raw string) (EpisodeStatus, error) {
	switch EpisodeStatus(strings.ToLower(strings.TrimSpace(raw))) {
	case EpisodeStatusDraft:
		return EpisodeStatusDraft, nil
	case EpisodeStatusPublished:
		return EpisodeStatusPublished, nil
	default:
		return "", ErrInvalidEpisode
	}
}

// EpisodeAttributes are the editable attributes of an episode. Number is
// unique per series and must be at least 1.
type EpisodeAttributes struct {
	Number      int
	Title       string
	Description string
}

// Episode is a podcast episode aggregate. Its version is the optimistic lock
// maintained by the repository: the entity never bumps it itself.
type Episode struct {
	id          EpisodeID
	seriesID    SeriesID
	attrs       EpisodeAttributes
	status      EpisodeStatus
	audio       *AudioFile
	chapters    []Chapter
	publishedAt *time.Time
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewEpisode validates attrs and creates a draft episode with the given id.
func NewEpisode(id EpisodeID, seriesID SeriesID, attrs EpisodeAttributes, now time.Time) (*Episode, error) {
	e := &Episode{
		id:        id,
		seriesID:  seriesID,
		status:    EpisodeStatusDraft,
		createdAt: now,
	}
	if err := e.applyAttributes(attrs, now); err != nil {
		return nil, err
	}
	return e, nil
}

// ReconstituteEpisode rebuilds an episode from trusted storage without
// validation.
func ReconstituteEpisode(
	id EpisodeID,
	seriesID SeriesID,
	attrs EpisodeAttributes,
	status EpisodeStatus,
	audio *AudioFile,
	chapters []Chapter,
	publishedAt *time.Time,
	createdAt, updatedAt time.Time,
	version int,
) *Episode {
	e := &Episode{
		id:        id,
		seriesID:  seriesID,
		attrs:     attrs,
		status:    status,
		chapters:  cloneChapters(chapters),
		createdAt: createdAt,
		updatedAt: updatedAt,
		version:   version,
	}
	if audio != nil {
		a := *audio
		e.audio = &a
	}
	if publishedAt != nil {
		t := *publishedAt
		e.publishedAt = &t
	}
	return e
}

// ID returns the episode identifier.
func (e *Episode) ID() EpisodeID { return e.id }

// SeriesID returns the identifier of the owning series.
func (e *Episode) SeriesID() SeriesID { return e.seriesID }

// Number returns the episode number within its series.
func (e *Episode) Number() int { return e.attrs.Number }

// Title returns the episode title.
func (e *Episode) Title() string { return e.attrs.Title }

// Description returns the episode description.
func (e *Episode) Description() string { return e.attrs.Description }

// Status returns the publication status.
func (e *Episode) Status() EpisodeStatus { return e.status }

// Attributes returns a copy of the editable attributes.
func (e *Episode) Attributes() EpisodeAttributes { return e.attrs }

// Audio returns a copy of the audio file, or nil when the episode has none.
func (e *Episode) Audio() *AudioFile {
	if e.audio == nil {
		return nil
	}
	a := *e.audio
	return &a
}

// HasAudio reports whether audio is attached.
func (e *Episode) HasAudio() bool { return e.audio != nil }

// Chapters returns a copy of the chapter list.
func (e *Episode) Chapters() []Chapter { return cloneChapters(e.chapters) }

// HasChapters reports whether at least one chapter is defined.
func (e *Episode) HasChapters() bool { return len(e.chapters) > 0 }

// PublishedAt returns when the episode was published, or nil.
func (e *Episode) PublishedAt() *time.Time {
	if e.publishedAt == nil {
		return nil
	}
	t := *e.publishedAt
	return &t
}

// CreatedAt returns the creation time.
func (e *Episode) CreatedAt() time.Time { return e.createdAt }

// UpdatedAt returns the last modification time.
func (e *Episode) UpdatedAt() time.Time { return e.updatedAt }

// Version returns the optimistic-lock version.
func (e *Episode) Version() int { return e.version }

// Update validates and replaces every editable attribute.
func (e *Episode) Update(attrs EpisodeAttributes, now time.Time) error {
	if err := e.applyAttributes(attrs, now); err != nil {
		return err
	}
	e.updatedAt = now
	return nil
}

// AttachAudio validates and replaces the episode audio. When a duration is
// known and chapters already exist, every chapter must start before it.
func (e *Episode) AttachAudio(audio AudioFile, now time.Time) error {
	if err := ValidateAudioContentType(audio.ContentType); err != nil {
		return err
	}
	if math.IsNaN(audio.DurationSeconds) || math.IsInf(audio.DurationSeconds, 0) || audio.DurationSeconds < 0 {
		return ErrInvalidEpisode
	}
	if audio.DurationSeconds > 0 {
		for _, c := range e.chapters {
			if c.StartSeconds >= audio.DurationSeconds {
				return ErrChapterOutOfRange
			}
		}
	}
	a := audio
	e.audio = &a
	e.updatedAt = now
	return nil
}

// SetChapters validates and replaces the chapter list. When the audio duration
// is known, every chapter must start before it.
func (e *Episode) SetChapters(chapters []Chapter, now time.Time) error {
	validated, err := NewChapters(chapters)
	if err != nil {
		return err
	}
	if e.audio != nil && e.audio.DurationSeconds > 0 {
		if last := validated[len(validated)-1]; last.StartSeconds >= e.audio.DurationSeconds {
			return ErrChapterOutOfRange
		}
	}
	e.chapters = validated
	e.updatedAt = now
	return nil
}

// Publish marks the episode published. It requires audio and at least one
// chapter; publishing an already published episode is a no-op.
func (e *Episode) Publish(now time.Time) error {
	if e.status == EpisodeStatusPublished {
		return nil
	}
	if e.audio == nil {
		return ErrEpisodeAudioMissing
	}
	if len(e.chapters) == 0 {
		return ErrEpisodeChaptersMissing
	}
	e.status = EpisodeStatusPublished
	published := now
	e.publishedAt = &published
	e.updatedAt = now
	return nil
}

// Unpublish moves the episode back to draft. Unpublishing a draft is a no-op.
func (e *Episode) Unpublish(now time.Time) {
	if e.status == EpisodeStatusDraft {
		return
	}
	e.status = EpisodeStatusDraft
	e.publishedAt = nil
	e.updatedAt = now
}

func (e *Episode) applyAttributes(attrs EpisodeAttributes, now time.Time) error {
	if attrs.Number < 1 {
		return ErrInvalidEpisode
	}
	title := strings.TrimSpace(attrs.Title)
	if n := utf8.RuneCountInString(title); n < 1 || n > MaxEpisodeTitleLength {
		return ErrInvalidEpisode
	}
	description := strings.TrimSpace(attrs.Description)
	if utf8.RuneCountInString(description) > MaxEpisodeDescriptionLength {
		return ErrInvalidEpisode
	}
	e.attrs = EpisodeAttributes{Number: attrs.Number, Title: title, Description: description}
	e.updatedAt = now
	return nil
}

func cloneChapters(chapters []Chapter) []Chapter {
	if len(chapters) == 0 {
		return nil
	}
	out := make([]Chapter, len(chapters))
	copy(out, chapters)
	for i := range out {
		if out[i].EndSeconds != nil {
			end := *out[i].EndSeconds
			out[i].EndSeconds = &end
		}
	}
	return out
}

// SetVersion is reserved for persistence adapters applying optimistic locking:
// they store version+1 and write the stored value back on the aggregate.
func (e *Episode) SetVersion(v int) { e.version = v }
