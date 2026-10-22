package domain_test

import (
	"errors"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/listening/domain"
)

var testNow = time.Date(2026, 10, 22, 10, 0, 0, 0, time.UTC)

const (
	testListener = domain.ListenerID("local")
	testEpisode  = domain.EpisodeRef("018f0000-0000-7000-8000-000000000001")
)

func TestNewProgress(t *testing.T) {
	p := domain.NewProgress(testListener, testEpisode, testNow)

	if p.ListenerID() != testListener {
		t.Errorf("ListenerID = %q, want %q", p.ListenerID(), testListener)
	}
	if p.EpisodeRef() != testEpisode {
		t.Errorf("EpisodeRef = %q, want %q", p.EpisodeRef(), testEpisode)
	}
	if p.PositionSeconds() != 0 {
		t.Errorf("PositionSeconds = %v, want 0", p.PositionSeconds())
	}
	if p.Completed() {
		t.Error("Completed = true, want false")
	}
	if !p.CreatedAt().Equal(testNow) || !p.UpdatedAt().Equal(testNow) {
		t.Errorf("timestamps = %v/%v, want %v", p.CreatedAt(), p.UpdatedAt(), testNow)
	}
}

func TestReconstituteProgress(t *testing.T) {
	created := testNow.Add(-time.Hour)
	p := domain.ReconstituteProgress(testListener, testEpisode, 42.5, true, created, testNow)

	if p.PositionSeconds() != 42.5 || !p.Completed() {
		t.Errorf("progress = %v/%v, want 42.5/true", p.PositionSeconds(), p.Completed())
	}
	if !p.CreatedAt().Equal(created) || !p.UpdatedAt().Equal(testNow) {
		t.Errorf("timestamps = %v/%v, want %v/%v", p.CreatedAt(), p.UpdatedAt(), created, testNow)
	}
}

func TestProgressRecord(t *testing.T) {
	tests := []struct {
		name           string
		startPosition  float64
		startCompleted bool
		position       float64
		duration       float64
		wantPosition   float64
		wantCompleted  bool
		wantErr        error
	}{
		{
			name:         "stores a mid-episode position",
			position:     12.5,
			duration:     60,
			wantPosition: 12.5,
		},
		{
			name:          "clamps to the duration and completes",
			position:      120,
			duration:      60,
			wantPosition:  60,
			wantCompleted: true,
		},
		{
			name:         "keeps the position when the duration is unknown",
			position:     120,
			duration:     0,
			wantPosition: 120,
		},
		{
			name:          "completes inside the two second window",
			position:      58.5,
			duration:      60,
			wantPosition:  58.5,
			wantCompleted: true,
		},
		{
			name:          "completes exactly on the window edge",
			position:      58,
			duration:      60,
			wantPosition:  58,
			wantCompleted: true,
		},
		{
			name:         "does not complete just before the window",
			position:     57.9,
			duration:     60,
			wantPosition: 57.9,
		},
		{
			name:           "reopens a completed episode",
			startPosition:  60,
			startCompleted: true,
			position:       30,
			duration:       60,
			wantPosition:   30,
		},
		{
			name:           "keeps a completed episode completed",
			startPosition:  60,
			startCompleted: true,
			position:       59,
			duration:       60,
			wantPosition:   59,
			wantCompleted:  true,
		},
		{
			name:     "rejects a negative position",
			position: -1,
			duration: 60,
			wantErr:  domain.ErrInvalidPosition,
		},
		{
			name:     "rejects a negative position without a duration",
			position: -0.5,
			wantErr:  domain.ErrInvalidPosition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := domain.ReconstituteProgress(testListener, testEpisode, tt.startPosition, tt.startCompleted, testNow.Add(-time.Hour), testNow)
			recorded := testNow.Add(time.Minute)

			err := p.Record(tt.position, tt.duration, recorded)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("Record error = %v, want %v", err, tt.wantErr)
				}
				if p.PositionSeconds() != tt.startPosition || p.Completed() != tt.startCompleted {
					t.Errorf("rejected Record changed the progress: %v/%v", p.PositionSeconds(), p.Completed())
				}
				if !p.UpdatedAt().Equal(testNow) {
					t.Errorf("rejected Record changed updatedAt to %v", p.UpdatedAt())
				}
				return
			}
			if err != nil {
				t.Fatalf("Record error = %v, want nil", err)
			}
			if p.PositionSeconds() != tt.wantPosition {
				t.Errorf("PositionSeconds = %v, want %v", p.PositionSeconds(), tt.wantPosition)
			}
			if p.Completed() != tt.wantCompleted {
				t.Errorf("Completed = %v, want %v", p.Completed(), tt.wantCompleted)
			}
			if !p.UpdatedAt().Equal(recorded) {
				t.Errorf("UpdatedAt = %v, want %v", p.UpdatedAt(), recorded)
			}
		})
	}
}

func TestProgressMarkCompleted(t *testing.T) {
	tests := []struct {
		name         string
		duration     float64
		wantPosition float64
	}{
		{name: "places the position at the end", duration: 95.5, wantPosition: 95.5},
		{name: "unknown duration becomes zero", duration: 0, wantPosition: 0},
		{name: "negative duration becomes zero", duration: -10, wantPosition: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := domain.NewProgress(testListener, testEpisode, testNow)
			marked := testNow.Add(time.Minute)

			p.MarkCompleted(tt.duration, marked)

			if p.PositionSeconds() != tt.wantPosition {
				t.Errorf("PositionSeconds = %v, want %v", p.PositionSeconds(), tt.wantPosition)
			}
			if !p.Completed() {
				t.Error("Completed = false, want true")
			}
			if !p.UpdatedAt().Equal(marked) {
				t.Errorf("UpdatedAt = %v, want %v", p.UpdatedAt(), marked)
			}
		})
	}
}

func TestProgressReset(t *testing.T) {
	p := domain.ReconstituteProgress(testListener, testEpisode, 60, true, testNow.Add(-time.Hour), testNow)
	reset := testNow.Add(time.Minute)

	p.Reset(reset)

	if p.PositionSeconds() != 0 {
		t.Errorf("PositionSeconds = %v, want 0", p.PositionSeconds())
	}
	if p.Completed() {
		t.Error("Completed = true, want false")
	}
	if !p.UpdatedAt().Equal(reset) {
		t.Errorf("UpdatedAt = %v, want %v", p.UpdatedAt(), reset)
	}
	if !p.CreatedAt().Equal(testNow.Add(-time.Hour)) {
		t.Errorf("CreatedAt = %v, want the original creation time", p.CreatedAt())
	}
}
