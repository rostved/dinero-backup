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

func BackupCreditNotes(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool) error {
	log.Println("Backing up Credit Notes...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "creditnotes"), 0755); err != nil {
			return err
		}
        if err := os.MkdirAll(filepath.Join(outDir, "deleted/creditnotes"), 0755); err != nil {
            return err
        }
	}

	lastSync := stateManager.GetLastSyncCreditNotes()
	now := time.Now().UTC().Format(time.RFC3339)

	params := url.Values{}
	params.Set("changesSince", lastSync)

	// Fetch Active Credit Notes
	data, err := client.Get("/v1/{organizationId}/sales/creditnotes", params)
	if err != nil {
		return err
	}

	var response PaginatedResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}

	if len(response.Collection) > 0 {
		filename := filepath.Join(outDir, "creditnotes", fmt.Sprintf("creditnotes_%s.json", time.Now().Format("20060102150405")))
		if !dryRun {
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}
			log.Printf("Fetched %d credit notes.", len(response.Collection))
		} else {
			log.Printf("[Dry Run] Would save %d credit notes to %s", len(response.Collection), filename)
		}
	}

	// Fetch Deleted Credit Notes
	params.Set("deletedOnly", "true")
	if deletedData, err := client.Get("/v1/{organizationId}/sales/creditnotes", params); err == nil {
		var deletedResponse PaginatedResponse
		if err := json.Unmarshal(deletedData, &deletedResponse); err == nil && len(deletedResponse.Collection) > 0 {
			filename := filepath.Join(outDir, "deleted/creditnotes", fmt.Sprintf("deleted_creditnotes_%s.json", time.Now().Format("20060102150405")))
			if !dryRun {
				if err := os.WriteFile(filename, deletedData, 0644); err != nil {
					return err
				}
				log.Printf("Fetched %d deleted credit notes.", len(deletedResponse.Collection))
			} else {
				log.Printf("[Dry Run] Would save %d deleted credit notes to %s", len(deletedResponse.Collection), filename)
			}
		}
	}

	if !dryRun {
		stateManager.UpdateCreditNotes(now)
		if err := stateManager.Save(); err != nil {
			return err
		}
	} else {
		log.Printf("[Dry Run] Would update state.creditNotes to %s", now)
	}

	return nil
}
