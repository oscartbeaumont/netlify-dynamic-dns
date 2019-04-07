package pkg

import (
	"net/http"
	"strings"

	"github.com/oscartbeaumont/netlify-dynamic-dns/internal"
	"github.com/pkg/errors"
)

// DeleteRecord deletes the specified record. Please note the specified record must contain a valid 'ID' field.
func DeleteRecord(domain string, accessToken string, record DNSRecord) error {
	// Construct The Request Data
	endpoint := "https://api.netlify.com/api/v1/dns_zones/" + strings.Replace(domain, ".", "_", -1) + "/dns_records/" + record.ID + "?access_token=" + accessToken
	req, err := http.NewRequest("DELETE", endpoint, nil)
	if err != nil {
		return errors.Wrap(err, "error: creating the 'DELETE' http request")
	}

	// Do The Request
	resp, err := internal.Client.Do(req)
	if err != nil {
		return errors.Wrap(err, "error: sending the http request to the Netlify api")
	}
	defer resp.Body.Close()

	// Handle The Response
	if resp.StatusCode == http.StatusNotFound {
		return errors.New("error: the domain you requested could not be found on your account")
	} else if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("error: unauthorized Netlify credentials. Please check your api key")
	} else if resp.StatusCode != http.StatusOK {
		return errors.New("error: something failed. The server returned the status " + resp.Status)
	}
	return nil
}
