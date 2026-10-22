package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name  string
		title string
		want  string
	}{
		{"simple words", "My Great Show", "my-great-show"},
		{"keeps digits", "Show 42", "show-42"},
		{"collapses punctuation runs", "  --Hello,  World!--  ", "hello-world"},
		{"strips accents", "Cafe Deja Vu", "cafe-deja-vu"},
		{"non ascii becomes separator", "Café Déjà Vu", "caf-d-j-vu"},
		{"only separators", "!!!", ""},
		{"empty", "", ""},
		{"truncates long titles", strings.Repeat("a", 130), strings.Repeat("a", domain.MaxSlugLength)},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := domain.Slugify(tc.title); got != tc.want {
				t.Fatalf("Slugify(%q) = %q, want %q", tc.title, got, tc.want)
			}
		})
	}
}

func TestNewSeriesSlug(t *testing.T) {
	now := fixedTime()

	t.Run("derives from title", func(t *testing.T) {
		s, err := domain.NewSeries(seriesID, domain.SeriesAttributes{Title: "The Daily Show"}, now)
		if err != nil {
			t.Fatalf("NewSeries() error = %v", err)
		}
		if s.Slug() != "the-daily-show" {
			t.Fatalf("Slug() = %q", s.Slug())
		}
	})

	t.Run("keeps explicit slug", func(t *testing.T) {
		s, err := domain.NewSeries(seriesID, domain.SeriesAttributes{Title: "The Daily Show", Slug: "daily"}, now)
		if err != nil {
			t.Fatalf("NewSeries() error = %v", err)
		}
		if s.Slug() != "daily" {
			t.Fatalf("Slug() = %q", s.Slug())
		}
	})

	invalid := map[string]string{
		"uppercase":        "My-Slug",
		"leading dash":     "-slug",
		"trailing dash":    "slug-",
		"double dash":      "my--slug",
		"space":            "my slug",
		"underscore":       "my_slug",
		"too long":         strings.Repeat("a", domain.MaxSlugLength+1),
		"empty from title": "!!!",
	}
	for name, slug := range invalid {
		t.Run("rejects "+name, func(t *testing.T) {
			_, err := domain.NewSeries(seriesID, domain.SeriesAttributes{Title: "Title", Slug: slug}, now)
			if !errors.Is(err, domain.ErrInvalidSeries) {
				t.Fatalf("NewSeries(slug=%q) error = %v, want ErrInvalidSeries", slug, err)
			}
		})
	}
}

func TestSeriesUpdateKeepsSlug(t *testing.T) {
	now := fixedTime()
	s, err := domain.NewSeries(seriesID, domain.SeriesAttributes{Title: "First Title"}, now)
	if err != nil {
		t.Fatalf("NewSeries() error = %v", err)
	}
	later := now.Add(hourSpan)
	if err := s.Update(domain.SeriesAttributes{Title: "Second Title"}, later); err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if s.Title() != "Second Title" {
		t.Fatalf("Title() = %q", s.Title())
	}
	if s.Slug() != "first-title" {
		t.Fatalf("Slug() = %q, want it preserved", s.Slug())
	}
	if !s.UpdatedAt().Equal(later) {
		t.Fatalf("UpdatedAt() = %v, want %v", s.UpdatedAt(), later)
	}
}
