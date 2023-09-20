package config_test

import (
	"testing"

	"git.maronato.dev/maronato/finger/internal/config"
)

func TestConfig_GetAddr(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		cfg  *config.Config
		want string
	}{
		{
			name: "default",
			cfg:  config.NewConfig(),
			want: "localhost:8080",
		},
		{
			name: "custom",
			cfg: &config.Config{
				Host: "example.com",
				Port: "1234",
			},
			want: "example.com:1234",
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tc.cfg.GetAddr()
			if got != tc.want {
				t.Errorf("Config.GetAddr() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     *config.Config
		wantErr bool
	}{
		{
			name:    "default",
			cfg:     config.NewConfig(),
			wantErr: false,
		},
		{
			name: "empty host",
			cfg: &config.Config{
				Host: "",
				Port: config.DefaultPort,
			},
			wantErr: true,
		},
		{
			name: "empty port",
			cfg: &config.Config{
				Host: config.DefaultHost,
				Port: "",
			},
			wantErr: true,
		},
		{
			name: "invalid addr",
			cfg: &config.Config{
				Host: config.DefaultHost,
				Port: "invalid",
			},
			wantErr: true,
		},
		{
			name: "empty urn path",
			cfg: &config.Config{
				Host:    config.DefaultHost,
				Port:    config.DefaultPort,
				URNPath: "",
			},
			wantErr: true,
		},
		{
			name: "empty finger path",
			cfg: &config.Config{
				Host:       config.DefaultHost,
				Port:       config.DefaultPort,
				URNPath:    config.DefaultURNPath,
				FingerPath: "",
			},
			wantErr: true,
		},
		{
			name: "valid",
			cfg: &config.Config{
				Host:       config.DefaultHost,
				Port:       config.DefaultPort,
				URNPath:    config.DefaultURNPath,
				FingerPath: config.DefaultFingerPath,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Config.Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}
