package cmd

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"git.maronato.dev/maronato/finger/internal/config"
	"github.com/peterbourgon/ff/v4"
)

func newHealthcheckCmd(cfg *config.Config) *ff.Command {
	return &ff.Command{
		Name:      "healthcheck",
		Usage:     "healthcheck [flags]",
		ShortHelp: "Check if the server is running",
		Exec: func(ctx context.Context, args []string) error {
			// Create a new client
			client := &http.Client{
				Timeout: 5 * time.Second, //nolint:gomnd // We want to use a constant
			}

			// Create a new request
			reqURL := url.URL{
				Scheme: "http",
				Host:   cfg.GetAddr(),
				Path:   "/healthz",
			}

			req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL.String(), http.NoBody)
			if err != nil {
				return fmt.Errorf("error creating request: %w", err)
			}

			// Send the request
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("error sending request: %w", err)
			}

			defer resp.Body.Close()

			// Check the response
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("server returned status %d", resp.StatusCode) //nolint:goerr113 // We want to return an error
			}

			return nil
		},
	}
}
