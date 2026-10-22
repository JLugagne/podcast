package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

func floatPtr(v float64) *float64 { return &v }

func TestNewChapters(t *testing.T) {
	t.Run("sorts by start", func(t *testing.T) {
		got, err := domain.NewChapters([]domain.Chapter{
			{StartSeconds: 60, Title: "C"},
			{StartSeconds: 0, Title: "A"},
			{StartSeconds: 30, Title: "B", EndSeconds: floatPtr(60)},
		})
		if err != nil {
			t.Fatalf("NewChapters() error = %v", err)
		}
		want := []string{"A", "B", "C"}
		for i, title := range want {
			if got[i].Title != title {
				t.Fatalf("chapter %d title = %q, want %q", i, got[i].Title, title)
			}
		}
	})

	t.Run("does not mutate input", func(t *testing.T) {
		in := []domain.Chapter{{StartSeconds: 10, Title: "B"}, {StartSeconds: 0, Title: "A"}}
		if _, err := domain.NewChapters(in); err != nil {
			t.Fatalf("NewChapters() error = %v", err)
		}
		if in[0].Title != "B" || in[1].Title != "A" {
			t.Fatalf("input was reordered: %+v", in)
		}
	})

	invalid := map[string][]domain.Chapter{
		"empty":            nil,
		"first not zero":   {{StartSeconds: 5, Title: "A"}},
		"duplicate starts": {{StartSeconds: 0, Title: "A"}, {StartSeconds: 0, Title: "B"}},
		"negative start":   {{StartSeconds: -5, Title: "A"}},
		"empty title":      {{StartSeconds: 0, Title: "   "}},
		"end equals start": {{StartSeconds: 0, Title: "A", EndSeconds: floatPtr(0)}},
		"end before start": {{StartSeconds: 0, Title: "A"}, {StartSeconds: 10, Title: "B", EndSeconds: floatPtr(5)}},
		"title too long":   {{StartSeconds: 0, Title: strings.Repeat("a", domain.MaxChapterTitleLength+1)}},
	}
	for name, chapters := range invalid {
		t.Run("rejects "+name, func(t *testing.T) {
			if _, err := domain.NewChapters(chapters); !errors.Is(err, domain.ErrInvalidChapters) {
				t.Fatalf("NewChapters() error = %v, want ErrInvalidChapters", err)
			}
		})
	}

	t.Run("rejects too many chapters", func(t *testing.T) {
		chapters := make([]domain.Chapter, domain.MaxChapters+1)
		for i := range chapters {
			chapters[i] = domain.Chapter{StartSeconds: float64(i), Title: "Chapter"}
		}
		if _, err := domain.NewChapters(chapters); !errors.Is(err, domain.ErrInvalidChapters) {
			t.Fatalf("NewChapters() error = %v, want ErrInvalidChapters", err)
		}
	})

	t.Run("accepts the maximum", func(t *testing.T) {
		chapters := make([]domain.Chapter, domain.MaxChapters)
		for i := range chapters {
			chapters[i] = domain.Chapter{StartSeconds: float64(i), Title: "Chapter"}
		}
		if _, err := domain.NewChapters(chapters); err != nil {
			t.Fatalf("NewChapters() error = %v", err)
		}
	})
}
