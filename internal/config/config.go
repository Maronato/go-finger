package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
)

const (
	// DefaultHost is the default host to listen on.
	DefaultHost = "localhost"
	// DefaultPort is the default port to listen on.
	DefaultPort = "8080"
	// DefaultURNPath is the default file path to the URN alias file.
	DefaultURNPath = "urns.yml"
	// DefaultFingerPath is the default file path to the webfinger definition file.
	DefaultFingerPath = "finger.yml"
)

// ErrInvalidConfig is returned when the config is invalid.
var ErrInvalidConfig = errors.New("invalid config")

type Config struct {
	Debug      bool
	Host       string
	Port       string
	URNPath    string
	FingerPath string
}

func NewConfig() *Config {
	return &Config{
		Host:       DefaultHost,
		Port:       DefaultPort,
		URNPath:    DefaultURNPath,
		FingerPath: DefaultFingerPath,
	}
}

func (c *Config) GetAddr() string {
	return net.JoinHostPort(c.Host, c.Port)
}

func (c *Config) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("%w: host is empty", ErrInvalidConfig)
	}

	if c.Port == "" {
		return fmt.Errorf("%w: port is empty", ErrInvalidConfig)
	}

	if _, err := url.Parse(c.GetAddr()); err != nil {
		return fmt.Errorf("%w: %w", ErrInvalidConfig, err)
	}

	if c.URNPath == "" {
		return fmt.Errorf("%w: urn path is empty", ErrInvalidConfig)
	}

	if c.FingerPath == "" {
		return fmt.Errorf("%w: finger path is empty", ErrInvalidConfig)
	}

	return nil
}
