package converters_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/usecases/seriesuc"
	"github.com/JLugagne/podcast/internal/catalog/inbound/http/converters"
	v1 "github.com/JLugagne/podcast/pkg/catalog/http/v1"
)

var (
	cestZone = time.FixedZone("CEST", 2*60*60)
	at       = time.Date(2026, 10, 21, 14, 30, 0, 0, cestZone)
)

const (
	seriesID  = "018f0000-0000-7000-8000-000000000001"
	episodeID = "018f0000-0000-7000-8000-000000000002"
)

// seriesView builds a series view with a cover.
func seriesView(cover *domain.Asset) *seriesuc.SeriesView {
	return &seriesuc.SeriesView{
		Series: domain.ReconstituteSeries(
			domain.SeriesID(seriesID),
			domain.SeriesAttributes{Title: "Show", Author: "Host", Description: "About", AccentColor: "#7A9CFF", Slug: "show"},
			cover, at, at.Add(time.Hour), 5,
		),
		EpisodeCount:         3,
		PublishedCount:       1,
		TotalDurationSeconds: 42.5,
	}
}

func TestToSeries(t *testing.T) {
	view := seriesView(&domain.Asset{Key: "series/x/cover.png", ContentType: "image/png", SizeBytes: 8})

	out := converters.ToSeries(view)

	if out.ID != seriesID || out.Slug != "show" || out.Title != "Show" || out.Author != "Host" {
		t.Fatalf("unexpected series: %+v", out)
	}
	if out.AccentColor != "#7A9CFF" || out.Version != 5 {
		t.Fatalf("unexpected series: %+v", out)
	}
	if out.EpisodeCount != 3 || out.PublishedCount != 1 || out.TotalDurationSeconds != 42.5 {
		t.Fatalf("unexpected counters: %+v", out)
	}
	if out.CreatedAt != "2026-10-21T12:30:00Z" || out.UpdatedAt != "2026-10-21T13:30:00Z" {
		t.Fatalf("timestamps are not RFC3339 UTC: %q %q", out.CreatedAt, out.UpdatedAt)
	}
	if out.FeedURL != "/feeds/show.xml" {
		t.Fatalf("feed_url = %q, want /feeds/show.xml", out.FeedURL)
	}
	if out.CoverURL == nil || *out.CoverURL != "/api/v1/series/"+seriesID+"/cover" {
		t.Fatalf("cover_url = %v", out.CoverURL)
	}
}

func TestToSeriesWithoutCover(t *testing.T) {
	out := converters.ToSeries(seriesView(nil))

	if out.CoverURL != nil {
		t.Fatalf("cover_url = %q, want nil", *out.CoverURL)
	}
}

func TestToSeriesListNeverNil(t *testing.T) {
	out := converters.ToSeriesList(nil)

	if out.Series == nil {
		t.Fatal("Series is nil, want an empty slice")
	}
	if len(out.Series) != 0 {
		t.Fatalf("length = %d, want 0", len(out.Series))
	}
}

func TestToSeriesAttributes(t *testing.T) {
	in := v1.SeriesInput{Slug: "s", Title: "T", Author: "A", Description: "D", AccentColor: "#000000"}

	out := converters.ToSeriesAttributes(in)

	want := domain.SeriesAttributes{Title: "T", Author: "A", Description: "D", AccentColor: "#000000", Slug: "s"}
	if out != want {
		t.Fatalf("attributes = %+v, want %+v", out, want)
	}
}

// episode builds an episode with audio and optional chapters.
func episode(chapters []domain.Chapter, publishedAt *time.Time) *domain.Episode {
	audio := &domain.AudioFile{
		Asset:           domain.Asset{Key: "episodes/x/audio.mp3", ContentType: "audio/mpeg", SizeBytes: 42},
		DurationSeconds: 95.5,
		OriginalName:    "episode.mp3",
	}
	return domain.ReconstituteEpisode(
		domain.EpisodeID(episodeID), domain.SeriesID(seriesID),
		domain.EpisodeAttributes{Number: 1, Title: "Episode", Description: "Details"},
		domain.EpisodeStatusDraft, audio, chapters, publishedAt,
		at, at.Add(time.Hour), 4,
	)
}

