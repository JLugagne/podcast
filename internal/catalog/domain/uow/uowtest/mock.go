// Package uowtest provides function-based test doubles for the uow ports. An
// unset func panics so unexpected calls surface.
package uowtest

import (
	"context"

	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
)

// MockUnitOfWork is a function-field test double for uow.UnitOfWork. The usual
// DoFn runs the callback against MockRepositories, which simulates a commit.
type MockUnitOfWork struct {
	DoFn func(ctx context.Context, fn func(ctx context.Context, r uow.Repositories) error) error
}

var _ uow.UnitOfWork = (*MockUnitOfWork)(nil)

// Do calls DoFn, panicking when it is not set.
func (m *MockUnitOfWork) Do(ctx context.Context, fn func(ctx context.Context, r uow.Repositories) error) error {
	if m.DoFn == nil {
		panic("MockUnitOfWork.DoFn not set")
	}
	return m.DoFn(ctx, fn)
}
