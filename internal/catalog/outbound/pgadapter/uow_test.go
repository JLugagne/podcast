package pgadapter_test

import (
	"context"
	"errors"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/domain"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
	"github.com/JLugagne/podcast/internal/catalog/outbound/pgadapter"
)

func TestUnitOfWorkCommit(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	unit := pgadapter.NewUnitOfWork(pool)

	s := mustSeriesAt(t, "my-show", "My Show", testNow)
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)

	if err := unit.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		if err := r.Series().Create(ctx, s); err != nil {
			return err
		}
		return r.Episodes().Create(ctx, e)
	}); err != nil {
		t.Fatalf("Do: %v", err)
	}

	if _, err := pgadapter.NewSeriesRepository(pool).Get(ctx, s.ID()); err != nil {
		t.Errorf("series after commit: %v", err)
	}
	count, err := pgadapter.NewEpisodeRepository(pool).CountBySeries(ctx, s.ID())
	if err != nil {
		t.Fatalf("CountBySeries after commit: %v", err)
	}
	if count != 1 {
		t.Errorf("episode count after commit = %d, want 1", count)
	}
}

func TestUnitOfWorkRollback(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	unit := pgadapter.NewUnitOfWork(pool)

	s := mustSeriesAt(t, "my-show", "My Show", testNow)
	e := mustEpisodeAt(t, s.ID(), 1, "Pilot", testNow)
	sentinel := errors.New("rollback requested")

	err := unit.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		if err := r.Series().Create(ctx, s); err != nil {
			return err
		}
		if err := r.Episodes().Create(ctx, e); err != nil {
			return err
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Do error = %v, want the callback error unchanged", err)
	}

	if _, err := pgadapter.NewSeriesRepository(pool).Get(ctx, s.ID()); !errors.Is(err, domain.ErrSeriesNotFound) {
		t.Errorf("series after rollback = %v, want ErrSeriesNotFound", err)
	}
	count, err := pgadapter.NewEpisodeRepository(pool).CountBySeries(ctx, s.ID())
	if err != nil {
		t.Fatalf("CountBySeries after rollback: %v", err)
	}
	if count != 0 {
		t.Errorf("episode count after rollback = %d, want 0", count)
	}
}

func TestUnitOfWorkPropagatesRepositoryErrors(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()
	unit := pgadapter.NewUnitOfWork(pool)

	s := mustSeriesAt(t, "my-show", "My Show", testNow)
	sentinel := errors.New("stop here")
	err := unit.Do(ctx, func(ctx context.Context, r uow.Repositories) error {
		if err := r.Series().Create(ctx, s); err != nil {
			return err
		}
		_, err := r.Series().Get(ctx, domain.SeriesID(domain.NewID()))
		if !errors.Is(err, domain.ErrSeriesNotFound) {
			t.Fatalf("Get missing inside transaction = %v, want ErrSeriesNotFound", err)
		}
		return sentinel
	})
	if !errors.Is(err, sentinel) {
		t.Fatalf("Do error = %v, want %v", err, sentinel)
	}
}
