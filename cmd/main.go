package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/alexflint/go-arg"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	"github.com/janeczku/go-spinner"
	"github.com/netlify/open-api/go/models"
	"github.com/netlify/open-api/go/plumbing/operations"
	"github.com/netlify/open-api/go/porcelain"
	"github.com/oscartbeaumont/netlify-dynamic-dns/internal/analytics"
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
	arg.MustParse(&args)
	args.zoneID = strings.ReplaceAll(args.Zone, ".", "_")
	args.recordHostname = args.Record + "." + args.Zone

	var lastAnalyticsReport time.Time
	var forBreak = true
	for forBreak {
		var analyticsChan = make(chan *struct{}, 1)
		if lastAnalyticsReport.IsZero() || time.Since(lastAnalyticsReport) > time.Hour*24 {
			go func() {
				if err := analytics.Report(Version); err != nil {
					fmt.Println(Red+"Error reporting anonymous analytics data: ", err, Reset)
				} else {
					lastAnalyticsReport = time.Now()
				}

				analyticsChan <- nil
			}()
		} else {
			analyticsChan <- nil
		}

		s := spinner.StartNew("Updating DNS record")
		err := doUpdate()
		<-analyticsChan
		s.Stop()

		if err != nil {
			log.Println(Red + "Error: Error Updating DNS Record " + err.Error() + Reset)
		} else if args.Interval == 0 {
			log.Println(Green + "DNS records updated successfully." + Reset + "")
			forBreak = false
		} else {
			log.Println(Green + "DNS records updated successfully. Next update in " + strconv.Itoa(args.Interval) + " minutes" + Reset + "")
			time.Sleep(time.Duration(args.Interval) * time.Minute)
		}
	}
}

// doUpdate updates the DNS records with the public IP address
func doUpdate() error {
	// Get the Public IP
	ipv4, err := ipProvider.GetIPv4()
	if err != nil {
		return fmt.Errorf("error retrieving your public ipv4 address: %w", err)
	}

	var ipv6 string
	if args.IPv6 {
		ipv6, err = ipProvider.GetIPv6()
		if err != nil {
			return fmt.Errorf("error retrieving your public ipv6 address: %w", err)
		}
	}

	getparams := operations.NewGetDNSRecordsParams()
	getparams.ZoneID = args.zoneID
	resp, err := netlify.Operations.GetDNSRecords(getparams, netlifyAuth)
	if err != nil {
		errr, apiError := err.(*operations.GetDNSRecordsDefault)
		if apiError && errr.Code() == http.StatusUnauthorized {
			log.Fatalln("\r" + Red + "Fatal Error: Netlify API access token unauthorised" + Reset + "")
		} else {
			return fmt.Errorf("error retrieving your records from Netlify DNS: %w", err)
		}
	}

	// Get existing records if they exist
	var existingARecord *models.DNSRecord
	var existingAAAARecord *models.DNSRecord
	for _, record := range resp.Payload {
		if record.Hostname == args.recordHostname {
			if record.Type == "A" {
				existingARecord = record
			} else if record.Type == "AAAA" {
				existingAAAARecord = record
			}
		}
	}

	// Delete existing records if they exist (Netlify DNS API has no update feature)
	if existingARecord != nil {
		deleteparams := operations.NewDeleteDNSRecordParams()
		deleteparams.ZoneID = existingARecord.DNSZoneID
		deleteparams.DNSRecordID = existingARecord.ID
		if _, err := netlify.Operations.DeleteDNSRecord(deleteparams, netlifyAuth); err != nil {
			return fmt.Errorf("error deleting existing record from Netlify DNS: %w", err)
		}
	}

	if existingAAAARecord != nil {
		deleteparams := operations.NewDeleteDNSRecordParams()
		deleteparams.ZoneID = existingAAAARecord.DNSZoneID
		deleteparams.DNSRecordID = existingAAAARecord.ID
		if _, err := netlify.Operations.DeleteDNSRecord(deleteparams, netlifyAuth); err != nil {
			return fmt.Errorf("error deleting existing record from Netlify DNS: %w", err)
		}
	}

	// Create new record
	var ipv4Record = &models.DNSRecordCreate{
		Hostname: args.recordHostname,
		Type:     "A",
		Value:    ipv4,
	}
	if existingARecord != nil {
		ipv4Record.TTL = existingARecord.TTL
	}

	createparams := operations.NewCreateDNSRecordParams()
	createparams.ZoneID = args.zoneID
	createparams.DNSRecord = ipv4Record
	if _, err := netlify.Operations.CreateDNSRecord(createparams, netlifyAuth); err != nil {
		return fmt.Errorf("error creating new DNS record on Netlify DNS: %w", err)
	}

	if args.IPv6 {
		var ipv6Record = &models.DNSRecordCreate{
			Hostname: args.recordHostname,
			Type:     "AAAA",
			Value:    ipv6,
		}
		if existingAAAARecord != nil {
			ipv6Record.TTL = existingAAAARecord.TTL
		}

		create2params := operations.NewCreateDNSRecordParams()
		create2params.ZoneID = args.zoneID
		create2params.DNSRecord = ipv6Record
		if _, err := netlify.Operations.CreateDNSRecord(create2params, netlifyAuth); err != nil {
			return fmt.Errorf("error creating new DNS record on Netlify DNS: %w", err)
		}
	}

	return nil
}
