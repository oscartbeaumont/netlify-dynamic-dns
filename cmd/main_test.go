package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/netlify/open-api/go/models"
	"github.com/netlify/open-api/go/plumbing/operations"
)

var myIPv4 string
var myIPv6 string

var (
	buildOnce sync.Once
	buildDir  string
	buildPath string
	buildErr  error
)

// TestMain sets up and shuts down the testing environment
func TestMain(m *testing.M) {
	code := runTests(m)
	if buildDir != "" {
		_ = os.RemoveAll(buildDir)
	}
	os.Exit(code)
}

// runTests prepares the integration environment when the Netlify credentials are available and
// runs the tests. Without the credentials the integration tests skip themselves while the unit
// tests still run.
func runTests(m *testing.M) int {
	if os.Getenv("NDDNS_TEST_ACCESS_TOKEN") == "" || os.Getenv("NDDNS_TEST_ZONE") == "" {
		return m.Run()
	}

	args.AccessToken = os.Getenv("NDDNS_TEST_ACCESS_TOKEN")
	args.zoneID = strings.ReplaceAll(os.Getenv("NDDNS_TEST_ZONE"), ".", "_")

	// Get the Public IP
	var err error
	if myIPv4, err = ipProvider.GetIPv4(); err != nil {
		log.Fatalf("error retrieving your public ipv4 address: %v", err)
	}
	if os.Getenv("NDDNS_IPv6_ENABLED") != "false" {
		if myIPv6, err = ipProvider.GetIPv6(); err != nil {
			log.Fatalf("error retrieving your public ipv6 address: %v", err)
		}
	}

	return m.Run()
}

// skipWithoutNetlify skips an integration test when the Netlify test credentials are not set.
func skipWithoutNetlify(t *testing.T) {
	t.Helper()
	if os.Getenv("NDDNS_TEST_ACCESS_TOKEN") == "" || os.Getenv("NDDNS_TEST_ZONE") == "" {
		t.Skip("NDDNS_TEST_ACCESS_TOKEN and NDDNS_TEST_ZONE are required for integration tests")
	}
}

// testBinary builds the command once and returns the path to the binary. Running the binary
// instead of `go run` reports the program's own exit status (`go run` always exits 1) and is a
// lot faster for the many argument combinations tested below.
func testBinary(t *testing.T) string {
	t.Helper()

	buildOnce.Do(func() {
		if buildDir, buildErr = os.MkdirTemp("", "nddns-test"); buildErr != nil {
			return
		}
		buildPath = filepath.Join(buildDir, "nddns")
		out, err := exec.Command("go", "build", "-o", buildPath, ".").CombinedOutput()
		if err != nil {
			buildErr = fmt.Errorf("go build failed: %v: %s", err, out)
		}
	})

	if buildErr != nil {
		t.Fatalf("could not build the test binary: %v", buildErr)
	}

	return buildPath
}

var invalidCommands = [][]string{
	{},
	{"--access-token=xxxx"},
	{"--zone=example.com"},
	{"--access-token=xxxx", "--zone=example.com"},
	{"--record=example.com"},
	{"--access-token=xxxx", "--record=example.com"},
	{"--zone=example.com", "--record=example.com"},
	{"--access-token=xxxx", "--zone=example com", "--record=home"},
	{"--access-token=xxxx", "--zone=https://example.com", "--record=home"},
	{"--access-token=xxxx", "--zone=example.com", "--record=home.example.com"},
	{"--access-token=xxxx", "--zone=example.com", "--record=home", "--ttl=59"},
	{"--access-token=xxxx", "--zone=example.com", "--record=home", "--ttl=86401"},
	{"--access-token=xxxx", "--zone=example.com", "--record=home", "--interval=-1"},
}

// TestInvalidArguments tests different combinations of arguments that are not valid
func TestInvalidArguments(t *testing.T) {
	binary := testBinary(t)

	for _, command := range invalidCommands {
		cmd := exec.Command(binary, command...)
		// go-arg prints the explanation to stdout, so capture both streams.
		output, err := cmd.CombinedOutput()

		if err == nil {
			t.Errorf("command %v succeeded, expecting a failure exit status. Output: %s", command, output)
		} else if !strings.Contains(string(output), "error:") {
			t.Errorf("command %v failed without an explanation. Output: %s", command, output)
		}
	}
}

// TestNormal tests the normal functionality of the project including TTL passthrough, multirecord based issues, subsubdomains
func TestNormal(t *testing.T) {
	skipWithoutNetlify(t)

	binary := testBinary(t)

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
		cmd := exec.Command(binary, "--accesstoken="+os.Getenv("NDDNS_TEST_ACCESS_TOKEN"), "--zone="+os.Getenv("NDDNS_TEST_ZONE"), "--record="+recordName)
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
		if err := cmd.Wait(); err != nil {
			t.Errorf("error running command: %v", err)
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
			if record.Hostname != recordHostname {
				continue
			}

			switch record.Type {
			case "A":
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
			case "AAAA":
				if foundAAAA {
					t.Fatalf("Duplicate 'AAAA' record found for hostname '%v'", recordHostname)
				}

				if myIPv6 == "" {
					// IPv6 updates are disabled for this run (CI sets NDDNS_IPv6_ENABLED=false),
					// so an existing AAAA record must be left exactly as it was.
					if record.Value != "0000:0000:0000:0000:0000:0000:0000:0000" {
						t.Fatalf("error existing 'AAAA' record was modified while IPv6 updates are disabled")
					}
				} else if record.Value != myIPv6 {
					t.Fatalf("error record value != your public IPv6 address")
				} else if recordName == "nddns-test-01" && record.TTL != 670 {
					t.Fatalf("error record has TTL of '%v' expected '%v' from existing record", record.TTL, 670)
				}

				foundAAAA = true
				recordsToRemove = append(recordsToRemove, record)
			}
		}

		if !foundA {
			t.Fatalf("error 'A' record was not found for domain '%v'", recordHostname)
		}
		if myIPv6 != "" && !foundAAAA {
			t.Fatalf("error 'AAAA' record was not found for domain '%v'", recordHostname)
		}
		if myIPv6 == "" && recordName == "nddns-test-01" && !foundAAAA {
			t.Fatalf("error existing 'AAAA' record was deleted while IPv6 updates are disabled")
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
