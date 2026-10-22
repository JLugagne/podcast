// Package converters maps catalog domain types to their public HTTP
// representation and back.
package converters

import (
	"regexp"
	"strings"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

// chaptersDocumentVersion is the version written in the Podcasting 2.0 JSON
// chapters documents served by the API.
const chaptersDocumentVersion = "1.2.0"

// chaptersVersionPattern accepts any 1.x chapters document version.
var chaptersVersionPattern = regexp.MustCompile(`^1\.[0-9]+(\.[0-9]+)*$`)

// ErrInvalidChaptersVersion reports a chapters document whose version is not a
// 1.x version.
var ErrInvalidChaptersVersion = domain.Wrap(domain.ErrValidation, "INVALID_CHAPTERS_VERSION", "chapters document version must be a 1.x version")

const (
	seriesCoverURLPrefix  = "/api/v1/series/"
	episodeAudioURLPrefix = "/api/v1/episodes/"
	feedURLPrefix         = "/feeds/"
	feedFileExtension     = ".xml"
)

// ToSeries maps a series view to its public representation.
func ToSeries(view *seriesuc.SeriesView) v1.Series {
	s := view.Series
	out := v1.Series{
		ID:                   string(s.ID()),
		Slug:                 s.Slug(),
		FeedURL:              feedURLPrefix + s.Slug() + feedFileExtension,
		Title:                s.Title(),
		Author:               s.Author(),
		Description:          s.Description(),
		AccentColor:          s.AccentColor(),
		EpisodeCount:         view.EpisodeCount,
		PublishedCount:       view.PublishedCount,
		TotalDurationSeconds: view.TotalDurationSeconds,
		CreatedAt:            ts(s.CreatedAt()),
		UpdatedAt:            ts(s.UpdatedAt()),
		Version:              s.Version(),
	}
	if s.HasCover() {
		url := seriesCoverURLPrefix + string(s.ID()) + "/cover"
		out.CoverURL = &url
	}
	return out
}

// ToSeriesList maps series views to their public representation. The result is
// never nil so the JSON body always carries a list.
func ToSeriesList(views []*seriesuc.SeriesView) v1.SeriesList {
	list := make([]v1.Series, 0, len(views))
	for _, view := range views {
		list = append(list, ToSeries(view))
	}
	return v1.SeriesList{Series: list}
}

// ToSeriesAttributes maps a series input to domain attributes.
func ToSeriesAttributes(in v1.SeriesInput) domain.SeriesAttributes {
	return domain.SeriesAttributes{
		Title:       in.Title,
		Author:      in.Author,
		Description: in.Description,
		AccentColor: in.AccentColor,
		Slug:        in.Slug,
	}
}

// ToEpisode maps an episode to its public representation.
func ToEpisode(e *domain.Episode) v1.Episode {
	out := v1.Episode{
		ID:          string(e.ID()),
		SeriesID:    string(e.SeriesID()),
		Number:      e.Number(),
		Title:       e.Title(),
		Description: e.Description(),
		Status:      string(e.Status()),
		CreatedAt:   ts(e.CreatedAt()),
		UpdatedAt:   ts(e.UpdatedAt()),
		Version:     e.Version(),
	}
	if a := e.Audio(); a != nil {
		out.Audio = &v1.Audio{
			URL:             episodeAudioURLPrefix + string(e.ID()) + "/audio",
			ContentType:     a.ContentType,
			SizeBytes:       a.SizeBytes,
			DurationSeconds: a.DurationSeconds,
			OriginalName:    a.OriginalName,
		}
	}
	chapters := e.Chapters()
	out.Chapters = make([]v1.Chapter, 0, len(chapters))
	for _, c := range chapters {
		out.Chapters = append(out.Chapters, ToChapter(c))
	}
	if published := e.PublishedAt(); published != nil {
		value := ts(*published)
		out.PublishedAt = &value
	}
	return out
}

// ToEpisodeList maps episodes to their public representation. The result is
// never nil so the JSON body always carries a list.
func ToEpisodeList(episodes []*domain.Episode) v1.EpisodeList {
	list := make([]v1.Episode, 0, len(episodes))
	for _, e := range episodes {
		list = append(list, ToEpisode(e))
	}
	return v1.EpisodeList{Episodes: list}
}

// ToEpisodeAttributes maps an episode input to domain attributes.
func ToEpisodeAttributes(in v1.EpisodeInput) domain.EpisodeAttributes {
	out := domain.EpisodeAttributes{Title: in.Title, Description: in.Description}
	if in.Number != nil {
		out.Number = *in.Number
	}
	return out
}

// ToChapter maps a domain chapter to its public representation.
func ToChapter(c domain.Chapter) v1.Chapter {
	out := v1.Chapter{
		StartSeconds: c.StartSeconds,
		Title:        c.Title,
		URL:          c.URL,
		ImageURL:     c.ImageURL,
		TOC:          c.TOC,
	}
	if c.EndSeconds != nil {
		end := *c.EndSeconds
		out.EndSeconds = &end
	}
	return out
}

// ToChapters maps a Podcasting 2.0 chapters document to domain chapters. The
// document version must be a 1.x version; a missing toc flag defaults to true.
func ToChapters(doc v1.ChaptersDocument) ([]domain.Chapter, error) {
	if !chaptersVersionPattern.MatchString(strings.TrimSpace(doc.Version)) {
		return nil, ErrInvalidChaptersVersion
	}
	out := make([]domain.Chapter, 0, len(doc.Chapters))
	for _, item := range doc.Chapters {
		chapter := domain.Chapter{
			StartSeconds: item.StartTime,
			Title:        item.Title,
			URL:          item.URL,
			ImageURL:     item.Img,
			TOC:          true,
		}
		if item.TOC != nil {
			chapter.TOC = *item.TOC
		}
		if item.EndTime != nil {
			end := *item.EndTime
			chapter.EndSeconds = &end
		}
		out = append(out, chapter)
	}
	return out, nil
}

// ToChaptersDocument maps domain chapters to a Podcasting 2.0 chapters
// document. The result is never nil so the JSON body always carries a list.
func ToChaptersDocument(chapters []domain.Chapter) v1.ChaptersDocument {
	items := make([]v1.ChapterItem, 0, len(chapters))
	for _, c := range chapters {
		toc := c.TOC
		items = append(items, v1.ChapterItem{
			StartTime: c.StartSeconds,
			Title:     c.Title,
			Img:       c.ImageURL,
			URL:       c.URL,
			TOC:       &toc,
			EndTime:   c.EndSeconds,
		})
	}
	return v1.ChaptersDocument{Version: chaptersDocumentVersion, Chapters: items}
}

// ts formats a timestamp as an RFC3339 string in UTC.
func ts(t time.Time) string { return t.UTC().Format(time.RFC3339) }
