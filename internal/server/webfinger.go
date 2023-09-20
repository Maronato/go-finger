package server

import (
	"encoding/json"
	"net/http"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/webfinger"
)

func WebfingerHandler(_ *config.Config, webfingers webfinger.WebFingers) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		l := log.FromContext(ctx)

		// Only handle GET requests
		if r.Method != http.MethodGet {
			l.Debug("Method not allowed")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		// Get the query params
		q := r.URL.Query()

		// Get the resource
		resource := q.Get("resource")
		if resource == "" {
			l.Debug("No resource provided")
			http.Error(w, "No resource provided", http.StatusBadRequest)

			return
		}

		// Get and validate resource
		finger, ok := webfingers[resource]
		if !ok {
			l.Debug("Resource not found")
			http.Error(w, "Resource not found", http.StatusNotFound)

			return
		}

		// Set the content type
		w.Header().Set("Content-Type", "application/jrd+json")

		// Write the response
		if err := json.NewEncoder(w).Encode(finger); err != nil {
			l.Debug("Error encoding json")
			http.Error(w, "Error encoding json", http.StatusInternalServerError)

			return
		}

		l.Debug("Webfinger request successful")
	})
}
