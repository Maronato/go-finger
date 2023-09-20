package webfingers

import (
	"fmt"
	"net/mail"
	"net/url"
)

// Link is a link in a webfinger.
type Link struct {
	Rel  string `json:"rel"`
	Href string `json:"href,omitempty"`
}

// WebFinger is a webfinger.
type WebFinger struct {
	Subject    string            `json:"subject"`
	Links      []Link            `json:"links,omitempty"`
	Properties map[string]string `json:"properties,omitempty"`
}

// Resources is a simplified webfinger map.
type Resources map[string]map[string]string

// URNAliases is a map of URN aliases.
type URNAliases map[string]string

// WebFingers is a map of webfingers.
type WebFingers map[string]*WebFinger

// NewWebFingers creates a new webfinger map from a simplified webfinger map and an optional URN aliases map.
func NewWebFingers(resources Resources, urnAliases URNAliases) (WebFingers, error) {
	fingers := make(WebFingers)

	// If the aliases map is nil, create an empty one.
	if urnAliases == nil {
		urnAliases = make(URNAliases)
	}

	// Parse the resources.
	for k, v := range resources {
		subject := k

		// Remove leading acct: if present.
		if len(k) > 5 && subject[:5] == "acct:" {
			subject = subject[5:]
		}

		// The subject must be a URL or email address.
		if _, err := mail.ParseAddress(subject); err != nil {
			if _, err := url.ParseRequestURI(subject); err != nil {
				return nil, fmt.Errorf("error parsing resource subject (%s): %w", k, err)
			}
		} else {
			// Add acct: back to the subject if it is an email address.
			subject = fmt.Sprintf("acct:%s", subject)
		}

		// Create a new webfinger.
		finger := &WebFinger{
			Subject: subject,
		}

		// Parse the resource fields.
		for field, value := range v {
			fieldUrn := field

			// If the key is present in the aliases map, use its value.
			if _, ok := urnAliases[field]; ok {
				fieldUrn = urnAliases[field]
			}

			// If the value is a valid URI, add it to the links.
			if _, err := url.ParseRequestURI(value); err == nil {
				finger.Links = append(finger.Links, Link{
					Rel:  fieldUrn,
					Href: value,
				})
			} else {
				// Otherwise add it to the properties.
				if finger.Properties == nil {
					finger.Properties = make(map[string]string)
				}

				finger.Properties[fieldUrn] = value
			}
		}

		// Add the webfinger to the map.
		fingers[subject] = finger
	}

	return fingers, nil
}