func TestToEpisode(t *testing.T) {
	end := 30.5
	chapters := []domain.Chapter{
		{StartSeconds: 0, Title: "Intro", URL: "https://example.com/0", TOC: true},
		{StartSeconds: 15, Title: "Second", ImageURL: "https://example.com/i.png", EndSeconds: &end, TOC: false},
	}
	published := at.Add(2 * time.Hour)

	out := converters.ToEpisode(episode(chapters, &published))

	if out.ID != episodeID || out.SeriesID != seriesID || out.Number != 1 || out.Status != "draft" {
		t.Fatalf("unexpected episode: %+v", out)
	}
	if out.Audio == nil {
		t.Fatal("audio is nil")
	}
	if out.Audio.URL != "/api/v1/episodes/"+episodeID+"/audio" {
		t.Fatalf("audio url = %q", out.Audio.URL)
	}
	if out.Audio.ContentType != "audio/mpeg" || out.Audio.SizeBytes != 42 || out.Audio.DurationSeconds != 95.5 || out.Audio.OriginalName != "episode.mp3" {
		t.Fatalf("unexpected audio: %+v", out.Audio)
	}
	if len(out.Chapters) != 2 {
		t.Fatalf("chapters = %+v", out.Chapters)
	}
	if out.Chapters[1].StartSeconds != 15 || out.Chapters[1].ImageURL != "https://example.com/i.png" {
		t.Fatalf("second chapter = %+v", out.Chapters[1])
	}
	if out.Chapters[1].EndSeconds == nil || *out.Chapters[1].EndSeconds != end {
		t.Fatalf("end_seconds = %v", out.Chapters[1].EndSeconds)
	}
	if out.Chapters[1].TOC {
		t.Fatal("toc = true, want false")
	}
	if out.PublishedAt == nil || *out.PublishedAt != "2026-10-21T14:30:00Z" {
		t.Fatalf("published_at = %v, want 2026-10-21T14:30:00Z", out.PublishedAt)
	}
	if out.CreatedAt != "2026-10-21T12:30:00Z" {
		t.Fatalf("created_at = %q", out.CreatedAt)
	}
}

func TestToEpisodeWithoutAudioOrChapters(t *testing.T) {
	e := domain.ReconstituteEpisode(
		domain.EpisodeID(episodeID), domain.SeriesID(seriesID),
		domain.EpisodeAttributes{Number: 1, Title: "Episode"},
		domain.EpisodeStatusDraft, nil, nil, nil, at, at, 1,
	)

	out := converters.ToEpisode(e)

	if out.Audio != nil {
		t.Fatalf("audio = %+v, want nil", out.Audio)
	}
	if out.Chapters == nil || len(out.Chapters) != 0 {
		t.Fatalf("chapters = %+v, want an empty slice", out.Chapters)
	}
	if out.PublishedAt != nil {
		t.Fatalf("published_at = %v, want nil", out.PublishedAt)
	}
}

func TestToEpisodeListNeverNil(t *testing.T) {
	out := converters.ToEpisodeList(nil)

	if out.Episodes == nil || len(out.Episodes) != 0 {
		t.Fatalf("episodes = %+v, want an empty slice", out.Episodes)
	}
}

func TestToEpisodeAttributes(t *testing.T) {
	number := 7
	out := converters.ToEpisodeAttributes(v1.EpisodeInput{Number: &number, Title: "T", Description: "D"})

	want := domain.EpisodeAttributes{Number: 7, Title: "T", Description: "D"}
	if out != want {
		t.Fatalf("attributes = %+v, want %+v", out, want)
	}

	if got := converters.ToEpisodeAttributes(v1.EpisodeInput{Title: "T"}).Number; got != 0 {
		t.Fatalf("number = %d, want 0 when omitted", got)
	}
}

