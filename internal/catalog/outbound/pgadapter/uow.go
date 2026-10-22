package pgadapter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/episoderepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/repositories/seriesrepo"
	"github.com/JLugagne/podcast/internal/catalog/domain/uow"
)

// unitOfWork runs work in one PostgreSQL transaction.
type unitOfWork struct{ pool *pgxpool.Pool }

// NewUnitOfWork returns the catalog unit of work backed by pool.
func NewUnitOfWork(pool *pgxpool.Pool) uow.UnitOfWork { return &unitOfWork{pool: pool} }

func (u *unitOfWork) Do(ctx context.Context, fn func(ctx context.Context, r uow.Repositories) error) error {
	tx, err := u.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return wrap("begin transaction", err)
	}
	if err := fn(ctx, txRepositories{tx: tx}); err != nil {
		if rbErr := tx.Rollback(context.WithoutCancel(ctx)); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
			return errors.Join(err, wrap("rollback", rbErr))
		}
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return wrap("commit", err)
	}
	return nil
}

// txRepositories exposes the repositories bound to the running transaction.
type txRepositories struct{ tx pgx.Tx }

var _ uow.Repositories = txRepositories{}

func (r txRepositories) Series() seriesrepo.SeriesRepository { return NewSeriesRepository(r.tx) }

func (r txRepositories) Episodes() episoderepo.EpisodeRepository { return NewEpisodeRepository(r.tx) }
