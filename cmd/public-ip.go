package main

import (
	"encoding/json"
)

const ipv4ApiEndpoint = "https://v4.ident.me/.json"
const ipv6ApiEndpoint = "https://v6.ident.me/.json"

type apiResponse struct {
	Address string
}

func getPublicIP() (ipv4 string, ipv6 string, err error) {
	ipv4res, err := pingAPI(ipv4ApiEndpoint)
	if err != nil {
		return "", "", err
	}

	ipv6res, err := pingAPI(ipv6ApiEndpoint)
	if err != nil {
		return "", "", err
	}

	return ipv4res.Address, ipv6res.Address, nil
}

func pingAPI(url string) (apiResponse, error) {
	res, err := newHTTPClient().Get(url)
	if err != nil {
		return apiResponse{}, err
	}
	var response apiResponse
	return response, json.NewDecoder(res.Body).Decode(&response)
}
