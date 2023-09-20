package cmd

import (
	"context"
	"fmt"
	"os"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/server"
	"git.maronato.dev/maronato/finger/internal/webfinger"
	"github.com/peterbourgon/ff/v4"
)

const appName = "finger"

func newServerCmd(cfg *config.Config) *ff.Command {
	return &ff.Command{
		Name:      "serve",
		Usage:     "serve [flags]",
		ShortHelp: "Start the webfinger server",
		Exec: func(ctx context.Context, args []string) error {
			// Create a logger and add it to the context
			l := log.NewLogger(os.Stderr, cfg)
			ctx = log.WithLogger(ctx, l)

			// Read the webfinger files
			r := webfinger.NewFingerReader()
			err := r.ReadFiles(cfg)
			if err != nil {
				return fmt.Errorf("error reading finger files: %w", err)
			}

			webfingers, err := r.ReadFingerFile(ctx)
			if err != nil {
				return fmt.Errorf("error parsing finger files: %w", err)
			}

			l.Info(fmt.Sprintf("Loaded %d webfingers", len(webfingers)))

			// Start the server
			if err := server.StartServer(ctx, cfg, webfingers); err != nil {
				return fmt.Errorf("error running server: %w", err)
			}

			return nil
		},
	}
}
