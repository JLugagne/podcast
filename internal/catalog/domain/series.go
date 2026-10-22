package domain

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	// DefaultAccentColor is used when no accent color is provided.
	DefaultAccentColor = "#7A9CFF"
	// MaxSeriesTitleLength is the maximum series title length in runes.
	MaxSeriesTitleLength = 200
	// MaxSeriesAuthorLength is the maximum series author length in runes.
	MaxSeriesAuthorLength = 200
	// MaxSeriesDescriptionLength is the maximum series description length in runes.
	MaxSeriesDescriptionLength = 5000
)

var accentColorPattern = regexp.MustCompile(`^#[0-9A-Fa-f]{6}$`)

// SeriesAttributes are the editable attributes of a series.
//
// Slug is optional: an empty slug derives one from the title on creation, and
// keeps the current slug on update.
type SeriesAttributes struct {
	Title       string
	Author      string
	Description string
	AccentColor string
	Slug        string
}

// Series is a podcast series aggregate. Its version is the optimistic lock
// maintained by the repository: the entity never bumps it itself.
type Series struct {
	id          SeriesID
	attrs       SeriesAttributes
	accentColor string
	cover       *Asset
	createdAt   time.Time
	updatedAt   time.Time
	version     int
}

// NewSeries validates attrs and creates a series with the given id.
func NewSeries(id SeriesID, attrs SeriesAttributes, now time.Time) (*Series, error) {
	s := &Series{id: id, createdAt: now}
	if err := s.applyAttributes(attrs, now); err != nil {
		return nil, err
	}
	return s, nil
}

// ReconstituteSeries rebuilds a series from trusted storage without validation.
func ReconstituteSeries(id SeriesID, attrs SeriesAttributes, cover *Asset, createdAt, updatedAt time.Time, version int) *Series {
	s := &Series{
		id:          id,
		attrs:       attrs,
		accentColor: attrs.AccentColor,
		createdAt:   createdAt,
		updatedAt:   updatedAt,
		version:     version,
	}
	if cover != nil {
		c := *cover
		s.cover = &c
	}
	return s
}

// ID returns the series identifier.
func (s *Series) ID() SeriesID { return s.id }

// Slug returns the series slug.
func (s *Series) Slug() string { return s.attrs.Slug }

// Title returns the series title.
func (s *Series) Title() string { return s.attrs.Title }

// Author returns the series author.
func (s *Series) Author() string { return s.attrs.Author }

// Description returns the series description.
func (s *Series) Description() string { return s.attrs.Description }

// AccentColor returns the series accent color in #RRGGBB form.
func (s *Series) AccentColor() string { return s.accentColor }

// Attributes returns a copy of the editable attributes.
func (s *Series) Attributes() SeriesAttributes { return s.attrs }

// Cover returns a copy of the cover asset, or nil when the series has none.
func (s *Series) Cover() *Asset {
	if s.cover == nil {
		return nil
	}
	c := *s.cover
	return &c
}

// HasCover reports whether a cover is attached.
func (s *Series) HasCover() bool { return s.cover != nil }

// CreatedAt returns the creation time.
func (s *Series) CreatedAt() time.Time { return s.createdAt }

// UpdatedAt returns the last modification time.
func (s *Series) UpdatedAt() time.Time { return s.updatedAt }

// Version returns the optimistic-lock version.
func (s *Series) Version() int { return s.version }

// Update validates and replaces every editable attribute.
func (s *Series) Update(attrs SeriesAttributes, now time.Time) error {
	if err := s.applyAttributes(attrs, now); err != nil {
		return err
	}
	s.updatedAt = now
	return nil
}

// SetCover attaches a cover asset and marks the series updated.
func (s *Series) SetCover(asset Asset, now time.Time) {
	a := asset
	s.cover = &a
	s.updatedAt = now
}

// ClearCover removes the cover asset and marks the series updated.
func (s *Series) ClearCover(now time.Time) {
	s.cover = nil
	s.updatedAt = now
}

func (s *Series) applyAttributes(attrs SeriesAttributes, now time.Time) error {
	title := strings.TrimSpace(attrs.Title)
	if n := utf8.RuneCountInString(title); n < 1 || n > MaxSeriesTitleLength {
		return ErrInvalidSeries
	}
	author := strings.TrimSpace(attrs.Author)
	if utf8.RuneCountInString(author) > MaxSeriesAuthorLength {
		return ErrInvalidSeries
	}
	description := strings.TrimSpace(attrs.Description)
	if utf8.RuneCountInString(description) > MaxSeriesDescriptionLength {
		return ErrInvalidSeries
	}
	accent := strings.TrimSpace(attrs.AccentColor)
	if accent == "" {
		accent = DefaultAccentColor
	}
	if !accentColorPattern.MatchString(accent) {
		return ErrInvalidSeries
	}
	slug := strings.TrimSpace(attrs.Slug)
	if slug == "" {
		slug = s.attrs.Slug
		if slug == "" {
			slug = Slugify(title)
		}
	}
	if err := validateSlug(slug); err != nil {
		return err
	}

	s.attrs = SeriesAttributes{
		Title:       title,
		Author:      author,
		Description: description,
		AccentColor: accent,
		Slug:        slug,
	}
	s.accentColor = accent
	s.updatedAt = now
	return nil
}

// SetVersion is reserved for persistence adapters applying optimistic locking:
// they store version+1 and write the stored value back on the aggregate.
func (s *Series) SetVersion(v int) { s.version = v }
