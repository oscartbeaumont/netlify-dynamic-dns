package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	flgDomain      = flag.String("domain", os.Getenv("DOMAIN"), "your domain name. eg. 'example.com'")
	flgHost        = flag.String("host", os.Getenv("HOST"), "the record name. eg. 'home'")
	flgAccessToken = flag.String("access-token", os.Getenv("ACCESS_TOKEN"), "the authentication token for the Netlify API.")
	flgInterval    = flag.Int("interval", 5, "the time in minutes between checking for DNS changes. Default is 5 minutes.")
)

func main() {
	flag.Parse()

	if *flgDomain == "" || *flgHost == "" || *flgAccessToken == "" {
		fmt.Println("You are required to set the domain, host and access_token. Refer to './netlify-dynamic-dns -help' for more information!")
		return
	}
	if *flgInterval == 5 && os.Getenv("INTERVAL") != "" {
		i, err := strconv.Atoi(os.Getenv("INTERVAL"))
		if err != nil {
			log.Println("Error Setting The INTERVAL Environment Varible.", err)
		} else {
			*flgInterval = i
		}
	}

	for {
		doDNSUpdate()
		time.Sleep(time.Duration(*flgInterval) * time.Minute)
	}
}

func doDNSUpdate() {
	ipv4, ipv6, err := getPublicIP()
	if err != nil {
		log.Fatalln(err, "Error Getting The Clients Public IP Address")
	}
	log.Println("Retrieved Your Public Ip's -", "IPv4:", ipv4, "IPv6:", ipv6)

	dnsRecords, err := getDNSRecords(*flgDomain, *flgAccessToken)
	if err != nil {
		log.Fatalln(err, "Error Retrieving The Netlify DNS Entrys")
	}

	ipv4Correct := false
	ipv6Correct := false

	for _, r := range dnsRecords {
		if r.Hostname == *flgHost+"."+*flgDomain {
			if r.Type == "A" {
				if r.Value == ipv4 && !ipv4Correct {
					ipv4Correct = true
				} else {
					log.Println("Deleting The Incorrect or Duplicate IPv4 DNS Record!", r)
					deleteDNSRecord(*flgDomain, *flgAccessToken, r.ID)
				}
			} else if r.Type == "AAAA" {
				if r.Value == ipv6 && !ipv6Correct {
					ipv6Correct = true
				} else {
					log.Println("Deleting The Incorrect or Duplicate IPv6 DNS Record!", r)
					deleteDNSRecord(*flgDomain, *flgAccessToken, r.ID)
				}
			}
		}
	}

	if ipv4Correct && ipv6Correct {
		log.Println("DNS Records Are Correct. No Changes Are Being Made.")
		return
	}

	if !ipv4Correct {
		log.Println("Updating The IPv4 DNS Record!")

		err = updateDNSRecord(*flgDomain, *flgAccessToken, netlifyDNSRecord{
			Type:     "A",
			Hostname: *flgHost,
			Value:    ipv4,
		})
		if err != nil {
			log.Fatalln(err, "Error Updating The Netlify DNS Entry")
		}
	}

	if !ipv6Correct {
		log.Println("Updating The IPv6 DNS Record!")

		err = updateDNSRecord(*flgDomain, *flgAccessToken, netlifyDNSRecord{
			Type:     "AAAA",
			Hostname: *flgHost,
			Value:    ipv6,
		})
		if err != nil {
			log.Fatalln(err, "Error Updating The Netlify DNS Entry")
		}
	}
}

func newHTTPClient() *http.Client {
	return &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			Dial: (&net.Dialer{
				Timeout: 5 * time.Second,
			}).Dial,
			TLSHandshakeTimeout: 5 * time.Second,
		},
	}
}
