package pkg

// UpdateRecord will find all current records and removes them if they are not valid. If no valid records are found it will create a new one.
func UpdateRecord(domain string, accessToken string, record DNSRecord, records []DNSRecord) error {
	correctRecordExists := false
        var newName = record.Name+"."+domain
	for _, r := range records {
		if r.Type == record.Type {
			if r.Name == newName && r.Value == record.Value {
				correctRecordExists = true
			} else if r.Name == newName {
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
