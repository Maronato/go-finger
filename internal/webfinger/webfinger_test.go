package webfinger_test

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/internal/webfinger"
)

func newTempFile(t *testing.T, content string) (name string, remove func()) {
	t.Helper()

	f, err := os.CreateTemp("", "finger-test")
	if err != nil {
		t.Fatalf("error creating temp file: %v", err)
	}

	_, err = f.WriteString(content)
	if err != nil {
		t.Fatalf("error writing to temp file: %v", err)
	}

	return f.Name(), func() {
		err = os.Remove(f.Name())
		if err != nil {
			t.Fatalf("error removing temp file: %v", err)
		}
	}
}

func TestNewFingerReader(t *testing.T) {
	t.Parallel()

	f := webfinger.NewFingerReader()

	if f == nil {
		t.Errorf("NewFingerReader() = %v, want: %v", f, nil)
	}
}

func TestFingerReader_ReadFiles(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		urnsContent    string
		fingersContent string
		useURNFile     bool
		useFingerFile  bool
		wantErr        bool
	}{
		{
			name:           "reads files",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "user@example.com:\n  name: John Doe",
			useURNFile:     true,
			useFingerFile:  true,
			wantErr:        false,
		},
		{
			name:           "errors on missing URNs file",
			urnsContent:    "invalid",
			fingersContent: "user@example.com:\n  name: John Doe",
			useURNFile:     false,
			useFingerFile:  true,
			wantErr:        true,
		},
		{
			name:           "errors on missing fingers file",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "invalid",
			useFingerFile:  false,
			useURNFile:     true,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cfg := config.NewConfig()

			urnsFileName, urnsCleanup := newTempFile(t, tc.urnsContent)
			defer urnsCleanup()

			fingersFileName, fingersCleanup := newTempFile(t, tc.fingersContent)
			defer fingersCleanup()

			if !tc.useURNFile {
				cfg.URNPath = "invalid"
			} else {
				cfg.URNPath = urnsFileName
			}

			if !tc.useFingerFile {
				cfg.FingerPath = "invalid"
			} else {
				cfg.FingerPath = fingersFileName
			}

			f := webfinger.NewFingerReader()

			err := f.ReadFiles(cfg)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("ReadFiles() error = %v", err)
				}

				return
			} else if tc.wantErr {
				t.Errorf("ReadFiles() error = %v, wantErr %v", err, tc.wantErr)
			}

			if !reflect.DeepEqual(f.URNSFile, []byte(tc.urnsContent)) {
				t.Errorf("ReadFiles() URNsFile = %v, want: %v", f.URNSFile, tc.urnsContent)
			}

			if !reflect.DeepEqual(f.FingersFile, []byte(tc.fingersContent)) {
				t.Errorf("ReadFiles() FingersFile = %v, want: %v", f.FingersFile, tc.fingersContent)
			}
		})
	}
}

