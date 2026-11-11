package main

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/netlify/open-api/go/models"
	"github.com/netlify/open-api/go/plumbing/operations"
)

func TestResolveTTL(t *testing.T) {
	tests := []struct {
		name     string
		explicit int64
		existing int64
		interval int
		wantTTL  int64
	}{
		{name: "explicit wins over existing", explicit: 120, existing: 670, interval: 5, wantTTL: 120},
		{name: "explicit minimum", explicit: 60, interval: 5, wantTTL: 60},
		{name: "existing record wins", existing: 670, interval: 5, wantTTL: 670},
		{name: "existing record at the minimum", existing: 60, interval: 5, wantTTL: 60},
		{name: "existing record below the minimum is ignored", existing: 30, interval: 5, wantTTL: 300},
		{name: "existing record above the maximum is ignored", existing: 90000, interval: 5, wantTTL: 300},
		{name: "existing record with no TTL falls back", interval: 5, wantTTL: 300},
		{name: "derived from interval", interval: 5, wantTTL: 300},
		{name: "single update", interval: 0, wantTTL: 60},
		{name: "derived clamped to minimum", interval: 1, wantTTL: 60},
		{name: "derived within range", interval: 600, wantTTL: 36000},
		{name: "derived at maximum", interval: 1440, wantTTL: 86400},
		{name: "derived clamped to maximum", interval: 2000, wantTTL: 86400},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := resolveTTL(test.explicit, test.existing, test.interval); got != test.wantTTL {
				t.Fatalf("resolveTTL(%d, %d, %d) = %d, want %d", test.explicit, test.existing, test.interval, got, test.wantTTL)
			}
		})
	}
}

func TestRecordTTLFor(t *testing.T) {
	payload := []*models.DNSRecord{
		{ID: "a1", Hostname: "home.example.com", Type: "A", TTL: 670},
		{ID: "aaaa1", Hostname: "home.example.com", Type: "AAAA", TTL: 900},
		{ID: "a2", Hostname: "other.example.com", Type: "A", TTL: 1200},
	}

	// Each record type keeps the TTL of the record it replaces.
	if got := recordTTLFor(0, payload, "home.example.com", "A", 5); got != 670 {
		t.Fatalf("A TTL = %d, want the existing 670", got)
	}
	if got := recordTTLFor(0, payload, "home.example.com", "AAAA", 5); got != 900 {
		t.Fatalf("AAAA TTL = %d, want the existing 900", got)
	}

	// An explicit TTL overrides both, and a type with no existing record falls back to the interval.
	if got := recordTTLFor(120, payload, "home.example.com", "AAAA", 5); got != 120 {
		t.Fatalf("AAAA TTL with --ttl = %d, want 120", got)
	}
	if got := recordTTLFor(0, nil, "home.example.com", "A", 5); got != 300 {
		t.Fatalf("A TTL without an existing record = %d, want 300", got)
	}
}

func TestTrimInputs(t *testing.T) {
	a := Arguments{
		AccessToken: "  token\n",
		Zone:        " neugartf.com ",
		Record:      "\thome ",
	}
	trimInputs(&a)

	if a.AccessToken != "token" || a.Zone != "neugartf.com" || a.Record != "home" {
		t.Fatalf("trimInputs left %#v", a)
	}
	if err := validateZone(a.Zone); err != nil {
		t.Fatalf("trimmed zone is invalid: %v", err)
	}
	if err := validateRecord(a.Record, a.Zone); err != nil {
		t.Fatalf("trimmed record is invalid: %v", err)
	}
}

