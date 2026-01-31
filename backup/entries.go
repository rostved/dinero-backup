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
	var uninitializedYears []time.Time
	var initializedYears []time.Time
	for _, year := range years {
		if stateManager.IsEntryYearInitialized(year.Year()) {
			initializedYears = append(initializedYears, year)
		} else {
			uninitializedYears = append(uninitializedYears, year)
		}
	}

	// Process uninitialized years - fetch full entries including primo
	for _, year := range uninitializedYears {
		if err := fetchFullYear(client, stateManager, outDir, year, dryRun, csvOutput); err != nil {
			log.Printf("Error fetching entries for year %d: %v", year.Year(), err)
			continue
		}
	}

	// Process initialized years - fetch changes once and merge into each year
	if len(initializedYears) > 0 {
		if err := fetchAndMergeAllChanges(client, stateManager, outDir, initializedYears, dryRun, csvOutput); err != nil {
			log.Printf("Error fetching entry changes: %v", err)
		}
	}

	return nil
}

// fetchFullYear fetches all entries for a year using /entries endpoint (includes primo)
func fetchFullYear(client *dinero.Client, stateManager *state.Manager, outDir string, year time.Time, dryRun bool, csvOutput bool) error {
	yearNum := year.Year()
	fromDate := time.Date(yearNum, 1, 1, 0, 0, 0, 0, time.UTC)
	toDate := time.Date(yearNum, 12, 31, 0, 0, 0, 0, time.UTC)

	log.Printf("Fetching full entries for year %d (first run, includes primo)", yearNum)

	params := url.Values{}
	params.Set("fromDate", fromDate.Format("2006-01-02"))
	params.Set("toDate", toDate.Format("2006-01-02"))

	data, err := client.Get("/v1/{organizationId}/entries", params)
	if err != nil {
		return fmt.Errorf("failed to fetch entries: %w", err)
	}

	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to parse entries: %w", err)
	}

	if len(entries) == 0 {
		log.Printf("No entries found for year %d.", yearNum)
		// Still mark as initialized even if empty
		if !dryRun {
			stateManager.MarkEntryYearInitialized(yearNum)
			stateManager.UpdateEntries(time.Now().UTC().Format(time.RFC3339))
			if err := stateManager.Save(); err != nil {
				return err
			}
		}
		return nil
	}

	// Save to file
	if err := saveEntriesFile(outDir, yearNum, entries, csvOutput, dryRun); err != nil {
		return err
	}

	log.Printf("Saved %d entries for year %d.", len(entries), yearNum)

	if !dryRun {
		stateManager.MarkEntryYearInitialized(yearNum)
		stateManager.UpdateEntries(time.Now().UTC().Format(time.RFC3339))
		if err := stateManager.Save(); err != nil {
			return err
		}
	}

	return nil
}

// fetchAndMergeAllChanges fetches all changes once and merges them into the appropriate year files
func fetchAndMergeAllChanges(client *dinero.Client, stateManager *state.Manager, outDir string, years []time.Time, dryRun bool, csvOutput bool) error {
	lastSync := stateManager.GetLastSyncEntries()

	log.Printf("Fetching entry changes since %s", lastSync)

	params := url.Values{}
	params.Set("changesSince", lastSync)

	data, err := client.Get("/v1/{organizationId}/entries/changes", params)
	if err != nil {
		return fmt.Errorf("failed to fetch entry changes: %w", err)
	}

	var allChanges []Entry
	if err := json.Unmarshal(data, &allChanges); err != nil {
		return fmt.Errorf("failed to parse entry changes: %w", err)
	}

	if len(allChanges) == 0 {
		log.Println("No entry changes found.")
		if !dryRun {
			stateManager.UpdateEntries(time.Now().UTC().Format(time.RFC3339))
			if err := stateManager.Save(); err != nil {
				return err
			}
		}
		return nil
	}

	log.Printf("Found %d total entry changes.", len(allChanges))

	// Group changes by year
	changesByYear := make(map[int][]Entry)
	for _, entry := range allChanges {
		entryDate, err := time.Parse("2006-01-02", entry.Date)
		if err != nil {
			continue
		}
		yearNum := entryDate.Year()
		changesByYear[yearNum] = append(changesByYear[yearNum], entry)
	}

	// Process each year that has changes
	for _, year := range years {
		yearNum := year.Year()
		yearChanges := changesByYear[yearNum]

		if len(yearChanges) == 0 {
			log.Printf("No changes for year %d.", yearNum)
			continue
		}

		log.Printf("Found %d changes for year %d, merging...", len(yearChanges), yearNum)

		// Load existing entries
		existingEntries, err := loadExistingEntries(outDir, yearNum)
		if err != nil {
			// If file doesn't exist, fetch full year
			log.Printf("Could not load existing entries for year %d, fetching full year: %v", yearNum, err)
			if err := fetchFullYear(client, stateManager, outDir, year, dryRun, csvOutput); err != nil {
				log.Printf("Error fetching full year %d: %v", yearNum, err)
			}
			continue
		}

		// Merge changes into existing entries
		mergedEntries := mergeEntries(existingEntries, yearChanges)

		// Save merged entries
		if err := saveEntriesFile(outDir, yearNum, mergedEntries, csvOutput, dryRun); err != nil {
			log.Printf("Error saving year %d: %v", yearNum, err)
			continue
		}

		log.Printf("Merged %d changes into year %d (total: %d entries).", len(yearChanges), yearNum, len(mergedEntries))
	}

	if !dryRun {
		stateManager.UpdateEntries(time.Now().UTC().Format(time.RFC3339))
		if err := stateManager.Save(); err != nil {
			return err
		}
	}

	return nil
}

// loadExistingEntries loads entries from an existing JSON file
func loadExistingEntries(outDir string, year int) ([]Entry, error) {
	// Always read from JSON file (source of truth)
	filename := filepath.Join(outDir, "entries", fmt.Sprintf("entries_%d.json", year))
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
func mergeEntries(existing, changes []Entry) []Entry {
	// Create map of existing entries by GUID
	entryMap := make(map[string]Entry)
	for _, e := range existing {
		entryMap[e.EntryGuid] = e
	}

	// Update/add changed entries
	for _, e := range changes {
		entryMap[e.EntryGuid] = e
	}

	// Convert back to slice
	result := make([]Entry, 0, len(entryMap))
	for _, e := range entryMap {
		result = append(result, e)
	}

	return result
}

// saveEntriesFile saves entries to a file in JSON and optionally CSV format
func saveEntriesFile(outDir string, year int, entries []Entry, csvOutput bool, dryRun bool) error {
	// Always save JSON as source of truth
	jsonData, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal entries: %w", err)
	}

	jsonFilename := filepath.Join(outDir, "entries", fmt.Sprintf("entries_%d.json", year))

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

		csvFilename := filepath.Join(outDir, "entries", fmt.Sprintf("entries_%d.csv", year))
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

// GetAccountingYears fetches all accounting years and returns their start dates
func GetAccountingYears(client *dinero.Client) ([]time.Time, error) {
	data, err := client.Get("/v1/{organizationId}/accountingyears", nil)
	if err != nil {
		return nil, err
	}

	var years []AccountingYear
	if err := json.Unmarshal(data, &years); err != nil {
		return nil, err
	}

	var result []time.Time
	for _, year := range years {
		dateStr := year.FromDate
		if dateStr == "" {
			dateStr = year.DateStart
		}
		if dateStr == "" {
			continue
		}

		t, err := time.ParseInLocation("2006-01-02", dateStr, time.UTC)
		if err != nil {
			continue
		}

		result = append(result, t)
	}

	return result, nil
}
