// Package postgres builds the PostgreSQL connection pool and runs goose
// migrations.
package postgres

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// connectRetryDelay is the pause between two connection attempts; it doubles
// on every failure up to connectMaxDelay.
const (
	connectRetryDelay = 500 * time.Millisecond
	connectMaxDelay   = 5 * time.Second
)

// NewPool opens a pgx pool on dsn and waits until the database answers or ctx
// expires. It retries with a capped exponential backoff so a compose stack can
// start the app before PostgreSQL is ready.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse dsn: %w", err)
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: create pool: %w", err)
	}

	delay := connectRetryDelay
	for {
		pingCtx, cancel := context.WithTimeout(ctx, connectMaxDelay)
		err := pool.Ping(pingCtx)
		cancel()
		if err == nil {
			return pool, nil
		}
		select {
		case <-ctx.Done():
			pool.Close()
			return nil, fmt.Errorf("postgres: connect: %w", err)
		case <-time.After(delay):
		}
		if delay < connectMaxDelay {
			delay *= 2
			if delay > connectMaxDelay {
				delay = connectMaxDelay
			}
		}
	}
}

// Migrate applies every pending migration of migrations with goose, tracking
// the version in the goose_<context>_version table so each bounded context
// migrates independently. It is idempotent.
func Migrate(ctx context.Context, pool *pgxpool.Pool, migrations fs.FS, context string, log *slog.Logger) error {
	table := "goose_" + context + "_version"
	db := stdlib.OpenDBFromPool(pool)
	defer func() { _ = db.Close() }()

	opts := []goose.ProviderOption{goose.WithTableName(table)}
	if log != nil {
		opts = append(opts, goose.WithSlog(log))
	}
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, opts...)
	if err != nil {
		return fmt.Errorf("postgres: create migration provider for %s: %w", context, err)
	}
	results, err := provider.Up(ctx)
	if err != nil {
		return fmt.Errorf("postgres: migrate %s: %w", context, err)
	}
	if log != nil {
		log.Info("postgres: migrations applied",
			slog.String("context", context),
			slog.Int("count", len(results)),
		)
	}
	return nil
}
