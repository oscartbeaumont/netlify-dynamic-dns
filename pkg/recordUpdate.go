package pkg

import (
	"strings"
)

// UpdateRecord will find all current records and removes them if they are not valid. If no valid records are found it will create a new one.
func UpdateRecord(domain string, accessToken string, record DNSRecord, records []DNSRecord) error {
	correctRecordExists := false
	for _, r := range records {
		if r.Type == record.Type && strings.TrimSuffix(r.Name, "."+domain) == record.Name { // If record has the correct type and name
			if r.Value == record.Value { // If the record contains the correct value set correctRecordExists to true
				correctRecordExists = true
			} else { // Else delete the record due to the incorrect value
				DeleteRecord(domain, accessToken, r)
			}
		}
	}

	if !correctRecordExists {
		record := DNSRecord{
			Type:  record.Type,
			Name:  record.Name,
			Value: record.Value,
			TTL:   record.TTL,
		}
		err := AddRecord(domain, accessToken, record)
		if err != nil {
			return err
		}
	}

	return nil
}
