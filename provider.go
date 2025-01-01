package cloudns

import (
	"context"
	"errors"
	"github.com/libdns/libdns"
	"strings"
)

// ClouDNS API docs: https://www.cloudns.net/wiki/article/41/

var baseUrl = "https://api.cloudns.net/dns/"

// Provider facilitates DNS record manipulation with <TODO: PROVIDER NAME>.
type Provider struct {
	AuthId       string `json:"auth_id"`
	SubAuthId    string `json:"sub_auth_id"`
	AuthPassword string `json:"auth_password"`
}

// GetRecords lists all the records in the zone.
func (p *Provider) GetRecords(ctx context.Context, zone string) ([]libdns.Record, error) {
	if strings.HasSuffix(zone, ".") {
		zone = strings.TrimSuffix(zone, ".")
	}
	records, err := UseClient(p.AuthId, p.SubAuthId, p.AuthPassword).GetRecords(ctx, zone)
	if err != nil {
		return nil, err
	}
	return records, nil
}

// AppendRecords adds records to the zone. It returns the records that were added.
func (p *Provider) AppendRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if strings.HasSuffix(zone, ".") {
		zone = strings.TrimSuffix(zone, ".")
	}
	var createdRecords []libdns.Record
	
	for _, record := range records {
		// Check if this is an ACME challenge record
		if strings.HasPrefix(record.Name, "_acme-challenge.") {
			// Get existing records to check if we need to update
			existingRecords, err := p.GetRecords(ctx, zone)
			if err != nil {
				return nil, errors.New("failed to get existing records: " + err.Error())
			}

			// Clean up any stale ACME challenge records for this name
			var recordsToDelete []libdns.Record
			var currentRecord *libdns.Record
			for _, existing := range existingRecords {
				if existing.Type == record.Type && existing.Name == record.Name {
					if currentRecord == nil {
						// Keep the first one we find as current
						tmp := existing
						currentRecord = &tmp
					} else {
						// Mark any additional ones for deletion
						recordsToDelete = append(recordsToDelete, existing)
					}
				}
			}

			// Delete stale records if any found
			if len(recordsToDelete) > 0 {
				_, err = p.DeleteRecords(ctx, zone, recordsToDelete)
				if err != nil {
					return nil, errors.New("failed to delete stale ACME challenge records: " + err.Error())
				}
			}

			// Update existing record if found
			if currentRecord != nil {
				r, err := UseClient(p.AuthId, p.SubAuthId, p.AuthPassword).UpdateRecord(ctx, zone, currentRecord.ID, record.Name, record.Value, record.TTL)
				if err != nil {
					return nil, errors.New("failed to update ACME challenge record: " + err.Error())
				}
				createdRecords = append(createdRecords, *r)
				continue
			}
		}

		// Default behavior for non-ACME records or when no ACME record exists
		r, err := UseClient(p.AuthId, p.SubAuthId, p.AuthPassword).AddRecord(ctx, zone, record.Type, record.Name, record.Value, record.TTL)
		if err != nil {
			return nil, errors.New("failed to add record: " + err.Error())
		}
		createdRecords = append(createdRecords, *r)
	}
	return createdRecords, nil
}

// SetRecords sets the records in the zone, either by updating existing records or creating new ones.
// It returns the updated records.
func (p *Provider) SetRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if strings.HasSuffix(zone, ".") {
		zone = strings.TrimSuffix(zone, ".")
	}
	var updatedRecords []libdns.Record
	for _, record := range records {
		if len(record.ID) == 0 {
			// create
			r, err := UseClient(p.AuthId, p.SubAuthId, p.AuthPassword).AddRecord(ctx, zone, record.Type, record.Name, record.Value, record.TTL)
			if err != nil {
				return nil, errors.New("failed to add record: " + err.Error())
			}
			updatedRecords = append(updatedRecords, *r)
		} else {
			//update
			r, err := UseClient(p.AuthId, p.SubAuthId, p.AuthPassword).UpdateRecord(ctx, zone, record.ID, record.Name, record.Value, record.TTL)
			if err != nil {
				return nil, errors.New("failed to update record: " + err.Error())
			}
			updatedRecords = append(updatedRecords, *r)
		}
	}
	return updatedRecords, nil
}

// DeleteRecords deletes the records from the zone. It returns the records that were deleted.
func (p *Provider) DeleteRecords(ctx context.Context, zone string, records []libdns.Record) ([]libdns.Record, error) {
	if strings.HasSuffix(zone, ".") {
		zone = strings.TrimSuffix(zone, ".")
	}
	var deletedRecords []libdns.Record
	for _, record := range records {
		r, err := UseClient(p.AuthId, p.SubAuthId, p.AuthPassword).DeleteRecord(ctx, zone, record.ID)
		if err != nil {
			return nil, errors.New("failed to delete record: " + err.Error())
		}
		deletedRecords = append(deletedRecords, *r)
	}
	return deletedRecords, nil
}

// Interface guards
var (
	_ libdns.RecordGetter   = (*Provider)(nil)
	_ libdns.RecordAppender = (*Provider)(nil)
	_ libdns.RecordSetter   = (*Provider)(nil)
	_ libdns.RecordDeleter  = (*Provider)(nil)
)
