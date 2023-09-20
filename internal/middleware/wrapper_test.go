package middleware_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"git.maronato.dev/maronato/finger/internal/middleware"
)

func TestWrapResponseWriter(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	wrapped := middleware.WrapResponseWriter(w)

	if wrapped == nil {
		t.Error("wrapper is nil")
	}
}

func TestResponseWrapper_Status(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	wrapped := middleware.WrapResponseWriter(w)

	if wrapped.Status() != 0 {
		t.Error("status is not 0")
	}

	wrapped.WriteHeader(http.StatusOK)

	if wrapped.Status() != http.StatusOK {
		t.Error("status is not 200")
	}
}

type FailWriter struct{}

func (w *FailWriter) Write(b []byte) (int, error) {
	return 0, fmt.Errorf("error")
}

func (w *FailWriter) Header() http.Header {
	return http.Header{}
}

func (w *FailWriter) WriteHeader(_ int) {}

func TestResponseWrapper_Write(t *testing.T) {
	t.Parallel()

	t.Run("writes success messages", func(t *testing.T) {
		t.Parallel()

		w := httptest.NewRecorder()
		wrapped := middleware.WrapResponseWriter(w)

		size, err := wrapped.Write([]byte("test"))
		if err != nil {
			t.Errorf("error writing response: %v", err)
		}

		if size != 4 {
			t.Error("size is not 4")
		}

		if wrapped.Status() != http.StatusOK {
			t.Error("status is not 200")
		}
	})

	t.Run("returns error on fail write", func(t *testing.T) {
		t.Parallel()

		w := &FailWriter{}
		wrapped := middleware.WrapResponseWriter(w)

		_, err := wrapped.Write([]byte("test"))
		if err == nil {
			t.Error("error is nil")
		}
	})
}

func TestResponseWrapper_Unwrap(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	wrapped := middleware.WrapResponseWriter(w)

	if wrapped.Unwrap() != w {
		t.Error("unwrapped response is not the same")
	}
}
