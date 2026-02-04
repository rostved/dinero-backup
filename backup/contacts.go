package backup

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/rostved/dinero-backup/dinero"
	"github.com/rostved/dinero-backup/state"
)

// ContactsResponse represents Dinero's paginated contacts response
type ContactsResponse struct {
	Collection []json.RawMessage `json:"Collection"`
	Pagination struct {
		Page                int `json:"Page"`
		PageSize            int `json:"PageSize"`
		Result              int `json:"Result"`
		ResultWithoutFilter int `json:"ResultWithoutFilter"`
		MaxPageSize         int `json:"MaxPageSize"`
	} `json:"Pagination"`
}

func BackupContacts(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool) error {
	log.Println("Backing up Contacts...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "contacts"), 0755); err != nil {
			return err
		}
	}

	lastSync := stateManager.GetLastSyncContacts()
	now := time.Now().UTC()

	// Fetch all contacts with pagination
	var allContacts []json.RawMessage
	page := 0
	pageSize := 100

	for {
		fields := "" +
			"Name,ContactGuid,ExternalReference,IsPerson,Street,ZipCode,City,CountryKey,Phone," +
			"Email,Webpage,AttPerson,VatNumber,EanNumber,PaymentConditionType,PaymentConditionNumberOfDays," +
			"IsMember,MemberNumber,CompanyStatus,VatRegionKey,CreatedAt,UpdatedAt,DeletedAt,PreferredInvoiceLanguageKey," +
			"PreferredInvoiceCurrencyKey"

		params := url.Values{}
		params.Set("fields", fields)
		params.Set("changesSince", lastSync)
		params.Set("page", fmt.Sprintf("%d", page))
		params.Set("pageSize", fmt.Sprintf("%d", pageSize))

		data, err := client.Get("/v2/{organizationId}/contacts", params)
		if err != nil {
			return fmt.Errorf("failed to fetch contacts: %w", err)
		}

		var response ContactsResponse
		if err := json.Unmarshal(data, &response); err != nil {
			return fmt.Errorf("failed to parse contacts response: %w", err)
		}

		allContacts = append(allContacts, response.Collection...)

		log.Printf("Fetched page %d: %d contacts", page, len(response.Collection))

		// Check if we've fetched all pages
		if len(response.Collection) < pageSize {
			break
		}
		page++
	}

	if len(allContacts) == 0 {
		log.Println("No contact changes found (not updating lastSync).")
		return nil
	}

	log.Printf("Found %d total contacts.", len(allContacts))

	// Load existing contacts and merge
	existingContacts, err := loadExistingContacts(outDir)
	if err != nil {
		log.Printf("No existing contacts file, creating new one.")
		existingContacts = []json.RawMessage{}
	}

	mergedContacts := mergeContacts(existingContacts, allContacts)

	// Save to file
	filename := filepath.Join(outDir, "contacts", "contacts.json")
	if !dryRun {
		jsonData, err := json.MarshalIndent(mergedContacts, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal contacts: %w", err)
		}
		if err := os.WriteFile(filename, jsonData, 0644); err != nil {
			return err
		}
		log.Printf("Saved %d contacts to %s", len(mergedContacts), filename)

		stateManager.UpdateContacts(now.Format(time.RFC3339))
		if err := stateManager.Save(); err != nil {
			return err
		}
	} else {
		log.Printf("[Dry Run] Would save %d contacts to %s", len(mergedContacts), filename)
	}

	return nil
}

func loadExistingContacts(outDir string) ([]json.RawMessage, error) {
	filename := filepath.Join(outDir, "contacts", "contacts.json")
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var contacts []json.RawMessage
	if err := json.Unmarshal(data, &contacts); err != nil {
		return nil, err
	}

	return contacts, nil
}

// mergeContacts merges changed contacts into existing contacts by ContactGuid
// Preserves existing order and appends new contacts at the end
func mergeContacts(existing, changes []json.RawMessage) []json.RawMessage {
	// Helper to extract ContactGuid from raw JSON
	getGuid := func(raw json.RawMessage) string {
		var obj struct {
			ContactGuid string `json:"ContactGuid"`
		}
		json.Unmarshal(raw, &obj)
		return obj.ContactGuid
	}

	// Create map of changes by GUID for quick lookup
	changeMap := make(map[string]json.RawMessage)
	for _, c := range changes {
		guid := getGuid(c)
		if guid != "" {
			changeMap[guid] = c
		}
	}

	// Track which changes have been applied
	applied := make(map[string]bool)

	// Update existing contacts in place, preserving order
	result := make([]json.RawMessage, 0, len(existing)+len(changes))
	for _, c := range existing {
		guid := getGuid(c)
		if changed, ok := changeMap[guid]; ok {
			result = append(result, changed)
			applied[guid] = true
		} else {
			result = append(result, c)
		}
	}

	// Append new contacts that weren't updates to existing ones
	for _, c := range changes {
		guid := getGuid(c)
		if !applied[guid] {
			result = append(result, c)
		}
	}

	return result
}
