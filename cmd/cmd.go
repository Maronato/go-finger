package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"git.maronato.dev/maronato/finger/internal/config"
	"github.com/peterbourgon/ff/v4"
	"github.com/peterbourgon/ff/v4/ffhelp"
)

func Run(version string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Allow graceful shutdown
	trapSignalsCrossPlatform(cancel)

	cfg := &config.Config{}

	// Create a new root command
	subcommands := []*ff.Command{
		newServerCmd(cfg),
		newHealthcheckCmd(cfg),
	}
	cmd := newRootCmd(version, cfg, subcommands)

	// Parse and run
	if err := cmd.ParseAndRun(ctx, os.Args[1:], ff.WithEnvVarPrefix("WF")); err != nil {
		if errors.Is(err, ff.ErrHelp) || errors.Is(err, ff.ErrNoExec) {
			fmt.Fprintf(os.Stderr, "\n%s\n", ffhelp.Command(cmd))

			return nil
		}

		return fmt.Errorf("error running command: %w", err)
	}

	return nil
}

// https://github.com/caddyserver/caddy/blob/fbb0ecfa322aa7710a3448453fd3ae40f037b8d1/sigtrap.go#L37
// trapSignalsCrossPlatform captures SIGINT or interrupt (depending
// on the OS), which initiates a graceful shutdown. A second SIGINT
// or interrupt will forcefully exit the process immediately.
func trapSignalsCrossPlatform(cancel context.CancelFunc) {
	go func() {
		shutdown := make(chan os.Signal, 1)
		signal.Notify(shutdown, os.Interrupt, syscall.SIGINT)

		for i := 0; true; i++ {
			<-shutdown

			if i > 0 {
				fmt.Printf("\nForce quit\n") //nolint:forbidigo // We want to print to stdout
				os.Exit(1)
			}

			fmt.Printf("\nGracefully shutting down. Press Ctrl+C again to force quit\n") //nolint:forbidigo // We want to print to stdout
			cancel()
		}
	}()
}

// NewRootCmd parses the command line flags and returns a config.Config struct.
func newRootCmd(version string, cfg *config.Config, subcommands []*ff.Command) *ff.Command {
	fs := ff.NewFlagSet(appName)

	for _, cmd := range subcommands {
		cmd.Flags = ff.NewFlagSet(cmd.Name).SetParent(fs)
	}

	cmd := &ff.Command{
		Name:        appName,
		Usage:       fmt.Sprintf("%s <command> [flags]", appName),
		ShortHelp:   fmt.Sprintf("(%s) A webfinger server", version),
		Flags:       fs,
		Subcommands: subcommands,
	}

	// Use 0.0.0.0 as the default host if on docker
	defaultHost := "localhost"
	if os.Getenv("ENV_DOCKER") == "true" {
		defaultHost = "0.0.0.0"
	}

	fs.BoolVar(&cfg.Debug, 'd', "debug", "Enable debug logging")
	fs.StringVar(&cfg.Host, 'h', "host", defaultHost, "Host to listen on")
	fs.StringVar(&cfg.Port, 'p', "port", "8080", "Port to listen on")
	fs.StringVar(&cfg.URNPath, 'u', "urn-file", "urns.yml", "Path to the URNs file")
	fs.StringVar(&cfg.FingerPath, 'f', "finger-file", "fingers.yml", "Path to the fingers file")

	return cmd
}
