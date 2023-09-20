package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"git.maronato.dev/maronato/finger/internal/log"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := log.FromContext(ctx)

		start := time.Now()

		// Wrap the response writer
		wrapped := WrapResponseWriter(w)

		// Call the next handler
		next.ServeHTTP(wrapped, r)

		status := wrapped.Status()

		// Log the request
		lg := l.With(
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", status),
			slog.String("remote", r.RemoteAddr),
			slog.Duration("duration", time.Since(start)),
		)

		switch {
		case status >= http.StatusInternalServerError:
			lg.Error("Server error")
		case status >= http.StatusBadRequest:
			lg.Info("Client error")
		default:
			lg.Info("Request completed")
		}
	})
}
