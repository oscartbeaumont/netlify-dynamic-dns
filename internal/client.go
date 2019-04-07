package internal

import (
	"net"
	"net/http"
	"time"
)

// Client is a HTTP Client With Sane Timeouts
var Client = &http.Client{
	Timeout: time.Second * 10,
	Transport: &http.Transport{
		Dial: (&net.Dialer{
			Timeout: 5 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 5 * time.Second,
	},
}
