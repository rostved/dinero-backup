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

func BackupEntries(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool) error {
	log.Println("Backing up Entries...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "entries"), 0755); err != nil {
			return err
		}
	}

	lastSync := stateManager.GetLastSyncEntries()
	now := time.Now().UTC()
	currentFrom, err := time.Parse(time.RFC3339, lastSync)
	if err != nil {
        // Fallback or restart? default state is valid RFC3339
		currentFrom = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	}

	for currentFrom.Before(now) {
		currentTo := currentFrom.AddDate(0, 0, 31)
		if currentTo.After(now) {
			currentTo = now
		}

		log.Printf("Fetching entries changes from %s to %s", currentFrom.Format(time.RFC3339), currentTo.Format(time.RFC3339))

		params := url.Values{}
		params.Set("changesFrom", currentFrom.Format(time.RFC3339))
		params.Set("changesTo", currentTo.Format(time.RFC3339))
		params.Set("includePrimo", "true")

		data, err := client.Get("/v1/{organizationId}/entries/changes", params)
		if err != nil {
			log.Printf("Error fetching entries: %v", err)
			break // Stop loop on error
		}

		var entries []interface{}
		if err := json.Unmarshal(data, &entries); err != nil {
            log.Printf("Error unmarshaling entries: %v", err)
			break
		}

		if len(entries) > 0 {
			filename := filepath.Join(outDir, "entries", fmt.Sprintf("entries_%s_%s.json", currentFrom.Format("20060102"), currentTo.Format("20060102")))
			if !dryRun {
				if err := os.WriteFile(filename, data, 0644); err != nil {
					return err
				}
				log.Printf("Fetched %d entries.", len(entries))
			} else {
				log.Printf("[Dry Run] Would save %d entries to %s", len(entries), filename)
			}
		}

		currentFrom = currentTo
        
		if !dryRun {
			stateManager.UpdateEntries(currentTo.Format(time.RFC3339))
			if err := stateManager.Save(); err != nil {
				return err
			}
		} else {
			log.Printf("[Dry Run] Would update state.entries to %s", currentTo.Format(time.RFC3339))
		}
	}

	return nil
}
