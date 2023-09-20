package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/middleware"
	"git.maronato.dev/maronato/finger/internal/webfinger"
	"golang.org/x/sync/errgroup"
)

const (
	// ReadTimeout is the maximum duration for reading the entire
	// request, including the body.
	ReadTimeout = 5 * time.Second
	// WriteTimeout is the maximum duration before timing out
	// writes of the response.
	WriteTimeout = 10 * time.Second
	// IdleTimeout is the maximum amount of time to wait for the
	// next request when keep-alives are enabled.
	IdleTimeout = 30 * time.Second
	// ReadHeaderTimeout is the amount of time allowed to read
	// request headers.
	ReadHeaderTimeout = 2 * time.Second
	// RequestTimeout is the maximum duration for the entire
	// request.
	RequestTimeout = 7 * 24 * time.Hour
)

func StartServer(ctx context.Context, cfg *config.Config, webfingers webfinger.WebFingers) error {
	l := log.FromContext(ctx)

	// Create the server mux
	mux := http.NewServeMux()
	mux.Handle("/.well-known/webfinger", WebfingerHandler(cfg, webfingers))
	mux.Handle("/healthz", HealthCheckHandler(cfg))

	// Create a new server
	srv := &http.Server{
		Addr: cfg.GetAddr(),
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
		Handler: middleware.RequestLogger(
			middleware.Recoverer(
				http.TimeoutHandler(mux, RequestTimeout, "request timed out"),
			),
		),
		ReadHeaderTimeout: ReadHeaderTimeout,
		ReadTimeout:       ReadTimeout,
		WriteTimeout:      WriteTimeout,
		IdleTimeout:       IdleTimeout,
	}

	// Create the errorgroup that will manage the server execution
	eg, egCtx := errgroup.WithContext(ctx)

	// Start the server
	eg.Go(func() error {
		l.Info("Starting server", slog.String("addr", srv.Addr))

		// Use the global context for the server
		srv.BaseContext = func(_ net.Listener) context.Context {
			return egCtx
		}

		return srv.ListenAndServe() //nolint:wrapcheck // We wrap the error in the errgroup
	})
	// Gracefully shutdown the server when the context is done
	eg.Go(func() error {
		// Wait for the context to be done
		<-egCtx.Done()

		l.Info("Shutting down server")
		// Disable the cancel since we don't wan't to force
		// the server to shutdown if the context is canceled.
		noCancelCtx := context.WithoutCancel(egCtx)

		return srv.Shutdown(noCancelCtx) //nolint:wrapcheck // We wrap the error in the errgroup
	})

	// Log when the server is fully shutdown
	srv.RegisterOnShutdown(func() {
		l.Info("Server shutdown complete")
	})

	// Wait for the server to exit and check for errors that
	// are not caused by the context being canceled.
	if err := eg.Wait(); err != nil && ctx.Err() == nil {
		return fmt.Errorf("server exited with error: %w", err)
	}

	return nil
}
