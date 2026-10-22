package domain_test

import (
	"errors"
	"testing"

	"github.com/JLugagne/podcast/internal/listening/domain"
)

func TestParseEpisodeRef(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    domain.EpisodeRef
		wantErr error
	}{
		{
			name: "accepts a canonical uuid",
			raw:  "018f0000-0000-7000-8000-000000000001",
			want: domain.EpisodeRef("018f0000-0000-7000-8000-000000000001"),
		},
		{
			name: "canonicalizes the case",
			raw:  "018F0000-0000-7000-8000-000000000001",
			want: domain.EpisodeRef("018f0000-0000-7000-8000-000000000001"),
		},
		{
			name:    "rejects a non uuid",
			raw:     "episode-1",
			wantErr: domain.ErrInvalidID,
		},
		{
			name:    "rejects an empty value",
			raw:     "",
			wantErr: domain.ErrInvalidID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ParseEpisodeRef(tt.raw)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ParseEpisodeRef(%q) error = %v, want %v", tt.raw, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseEpisodeRef(%q) error = %v, want nil", tt.raw, err)
			}
			if got != tt.want {
				t.Errorf("ParseEpisodeRef(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestLocalListener(t *testing.T) {
	if domain.LocalListenerID != "local" {
		t.Errorf("LocalListenerID = %q, want %q", domain.LocalListenerID, "local")
	}
	actor := domain.LocalListener()
	if actor.ID() != "local" {
		t.Errorf("LocalListener().ID() = %q, want %q", actor.ID(), "local")
	}
	if actor.Kind() != domain.ActorKindUser {
		t.Errorf("LocalListener().Kind() = %q, want %q", actor.Kind(), domain.ActorKindUser)
	}
	var nilActor *domain.Actor
	if nilActor.ID() != "" || nilActor.Kind() != "" {
		t.Error("a nil actor should expose empty values")
	}
}
