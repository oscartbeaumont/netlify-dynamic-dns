package publicip

// Provider is a service capable of determining the users public IPv4 and IPv6 addresss
type Provider interface {
	GetIPv4() (string, error)
	GetIPv6() (string, error)
}
