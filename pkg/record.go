package pkg

// DNSRecord is a Netlify DNS Record Entry. It is serialised to JSON when talking to the Netlify API.
type DNSRecord struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Name  string `json:"hostname"`
	Value string `json:"value"`
	TTL   int    `json:"ttl,omitempty"`
}
