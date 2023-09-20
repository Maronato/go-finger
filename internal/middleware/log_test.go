package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/middleware"
)

func TestRequestLogger(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.NewConfig()

	stdout := &strings.Builder{}

	l := log.NewLogger(stdout, cfg)
	ctx = log.WithLogger(ctx, l)

	w := httptest.NewRecorder()
	r, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)

	if stdout.String() != "" {
		t.Error("logger logged before request")
	}

	middleware.RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})).ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Error("status is not 200")
	}

	if stdout.String() == "" {
		t.Error("logger did not log request")
	}
}
