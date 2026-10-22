package e2eapi_test

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

	"github.com/JLugagne/podcast/internal/catalog"
	"github.com/JLugagne/podcast/internal/listening"
	sharedpostgres "github.com/JLugagne/podcast/internal/pkg/postgres"
)

var (
	// testPool is the one PostgreSQL pool shared by every test of the package.
	testPool *pgxpool.Pool
	// skipReason is set when the container or the connection could not be
	// established; every test then skips with that reason.
	skipReason string
)

// TestMain starts one PostgreSQL container for the whole package and terminates
// it afterwards. The migrations themselves are exercised by the tests through
// catalog.New and listening.New. Tests are skipped in -short mode or
// when Docker is unavailable.
func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(run(m))
}

func run(m *testing.M) int {
	if testing.Short() {
		skipReason = "e2e test skipped in -short mode"
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

	// Create the schema once so requireDB can truncate it before every test.
	if err := catalog.Migrate(ctx, testPool, nil); err != nil {
		skipReason = fmt.Sprintf("migrate catalog: %v", err)
		return m.Run()
	}
	if err := listening.Migrate(ctx, testPool, nil); err != nil {
		skipReason = fmt.Sprintf("migrate listening: %v", err)
		return m.Run()
	}

	return m.Run()
}
