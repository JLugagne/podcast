package podcast

import (
	"context"
	"time"

	"github.com/spf13/cobra"

	"github.com/JLugagne/podcast/internal/catalog"
	"github.com/JLugagne/podcast/internal/listening"
	"github.com/JLugagne/podcast/internal/pkg/config"
	"github.com/JLugagne/podcast/internal/pkg/logger"
	"github.com/JLugagne/podcast/internal/pkg/postgres"
)

// migrateTimeout bounds the whole migration run.
const migrateTimeout = 2 * time.Minute

// newMigrateCommand builds the `migrate` subcommand.
func newMigrateCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "migrate",
		Short: "Apply the database migrations and exit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrate(cmd.Context())
		},
	}
}

// runMigrate applies every pending catalog and listening migration.
func runMigrate(parent context.Context) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	log, err := logger.New(cfg.Log.Format, cfg.Log.Level)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(parent, migrateTimeout)
	defer cancel()

	pool, err := postgres.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	if err := catalog.Migrate(ctx, pool, log); err != nil {
		return err
	}
	return listening.Migrate(ctx, pool, log)
}
