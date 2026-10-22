package domain

import (
	"math"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// MaxChapters is the maximum number of chapters of an episode.
	MaxChapters = 500
	// MaxChapterTitleLength is the maximum chapter title length in runes.
	MaxChapterTitleLength = 300
)

// Chapter is a Podcasting 2.0 chapter marker. EndSeconds is optional and, when
// set, must be greater than StartSeconds.
type Chapter struct {
	StartSeconds float64
	Title        string
	URL          string
	ImageURL     string
	EndSeconds   *float64
	TOC          bool
}

// NewChapters validates and normalizes chapters: 1 to MaxChapters entries,
// sorted by strictly increasing start time, the first one starting at 0, each
// title non-empty and short enough, and each optional end greater than its
// start. It returns a defensive copy.
func NewChapters(chapters []Chapter) ([]Chapter, error) {
	if len(chapters) == 0 || len(chapters) > MaxChapters {
		return nil, ErrInvalidChapters
	}

	out := make([]Chapter, len(chapters))
	copy(out, chapters)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].StartSeconds < out[j].StartSeconds
	})

	for i := range out {
		c := &out[i]
		c.Title = strings.TrimSpace(c.Title)
		if c.Title == "" || utf8.RuneCountInString(c.Title) > MaxChapterTitleLength {
			return nil, ErrInvalidChapters
		}
		if math.IsNaN(c.StartSeconds) || math.IsInf(c.StartSeconds, 0) || c.StartSeconds < 0 {
			return nil, ErrInvalidChapters
		}
		if c.EndSeconds != nil {
			end := *c.EndSeconds
			if math.IsNaN(end) || math.IsInf(end, 0) || end <= c.StartSeconds {
				return nil, ErrInvalidChapters
			}
			c.EndSeconds = &end
		}
		if i == 0 {
			if c.StartSeconds != 0 {
				return nil, ErrInvalidChapters
			}
			continue
		}
		if c.StartSeconds == out[i-1].StartSeconds {
			return nil, ErrInvalidChapters
		}
	}

	return out, nil
}
