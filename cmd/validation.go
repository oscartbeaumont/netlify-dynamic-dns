package main

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

// Netlify accepts TTLs between 60 seconds and one day.
const (
	minTTL int64 = 60
	maxTTL int64 = 86400
)

// maxIntervalMinutes is the largest --interval whose duration still fits in a time.Duration.
// Without the bound a huge value would overflow and turn the update loop into a busy loop.
const maxIntervalMinutes = int(math.MaxInt64 / int64(time.Minute))

// trimInputs removes surrounding whitespace from the user supplied values. A leading space in
// the zone produces an invalid Netlify zone id, so it is better to tolerate it than to leave
// the user with a configuration that can never work.
func trimInputs(a *Arguments) {
	a.AccessToken = strings.TrimSpace(a.AccessToken)
	a.Zone = strings.TrimSpace(a.Zone)
	a.Record = strings.TrimSpace(a.Record)
}

// validateZone checks that zone is a bare domain name which Netlify can turn into a zone id.
func validateZone(zone string) error {
	if zone == "" {
		return errors.New("must not be empty")
	}
	if strings.Contains(zone, "://") || strings.Contains(zone, "/") {
		return errors.New("must be a domain name without a scheme or path")
	}
	if !strings.Contains(zone, ".") {
		return errors.New(`must contain at least one dot, for example "example.com"`)
	}
	return validateLabels(zone, "zone")
}

// validateRecord checks that record is one or more DNS labels relative to zone. The record must
// not be a full hostname because the zone is appended to it when the record is created.
func validateRecord(record, zone string) error {
	if strings.Contains(record, "*") {
		return errors.New("must not contain a wildcard")
	}
	if record == zone || strings.HasSuffix(record, "."+zone) {
		return fmt.Errorf("must be a subdomain of %q, not a full hostname", zone)
	}
	return validateLabels(record, "record")
}

// validateLabels checks name is a dot separated set of lowercase DNS labels.
func validateLabels(name, kind string) error {
	if name == "" {
		return fmt.Errorf("%s must not be empty", kind)
	}
	if strings.ContainsAny(name, " \t\r\n") {
		return fmt.Errorf("%s must not contain whitespace", kind)
	}

	for _, label := range strings.Split(name, ".") {
		if label == "" {
			return fmt.Errorf("%s %q must not contain an empty label", kind, name)
		}
		if label[0] == '-' || label[len(label)-1] == '-' {
			return fmt.Errorf("%s %q must not have a label starting or ending with a hyphen", kind, name)
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return fmt.Errorf("%s %q contains the invalid character %q", kind, name, char)
			}
		}
	}

	return nil
}

// resolveTTL determines the TTL to use for a record. An explicit TTL always wins, then the TTL
// of the record being replaced is kept so a value configured in the Netlify dashboard survives,
// and otherwise the update interval is used so resolvers never cache an address for longer than
// the record can be stale.
func resolveTTL(explicit int64, existingTTL int64, intervalMinutes int) int64 {
	if explicit != 0 {
		return explicit
	}
	// A TTL outside Netlify's range would be rejected when the record is recreated, so it is
	// treated as if the record had none.
	if existingTTL >= minTTL && existingTTL <= maxTTL {
		return existingTTL
	}

	ttl := int64(intervalMinutes) * 60
	if ttl < minTTL {
		return minTTL
	}
	if ttl > maxTTL {
		return maxTTL
	}
	return ttl
}