func TestValidateZone(t *testing.T) {
	tests := []struct {
		name    string
		zone    string
		wantErr bool
	}{
		{name: "simple domain", zone: "example.com"},
		{name: "subdomain", zone: "dns.example.com"},
		{name: "hyphens", zone: "my-zone.example.com"},
		{name: "digits", zone: "123.example.com"},
		{name: "punycode", zone: "xn--bcher-kva.example"},
		{name: "leading space", zone: " neugartf.com", wantErr: true},
		{name: "trailing space", zone: "neugartf.com ", wantErr: true},
		{name: "internal space", zone: "neugartf .com", wantErr: true},
		{name: "scheme", zone: "https://example.com", wantErr: true},
		{name: "path", zone: "example.com/dns", wantErr: true},
		{name: "no dot", zone: "example", wantErr: true},
		{name: "empty", zone: "", wantErr: true},
		{name: "leading dot", zone: ".example.com", wantErr: true},
		{name: "trailing dot", zone: "example.com.", wantErr: true},
		{name: "double dot", zone: "example..com", wantErr: true},
		{name: "leading hyphen", zone: "-example.com", wantErr: true},
		{name: "trailing hyphen", zone: "example-.com", wantErr: true},
		{name: "uppercase", zone: "example.COM", wantErr: true},
		{name: "underscore", zone: "example_.com", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateZone(test.zone)
			if test.wantErr && err == nil {
				t.Fatalf("validateZone(%q) = nil, want an error", test.zone)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validateZone(%q) = %v, want nil", test.zone, err)
			}
		})
	}
}

func TestValidateRecord(t *testing.T) {
	tests := []struct {
		name    string
		record  string
		wantErr bool
	}{
		{name: "single label", record: "home"},
		{name: "multiple labels", record: "nddns.test-03"},
		{name: "digits", record: "123"},
		{name: "hyphens", record: "my-home"},
		{name: "leading space", record: " home", wantErr: true},
		{name: "trailing space", record: "home ", wantErr: true},
		{name: "internal space", record: "my home", wantErr: true},
		{name: "wildcard", record: "*", wantErr: true},
		{name: "wildcard label", record: "*.home", wantErr: true},
		{name: "full hostname", record: "home.example.com", wantErr: true},
		{name: "zone as record", record: "example.com", wantErr: true},
		{name: "empty", record: "", wantErr: true},
		{name: "leading dot", record: ".home", wantErr: true},
		{name: "trailing dot", record: "home.", wantErr: true},
		{name: "double dot", record: "home..sub", wantErr: true},
		{name: "leading hyphen", record: "-home", wantErr: true},
		{name: "trailing hyphen", record: "home-", wantErr: true},
		{name: "uppercase", record: "HOME", wantErr: true},
		{name: "underscore", record: "my_home", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateRecord(test.record, "example.com")
			if test.wantErr && err == nil {
				t.Fatalf("validateRecord(%q) = nil, want an error", test.record)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("validateRecord(%q) = %v, want nil", test.record, err)
			}
		})
	}
}

func TestRecordsToDelete(t *testing.T) {
	payload := []*models.DNSRecord{
		nil,
		{ID: "a1", Hostname: "home.example.com", Type: "A"},
		{ID: "a1", Hostname: "home.example.com", Type: "A"},
		{ID: "aaaa1", Hostname: "home.example.com", Type: "AAAA"},
		{ID: "aaaa1", Hostname: "home.example.com", Type: "AAAA"},
		{ID: "a2", Hostname: "other.example.com", Type: "A"},
		{ID: "txt1", Hostname: "home.example.com", Type: "TXT"},
		{ID: "", Hostname: "home.example.com", Type: "A"},
		{ID: "", Hostname: "home.example.com", Type: "A"},
	}

	got := recordsToDelete(payload, "home.example.com")
	want := map[string]int{"a1": 1, "aaaa1": 1}
	if len(got) != len(want) {
		t.Fatalf("recordsToDelete returned %d records, want %d: %v", len(got), len(want), got)
	}
	for _, record := range got {
		if want[record.ID] == 0 {
			t.Fatalf("unexpected record in result: %+v", record)
		}
		want[record.ID]--
	}
	for id, remaining := range want {
		if remaining != 0 {
			t.Fatalf("record with id %q was not returned the expected number of times", id)
		}
	}

	if got := recordsToDelete(payload, "missing.example.com"); len(got) != 0 {
		t.Fatalf("recordsToDelete for an unknown hostname returned %v, want nothing", got)
	}
	if got := recordsToDelete(nil, "home.example.com"); len(got) != 0 {
		t.Fatalf("recordsToDelete for nil payload returned %v, want nothing", got)
	}
}

