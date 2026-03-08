package server

import (
	"net/http"

	"github.com/Maronato/go-finger/internal/config"
)

func HealthCheckHandler(_ *config.Config) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}
