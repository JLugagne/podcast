package pgadapter_test

import (
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

var testNow = time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)

func seriesAttrs(slug, title string) domain.SeriesAttributes {
	return domain.SeriesAttributes{
		Title:       title,
		Author:      "Ada Lovelace",
		Description: "A show about numbers",
		AccentColor: "#123456",
		Slug:        slug,
	}
}

func mustSeriesAt(t *testing.T, slug, title string, at time.Time) *domain.Series {
	t.Helper()
	s, err := domain.NewSeries(domain.SeriesID(domain.NewID()), seriesAttrs(slug, title), at)
	if err != nil {
		t.Fatalf("NewSeries(%q): %v", slug, err)
	}
	return s
}

func mustEpisodeAt(t *testing.T, seriesID domain.SeriesID, number int, title string, at time.Time) *domain.Episode {
	t.Helper()
	e, err := domain.NewEpisode(domain.EpisodeID(domain.NewID()), seriesID, domain.EpisodeAttributes{
		Number:      number,
		Title:       title,
		Description: "Episode description",
	}, at)
	if err != nil {
		t.Fatalf("NewEpisode(%d): %v", number, err)
	}
	return e
}

func float64Ptr(v float64) *float64 { return &v }
