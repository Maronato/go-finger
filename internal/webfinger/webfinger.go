package webfinger

import (
	"context"
	"fmt"
	"log/slog"
	"net/mail"
	"net/url"
	"os"

	"git.maronato.dev/maronato/finger/internal/config"
	"git.maronato.dev/maronato/finger/internal/log"
	"gopkg.in/yaml.v3"
)

type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href,omitempty"`
}

type WebFinger struct {
	Subject    string            `json:"subject"`
	Links      []Link            `json:"links,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

type WebFingers map[string]*WebFinger

type (
	URNMap        = map[string]string
	RawFingersMap = map[string]map[string]string
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

func (f *FingerReader) ParseFingers(ctx context.Context, urns URNMap, rawFingers RawFingersMap) (WebFingers, error) {
	l := log.FromContext(ctx)

	webfingers := make(WebFingers)

	// Parse the webfinger file
	for k, v := range rawFingers {
		resource := k

		// Remove leading acct: if present
		if len(k) > 5 && resource[:5] == "acct:" {
			resource = resource[5:]
		}

		// The key must be a URL or email address
		if _, err := mail.ParseAddress(resource); err != nil {
			if _, err := url.ParseRequestURI(resource); err != nil {
				return nil, fmt.Errorf("error parsing webfinger key (%s): %w", k, err)
			}
		} else {
			// Add acct: back to the key if it is an email address
			resource = fmt.Sprintf("acct:%s", resource)
		}

		// Create a new webfinger
		webfinger := &WebFinger{
			Subject: resource,
		}

		// Parse the fields
		for field, value := range v {
			fieldUrn := field

			// If the key is present in the URNs file, use the value
			if _, ok := urns[field]; ok {
				fieldUrn = urns[field]
			}

			// If the value is a valid URI, add it to the links
			if _, err := url.ParseRequestURI(value); err == nil {
				webfinger.Links = append(webfinger.Links, Link{
					Rel:  fieldUrn,
					Href: value,
				})
			} else {
				// Otherwise add it to the properties
				if webfinger.Properties == nil {
					webfinger.Properties = make(map[string]string)
				}

				webfinger.Properties[fieldUrn] = value
			}
		}

		// Add the webfinger to the map
		webfingers[resource] = webfinger
	}

	l.Debug("Webfinger map built successfully", slog.Int("number", len(webfingers)), slog.Any("data", webfingers))

	return webfingers, nil
}

func (f *FingerReader) ReadFingerFile(ctx context.Context) (WebFingers, error) {
	l := log.FromContext(ctx)

	urnMap := make(URNMap)
	fingerData := make(RawFingersMap)

	// Parse the URNs file
	if err := yaml.Unmarshal(f.URNSFile, &urnMap); err != nil {
		return nil, fmt.Errorf("error unmarshalling URNs file: %w", err)
	}

	// The URNs file must be a map of strings to valid URLs
	for _, v := range urnMap {
		if _, err := url.ParseRequestURI(v); err != nil {
			return nil, fmt.Errorf("error parsing URN URIs: %w", err)
		}
	}

	l.Debug("URNs file parsed successfully", slog.Int("number", len(urnMap)), slog.Any("data", urnMap))

	// Parse the fingers file
	if err := yaml.Unmarshal(f.FingersFile, &fingerData); err != nil {
		return nil, fmt.Errorf("error unmarshalling fingers file: %w", err)
	}

	l.Debug("Fingers file parsed successfully", slog.Int("number", len(fingerData)), slog.Any("data", fingerData))

	// Parse raw data
	webfingers, err := f.ParseFingers(ctx, urnMap, fingerData)
	if err != nil {
		return nil, fmt.Errorf("error parsing raw fingers: %w", err)
	}

	return webfingers, nil
}