func TestToChapters(t *testing.T) {
	end := 30.5
	toc := false
	doc := v1.ChaptersDocument{
		Version: "1.2.0",
		Chapters: []v1.ChapterItem{
			{StartTime: 0, Title: "Intro", URL: "https://example.com/0"},
			{StartTime: 15, Title: "Second", Img: "https://example.com/i.png", EndTime: &end, TOC: &toc},
		},
	}

	chapters, err := converters.ToChapters(doc)
	if err != nil {
		t.Fatalf("ToChapters: %v", err)
	}
	if len(chapters) != 2 {
		t.Fatalf("chapters = %+v", chapters)
	}
	first := chapters[0]
	if first.StartSeconds != 0 || first.Title != "Intro" || first.URL != "https://example.com/0" {
		t.Fatalf("first chapter = %+v", first)
	}
	if !first.TOC {
		t.Fatal("toc = false, want the default true")
	}
	second := chapters[1]
	if second.ImageURL != "https://example.com/i.png" || second.TOC {
		t.Fatalf("second chapter = %+v", second)
	}
	if second.EndSeconds == nil || *second.EndSeconds != end {
		t.Fatalf("end_seconds = %v", second.EndSeconds)
	}
}

func TestToChaptersInvalidVersion(t *testing.T) {
	for _, version := range []string{"", "2.0.0", "1", "1.2.0-rc1", "v1.2.0"} {
		t.Run(version, func(t *testing.T) {
			_, err := converters.ToChapters(v1.ChaptersDocument{Version: version})
			if !errors.Is(err, converters.ErrInvalidChaptersVersion) {
				t.Fatalf("err = %v, want ErrInvalidChaptersVersion", err)
			}
		})
	}
}

func TestToChaptersDocument(t *testing.T) {
	end := 30.5
	chapters := []domain.Chapter{
		{StartSeconds: 0, Title: "Intro", URL: "https://example.com/0", TOC: true},
		{StartSeconds: 15, Title: "Second", ImageURL: "https://example.com/i.png", EndSeconds: &end, TOC: false},
	}

	out := converters.ToChaptersDocument(chapters)

	if out.Version != "1.2.0" {
		t.Fatalf("version = %q", out.Version)
	}
	if len(out.Chapters) != 2 {
		t.Fatalf("chapters = %+v", out.Chapters)
	}
	if out.Chapters[0].StartTime != 0 || out.Chapters[0].Title != "Intro" || out.Chapters[0].URL != "https://example.com/0" {
		t.Fatalf("first chapter = %+v", out.Chapters[0])
	}
	if out.Chapters[0].TOC == nil || !*out.Chapters[0].TOC {
		t.Fatalf("toc = %v", out.Chapters[0].TOC)
	}
	if out.Chapters[1].Img != "https://example.com/i.png" || out.Chapters[1].EndTime == nil || *out.Chapters[1].EndTime != end {
		t.Fatalf("second chapter = %+v", out.Chapters[1])
	}
	if out.Chapters[1].TOC == nil || *out.Chapters[1].TOC {
		t.Fatalf("toc = %v, want false", out.Chapters[1].TOC)
	}
}

func TestToChaptersDocumentEmpty(t *testing.T) {
	out := converters.ToChaptersDocument(nil)

	if out.Chapters == nil || len(out.Chapters) != 0 {
		t.Fatalf("chapters = %+v, want an empty slice", out.Chapters)
	}
	if out.Version != "1.2.0" {
		t.Fatalf("version = %q", out.Version)
	}
}

func TestChaptersRoundTrip(t *testing.T) {
	end := 42.25
	original := []domain.Chapter{
		{StartSeconds: 0, Title: "Intro", TOC: true},
		{StartSeconds: 12.5, Title: "Body", URL: "https://example.com/body", ImageURL: "https://example.com/i.png", EndSeconds: &end, TOC: false},
	}

	back, err := converters.ToChapters(converters.ToChaptersDocument(original))
	if err != nil {
		t.Fatalf("ToChapters: %v", err)
	}
	if !reflect.DeepEqual(back, original) {
		t.Fatalf("round trip = %+v, want %+v", back, original)
	}
}
