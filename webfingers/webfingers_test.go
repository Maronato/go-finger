package webfingers_test

import (
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"git.maronato.dev/maronato/finger/webfingers"
)

func TestNewWebFingers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resources  webfingers.Resources
		urnAliases webfingers.URNAliases
		want       webfingers.WebFingers
		wantErr    bool
	}{
		{
			name: "basic",
			resources: webfingers.Resources{
				"user@example.com": {
					"name": "Example User",
				},
			},
			urnAliases: webfingers.URNAliases{
				"name": "http://schema.org/name",
			},
			want: webfingers.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"http://schema.org/name": "Example User",
					},
				},
			},
		},
		{
			name: "parses links",
			resources: webfingers.Resources{
				"user@example.com": {
					"link1": "https://example.com/link1",
					"link2": "https://example.com/link2",
				},
			},
			want: webfingers.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Links: []webfingers.Link{
						{
							Rel:  "link1",
							Href: "https://example.com/link1",
						},
						{
							Rel:  "link2",
							Href: "https://example.com/link2",
						},
					},
				},
			},
		},
		{
			name: "parses links with URN aliases",
			resources: webfingers.Resources{
				"user@example.com": {
					"link1": "https://example.com/link1",
				},
			},
			urnAliases: webfingers.URNAliases{
				"link1": "http://schema.com/link",
			},
			want: webfingers.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Links: []webfingers.Link{
						{
							Rel:  "http://schema.com/link",
							Href: "https://example.com/link1",
						},
					},
				},
			},
		},
		{
			name: "parses properties",
			resources: webfingers.Resources{
				"user@example.com": {
					"prop1": "value1",
					"prop2": "value2",
				},
			},
			want: webfingers.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"prop1": "value1",
						"prop2": "value2",
					},
				},
			},
		},
		{
			name: "parses properties with URN aliases",
			resources: webfingers.Resources{
				"user@example.com": {
					"prop1": "value1",
				},
			},
			urnAliases: webfingers.URNAliases{
				"prop1": "http://schema.com/prop",
			},
			want: webfingers.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"http://schema.com/prop": "value1",
					},
				},
			},
		},
		{
			name: "parses multiple resources",
			resources: webfingers.Resources{
				"user@example.com": {
					"prop1": "value1",
				},
				"user2@example.com": {
					"prop2": "value2",
				},
			},
			want: webfingers.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"prop1": "value1",
					},
				},
				"acct:user2@example.com": {
					Subject: "acct:user2@example.com",
					Properties: map[string]string{
						"prop2": "value2",
					},
				},
			},
		},
		{
			name: "parses URI resources",
			resources: webfingers.Resources{
				"https://example.com": {
					"prop1": "value1",
				},
			},
			want: webfingers.WebFingers{
				"https://example.com": {
					Subject: "https://example.com",
					Properties: map[string]string{
						"prop1": "value1",
					},
				},
			},
		},
		{
			name: "parses email resource with acct:",
			resources: webfingers.Resources{
				"acct:user@example.com": {
					"prop1": "value1",
				},
			},
			want: webfingers.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"prop1": "value1",
					},
				},
			},
		},
		{
			name: "errors on invalid resource",
			resources: webfingers.Resources{
				"invalid": {
					"prop1": "value1",
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := webfingers.NewWebFingers(tc.resources, tc.urnAliases)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("unexpected error: %v", err)
				}

				return
			} else if tc.wantErr {
				t.Error("expected error, got nil")
			}

			// Sort the links.
			for _, finger := range got {
				sort.Slice(finger.Links, func(i, j int) bool {
					return finger.Links[i].Rel < finger.Links[j].Rel
				})
			}

			for _, finger := range tc.want {
				sort.Slice(finger.Links, func(i, j int) bool {
					return finger.Links[i].Rel < finger.Links[j].Rel
				})
			}

			if !reflect.DeepEqual(got, tc.want) {
				// Marshall both so we can visualize the differences.
				gotJSON, _ := json.MarshalIndent(got, "", "  ")
				wantJSON, _ := json.MarshalIndent(tc.want, "", "  ")

				t.Errorf("got:\n%s\nwant:\n%s", gotJSON, wantJSON)
			}
		})
	}
}
