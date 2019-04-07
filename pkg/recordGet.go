package pkg

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/oscartbeaumont/netlify-dynamic-dns/internal"
	"github.com/pkg/errors"
)

// GetRecords returns an array of all records for a domain
func GetRecords(domain string, accessToken string) ([]DNSRecord, error) {
	// Construct The Request Data
	endpoint := "https://api.netlify.com/api/v1/dns_zones/" + strings.Replace(domain, ".", "_", -1) + "/dns_records?access_token=" + accessToken

	// Do The Request
	resp, err := internal.Client.Get(endpoint)
	if err != nil {
		return nil, errors.Wrap(err, "error: sending the http request to the Netlify api")
	}
	defer resp.Body.Close()

	// Handle The Response
	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("error: the domain you requested could not be found on your account")
	} else if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("error: unauthorized Netlify credentials. Please check your api key")
	} else if resp.StatusCode != http.StatusOK {
		return nil, errors.New("error: something failed. The server returned the status " + resp.Status)
	}

	var output = []DNSRecord{}
	err = json.NewDecoder(resp.Body).Decode(&output)
	if err != nil {
		return nil, err
	}

	return output, nil
}
