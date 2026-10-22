package uowtest

import (
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
)

// MockRepositories is a function-field test double for uow.Repositories.
type MockRepositories struct {
	SeriesFn   func() seriesrepo.SeriesRepository
	EpisodesFn func() episoderepo.EpisodeRepository
}

var _ uow.Repositories = (*MockRepositories)(nil)

// Series calls SeriesFn, panicking when it is not set.
func (m *MockRepositories) Series() seriesrepo.SeriesRepository {
	if m.SeriesFn == nil {
		panic("MockRepositories.SeriesFn not set")
	}
	return m.SeriesFn()
}

// Episodes calls EpisodesFn, panicking when it is not set.
func (m *MockRepositories) Episodes() episoderepo.EpisodeRepository {
	if m.EpisodesFn == nil {
		panic("MockRepositories.EpisodesFn not set")
	}
	return m.EpisodesFn()
}
