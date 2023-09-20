package server_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"strings"
	"testing"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/server"
	"git.maronato.dev/maronato/finger/internal/webfinger"
)

func TestWebfingerHandler(t *testing.T) {
	t.Parallel()

	webfingers := webfinger.WebFingers{
		"acct:user@example.com": {
			Subject: "acct:user@example.com",
			Links: []webfinger.Link{
				{
					Rel:  "http://webfinger.net/rel/profile-page",
					Href: "https://example.com/user",
				},
			},
			Properties: map[string]string{
				"http://webfinger.net/rel/name": "John Doe",
			},
		},
		"acct:other@example.com": {
			Subject: "acct:other@example.com",
			Properties: map[string]string{
				"http://webfinger.net/rel/name": "Jane Doe",
			},
		},
		"https://example.com/user": {
			Subject: "https://example.com/user",
			Properties: map[string]string{
				"http://webfinger.net/rel/name": "John Baz",
			},
		},
	}

	tests := []struct {
		name            string
		resource        string
		wantCode        int
		alternateMethod string
	}{
		{
			name:     "valid resource",
			resource: "acct:user@example.com",
			wantCode: http.StatusOK,
		},
		{
			name:     "other valid resource",
			resource: "acct:other@example.com",
			wantCode: http.StatusOK,
		},
		{
			name:     "url resource",
			resource: "https://example.com/user",
			wantCode: http.StatusOK,
		},
		{
			name:     "resource missing acct:",
			resource: "user@example.com",
			wantCode: http.StatusNotFound,
		},
		{
			name:     "resource missing",
			resource: "",
			wantCode: http.StatusBadRequest,
		},
		{
			name:            "invalid method",
			resource:        "acct:user@example.com",
			wantCode:        http.StatusMethodNotAllowed,
			alternateMethod: http.MethodPost,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			cfg := config.NewConfig()
			l := log.NewLogger(&strings.Builder{}, cfg)

			ctx = log.WithLogger(ctx, l)

			// Create a new request
			r, _ := http.NewRequestWithContext(ctx, tc.alternateMethod, "/.well-known/webfinger?resource="+tc.resource, http.NoBody)

			// Create a new response
			w := httptest.NewRecorder()

			// Create a new handler
			h := server.WebfingerHandler(cfg, webfingers)

			// Serve the request
			h.ServeHTTP(w, r)

			// Check the status code
			if w.Code != tc.wantCode {
				t.Errorf("expected status code %d, got %d", tc.wantCode, w.Code)
			}

			// If the status code is 200, check the response body
			if tc.wantCode == http.StatusOK {
				// Check the content type
				if w.Header().Get("Content-Type") != "application/jrd+json" {
					t.Errorf("expected content type %s, got %s", "application/jrd+json", w.Header().Get("Content-Type"))
				}

				fingerWant := webfingers[tc.resource]
				fingerGot := &webfinger.WebFinger{}

				// Decode the response body
				if err := json.NewDecoder(w.Body).Decode(fingerGot); err != nil {
					t.Errorf("error decoding json: %v", err)
				}

				//  Sort links

				sort.Slice(fingerGot.Links, func(i, j int) bool {
					return fingerGot.Links[i].Rel < fingerGot.Links[j].Rel
				})

				sort.Slice(fingerWant.Links, func(i, j int) bool {
					return fingerWant.Links[i].Rel < fingerWant.Links[j].Rel
				})

				// Check the response body
				if !reflect.DeepEqual(fingerGot, fingerWant) {
					t.Errorf("expected body %v, got %v", fingerWant, fingerGot)
				}
			}
		})
	}
}
