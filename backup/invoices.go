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

func BackupInvoices(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool) error {
	log.Println("Backing up Invoices...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "invoices"), 0755); err != nil {
			return err
		}
        if err := os.MkdirAll(filepath.Join(outDir, "deleted/invoices"), 0755); err != nil {
            return err
        }
	}

	lastSync := stateManager.GetLastSyncInvoices()
	now := time.Now().UTC().Format(time.RFC3339)

	fields := "Guid,ContactName,Date,Description,TotalInclVat,Status,CreatedAt,UpdatedAt,DeletedAt,Number,ExternalReference,ContactGuid,PaymentDate,TotalExclVat,Currency"
	params := url.Values{}
	params.Set("fields", fields)
	params.Set("changesSince", lastSync)

	// Fetch Active Invoices
	data, err := client.Get("/v1/{organizationId}/invoices", params)
	if err != nil {
		return err
	}

	var invoices []interface{}
	if err := json.Unmarshal(data, &invoices); err != nil {
		return err
	}

	if len(invoices) > 0 {
		filename := filepath.Join(outDir, "invoices", fmt.Sprintf("invoices_%s.json", time.Now().Format("20060102150405")))
		if !dryRun {
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}
			log.Printf("Fetched %d invoices.", len(invoices))
		} else {
			log.Printf("[Dry Run] Would save %d invoices to %s", len(invoices), filename)
		}
	}
    
    // Fetch Deleted Invoices
    params.Set("deletedOnly", "true")
    deletedData, err := client.Get("/v1/{organizationId}/invoices", params)
    if err == nil {
         var deletedInvoices []interface{}
         if err := json.Unmarshal(deletedData, &deletedInvoices); err == nil && len(deletedInvoices) > 0 {
            filename := filepath.Join(outDir, "deleted/invoices", fmt.Sprintf("deleted_invoices_%s.json", time.Now().Format("20060102150405")))
            if !dryRun {
                if err := os.WriteFile(filename, deletedData, 0644); err != nil {
                    return err
                }
                log.Printf("Fetched %d deleted invoices.", len(deletedInvoices))
            } else {
                log.Printf("[Dry Run] Would save %d deleted invoices to %s", len(deletedInvoices), filename)
            }
         }
    }

	if !dryRun {
		stateManager.UpdateInvoices(now)
		if err := stateManager.Save(); err != nil {
			return err
		}
	} else {
		log.Printf("[Dry Run] Would update state.invoices to %s", now)
	}

	return nil
}
