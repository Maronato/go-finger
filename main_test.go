package main_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	finger "git.maronato.dev/maronato/finger"
)

func BenchmarkGetWebfinger(b *testing.B) {
	ctx := context.Background()
	cfg := &finger.Config{}
	l := finger.NewLogger(cfg)

	ctx = finger.WithLogger(ctx, l)
	resource := "acct:user@example.com"
	webmap := finger.WebFingerMap{
		resource: {
			Subject: resource,
			Links: []finger.Link{
				{
					Rel:  "http://webfinger.net/rel/avatar",
					Href: "https://example.com/avatar.png",
				},
			},
			Properties: map[string]string{
				"example": "value",
			},
		},
		"acct:other": {
			Subject: "acct:other",
			Links: []finger.Link{
				{
					Rel:  "http://webfinger.net/rel/avatar",
					Href: "https://example.com/avatar.png",
				},
			},
			Properties: map[string]string{
				"example": "value",
			},
		},
	}

	handler := finger.WebfingerHandler(&finger.Config{}, webmap)

	r, _ := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		fmt.Sprintf("/.well-known/webfinger?resource=%s", resource),
		http.NoBody,
	)

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
	}
}
