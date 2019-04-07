package pkg

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/oscartbeaumont/netlify-dynamic-dns/internal"
	"github.com/pkg/errors"
)

// AddRecord creates a new record
func AddRecord(domain string, accessToken string, record DNSRecord) error {
	// Construct The Request Data
	endpoint := "https://api.netlify.com/api/v1/dns_zones/" + strings.Replace(domain, ".", "_", -1) + "/dns_records?access_token=" + accessToken
	marshalledRecord, err := json.Marshal(record)
	if err != nil {
		return errors.Wrap(err, "error: marshalling the DNS record")
	}

	// Do The Request
	resp, err := internal.Client.Post(endpoint, "application/json", bytes.NewBuffer(marshalledRecord))
	if err != nil {
		return errors.Wrap(err, "error: sending the http request to the Netlify api")
	}
	defer resp.Body.Close()

	// Handle The Response
	if resp.StatusCode == http.StatusNotFound {
		return errors.New("error: the domain you requested could not be found on your account")
	} else if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("error: unauthorized Netlify credentials. Please check your api key")
	} else if resp.StatusCode != http.StatusCreated {
		return errors.New("error: something failed. The server returned the status " + resp.Status)
	}
	return nil
}
