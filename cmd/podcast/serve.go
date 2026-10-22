package podcast

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/JLugagne/podcast/internal/pkg/config"
	"github.com/JLugagne/podcast/internal/pkg/httpserver"
	"github.com/JLugagne/podcast/internal/pkg/logger"
	"github.com/JLugagne/podcast/internal/pkg/postgres"
)

// newServeCommand builds the `serve` subcommand.
func newServeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API, the listener and the admin UI",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runServe(cmd.Context())
		},
	}
}

// runServe loads the configuration and serves until SIGINT or SIGTERM.
func runServe(parent context.Context) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	log, err := logger.New(cfg.Log.Format, cfg.Log.Level)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(parent, os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.Database.URL)
	if err != nil {
		return err
	}
	defer pool.Close()

	handler, err := NewHandler(HandlerOptions{
		Ctx:           ctx,
		Pool:          pool,
		StorageDir:    cfg.Storage.Dir,
		MaxAudioBytes: cfg.Upload.MaxAudioBytes,
		MaxImageBytes: cfg.Upload.MaxImageBytes,
		PublicURL:     cfg.HTTP.PublicURL,
		Logger:        log,
	})
	if err != nil {
		return err
	}
	return httpserver.New(cfg.HTTP.Addr, handler, log).Run(ctx)
}
