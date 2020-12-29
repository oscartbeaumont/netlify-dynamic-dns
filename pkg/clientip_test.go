package pkg

import (
	"regexp"
	"testing"
)

func TestGetPublicIPv4(t *testing.T) {
	ipv4, err := GetPublicIPv4()
	if err != nil {
		t.Error(err)
		t.Fail()
	}
	matched, matchingErr := regexp.MatchString(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`, ipv4)
	if matchingErr != nil {
		t.Error(err)
		t.Fail()
	}
	if !matched {
		t.Error("IPv4 does not match the required format")
		t.Fail()
	}
}
