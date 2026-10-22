package pgadapter_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/repositories/progressrepo"
	"github.com/JLugagne/podcast/internal/listening/outbound/pgadapter"
)

var base = time.Date(2026, 10, 22, 12, 0, 0, 0, time.UTC)

const (
	listener      = domain.ListenerID("local")
	otherListener = domain.ListenerID("other")
	episodeA      = domain.EpisodeRef("018f0000-0000-7000-8000-000000000001")
	episodeB      = domain.EpisodeRef("018f0000-0000-7000-8000-000000000002")
	episodeC      = domain.EpisodeRef("018f0000-0000-7000-8000-000000000003")
	episodeD      = domain.EpisodeRef("018f0000-0000-7000-8000-000000000004")
)

// newRepository returns a repository on the shared database with a clean table.
func newRepository(t *testing.T) progressrepo.ProgressRepository {
	t.Helper()
	return pgadapter.NewProgressRepository(requireDB(t))
}

// storedProgress builds a progress row with base as creation time.
func storedProgress(listener domain.ListenerID, episode domain.EpisodeRef, position float64, completed bool, updated time.Time) *domain.Progress {
	return domain.ReconstituteProgress(listener, episode, position, completed, base, updated)
}

func TestSaveAndGet(t *testing.T) {
	repository := newRepository(t)
	want := storedProgress(listener, episodeA, 12.5, false, base.Add(time.Minute))

	if err := repository.Save(context.Background(), want); err != nil {
		t.Fatalf("Save error = %v, want nil", err)
	}
	got, err := repository.Get(context.Background(), listener, episodeA)
	if err != nil {
		t.Fatalf("Get error = %v, want nil", err)
	}

	if got.ListenerID() != want.ListenerID() || got.EpisodeRef() != want.EpisodeRef() {
		t.Errorf("identity = %q/%q, want %q/%q", got.ListenerID(), got.EpisodeRef(), want.ListenerID(), want.EpisodeRef())
	}
	if got.PositionSeconds() != want.PositionSeconds() {
		t.Errorf("PositionSeconds = %v, want %v", got.PositionSeconds(), want.PositionSeconds())
	}
	if got.Completed() != want.Completed() {
		t.Errorf("Completed = %v, want %v", got.Completed(), want.Completed())
	}
	if !got.CreatedAt().Equal(want.CreatedAt()) || !got.UpdatedAt().Equal(want.UpdatedAt()) {
		t.Errorf("timestamps = %v/%v, want %v/%v", got.CreatedAt(), got.UpdatedAt(), want.CreatedAt(), want.UpdatedAt())
	}
}

func TestSaveUpsertsAndKeepsCreatedAt(t *testing.T) {
	repository := newRepository(t)
	ctx := context.Background()

	first := storedProgress(listener, episodeA, 10, false, base.Add(time.Minute))
	if err := repository.Save(ctx, first); err != nil {
		t.Fatalf("first Save error = %v, want nil", err)
	}
	second := storedProgress(listener, episodeA, 60, true, base.Add(2*time.Minute))
	if err := repository.Save(ctx, second); err != nil {
		t.Fatalf("second Save error = %v, want nil", err)
	}

	got, err := repository.Get(ctx, listener, episodeA)
	if err != nil {
		t.Fatalf("Get error = %v, want nil", err)
	}
	if got.PositionSeconds() != 60 || !got.Completed() {
		t.Errorf("progress = %v/%v, want 60/true", got.PositionSeconds(), got.Completed())
	}
	if !got.CreatedAt().Equal(base) {
		t.Errorf("CreatedAt = %v, want the original creation time %v", got.CreatedAt(), base)
	}
	if !got.UpdatedAt().Equal(base.Add(2 * time.Minute)) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt(), base.Add(2*time.Minute))
	}
}

func TestGetNotFound(t *testing.T) {
	repository := newRepository(t)

	_, err := repository.Get(context.Background(), listener, episodeA)
	if !errors.Is(err, domain.ErrProgressNotFound) {
		t.Fatalf("Get error = %v, want %v", err, domain.ErrProgressNotFound)
	}
}

func TestDeleteIsIdempotent(t *testing.T) {
	repository := newRepository(t)
	ctx := context.Background()

	if err := repository.Save(ctx, storedProgress(listener, episodeA, 10, false, base)); err != nil {
		t.Fatalf("Save error = %v, want nil", err)
	}
	if err := repository.Delete(ctx, listener, episodeA); err != nil {
		t.Fatalf("Delete error = %v, want nil", err)
	}
	if _, err := repository.Get(ctx, listener, episodeA); !errors.Is(err, domain.ErrProgressNotFound) {
		t.Fatalf("Get after Delete error = %v, want %v", err, domain.ErrProgressNotFound)
	}
	if err := repository.Delete(ctx, listener, episodeA); err != nil {
		t.Fatalf("second Delete error = %v, want nil", err)
	}
}

func TestListOrdersFiltersAndLimits(t *testing.T) {
	repository := newRepository(t)
	ctx := context.Background()

	rows := []*domain.Progress{
		storedProgress(listener, episodeA, 10, false, base.Add(3*time.Minute)),
		storedProgress(listener, episodeB, 20, false, base.Add(2*time.Minute)),
		storedProgress(listener, episodeC, 60, true, base.Add(4*time.Minute)),
		storedProgress(listener, episodeD, 0, false, base.Add(time.Minute)),
	}
	for _, row := range rows {
		if err := repository.Save(ctx, row); err != nil {
			t.Fatalf("Save %q error = %v, want nil", row.EpisodeRef(), err)
		}
	}

	tests := []struct {
		name   string
		filter progressrepo.Filter
		want   []domain.EpisodeRef
	}{
		{
			name:   "all rows newest first",
			filter: progressrepo.Filter{},
			want:   []domain.EpisodeRef{episodeC, episodeA, episodeB, episodeD},
		},
		{
			name:   "in progress only",
			filter: progressrepo.Filter{InProgressOnly: true},
			want:   []domain.EpisodeRef{episodeA, episodeB},
		},
		{
			name:   "in progress limited",
			filter: progressrepo.Filter{InProgressOnly: true, Limit: 1},
			want:   []domain.EpisodeRef{episodeA},
		},
		{
			name:   "all limited",
			filter: progressrepo.Filter{Limit: 2},
			want:   []domain.EpisodeRef{episodeC, episodeA},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := repository.List(ctx, listener, tt.filter)
			if err != nil {
				t.Fatalf("List error = %v, want nil", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("List returned %d rows, want %d", len(got), len(tt.want))
			}
			for i, ref := range tt.want {
				if got[i].EpisodeRef() != ref {
					t.Errorf("row %d = %q, want %q", i, got[i].EpisodeRef(), ref)
				}
			}
		})
	}
}

func TestListKeepsListenersApart(t *testing.T) {
	repository := newRepository(t)
	ctx := context.Background()

	for _, row := range []*domain.Progress{
		storedProgress(listener, episodeA, 10, false, base),
		storedProgress(otherListener, episodeB, 20, false, base.Add(time.Minute)),
	} {
		if err := repository.Save(ctx, row); err != nil {
			t.Fatalf("Save %q error = %v, want nil", row.EpisodeRef(), err)
		}
	}

	got, err := repository.List(ctx, otherListener, progressrepo.Filter{})
	if err != nil {
		t.Fatalf("List error = %v, want nil", err)
	}
	if len(got) != 1 || got[0].EpisodeRef() != episodeB || got[0].ListenerID() != otherListener {
		t.Fatalf("List for %q = %v, want only %q", otherListener, got, episodeB)
	}
}
