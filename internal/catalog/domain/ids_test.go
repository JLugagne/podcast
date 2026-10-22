package domain_test

import (
	"errors"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
)

func TestParseIDs(t *testing.T) {
	t.Run("series id", func(t *testing.T) {
		id, err := domain.ParseSeriesID("018f0000-0000-7000-8000-000000000001")
		if err != nil {
			t.Fatalf("ParseSeriesID() error = %v", err)
		}
		if id != domain.SeriesID("018f0000-0000-7000-8000-000000000001") {
			t.Fatalf("ParseSeriesID() = %q", id)
		}
	})

	t.Run("canonicalizes", func(t *testing.T) {
		id, err := domain.ParseEpisodeID("018F0000-0000-7000-8000-00000000000A")
		if err != nil {
			t.Fatalf("ParseEpisodeID() error = %v", err)
		}
		if string(id) != "018f0000-0000-7000-8000-00000000000a" {
			t.Fatalf("ParseEpisodeID() = %q", id)
		}
	})

	tests := map[string]string{
		"empty":     "",
		"garbage":   "not-a-uuid",
		"truncated": "018f0000-0000-7000-8000",
	}
	for name, raw := range tests {
		t.Run("invalid "+name, func(t *testing.T) {
			if _, err := domain.ParseSeriesID(raw); !errors.Is(err, domain.ErrInvalidID) {
				t.Fatalf("ParseSeriesID(%q) error = %v, want ErrInvalidID", raw, err)
			}
			if _, err := domain.ParseEpisodeID(raw); !errors.Is(err, domain.ErrInvalidID) {
				t.Fatalf("ParseEpisodeID(%q) error = %v, want ErrInvalidID", raw, err)
			}
		})
	}
}

func TestNewIDIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 100)
	for range 100 {
		id := domain.NewID()
		if _, err := domain.ParseSeriesID(id); err != nil {
			t.Fatalf("NewID() = %q is not a valid uuid: %v", id, err)
		}
		if _, ok := seen[id]; ok {
			t.Fatalf("NewID() returned duplicate %q", id)
		}
		seen[id] = struct{}{}
	}
}

func TestLocalAdmin(t *testing.T) {
	actor := domain.LocalAdmin()
	if actor.ID() != domain.LocalAdminID {
		t.Fatalf("LocalAdmin().ID() = %q", actor.ID())
	}
	if actor.Kind() != domain.ActorKindUser {
		t.Fatalf("LocalAdmin().Kind() = %q", actor.Kind())
	}
	var nilActor *domain.Actor
	if nilActor.ID() != "" || nilActor.Kind() != "" {
		t.Fatal("nil actor getters must return empty values")
	}
}
