package publicip

import (
	"errors"
	"net"

	"github.com/miekg/dns"
)

const opendnsMyIP = "myip.opendns.com."
const opendnsResolver = "resolver1.opendns.com"

// OpenDNSProvider is a Public IP address provider which makes use of OpenDNS
type OpenDNSProvider struct{}

func (opdns OpenDNSProvider) doQuery(fqdn string, recordType uint16, server string) (*dns.Msg, error) {
	query := new(dns.Msg)
	query.SetQuestion(fqdn, recordType)

	res, err := dns.Exchange(query, server)
	if err != nil {
		return nil, err
	}

	if len(res.Answer) < 1 {
		return nil, errors.New("OpenDNS failed to return ipv4 address. Are your sure your client & internet connection supports it?")
	}

	return res, nil
}

// GetIPv4 returns the public IPv4 Address of the current machine
func (opdns OpenDNSProvider) GetIPv4() (string, error) {
	// Lookup OpenDNS IPv4 Addr
	opdnsRes, err := opdns.doQuery(dns.Fqdn(opendnsResolver), dns.TypeA, opendnsResolver+":53")
	if err != nil {
		return "", err
	}

	opdnsRecord, ok := opdnsRes.Answer[0].(*dns.A)
	if !ok {
		return "", errors.New("OpenDNS failed to return a valid IPv4 address for itself")
	}

	// Query OpenDNS for clients public ip
	res, err := opdns.doQuery(opendnsMyIP, dns.TypeA, opdnsRecord.A.String()+":53")
	if err != nil {
		return "", err
	}

	record, ok := res.Answer[0].(*dns.A)
	if !ok {
		return "", errors.New("OpenDNS failed to return a valid A record")
	}

	return record.A.String(), nil
}

// GetIPv6 returns the public IPv6 Address of the current machine
func (opdns OpenDNSProvider) GetIPv6() (string, error) {
	// Lookup OpenDNS IPv6 Addr
	opdnsRes, err := opdns.doQuery(dns.Fqdn(opendnsResolver), dns.TypeAAAA, opendnsResolver+":53")
	if err != nil {
		return "", err
	}

	opdnsRecord, ok := opdnsRes.Answer[0].(*dns.AAAA)
	if !ok {
		return "", errors.New("OpenDNS failed to return a valid IPv6 address for itself")
	}

	// Query OpenDNS for clients public ip
	res, err := opdns.doQuery(opendnsMyIP, dns.TypeAAAA, "["+opdnsRecord.AAAA.String()+"]:53")
	if erry, ok := err.(*net.OpError); ok && erry.Err.Error() == "connect: no route to host" {
		return "", errors.New("No route to OpenDNS IPv6. Does your connection support IPv6?")
	} else if err != nil {
		return "", err
	}

	record, ok := res.Answer[0].(*dns.AAAA)
	if !ok {
		return "", errors.New("OpenDNS failed to return a valid AAAA record")
	}

	return record.AAAA.String(), nil
}
