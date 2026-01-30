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

	var creditNotes []interface{}
	if err := json.Unmarshal(data, &creditNotes); err != nil {
		return err
	}

	if len(creditNotes) > 0 {
		filename := filepath.Join(outDir, "creditnotes", fmt.Sprintf("creditnotes_%s.json", time.Now().Format("20060102150405")))
		if !dryRun {
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}
			log.Printf("Fetched %d credit notes.", len(creditNotes))
		} else {
			log.Printf("[Dry Run] Would save %d credit notes to %s", len(creditNotes), filename)
		}
	}
    
    // Fetch Deleted Credit Notes
    params.Set("deletedOnly", "true")
    // endpoint might not support deletedOnly, wrapping in try/catch equivalent (ignoring error)
    if deletedData, err := client.Get("/v1/{organizationId}/sales/creditnotes", params); err == nil {
         var deletedCreditNotes []interface{}
         if err := json.Unmarshal(deletedData, &deletedCreditNotes); err == nil && len(deletedCreditNotes) > 0 {
            filename := filepath.Join(outDir, "deleted/creditnotes", fmt.Sprintf("deleted_creditnotes_%s.json", time.Now().Format("20060102150405")))
            if !dryRun {
                if err := os.WriteFile(filename, deletedData, 0644); err != nil {
                    return err
                }
                log.Printf("Fetched %d deleted credit notes.", len(deletedCreditNotes))
            } else {
                log.Printf("[Dry Run] Would save %d deleted credit notes to %s", len(deletedCreditNotes), filename)
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
