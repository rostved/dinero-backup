package backup

import (
	"encoding/json"
	"fmt"
	"io"
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

	var response InvoiceResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}

	if len(response.Collection) > 0 {
		filename := filepath.Join(outDir, "invoices", fmt.Sprintf("invoices_%s.json", time.Now().Format("20060102150405")))
		if !dryRun {
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}
			log.Printf("Fetched %d invoices.", len(response.Collection))
		} else {
			log.Printf("[Dry Run] Would save %d invoices to %s", len(response.Collection), filename)
		}

		// Download PDFs for booked invoices (all non-Draft invoices have been booked)
		for _, invoice := range response.Collection {
			if invoice.Status != "Draft" {
				pdfFilename := filepath.Join(outDir, "invoices", fmt.Sprintf("%d.pdf", invoice.Number))
				if !dryRun {
					stream, err := client.GetPDF(fmt.Sprintf("/v1/{organizationId}/invoices/%s", invoice.Guid))
					if err != nil {
						if client.Debug {
							log.Printf("Failed to download PDF for invoice %d: %v", invoice.Number, err)
						}
						continue
					}

					outFile, err := os.Create(pdfFilename)
					if err != nil {
						stream.Close()
						log.Printf("Failed to create PDF file %s: %v", pdfFilename, err)
						continue
					}

					_, err = io.Copy(outFile, stream)
					outFile.Close()
					stream.Close()

					if err != nil {
						log.Printf("Failed to write PDF %s: %v", pdfFilename, err)
					} else if client.Debug {
						log.Printf("Downloaded invoice PDF: %d", invoice.Number)
					}
				} else {
					log.Printf("[Dry Run] Would download PDF for invoice %d", invoice.Number)
				}
			}
		}
	}

	// Fetch Deleted Invoices
	params.Set("deletedOnly", "true")
	deletedData, err := client.Get("/v1/{organizationId}/invoices", params)
	if err == nil {
		var deletedResponse PaginatedResponse
		if err := json.Unmarshal(deletedData, &deletedResponse); err == nil && len(deletedResponse.Collection) > 0 {
			filename := filepath.Join(outDir, "deleted/invoices", fmt.Sprintf("deleted_invoices_%s.json", time.Now().Format("20060102150405")))
			if !dryRun {
				if err := os.WriteFile(filename, deletedData, 0644); err != nil {
					return err
				}
				log.Printf("Fetched %d deleted invoices.", len(deletedResponse.Collection))
			} else {
				log.Printf("[Dry Run] Would save %d deleted invoices to %s", len(deletedResponse.Collection), filename)
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
