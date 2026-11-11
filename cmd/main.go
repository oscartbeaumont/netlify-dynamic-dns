package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/netlify/open-api/go/models"
	"github.com/netlify/open-api/go/plumbing/operations"
	"github.com/netlify/open-api/go/porcelain"
	"github.com/oscartbeaumont/netlify-dynamic-dns/internal/publicip"
)

var args Arguments
var netlify = porcelain.NewRetryable(porcelain.Default.Transport, nil, porcelain.DefaultRetryAttempts)
var netlifyAuth = runtime.ClientAuthInfoWriterFunc(func(r runtime.ClientRequest, _ strfmt.Registry) error {
	if err := r.SetHeaderParam("User-Agent", "NetlifyDDNS"); err != nil {
		return err
	}
	if err := r.SetHeaderParam("Authorization", "Bearer "+args.AccessToken); err != nil {
		return err
	}
	return nil
})
var ipProvider publicip.Provider = publicip.OpenDNSProvider{}

func main() {
	validation := arg.MustParse(&args)
	trimInputs(&args)

	if !args.UpdateRootRecord && args.Record == "" {
		validation.Fail("Either --record or --updaterootrecord must be provided")
	}
	if err := validateZone(args.Zone); err != nil {
		validation.Fail(fmt.Sprintf("Invalid --zone: %v", err))
	}
	if !args.UpdateRootRecord {
		if err := validateRecord(args.Record, args.Zone); err != nil {
			validation.Fail(fmt.Sprintf("Invalid --record: %v", err))
		}
	}
	if args.TTL != 0 && args.TTL < minTTL {
		validation.Fail(fmt.Sprintf("Invalid --ttl: must be at least %d seconds", minTTL))
	}
	if args.TTL > maxTTL {
		validation.Fail(fmt.Sprintf("Invalid --ttl: must be at most %d seconds (one day)", maxTTL))
	}
	if args.Interval < 0 {
		validation.Fail("Invalid --interval: must not be negative")
	}
	if args.Interval > maxIntervalMinutes {
		validation.Fail(fmt.Sprintf("Invalid --interval: must be at most %d minutes", maxIntervalMinutes))
	}

	args.zoneID = strings.ReplaceAll(args.Zone, ".", "_")
	if args.UpdateRootRecord {
		args.recordHostname = args.Zone
	} else {
		args.recordHostname = args.Record + "." + args.Zone
	}

	if err := run(); err != nil {
		if errors.Is(err, errUnauthorized) {
			log.Println(Red + "Fatal error: " + err.Error() + Reset)
		} else {
			log.Println(Red + "Error updating DNS records: " + err.Error() + Reset)
		}
		os.Exit(1)
	}
}

// run performs the first update and then keeps updating at the configured interval. A failure on
// the first update is returned so a misconfiguration exits with an error instead of retrying
// forever, as is a rejected access token. Later failures are logged and retried.
func run() error {
	first := true
	// A daemon updating every few minutes would otherwise repeat the same warning forever.
	seenWarnings := make(map[string]struct{})
	for {
		s := startSpinner("Updating DNS records")
		warnings, err := doUpdate()
		s.Stop()
		for _, warning := range warnings {
			if _, seen := seenWarnings[warning]; seen {
				continue
			}
			seenWarnings[warning] = struct{}{}
			log.Println(warning)
		}

		switch {
		case err != nil && (first || errors.Is(err, errUnauthorized)):
			return err
		case err != nil:
			log.Println(Red + "Error updating DNS records: " + err.Error() + Reset)
		case args.Interval == 0:
			log.Println(Green + "DNS records updated successfully." + Reset)
			return nil
		default:
			log.Println(Green + "DNS records updated successfully. Next update in " + strconv.Itoa(args.Interval) + " minutes" + Reset)
		}

		first = false
		time.Sleep(time.Duration(args.Interval) * time.Minute)
	}
}

// publicIPs holds the addresses used for a single update. ipv6 is empty when the AAAA record is
// not being updated.
type publicIPs struct {
	ipv4     string
	ipv6     string
	warnings []string
}

