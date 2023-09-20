package middleware

import (
	"log/slog"
	"net/http"

	"git.maronato.dev/maronato/finger/internal/log"
)

func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := log.FromContext(ctx)

		defer func() {
			err := recover()
			if err != nil {
				l.Error("Panic", slog.Any("error", err))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)

				return
			}
		}()

		next.ServeHTTP(w, r)
	})
}
