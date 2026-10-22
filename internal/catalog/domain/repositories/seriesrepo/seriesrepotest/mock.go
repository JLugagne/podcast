// Package seriesrepotest provides function-based test doubles for the
// seriesrepo port. An unset func panics so unexpected calls surface.
package seriesrepotest

import (
	"context"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
)

// MockSeriesRepository is a function-field test double for
// seriesrepo.SeriesRepository.
type MockSeriesRepository struct {
	CreateFn    func(ctx context.Context, s *domain.Series) error
	UpdateFn    func(ctx context.Context, s *domain.Series) error
	GetFn       func(ctx context.Context, id domain.SeriesID) (*domain.Series, error)
	GetBySlugFn func(ctx context.Context, slug string) (*domain.Series, error)
	ListFn      func(ctx context.Context, f seriesrepo.Filter) ([]*domain.Series, error)
	DeleteFn    func(ctx context.Context, id domain.SeriesID) error
}

var _ seriesrepo.SeriesRepository = (*MockSeriesRepository)(nil)

// Create calls CreateFn, panicking when it is not set.
func (m *MockSeriesRepository) Create(ctx context.Context, s *domain.Series) error {
	if m.CreateFn == nil {
		panic("MockSeriesRepository.CreateFn not set")
	}
	return m.CreateFn(ctx, s)
}

// Update calls UpdateFn, panicking when it is not set.
func (m *MockSeriesRepository) Update(ctx context.Context, s *domain.Series) error {
	if m.UpdateFn == nil {
		panic("MockSeriesRepository.UpdateFn not set")
	}
	return m.UpdateFn(ctx, s)
}

// Get calls GetFn, panicking when it is not set.
func (m *MockSeriesRepository) Get(ctx context.Context, id domain.SeriesID) (*domain.Series, error) {
	if m.GetFn == nil {
		panic("MockSeriesRepository.GetFn not set")
	}
	return m.GetFn(ctx, id)
}

// GetBySlug calls GetBySlugFn, panicking when it is not set.
func (m *MockSeriesRepository) GetBySlug(ctx context.Context, slug string) (*domain.Series, error) {
	if m.GetBySlugFn == nil {
		panic("MockSeriesRepository.GetBySlugFn not set")
	}
	return m.GetBySlugFn(ctx, slug)
}

// List calls ListFn, panicking when it is not set.
func (m *MockSeriesRepository) List(ctx context.Context, f seriesrepo.Filter) ([]*domain.Series, error) {
	if m.ListFn == nil {
		panic("MockSeriesRepository.ListFn not set")
	}
	return m.ListFn(ctx, f)
}

// Delete calls DeleteFn, panicking when it is not set.
func (m *MockSeriesRepository) Delete(ctx context.Context, id domain.SeriesID) error {
	if m.DeleteFn == nil {
		panic("MockSeriesRepository.DeleteFn not set")
	}
	return m.DeleteFn(ctx, id)
}
