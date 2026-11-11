package main

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/netlify/open-api/go/models"
)

// netlifyAPIError is implemented by the generated operations.*Default response types, which are
// the only Netlify API errors that expose the HTTP status code and response payload.
type netlifyAPIError interface {
	error
	Code() int
	GetPayload() *models.Error
}

// errUnauthorized identifies a rejected Netlify access token so main can report it as fatal.
var errUnauthorized = errors.New("netlify API access token is unauthorised")

// isUnauthorized reports whether err is an API response with a 401 status code.
func isUnauthorized(err error) bool {
	var apiErr netlifyAPIError
	return errors.As(err, &apiErr) && apiErr.Code() == http.StatusUnauthorized
}

// unauthorizedError explains how to fix a rejected access token.
func unauthorizedError(zone string) error {
	return fmt.Errorf("%w. Check that --accesstoken is valid and has access to the DNS zone %q", errUnauthorized, zone)
}

// describeAPIError adds the affected record and the Netlify response's status and message to an
// error from the generated API client. Other errors are returned unchanged.
func describeAPIError(err error, hostname, recordType, recordID string) error {
	var apiErr netlifyAPIError
	if !errors.As(err, &apiErr) {
		return err
	}

	detail := fmt.Sprintf("Netlify API returned HTTP %d", apiErr.Code())
	if payload := apiErr.GetPayload(); payload != nil {
		if payload.Message != "" {
			detail += ": " + payload.Message
		}
		if payload.Code != 0 {
			detail += fmt.Sprintf(" (code %d)", payload.Code)
		}
	}

	var target string
	switch {
	case hostname != "" && recordType != "":
		target = fmt.Sprintf("%s record %q", recordType, hostname)
	case hostname != "":
		target = fmt.Sprintf("record %q", hostname)
	}
	if recordID != "" {
		if target != "" {
			target += " "
		}
		target += fmt.Sprintf("(id %s)", recordID)
	}
	if target == "" {
		return errors.New(detail)
	}
	return fmt.Errorf("%s: %s", target, detail)
}
