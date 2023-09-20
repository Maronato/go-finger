package server_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/server"
)

func TestHealthcheckHandler(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.NewConfig()
	l := log.NewLogger(&strings.Builder{}, cfg)

	ctx = log.WithLogger(ctx, l)

	// Create a new request
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/healthz", http.NoBody)

	// Create a new recorder
	rec := httptest.NewRecorder()

	// Create a new handler
	h := server.HealthCheckHandler(cfg)

	// Serve the request
	h.ServeHTTP(rec, req)

	// Check the status code
	if rec.Code != http.StatusOK {
		t.Errorf("expected status code %d, got %d", http.StatusOK, rec.Code)
	}
}
