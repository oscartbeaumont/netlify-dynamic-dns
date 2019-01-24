package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/pkg/errors"
)

type netlifyDNSRecord struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Hostname string `json:"hostname"`
	Value    string `json:"value"`
}

func getDNSRecords(domain string, accessToken string) ([]netlifyDNSRecord, error) {
	req, err := http.NewRequest(
		"GET",
		netlifyDNSEndpoint(domain, accessToken, ""),
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating the 'http.Request' for getDNSRecords()")
	}

	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var output = []netlifyDNSRecord{}
		err := json.NewDecoder(resp.Body).Decode(&output)
		if err != nil {
			return nil, err
		}
		return output, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("the domain you requested is not connected to your Netlify account")
	} else if resp.StatusCode == http.StatusUnauthorized {
		return nil, errors.New("unauthorized Netlify credentials. Please check your api key")
	}
	return nil, errors.New("something failed. the netlify api endpoint returned the http status code: " + resp.Status)
}

func deleteDNSRecord(domain, accessToken, recordID string) error {
	req, err := http.NewRequest(
		"DELETE",
		netlifyDNSEndpoint(domain, accessToken, recordID),
		nil,
	)
	if err != nil {
		return errors.Wrap(err, "Error creating the 'http.Request' for deletDNSRecord() with the record ID: "+recordID)
	}

	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	} else if resp.StatusCode == http.StatusNotFound {
		return errors.New("the domain you requested is not connected to your Netlify account")
	} else if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("unauthorized Netlify credentials. Please check your api key")
	}
	return errors.New("something failed. the netlify api endpoint returned the http status code: " + resp.Status)
}

func updateDNSRecord(domain string, accessToken string, body netlifyDNSRecord) error {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(
		"POST",
		netlifyDNSEndpoint(domain, accessToken, ""),
		bytes.NewBuffer(jsonBody),
	)
	if err != nil {
		return errors.Wrap(err, "Error creating the 'http.Request' for updateDNSRecord() for the record: "+body.ID)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := newHTTPClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusCreated {
		return nil
	} else if resp.StatusCode == http.StatusNotFound {
		return errors.New("the domain you requested is not connected to your Netlify account")
	} else if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("unauthorized Netlify credentials. Please check your api key")
	}
	return errors.New("something failed. the netlify api endpoint returned the http status code: " + resp.Status)
}

func netlifyDNSEndpoint(domain, accessToken, recordID string) string {
	url := "https://api.netlify.com/api/v1/dns_zones/" + strings.Replace(domain, ".", "_", -1) + "/dns_records"
	if recordID != "" {
		url += "/" + recordID
	}
	url += "?access_token=" + accessToken
	return url
}
