package handler

import (
	"encoding/json"
	"net/http"

	"git.maronato.dev/maronato/finger/webfingers"
)

func WebfingerHandler(fingers webfingers.WebFingers) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only handle GET requests
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)

			return
		}

		// Get the query params
		q := r.URL.Query()

		// Get the resource
		resource := q.Get("resource")
		if resource == "" {
			http.Error(w, "No resource provided", http.StatusBadRequest)

			return
		}

		// Get and validate resource
		finger, ok := fingers[resource]
		if !ok {
			http.Error(w, "Resource not found", http.StatusNotFound)

			return
		}

		// Set the content type
		w.Header().Set("Content-Type", "application/jrd+json")

		// Write the response
		if err := json.NewEncoder(w).Encode(finger); err != nil {
			http.Error(w, "Error encoding json", http.StatusInternalServerError)

			return
		}
	})
}
