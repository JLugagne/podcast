package pgadapter

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DBTX is the subset of the pgx API used by the repositories. It is satisfied
// by *pgxpool.Pool and pgx.Tx, so the same constructors build both standalone
// repositories and transaction-bound ones.
type DBTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PostgreSQL error codes handled by the adapter.
const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
	restrictViolation   = "23001"
)

// Named constraints the adapter maps to domain errors.
const (
	seriesSlugConstraint      = "catalog_series_slug_key"
	episodeNumberConstraint   = "catalog_episodes_series_number_key"
	episodeSeriesFKConstraint = "catalog_episodes_series_id_fkey"
)

// pgError extracts the SQLSTATE code and constraint name of a PostgreSQL error.
func pgError(err error) (code, constraint string) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code, pgErr.ConstraintName
	}
	return "", ""
}

// wrap prefixes err with the failed operation, keeping it inspectable with
// errors.Is/As.
func wrap(op string, err error) error {
	return errors.Join(errors.New("pgadapter: "+op), err)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func derefInt64(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

func derefFloat64(v *float64) float64 {
	if v == nil {
		return 0
	}
	return *v
}
