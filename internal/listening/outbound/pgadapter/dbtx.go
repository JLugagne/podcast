// Package pgadapter implements the listening persistence port on PostgreSQL
// (pgx).
package pgadapter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the subset of the pgx API used by the repository. It is satisfied by
// *pgxpool.Pool and pgx.Tx.
type DBTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// wrap prefixes err with the failed operation, keeping it inspectable with
// errors.Is/As.
func wrap(op string, err error) error {
	return errors.Join(errors.New("pgadapter: "+op), err)
}

// mapNotFound translates a pgx.ErrNoRows scan into notFound, wraps any other
// error with op, and returns nil when err is nil.
func mapNotFound(err error, notFound error, op string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	if err != nil {
		return wrap(op, err)
	}
	return nil
}
