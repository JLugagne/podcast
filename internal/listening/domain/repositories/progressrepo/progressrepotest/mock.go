// Package progressrepotest provides function-based test doubles for the
// progressrepo port. An unset func panics so unexpected calls surface.
package progressrepotest

import (
	"context"

	"github.com/JLugagne/podcast/internal/listening/domain"
	"github.com/JLugagne/podcast/internal/listening/domain/repositories/progressrepo"
)

// MockProgressRepository is a function-field double of
// progressrepo.ProgressRepository.
type MockProgressRepository struct {
	GetFn    func(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) (*domain.Progress, error)
	SaveFn   func(ctx context.Context, progress *domain.Progress) error
	DeleteFn func(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) error
	ListFn   func(ctx context.Context, listener domain.ListenerID, filter progressrepo.Filter) ([]*domain.Progress, error)
}

var _ progressrepo.ProgressRepository = (*MockProgressRepository)(nil)

// Get calls GetFn.
func (m *MockProgressRepository) Get(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) (*domain.Progress, error) {
	if m.GetFn == nil {
		panic("MockProgressRepository.GetFn not set")
	}
	return m.GetFn(ctx, listener, episode)
}

// Save calls SaveFn.
func (m *MockProgressRepository) Save(ctx context.Context, progress *domain.Progress) error {
	if m.SaveFn == nil {
		panic("MockProgressRepository.SaveFn not set")
	}
	return m.SaveFn(ctx, progress)
}

// Delete calls DeleteFn.
func (m *MockProgressRepository) Delete(ctx context.Context, listener domain.ListenerID, episode domain.EpisodeRef) error {
	if m.DeleteFn == nil {
		panic("MockProgressRepository.DeleteFn not set")
	}
	return m.DeleteFn(ctx, listener, episode)
}

// List calls ListFn.
func (m *MockProgressRepository) List(ctx context.Context, listener domain.ListenerID, filter progressrepo.Filter) ([]*domain.Progress, error) {
	if m.ListFn == nil {
		panic("MockProgressRepository.ListFn not set")
	}
	return m.ListFn(ctx, listener, filter)
}
