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

func assertNoPanic(t *testing.T, f func()) {
	t.Helper()

	defer func() {
		if r := recover(); r != nil {
			t.Error("function panicked")
		}
	}()

	f()
}

func TestRecoverer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cfg := config.NewConfig()
	l := log.NewLogger(&strings.Builder{}, cfg)
	ctx = log.WithLogger(ctx, l)

	t.Run("handles panics", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		r, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)

		h := middleware.Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic("test")
		}))

		assertNoPanic(t, func() {
			h.ServeHTTP(w, r)
		})

		if w.Code != http.StatusInternalServerError {
			t.Error("status is not 500")
		}

		if w.Body.String() != "Internal Server Error\n" {
			t.Error("response body is not 'Internal Server Error'")
		}
	})

	t.Run("handles successful requests", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		r, _ := http.NewRequestWithContext(ctx, http.MethodGet, "/", http.NoBody)

		h := middleware.Recoverer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		assertNoPanic(t, func() {
			h.ServeHTTP(w, r)
		})

		if w.Code != http.StatusOK {
			t.Error("status is not 200")
		}
	})
}
