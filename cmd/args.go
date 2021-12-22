package main

// Version contains the builds version. It is injected at build time
var Version = "v0.0.0-dev"

// Arguments stores the configuration which is determined from the command line arguments or environment varibles
type Arguments struct {
	AccessToken      string `arg:"env:NDDNS_ACCESS_TOKEN,required" help:"Netlify personal access token. Can be created at https://app.netlify.com/user/application"`
	Zone             string `placeholder:"\"example.com\"" arg:"env:NDDNS_ZONE,required" help:"The Netlify DNS zone domain name"`
	Record           string `placeholder:"\"home\"" arg:"env:NDDNS_RECORD" help:"The record in the DNS zone to set as your public IP"`
	IPv6             bool   `default:"true" arg:"env:NDDNS_IPv6_ENABLED" help:"Whether the IPv6 record (AAAA) should also be updated."`
	Interval         int    `placeholder:"5" arg:"env:NDDNS_INTERVAL" help:"The amount of minutes between sending updates. If 0 only a single update is done."`
	UpdateRootRecord bool   `placeholder:"false" arg:"env:NDDNS_IS_ROOT" help:"Use to update the root record instead of a subdomain"`
	zoneID           string `arg:"-"`
	recordHostname   string `arg:"-"`
}

// Description is for alexflint/go-args
func (Arguments) Description() string {
	return "Netlify Dynamic DNS Updated. Created by Oscar Beaumont!"
}

// Version is for alexflint/go-args
func (Arguments) Version() string {
	return "Version: " + Version
}
