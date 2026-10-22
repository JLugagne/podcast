// Package uow declares the unit of work used for catalog writes spanning
// several repositories.
package uow

import (
	"context"

	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
)

// Repositories are bound to the running transaction.
type Repositories interface {
	Series() seriesrepo.SeriesRepository
	Episodes() episoderepo.EpisodeRepository
}

// UnitOfWork runs fn in one transaction: committed when fn returns nil,
// rolled back otherwise, and fn's error is returned unchanged.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, r Repositories) error) error
}
