package main

import "github.com/netlify/open-api/go/models"

// recordsToDelete returns every A and AAAA record for hostname, with duplicates removed by id.
// Netlify can hold several records for one hostname, and deleting only the first one leaves the
// stale records behind forever.
func recordsToDelete(records []*models.DNSRecord, hostname string) []*models.DNSRecord {
	var result []*models.DNSRecord
	seen := make(map[string]struct{})

	for _, record := range records {
		if record == nil || record.Hostname != hostname {
			continue
		}
		if record.Type != "A" && record.Type != "AAAA" {
			continue
		}
		if record.ID != "" {
			if _, ok := seen[record.ID]; ok {
				continue
			}
			seen[record.ID] = struct{}{}
		}
		result = append(result, record)
	}

	return result
}

// firstRecordOfType returns the first record for hostname of the given type, or nil when there
// is none.
func firstRecordOfType(records []*models.DNSRecord, hostname, recordType string) *models.DNSRecord {
	for _, record := range records {
		if record != nil && record.Hostname == hostname && record.Type == recordType {
			return record
		}
	}
	return nil
}

// recordTTL returns the TTL of record, or 0 when there is no record.
func recordTTL(record *models.DNSRecord) int64 {
	if record == nil {
		return 0
	}
	return record.TTL
}
