// Package podcast runs the podcast catalog API and its admin UI. It is the only
// entry point of the service: Execute dispatches the cobra subcommands.
package podcast

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// configPath is bound to the persistent --config flag.
var configPath string

// newRootCommand builds the command tree.
func newRootCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "podcast",
		Short:         "Podcast catalog API and admin UI",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&configPath, "config", "", "path to an optional YAML configuration file")
	root.AddCommand(newServeCommand(), newMigrateCommand())
	return root
}

// Execute runs the CLI and exits with status 1 on failure.
func Execute() {
	if err := newRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
