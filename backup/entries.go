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

func BackupEntries(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool, csvOutput bool) error {
	log.Println("Backing up Entries...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "entries"), 0755); err != nil {
			return err
		}
	}

	// Get all accounting years
	years, err := GetAccountingYears(client)
	if err != nil {
		return fmt.Errorf("failed to get accounting years: %w", err)
	}

	if len(years) == 0 {
		log.Println("No accounting years found.")
		return nil
	}

	// Separate years into initialized and uninitialized
	var uninitializedYears []AccountingYear
	var initializedYears []AccountingYear
	for _, year := range years {
		yearName := year.GetName()
		if stateManager.IsEntryYearInitialized(yearName) {
			initializedYears = append(initializedYears, year)
		} else {
			uninitializedYears = append(uninitializedYears, year)
		}
	}

	// Process uninitialized years - fetch full entries including primo
	for _, year := range uninitializedYears {
		if err := fetchFullYear(client, stateManager, outDir, year, dryRun, csvOutput); err != nil {
			log.Printf("Error fetching entries for year %s: %v", year.GetName(), err)
			continue
		}
	}

	// Process initialized years - fetch changes once and merge into each year
	if len(initializedYears) > 0 {
		if err := fetchAndMergeAllChanges(client, stateManager, outDir, initializedYears, dryRun, csvOutput); err != nil {
			return fmt.Errorf("error fetching entry changes: %w", err)
		}
	}

	return nil
}

// fetchFullYear fetches all entries for a fiscal year using /entries endpoint (includes primo)
func fetchFullYear(client *dinero.Client, stateManager *state.Manager, outDir string, accYear AccountingYear, dryRun bool, csvOutput bool) error {
	yearName := accYear.GetName()
	fromDate := accYear.GetFromDate()
	toDate := accYear.GetToDate()

	log.Printf("Fetching full entries for fiscal year %s (%s to %s, first run, includes primo)", yearName, fromDate, toDate)

	params := url.Values{}
	params.Set("fromDate", fromDate)
	params.Set("toDate", toDate)

	data, err := client.Get("/v1/{organizationId}/entries", params)
	if err != nil {
		return fmt.Errorf("failed to fetch entries: %w", err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse entries: %w", err)
	}

	if len(entries) == 0 {
		log.Printf("No entries found for fiscal year %s.", yearName)
		// Still mark as initialized even if empty
		if !dryRun {
			stateManager.MarkEntryYearInitialized(yearName)
			stateManager.UpdateEntries(time.Now().UTC().Format(time.RFC3339))
			if err := stateManager.Save(); err != nil {
				return err
			}
		}
		return nil
	}

	// Save to file
	if err := saveEntriesFile(outDir, yearName, entries, csvOutput, dryRun); err != nil {
		return err
	}

	log.Printf("Saved %d entries for fiscal year %s.", len(entries), yearName)

	if !dryRun {
		stateManager.MarkEntryYearInitialized(yearName)
		stateManager.UpdateEntries(time.Now().UTC().Format(time.RFC3339))
		if err := stateManager.Save(); err != nil {
			return err
		}
	}

	return nil
}

// fetchAndMergeAllChanges fetches all changes once and merges them into the appropriate fiscal year files
func fetchAndMergeAllChanges(client *dinero.Client, stateManager *state.Manager, outDir string, years []AccountingYear, dryRun bool, csvOutput bool) error {
	lastSyncStr := stateManager.GetLastSyncEntries()

	lastSync, err := time.Parse(time.RFC3339, lastSyncStr)
	if err != nil {
		return fmt.Errorf("failed to parse lastSync time: %w", err)
	}

	now := time.Now().UTC()

	log.Printf("Fetching entry changes from %s to %s", lastSync.Format(time.RFC3339), now.Format(time.RFC3339))

	// API only allows 31 days at a time, so we need to chunk
	var allChanges []Entry
	chunkStart := lastSync

	for chunkStart.Before(now) {
		chunkEnd := chunkStart.AddDate(0, 0, 31)
		if chunkEnd.After(now) {
			chunkEnd = now
		}

		params := url.Values{}
		params.Set("changesFrom", chunkStart.Format(time.RFC3339))
		params.Set("changesTo", chunkEnd.Format(time.RFC3339))

		log.Printf("Fetching changes from %s to %s", chunkStart.Format("2006-01-02"), chunkEnd.Format("2006-01-02"))

		data, err := client.Get("/v1/{organizationId}/entries/changes", params)
		if err != nil {
			return fmt.Errorf("failed to fetch entry changes: %w", err)
		}

		var chunkChanges []Entry
		if err := json.Unmarshal(data, &chunkChanges); err != nil {
			return fmt.Errorf("failed to parse entry changes: %w", err)
		}

		allChanges = append(allChanges, chunkChanges...)
		chunkStart = chunkEnd
	}

	if len(allChanges) == 0 {
		log.Println("No entry changes found (not updating lastSync).")
		return nil
	}

	log.Printf("Found %d total entry changes.", len(allChanges))

	// Group changes by fiscal year (match entry date to fiscal year range)
	changesByYear := make(map[string][]Entry)
	for _, entry := range allChanges {
		entryDate, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			continue
		}
		// Find which fiscal year this entry belongs to
		for _, accYear := range years {
			fromDate, _ := time.Parse("2006-01-02", accYear.GetFromDate())
			toDate, _ := time.Parse("2006-01-02", accYear.GetToDate())
			if !entryDate.Before(fromDate) && !entryDate.After(toDate) {
				yearName := accYear.GetName()
				changesByYear[yearName] = append(changesByYear[yearName], entry)
				break
			}
		}
	}

	// Process each fiscal year that has changes
	for _, accYear := range years {
		yearName := accYear.GetName()
		yearChanges := changesByYear[yearName]

		if len(yearChanges) == 0 {
			log.Printf("No changes for fiscal year %s.", yearName)
			continue
		}

		log.Printf("Found %d changes for fiscal year %s, merging...", len(yearChanges), yearName)

		// Load existing entries
		existingEntries, err := loadExistingEntries(outDir, yearName)
		if err != nil {
			// If file doesn't exist, fetch full year
			log.Printf("Could not load existing entries for fiscal year %s, fetching full year: %v", yearName, err)
			if err := fetchFullYear(client, stateManager, outDir, accYear, dryRun, csvOutput); err != nil {
				log.Printf("Error fetching full fiscal year %s: %v", yearName, err)
			}
			continue
		}

		// Merge changes into existing entries
		mergedEntries := mergeEntries(existingEntries, yearChanges)

		// Save merged entries
		if err := saveEntriesFile(outDir, yearName, mergedEntries, csvOutput, dryRun); err != nil {
			log.Printf("Error saving fiscal year %s: %v", yearName, err)
			continue
		}

		log.Printf("Merged %d changes into fiscal year %s (total: %d entries).", len(yearChanges), yearName, len(mergedEntries))
	}

	if !dryRun {
		stateManager.UpdateEntries(now.Format(time.RFC3339))
		if err := stateManager.Save(); err != nil {
			return err
		}
	}

	return nil
}

