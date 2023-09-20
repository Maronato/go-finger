package fingerreader

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"os"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"git.maronato.dev/maronato/finger/webfingers"
	"gopkg.in/yaml.v3"
)

type FingerReader struct {
	URNSFile    []byte
	FingersFile []byte
}

func NewFingerReader() *FingerReader {
	return &FingerReader{}
}

func (f *FingerReader) ReadFiles(cfg *config.Config) error {
	// Read URNs file
	file, err := os.ReadFile(cfg.URNPath)
	if err != nil {
		// If the file does not exist and the path is the default, set the URNs to an empty map
		if os.IsNotExist(err) && cfg.URNPath == config.DefaultURNPath {
			f.URNSFile = []byte("")
		} else {
			return fmt.Errorf("error opening URNs file: %w", err)
		}
	}

	f.URNSFile = file

	// Read fingers file
	file, err = os.ReadFile(cfg.FingerPath)
	if err != nil {
		// If the file does not exist and the path is the default, set the fingers to an empty map
		if os.IsNotExist(err) && cfg.FingerPath == config.DefaultFingerPath {
			f.FingersFile = []byte("")
		} else {
			return fmt.Errorf("error opening fingers file: %w", err)
		}
	}

	f.FingersFile = file

	return nil
}

func (f *FingerReader) ReadFingerFile(ctx context.Context) (webfingers.WebFingers, error) {
	l := log.FromContext(ctx)

	urnAliases := make(webfingers.URNAliases)
	resources := make(webfingers.Resources)

	// Parse the URNs file
	if err := yaml.Unmarshal(f.URNSFile, &urnAliases); err != nil {
		return nil, fmt.Errorf("error unmarshalling URNs file: %w", err)
	}

	// The URNs file must be a map of strings to valid URLs
	for _, v := range urnAliases {
		if _, err := url.ParseRequestURI(v); err != nil {
			return nil, fmt.Errorf("error parsing URN URIs: %w", err)
		}
	}

	l.Debug("URNs file parsed successfully", slog.Int("number", len(urnAliases)), slog.Any("data", urnAliases))

	// Parse the fingers file
	if err := yaml.Unmarshal(f.FingersFile, &resources); err != nil {
		return nil, fmt.Errorf("error unmarshalling fingers file: %w", err)
	}

	l.Debug("Fingers file parsed successfully", slog.Int("number", len(resources)), slog.Any("data", resources))

	// Parse raw data
	fingers, err := webfingers.NewWebFingers(resources, urnAliases)
	if err != nil {
		return nil, fmt.Errorf("error parsing raw fingers: %w", err)
	}

	return fingers, nil
}