func TestRecordsToCreate(t *testing.T) {
	existing := []*models.DNSRecord{
		{ID: "a1", Hostname: "home.example.com", Type: "A", TTL: 670},
		{ID: "aaaa1", Hostname: "home.example.com", Type: "AAAA", TTL: 900},
	}

	records := recordsToCreate(
		publicIPs{ipv4: "1.2.3.4", ipv6: "2001:db8::1"},
		"home.example.com", 0, existing, 5,
	)
	if len(records) != 2 {
		t.Fatalf("recordsToCreate returned %d records, want an A and an AAAA record", len(records))
	}
	if records[0].Type != "A" || records[0].Value != "1.2.3.4" || records[0].TTL != 670 {
		t.Fatalf("A record = %+v, want the ipv4 address with the existing 670 TTL", records[0])
	}
	if records[1].Type != "AAAA" || records[1].Value != "2001:db8::1" || records[1].TTL != 900 {
		t.Fatalf("AAAA record = %+v, want the ipv6 address with the existing 900 TTL", records[1])
	}

	// Without an IPv6 address only the A record is published, and an explicit TTL wins.
	records = recordsToCreate(publicIPs{ipv4: "1.2.3.4"}, "home.example.com", 120, nil, 5)
	if len(records) != 1 {
		t.Fatalf("recordsToCreate returned %d records, want only an A record", len(records))
	}
	if records[0].Type != "A" || records[0].TTL != 120 {
		t.Fatalf("A record = %+v, want the explicit 120 TTL", records[0])
	}
}

func TestFirstRecordOfType(t *testing.T) {
	payload := []*models.DNSRecord{
		{ID: "a1", Hostname: "home.example.com", Type: "A", TTL: 670},
		{ID: "a2", Hostname: "home.example.com", Type: "A", TTL: 900},
	}

	record := firstRecordOfType(payload, "home.example.com", "A")
	if record == nil || record.ID != "a1" {
		t.Fatalf("firstRecordOfType returned %+v, want the first A record", record)
	}
	if got := recordTTL(record); got != 670 {
		t.Fatalf("recordTTL = %d, want 670", got)
	}
	if got := firstRecordOfType(payload, "home.example.com", "AAAA"); got != nil {
		t.Fatalf("firstRecordOfType returned %+v, want nil", got)
	}
	if got := recordTTL(nil); got != 0 {
		t.Fatalf("recordTTL(nil) = %d, want 0", got)
	}
}

func TestDescribeAPIError(t *testing.T) {
	t.Run("delete response", func(t *testing.T) {
		apiErr := operations.NewDeleteDNSRecordDefault(http.StatusInternalServerError)
		apiErr.Payload = &models.Error{Message: "default"}

		got := describeAPIError(apiErr, "home.example.com", "A", "abc123").Error()
		for _, want := range []string{"HTTP 500", "default", "A record", "home.example.com", "abc123"} {
			if !strings.Contains(got, want) {
				t.Fatalf("error %q does not contain %q", got, want)
			}
		}
	})

	t.Run("create response with payload code", func(t *testing.T) {
		apiErr := operations.NewCreateDNSRecordDefault(http.StatusBadRequest)
		apiErr.Payload = &models.Error{Code: 422, Message: "invalid ttl"}

		got := describeAPIError(apiErr, "home.example.com", "AAAA", "").Error()
		for _, want := range []string{"HTTP 400", "invalid ttl", "code 422", "AAAA record", "home.example.com"} {
			if !strings.Contains(got, want) {
				t.Fatalf("error %q does not contain %q", got, want)
			}
		}
	})

	t.Run("missing payload", func(t *testing.T) {
		apiErr := operations.NewDeleteDNSRecordDefault(http.StatusServiceUnavailable)

		got := describeAPIError(apiErr, "home.example.com", "A", "abc123").Error()
		if !strings.Contains(got, "HTTP 503") {
			t.Fatalf("error %q does not contain the HTTP status", got)
		}
	})

	t.Run("non api error is unchanged", func(t *testing.T) {
		original := errors.New("connection refused")
		got := describeAPIError(original, "home.example.com", "A", "abc123")
		if got != original {
			t.Fatalf("describeAPIError = %v, want the original error", got)
		}
	})
}