// loadExistingEntries loads entries from an existing JSON file
func loadExistingEntries(outDir string, yearName string) ([]Entry, error) {
	// Always read from JSON file (source of truth)
	filename := filepath.Join(outDir, "entries", fmt.Sprintf("entries_%s.json", yearName))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// mergeEntries merges changed entries into existing entries by EntryGuid
// Preserves existing order and appends new entries at the end
func mergeEntries(existing, changes []Entry) []Entry {
	// Create map of changes by GUID for quick lookup
	changeMap := make(map[string]Entry)
	for _, e := range changes {
		changeMap[e.EntryGuid] = e
	}

	// Track which changes have been applied
	applied := make(map[string]bool)

	// Update existing entries in place, preserving order
	result := make([]Entry, 0, len(existing)+len(changes))
	for _, e := range existing {
		if changed, ok := changeMap[e.EntryGuid]; ok {
			result = append(result, changed)
			applied[e.EntryGuid] = true
		} else {
			result = append(result, e)
		}
	}

	// Append new entries that weren't updates to existing ones
	for _, e := range changes {
		if !applied[e.EntryGuid] {
			result = append(result, e)
		}
	}

	return result
}

// saveEntriesFile saves entries to a file in JSON and optionally CSV format
func saveEntriesFile(outDir string, yearName string, entries []Entry, csvOutput bool, dryRun bool) error {
	// Always save JSON as source of truth
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	jsonFilename := filepath.Join(outDir, "entries", fmt.Sprintf("entries_%s.json", yearName))

	if !dryRun {
		if err := os.WriteFile(jsonFilename, jsonData, 0644); err != nil {
			return err
		}
	} else {
		log.Printf("[Dry Run] Would save %d entries to %s", len(entries), jsonFilename)
	}

	// Optionally also save CSV
	if csvOutput {
		csvData, err := EntriesToCSV(jsonData)
		if err != nil {
			return fmt.Errorf("failed to convert to CSV: %w", err)
		}

		csvFilename := filepath.Join(outDir, "entries", fmt.Sprintf("entries_%s.csv", yearName))
		if !dryRun {
			if err := os.WriteFile(csvFilename, csvData, 0644); err != nil {
				return err
			}
		} else {
			log.Printf("[Dry Run] Would save CSV to %s", csvFilename)
		}
	}

	return nil
}

// GetAccountingYears fetches all accounting years from the API
func GetAccountingYears(client *dinero.Client) ([]AccountingYear, error) {
	data, err := client.Get("/v1/{organizationId}/accountingyears", nil)
	if err != nil {
		return nil, err
	}

	var years []AccountingYear
	if err := json.Unmarshal(data, &years); err != nil {
		return nil, err
	}

	// Filter out years without valid dates
	var result []AccountingYear
	for _, year := range years {
		if year.GetFromDate() != "" && year.GetToDate() != "" {
			result = append(result, year)
		}
	}

	return result, nil
}
