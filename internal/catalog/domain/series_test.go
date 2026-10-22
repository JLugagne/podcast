package domain_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

var (
	seriesID = domain.SeriesID("018f0000-0000-7000-8000-000000000001")
	hourSpan = time.Hour
)

func fixedTime() time.Time {
	return time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
}

func TestNewSeries(t *testing.T) {
	now := fixedTime()

	t.Run("defaults and trims", func(t *testing.T) {
		s, err := domain.NewSeries(seriesID, domain.SeriesAttributes{Title: "  My Show  "}, now)
		if err != nil {
			t.Fatalf("NewSeries() error = %v", err)
		}
		if s.Title() != "My Show" {
			t.Fatalf("Title() = %q", s.Title())
		}
		if s.Slug() != "my-show" {
			t.Fatalf("Slug() = %q", s.Slug())
		}
		if s.AccentColor() != domain.DefaultAccentColor {
			t.Fatalf("AccentColor() = %q", s.AccentColor())
		}
		if s.Cover() != nil {
			t.Fatal("Cover() must be nil")
		}
		if s.Version() != 0 {
			t.Fatalf("Version() = %d", s.Version())
		}
		if !s.CreatedAt().Equal(now) || !s.UpdatedAt().Equal(now) {
			t.Fatalf("timestamps = %v / %v", s.CreatedAt(), s.UpdatedAt())
		}
	})

	t.Run("accepts a valid accent color", func(t *testing.T) {
		s, err := domain.NewSeries(seriesID, domain.SeriesAttributes{Title: "Show", AccentColor: "#a1B2c3"}, now)
		if err != nil {
			t.Fatalf("NewSeries() error = %v", err)
		}
		if s.AccentColor() != "#a1B2c3" {
			t.Fatalf("AccentColor() = %q", s.AccentColor())
		}
	})

	invalid := map[string]domain.SeriesAttributes{
		"empty title":          {Title: "   "},
		"title too long":       {Title: strings.Repeat("a", domain.MaxSeriesTitleLength+1)},
		"author too long":      {Title: "Show", Author: strings.Repeat("a", domain.MaxSeriesAuthorLength+1)},
		"description too long": {Title: "Show", Description: strings.Repeat("a", domain.MaxSeriesDescriptionLength+1)},
		"bad accent word":      {Title: "Show", AccentColor: "red"},
		"short accent":         {Title: "Show", AccentColor: "#12345"},
		"invalid hex":          {Title: "Show", AccentColor: "#GGGGGG"},
	}
	for name, attrs := range invalid {
		t.Run("rejects "+name, func(t *testing.T) {
			if _, err := domain.NewSeries(seriesID, attrs, now); !errors.Is(err, domain.ErrInvalidSeries) {
				t.Fatalf("NewSeries() error = %v, want ErrInvalidSeries", err)
			}
		})
	}
}

func TestSeriesCover(t *testing.T) {
	now := fixedTime()
	s, err := domain.NewSeries(seriesID, domain.SeriesAttributes{Title: "Show"}, now)
	if err != nil {
		t.Fatalf("NewSeries() error = %v", err)
	}

	cover := domain.Asset{Key: "series/1/cover-1.png", ContentType: "image/png", SizeBytes: 42}
	later := now.Add(hourSpan)
	s.SetCover(cover, later)
	if got := s.Cover(); got == nil || got.Key != cover.Key || got.SizeBytes != 42 {
		t.Fatalf("Cover() = %+v", got)
	}
	if !s.UpdatedAt().Equal(later) {
		t.Fatalf("UpdatedAt() = %v", s.UpdatedAt())
	}

	s.ClearCover(later.Add(hourSpan))
	if s.Cover() != nil {
		t.Fatal("Cover() must be nil after ClearCover")
	}
}

func TestReconstituteSeries(t *testing.T) {
	created := fixedTime()
	updated := created.Add(hourSpan)
	attrs := domain.SeriesAttributes{Title: "Show", Author: "Me", Description: "Desc", AccentColor: "#123456", Slug: "show"}
	cover := &domain.Asset{Key: "k", ContentType: "image/png", SizeBytes: 1}

	s := domain.ReconstituteSeries(seriesID, attrs, cover, created, updated, 7)
	if s.ID() != seriesID || s.Title() != "Show" || s.Slug() != "show" || s.AccentColor() != "#123456" {
		t.Fatalf("unexpected series: %+v", s.Attributes())
	}
	if s.Version() != 7 {
		t.Fatalf("Version() = %d", s.Version())
	}
	if !s.CreatedAt().Equal(created) || !s.UpdatedAt().Equal(updated) {
		t.Fatalf("timestamps = %v / %v", s.CreatedAt(), s.UpdatedAt())
	}
	if s.Cover() == nil || s.Cover().Key != "k" {
		t.Fatalf("Cover() = %+v", s.Cover())
	}
}
