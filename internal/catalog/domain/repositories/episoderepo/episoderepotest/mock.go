// Package episoderepotest provides function-based test doubles for the
// episoderepo port. An unset func panics so unexpected calls surface.
package episoderepotest

import (
	"context"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
)

// MockEpisodeRepository is a function-field test double for
// episoderepo.EpisodeRepository.
type MockEpisodeRepository struct {
	CreateFn        func(ctx context.Context, e *domain.Episode) error
	UpdateFn        func(ctx context.Context, e *domain.Episode) error
	GetFn           func(ctx context.Context, id domain.EpisodeID) (*domain.Episode, error)
	DeleteFn        func(ctx context.Context, id domain.EpisodeID) error
	ListBySeriesFn  func(ctx context.Context, seriesID domain.SeriesID, f episoderepo.Filter) ([]*domain.Episode, error)
	CountBySeriesFn func(ctx context.Context, seriesID domain.SeriesID) (int, error)
	NextNumberFn    func(ctx context.Context, seriesID domain.SeriesID) (int, error)
}

var _ episoderepo.EpisodeRepository = (*MockEpisodeRepository)(nil)

// Create calls CreateFn, panicking when it is not set.
func (m *MockEpisodeRepository) Create(ctx context.Context, e *domain.Episode) error {
	if m.CreateFn == nil {
		panic("MockEpisodeRepository.CreateFn not set")
	}
	return m.CreateFn(ctx, e)
}

// Update calls UpdateFn, panicking when it is not set.
func (m *MockEpisodeRepository) Update(ctx context.Context, e *domain.Episode) error {
	if m.UpdateFn == nil {
		panic("MockEpisodeRepository.UpdateFn not set")
	}
	return m.UpdateFn(ctx, e)
}

// Get calls GetFn, panicking when it is not set.
func (m *MockEpisodeRepository) Get(ctx context.Context, id domain.EpisodeID) (*domain.Episode, error) {
	if m.GetFn == nil {
		panic("MockEpisodeRepository.GetFn not set")
	}
	return m.GetFn(ctx, id)
}

// Delete calls DeleteFn, panicking when it is not set.
func (m *MockEpisodeRepository) Delete(ctx context.Context, id domain.EpisodeID) error {
	if m.DeleteFn == nil {
		panic("MockEpisodeRepository.DeleteFn not set")
	}
	return m.DeleteFn(ctx, id)
}

// ListBySeries calls ListBySeriesFn, panicking when it is not set.
func (m *MockEpisodeRepository) ListBySeries(ctx context.Context, seriesID domain.SeriesID, f episoderepo.Filter) ([]*domain.Episode, error) {
	if m.ListBySeriesFn == nil {
		panic("MockEpisodeRepository.ListBySeriesFn not set")
	}
	return m.ListBySeriesFn(ctx, seriesID, f)
}

// CountBySeries calls CountBySeriesFn, panicking when it is not set.
func (m *MockEpisodeRepository) CountBySeries(ctx context.Context, seriesID domain.SeriesID) (int, error) {
	if m.CountBySeriesFn == nil {
		panic("MockEpisodeRepository.CountBySeriesFn not set")
	}
	return m.CountBySeriesFn(ctx, seriesID)
}

// NextNumber calls NextNumberFn, panicking when it is not set.
func (m *MockEpisodeRepository) NextNumber(ctx context.Context, seriesID domain.SeriesID) (int, error) {
	if m.NextNumberFn == nil {
		panic("MockEpisodeRepository.NextNumberFn not set")
	}
	return m.NextNumberFn(ctx, seriesID)
}
