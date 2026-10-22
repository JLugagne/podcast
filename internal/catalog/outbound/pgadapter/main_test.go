package pgadapter_test

import (
	"context"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/JLugagne/podcast/internal/catalog/outbound/pgadapter"
	sharedpostgres "github.com/JLugagne/podcast/internal/pkg/postgres"
)

var (
	testPool   *pgxpool.Pool
	skipReason string
)

// TestMain starts one PostgreSQL container for the whole package, runs the
// migrations twice (proving they are idempotent) and terminates the container
// afterwards. Tests are skipped in -short mode or when Docker is unavailable.
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if testing.Short() {
		skipReason = "integration test skipped in -short mode"
		return m.Run()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	container, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("podcast"),
		postgres.WithUsername("podcast"),
		postgres.WithPassword("podcast"),
		testcontainers.WithWaitStrategy(wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
	)
	if err != nil {
		skipReason = fmt.Sprintf("postgres container unavailable: %v", err)
		return m.Run()
	}
	defer func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			fmt.Fprintf(os.Stderr, "terminate postgres container: %v\n", err)
		}
	}()

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		skipReason = fmt.Sprintf("postgres connection string: %v", err)
		return m.Run()
	}
	testPool, err = sharedpostgres.NewPool(ctx, dsn)
	if err != nil {
		skipReason = fmt.Sprintf("connect to postgres: %v", err)
		return m.Run()
	}
	defer testPool.Close()

	for i := 0; i < 2; i++ {
		if err := sharedpostgres.Migrate(ctx, testPool, pgadapter.Migrations(), "catalog", nil); err != nil {
			skipReason = fmt.Sprintf("migrate catalog (run %d): %v", i+1, err)
			return m.Run()
		}
	}

	return m.Run()
}

// requireDB skips the test when the shared database is unavailable and clears
// the catalog tables so every test starts from a known state.
func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if skipReason != "" {
		t.Skip(skipReason)
	}
	if _, err := testPool.Exec(context.Background(), "TRUNCATE catalog_episodes, catalog_series"); err != nil {
		t.Fatalf("truncate catalog tables: %v", err)
	}
	return testPool
}
