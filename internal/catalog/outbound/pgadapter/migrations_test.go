package pgadapter_test

import (
	"context"
	"testing"

	"github.com/JLugagne/podcast/internal/catalog/outbound/pgadapter"
	sharedpostgres "github.com/JLugagne/podcast/internal/pkg/postgres"
)

func TestMigrationsAreIdempotent(t *testing.T) {
	pool := requireDB(t)
	ctx := context.Background()

	if err := sharedpostgres.Migrate(ctx, pool, pgadapter.Migrations(), "catalog", nil); err != nil {
		t.Fatalf("Migrate on an already migrated database: %v", err)
	}

	var count int
	if err := pool.QueryRow(ctx, `SELECT COUNT(*) FROM information_schema.tables
		WHERE table_schema = 'public'
		AND table_name IN ('catalog_series', 'catalog_episodes', 'goose_catalog_version')`).Scan(&count); err != nil {
		t.Fatalf("count catalog tables: %v", err)
	}
	if count != 3 {
		t.Fatalf("found %d of the 3 expected tables", count)
	}
}