// resolvePublicIPs looks up the public addresses to publish. IPv6 is best effort: a machine
// without IPv6 connectivity (common on residential connections and on hosts with only a link
// local address) must still get its A record updated, so a failed lookup only disables the AAAA
// record for this update.
func resolvePublicIPs(provider publicip.Provider, wantIPv6 bool) (publicIPs, error) {
	var ips publicIPs

	var err error
	if ips.ipv4, err = provider.GetIPv4(); err != nil {
		return ips, fmt.Errorf("error retrieving your public ipv4 address: %w", err)
	}

	if !wantIPv6 {
		return ips, nil
	}

	if ips.ipv6, err = provider.GetIPv6(); err != nil {
		ips.ipv6 = ""
		ips.warnings = append(ips.warnings, "Warning: skipping the IPv6 (AAAA) record: "+err.Error()+". Use --ipv6=false to silence this.")
	}

	return ips, nil
}

// recordsToCreate builds the records which publish ips. Both record types share the TTL rules,
// and the TTL of the record being replaced is passed in so a value configured in the Netlify
// dashboard survives.
func recordsToCreate(ips publicIPs, hostname string, explicitTTL int64, existing []*models.DNSRecord, intervalMinutes int) []*models.DNSRecordCreate {
	records := []*models.DNSRecordCreate{{
		Hostname: hostname,
		Type:     "A",
		Value:    ips.ipv4,
		TTL:      recordTTLFor(explicitTTL, existing, hostname, "A", intervalMinutes),
	}}

	if ips.ipv6 != "" {
		records = append(records, &models.DNSRecordCreate{
			Hostname: hostname,
			Type:     "AAAA",
			Value:    ips.ipv6,
			TTL:      recordTTLFor(explicitTTL, existing, hostname, "AAAA", intervalMinutes),
		})
	}

	return records
}

// doUpdate updates the DNS records with the public IP address. Problems that do not stop the
// update are returned as warnings so they can be logged after the spinner has been cleared.
func doUpdate() ([]string, error) {
	ips, err := resolvePublicIPs(ipProvider, args.IPv6)
	if err != nil {
		return ips.warnings, err
	}
	warnings := ips.warnings
	updateIPv6 := ips.ipv6 != ""

	getparams := operations.NewGetDNSRecordsParams()
	getparams.ZoneID = args.zoneID
	resp, err := netlify.Operations.GetDNSRecords(getparams, netlifyAuth)
	if err != nil {
		if isUnauthorized(err) {
			return warnings, unauthorizedError(args.Zone)
		}
		return warnings, fmt.Errorf("error retrieving your records from Netlify DNS: %w", describeAPIError(err, "", "", ""))
	}

	// Only touch the types being replaced, so disabling IPv6 leaves any AAAA records alone.
	var existing []*models.DNSRecord
	for _, record := range recordsToDelete(resp.Payload, args.recordHostname) {
		if record.Type == "AAAA" && !updateIPv6 {
			continue
		}
		existing = append(existing, record)
	}

	// Netlify's DNS API cannot update a record in place, so the record is recreated. The new
	// records are created before the old ones are deleted: if a create fails the hostname keeps
	// serving its previous address instead of going blank. The duplicates this leaves behind are
	// deleted below, or by the next update if that delete fails.
	for _, record := range recordsToCreate(ips, args.recordHostname, args.TTL, resp.Payload, args.Interval) {
		createparams := operations.NewCreateDNSRecordParams()
		createparams.ZoneID = args.zoneID
		createparams.DNSRecord = record
		if _, err := netlify.Operations.CreateDNSRecord(createparams, netlifyAuth); err != nil {
			if isUnauthorized(err) {
				return warnings, unauthorizedError(args.Zone)
			}
			return warnings, fmt.Errorf("error creating new DNS record: %w", describeAPIError(err, record.Hostname, record.Type, ""))
		}
	}

	for _, record := range existing {
		deleteparams := operations.NewDeleteDNSRecordParams()
		deleteparams.ZoneID = record.DNSZoneID
		deleteparams.DNSRecordID = record.ID
		if _, err := netlify.Operations.DeleteDNSRecord(deleteparams, netlifyAuth); err != nil {
			if isUnauthorized(err) {
				return warnings, unauthorizedError(args.Zone)
			}
			// The new records are already live, so a failed delete is a duplicate record rather
			// than a broken hostname (issue #18). It is retried by the next update.
			warnings = append(warnings, "Warning: could not delete the old DNS record, it will be retried on the next update: "+
				describeAPIError(err, record.Hostname, record.Type, record.ID).Error())
		}
	}

	return warnings, nil
}