func TestUnauthorizedError(t *testing.T) {
	unauthorized := operations.NewGetDNSRecordsDefault(http.StatusUnauthorized)
	if !isUnauthorized(unauthorized) {
		t.Fatal("isUnauthorized(401) = false, want true")
	}
	if !isUnauthorized(fmt.Errorf("get records: %w", unauthorized)) {
		t.Fatal("isUnauthorized(wrapped 401) = false, want true")
	}
	if isUnauthorized(operations.NewGetDNSRecordsDefault(http.StatusInternalServerError)) {
		t.Fatal("isUnauthorized(500) = true, want false")
	}
	if isUnauthorized(errors.New("connection refused")) {
		t.Fatal("isUnauthorized(non API error) = true, want false")
	}

	err := unauthorizedError("example.com")
	if !errors.Is(err, errUnauthorized) {
		t.Fatalf("unauthorizedError does not wrap errUnauthorized: %v", err)
	}
	if !strings.Contains(err.Error(), "example.com") {
		t.Fatalf("unauthorizedError does not name the zone: %v", err)
	}
}

// fakeIPProvider is a publicip.Provider with canned results.
type fakeIPProvider struct {
	ipv4, ipv6       string
	ipv4Err, ipv6Err error
	ipv6Calls        int
}

func (f *fakeIPProvider) GetIPv4() (string, error) { return f.ipv4, f.ipv4Err }
func (f *fakeIPProvider) GetIPv6() (string, error) {
	f.ipv6Calls++
	return f.ipv6, f.ipv6Err
}

func TestResolvePublicIPs(t *testing.T) {
	t.Run("both addresses", func(t *testing.T) {
		provider := &fakeIPProvider{ipv4: "1.2.3.4", ipv6: "2001:db8::1"}
		ips, err := resolvePublicIPs(provider, true)
		if err != nil {
			t.Fatalf("resolvePublicIPs returned %v", err)
		}
		if ips.ipv4 != "1.2.3.4" || ips.ipv6 != "2001:db8::1" {
			t.Fatalf("resolvePublicIPs returned %+v", ips)
		}
		if len(ips.warnings) != 0 {
			t.Fatalf("resolvePublicIPs returned warnings %v, want none", ips.warnings)
		}
	})

	t.Run("ipv6 disabled is never looked up", func(t *testing.T) {
		provider := &fakeIPProvider{ipv4: "1.2.3.4", ipv6: "2001:db8::1"}
		ips, err := resolvePublicIPs(provider, false)
		if err != nil {
			t.Fatalf("resolvePublicIPs returned %v", err)
		}
		if provider.ipv6Calls != 0 {
			t.Fatalf("GetIPv6 was called %d times, want 0", provider.ipv6Calls)
		}
		if ips.ipv6 != "" || len(ips.warnings) != 0 {
			t.Fatalf("resolvePublicIPs returned %+v, want only an IPv4 address", ips)
		}
	})

	t.Run("ipv6 failure keeps the ipv4 update alive", func(t *testing.T) {
		provider := &fakeIPProvider{ipv4: "1.2.3.4", ipv6Err: errors.New("network is unreachable")}
		ips, err := resolvePublicIPs(provider, true)
		if err != nil {
			t.Fatalf("resolvePublicIPs returned %v, want the update to continue", err)
		}
		if ips.ipv4 != "1.2.3.4" || ips.ipv6 != "" {
			t.Fatalf("resolvePublicIPs returned %+v", ips)
		}
		if len(ips.warnings) != 1 || !strings.Contains(ips.warnings[0], "network is unreachable") {
			t.Fatalf("resolvePublicIPs returned warnings %v, want one mentioning the cause", ips.warnings)
		}
	})

	t.Run("ipv4 failure is fatal", func(t *testing.T) {
		provider := &fakeIPProvider{ipv4Err: errors.New("no route to host"), ipv6: "2001:db8::1"}
		ips, err := resolvePublicIPs(provider, true)
		if err == nil || !strings.Contains(err.Error(), "no route to host") {
			t.Fatalf("resolvePublicIPs returned %v, want an error", err)
		}
		if provider.ipv6Calls != 0 {
			t.Fatalf("GetIPv6 was called %d times after a failed IPv4 lookup, want 0", provider.ipv6Calls)
		}
		if len(ips.warnings) != 0 {
			t.Fatalf("resolvePublicIPs returned warnings %v, want none", ips.warnings)
		}
	})
}
