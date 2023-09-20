package fingerreader_test

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/fingerreader"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/webfingers"
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

	f := fingerreader.NewFingerReader()

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

			f := fingerreader.NewFingerReader()

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

func TestReadFingerFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		urnsContent    string
		fingersContent string
		wantURN        webfingers.URNAliases
		wantFinger     webfingers.Resources
		returns        webfingers.WebFingers
		wantErr        bool
	}{
		{
			name:           "reads files",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "user@example.com:\n  name: John Doe",
			wantURN: webfingers.URNAliases{
				"name":    "https://schema/name",
				"profile": "https://schema/profile",
			},
			wantFinger: webfingers.Resources{
				"user@example.com": {
					"name": "John Doe",
				},
			},
			returns: webfingers.WebFingers{
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
			wantURN: webfingers.URNAliases{
				"favorite_food": "https://schema/favorite_food",
			},
			wantFinger: webfingers.Resources{
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
			wantErr:        true,
		},
		{
			name:           "errors on invalid fingers file",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "invalid",
			wantErr:        true,
		},
		{
			name:           "errors on invalid URNs values",
			urnsContent:    "name: invalid",
			fingersContent: "user@example.com:\n  name: John Doe",
			wantErr:        true,
		},
		{
			name:           "errors on invalid fingers values",
			urnsContent:    "name: https://schema/name\nprofile: https://schema/profile",
			fingersContent: "invalid:\n  name: John Doe",
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

			f := fingerreader.NewFingerReader()

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

			if tc.returns != nil && !reflect.DeepEqual(got, tc.returns) {
				t.Errorf("ReadFingerFile() got = %v, want: %v", got, tc.returns)
			}
		})
	}
}
