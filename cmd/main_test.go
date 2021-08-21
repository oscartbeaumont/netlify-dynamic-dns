package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/netlify/open-api/go/models"
	"github.com/netlify/open-api/go/plumbing/operations"
)

var commandBaseArgs = []string{"run", "./"}

var myIPv4 string
var myIPv6 string

// TestMain sets up and shuts down the testing environment
func TestMain(m *testing.M) {
	if os.Getenv("NDDNS_TEST_ACCESS_TOKEN") == "" {
		log.Fatalln("Error 'NDDNS_TEST_ACCESS_TOKEN' environment varible is required to run tests")
	}
	if os.Getenv("NDDNS_TEST_ZONE") == "" {
		log.Fatalln("Error 'NDDNS_TEST_ZONE' environment varible is required to run tests")
	}
	os.Setenv("NDDNS_DISABLE_ANALYTICS", "true")
	args.AccessToken = os.Getenv("NDDNS_TEST_ACCESS_TOKEN")
	args.zoneID = strings.ReplaceAll(os.Getenv("NDDNS_TEST_ZONE"), ".", "_")

	// Get the Public IP
	var err error
	if myIPv4, err = ipProvider.GetIPv4(); err != nil {
		log.Fatalln("error retrieving your public ipv4 address: %w", err)
	}
	if os.Getenv("NDDNS_IPv6_ENABLED") != "false" {
		if myIPv6, err = ipProvider.GetIPv6(); err != nil {
			log.Fatalln("error retrieving your public ipv6 address: %w", err)
		}
	}
	code := m.Run()
	os.Setenv("NDDNS_DISABLE_ANALYTICS", "")
	os.Exit(code)
}

var invalidCommands = [][]string{
	{},
	{"--access-token=xxxx"},
	{"--zone=example.com"},
	{"--access-token=xxxx", "--zone=example.com"},
	{"--record=example.com"},
	{"--access-token=xxxx", "--record=example.com"},
	{"--zone=example.com", "--record=example.com"},
}

// TestInvalidArguments tests different combinations of arguments that are not valid
func TestInvalidArguments(t *testing.T) {
	for i, command := range invalidCommands {
		cmd := exec.Command("go", append(commandBaseArgs, command...)...)
		if err := cmd.Run(); err == nil || err.Error() != "exit status 1" {
			t.Errorf("error invalid return status for command %v. Expecting failure exit status!", i)
		}
	}
}

// TestNormal tests the normal functionality of the project including TTL passthrough, multirecord based issues, subsubdomains
func TestNormal(t *testing.T) {
	// Create existing record 'nddns-test-01' with weird TTL to test passthrough
	for _, record := range [][]string{{"A", "0.0.0.0"}, {"AAAA", "0000:0000:0000:0000:0000:0000:0000:0000"}} {
		createparams := operations.NewCreateDNSRecordParams()
		createparams.ZoneID = args.zoneID
		createparams.DNSRecord = &models.DNSRecordCreate{
			Hostname: "nddns-test-01",
			Type:     record[0],
			Value:    record[1],
			TTL:      670,
		}
		if _, err := netlify.Operations.CreateDNSRecord(createparams, netlifyAuth); err != nil {
			t.Errorf("error creating new DNS record on Netlify DNS: %v", err)
		}
	}

	var recordsToRemove = []*models.DNSRecord{}
	for _, recordName := range []string{"nddns-test-01", "nddns-test-02", "nddns.test-03"} {
		cmd := exec.Command("go", append(commandBaseArgs, []string{"--accesstoken=" + os.Getenv("NDDNS_TEST_ACCESS_TOKEN"), "--zone=" + os.Getenv("NDDNS_TEST_ZONE"), "--record=" + recordName}...)...)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			t.Errorf("error creating command stderr pipe: %s", err)
		}
		if err := cmd.Start(); err != nil {
			t.Errorf("error running command: %s", err)
		}

		scanner := bufio.NewScanner(stderr)
		scanner.Scan()
		if !strings.Contains(scanner.Text(), "successfully") {
			t.Errorf("error running command: command didn't report success to terminal")
		}

		getparams := operations.NewGetDNSRecordsParams()
		getparams.ZoneID = args.zoneID
		resp, err := netlify.Operations.GetDNSRecords(getparams, netlifyAuth)
		if err != nil {
			errr, apiError := err.(*operations.GetDNSRecordsDefault)
			if apiError && errr.Code() == http.StatusUnauthorized {
				t.Fatal("error: Netlify API access token unauthorised")
			} else {
				t.Error(fmt.Errorf("error retrieving records from Netlify DNS: %w", err))
			}
		}

		var recordHostname = recordName + "." + os.Getenv("NDDNS_TEST_ZONE")
		var foundA bool
		var foundAAAA bool
		for _, record := range resp.Payload {
			if record.Hostname == recordHostname {
				if record.Type == "A" {
					if foundA {
						t.Fatalf("Duplicate 'A' record found for hostname '%v'", recordHostname)
					}

					if record.Value != myIPv4 {
						t.Fatalf("error record value != your public IPv4 address")
					} else if recordName == "nddns-test-01" && record.TTL != 670 {
						t.Fatalf("error record has TTL of '%v' expected '%v' from existing record", record.TTL, 670)
					}

					foundA = true
					recordsToRemove = append(recordsToRemove, record)
				} else if record.Type == "AAAA" {
					if foundAAAA {
						t.Fatalf("Duplicate 'AAAA' record found for hostname '%v'", recordHostname)
					}

					if record.Value != myIPv6 {
						t.Fatalf("error record value != your public IPv6 address")
					} else if recordName == "nddns-test-01" && record.TTL != 670 {
						t.Fatalf("error record has TTL of '%v' expected '%v' from existing record", record.TTL, 670)
					}

					foundAAAA = true
					recordsToRemove = append(recordsToRemove, record)
				}
			}
		}

		if !foundA {
			t.Fatalf("error 'A' record was not found for domain '%v'", recordHostname)
		} else if myIPv6 != "" && !foundAAAA {
			t.Fatalf("error 'AAAA' record was not found for domain '%v'", recordHostname)
		}
	}

	// Cleanup
	for _, record := range recordsToRemove {
		t.Log(record.Hostname)
		deleteparams := operations.NewDeleteDNSRecordParams()
		deleteparams.ZoneID = record.DNSZoneID
		deleteparams.DNSRecordID = record.ID
		if _, err := netlify.Operations.DeleteDNSRecord(deleteparams, netlifyAuth); err != nil {
			t.Errorf("error deleting record '%s' from Netlify DNS: %v", record.Hostname, err)
		}
	}
}