func TestParseFingers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		rawFingers webfinger.RawFingersMap
		want       webfinger.WebFingers
		wantErr    bool
	}{
		{
			name: "parses links",
			rawFingers: webfinger.RawFingersMap{
				"user@example.com": {
					"profile":           "https://example.com/profile",
					"invalidalias":      "https://example.com/invalidalias",
					"https://something": "https://somethingelse",
				},
			},
			want: webfinger.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Links: []webfinger.Link{
						{
							Rel:  "https://schema/profile",
							Href: "https://example.com/profile",
						},
						{
							Rel:  "invalidalias",
							Href: "https://example.com/invalidalias",
						},
						{
							Rel:  "https://something",
							Href: "https://somethingelse",
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "parses properties",
			rawFingers: webfinger.RawFingersMap{
				"user@example.com": {
					"name":           "John Doe",
					"invalidalias":   "value1",
					"https://mylink": "value2",
				},
			},
			want: webfinger.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"https://schema/name": "John Doe",
						"invalidalias":        "value1",
						"https://mylink":      "value2",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "accepts acct: prefix",
			rawFingers: webfinger.RawFingersMap{
				"acct:user@example.com": {
					"name": "John Doe",
				},
			},
			want: webfinger.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"https://schema/name": "John Doe",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "accepts urls as resource",
			rawFingers: webfinger.RawFingersMap{
				"https://example.com": {
					"name": "John Doe",
				},
			},
			want: webfinger.WebFingers{
				"https://example.com": {
					Subject: "https://example.com",
					Properties: map[string]string{
						"https://schema/name": "John Doe",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "accepts multiple resources",
			rawFingers: webfinger.RawFingersMap{
				"user@example.com": {
					"name": "John Doe",
				},
				"other@example.com": {
					"name": "Jane Doe",
				},
			},
			want: webfinger.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"https://schema/name": "John Doe",
					},
				},
				"acct:other@example.com": {
					Subject: "acct:other@example.com",
					Properties: map[string]string{
						"https://schema/name": "Jane Doe",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "errors on invalid resource",
			rawFingers: webfinger.RawFingersMap{
				"invalid": {
					"name": "John Doe",
				},
			},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// Create a urn map
			urns := webfinger.URNMap{
				"name":    "https://schema/name",
				"profile": "https://schema/profile",
			}

			ctx := context.Background()
			cfg := config.NewConfig()
			l := log.NewLogger(&strings.Builder{}, cfg)

			ctx = log.WithLogger(ctx, l)

			f := webfinger.NewFingerReader()

			got, err := f.ParseFingers(ctx, urns, tc.rawFingers)
			if (err != nil) != tc.wantErr {
				t.Errorf("ParseFingers() error = %v, wantErr %v", err, tc.wantErr)

				return
			}

			// Sort links to make it easier to compare
			for _, v := range got {
				for range v.Links {
					sort.Slice(v.Links, func(i, j int) bool {
						return v.Links[i].Rel < v.Links[j].Rel
					})
				}
			}

			for _, v := range tc.want {
				for range v.Links {
					sort.Slice(v.Links, func(i, j int) bool {
						return v.Links[i].Rel < v.Links[j].Rel
					})
				}
			}

			if !reflect.DeepEqual(got, tc.want) {
				// Unmarshal the structs to JSON to make it easier to print
				gotstr := &strings.Builder{}
				gotenc := json.NewEncoder(gotstr)

				wantstr := &strings.Builder{}
				wantenc := json.NewEncoder(wantstr)

				_ = gotenc.Encode(got)
				_ = wantenc.Encode(tc.want)

				t.Errorf("ParseFingers() got = \n%s want: \n%s", gotstr.String(), wantstr.String())
			}
		})
	}
}

func TestReadFingerFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		urnsContent    string
		fingersContent string
		wantURN        webfinger.URNMap
		wantFinger     webfinger.RawFingersMap
		returns        *webfinger.WebFingers
		wantErr        bool
	}{
		{
			name:           "reads files",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "user@example.com:\n  name: John Doe",
			wantURN: webfinger.URNMap{
				"name":    "https://schema/name",
				"profile": "https://schema/profile",
			},
			wantFinger: webfinger.RawFingersMap{
				"user@example.com": {
					"name": "John Doe",
				},
			},
			returns: &webfinger.WebFingers{
				"acct:user@example.com": {
					Subject: "acct:user@example.com",
					Properties: map[string]string{
						"https://schema/name": "John Doe",
					},
				},
			},
			wantErr: false,
		},
		{
			name:           "uses custom URNs",
			urnsContent:    "favorite_food: https://schema/favorite_food",
			fingersContent: "user@example.com:\n  favorite_food: Apple",
			wantURN: webfinger.URNMap{
				"favorite_food": "https://schema/favorite_food",
			},
			wantFinger: webfinger.RawFingersMap{
				"user@example.com": {
					"https://schema/favorite_food": "Apple",
				},
			},
			wantErr: false,
		},
		{
			name:           "errors on invalid URNs file",
			urnsContent:    "invalid",
			fingersContent: "user@example.com:\n  name: John Doe",
			wantURN:        webfinger.URNMap{},
			wantFinger:     webfinger.RawFingersMap{},
			wantErr:        true,
		},
		{
			name:           "errors on invalid fingers file",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "invalid",
			wantURN:        webfinger.URNMap{},
			wantFinger:     webfinger.RawFingersMap{},
			wantErr:        true,
		},
		{
			name:           "errors on invalid URNs values",
			urnsContent:    "name: invalid",
			fingersContent: "user@example.com:\n  name: John Doe",
			wantURN:        webfinger.URNMap{},
			wantFinger:     webfinger.RawFingersMap{},
			wantErr:        true,
		},
		{
			name:           "errors on invalid fingers values",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "invalid:\n  name: John Doe",
			wantURN:        webfinger.URNMap{},
			wantFinger:     webfinger.RawFingersMap{},
			wantErr:        true,
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

			f := webfinger.NewFingerReader()

			f.FingersFile = []byte(tc.fingersContent)
			f.URNSFile = []byte(tc.urnsContent)

			got, err := f.ReadFingerFile(ctx)
			if err != nil {
				if !tc.wantErr {
					t.Errorf("ReadFingerFile() error = %v", err)
				}

				return
			} else if tc.wantErr {
				t.Errorf("ReadFingerFile() error = %v, wantErr %v", err, tc.wantErr)
			}

			if tc.returns != nil && !reflect.DeepEqual(got, *tc.returns) {
				t.Errorf("ReadFingerFile() got = %v, want: %v", got, *tc.returns)
			}
		})
	}
}
