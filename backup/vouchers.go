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

type Voucher struct {
    Guid string `json:"Guid"`
    FileGuid string `json:"FileGuid"`
}

func BackupVouchers(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool) error {
	log.Println("Backing up Vouchers...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "vouchers"), 0755); err != nil {
			return err
		}
	}

	lastSync := stateManager.GetLastSyncVouchers()
	now := time.Now().UTC().Format(time.RFC3339)

	params := url.Values{}
	params.Set("changesSince", lastSync)

	// Fetch Purchase Vouchers
	data, err := client.Get("/v1/{organizationId}/vouchers/purchase", params)
	if err != nil {
		return err
	}

	var vouchers []Voucher
	if err := json.Unmarshal(data, &vouchers); err != nil {
		return err
	}

	if len(vouchers) > 0 {
		filename := filepath.Join(outDir, "vouchers", fmt.Sprintf("vouchers_meta_%s.json", time.Now().Format("20060102150405")))
		if !dryRun {
			if err := os.WriteFile(filename, data, 0644); err != nil {
				return err
			}
			log.Printf("Fetched %d vouchers metadata.", len(vouchers))
		} else {
			log.Printf("[Dry Run] Would save %d vouchers metadata to %s", len(vouchers), filename)
		}
        
        // Download files
        for _, voucher := range vouchers {
            if voucher.FileGuid != "" {
                if !dryRun {
                    stream, err := client.GetStream(fmt.Sprintf("/v1/{organizationId}/files/%s", voucher.FileGuid))
                    if err != nil {
                         if client.Debug {
                             log.Printf("Failed to download file for voucher %s: %v", voucher.Guid, err)
                         }
                         continue
                    }
                    defer stream.Close()
                    
                    filePath := filepath.Join(outDir, "vouchers", fmt.Sprintf("%s_%s.pdf", voucher.Guid, voucher.FileGuid))
                    outFile, err := os.Create(filePath)
                    if err != nil {
                        log.Printf("Failed to create file %s: %v", filePath, err)
                        stream.Close()
                        continue
                    }
                    
                    _, err = io.Copy(outFile, stream)
                    outFile.Close()
                    stream.Close()
                    
                    if err != nil {
                        log.Printf("Failed to write to file %s: %v", filePath, err)
                    } else {
                        if client.Debug {
                            log.Printf("Downloaded voucher file: %s", voucher.Guid)
                        }
                    }
                } else {
                    log.Printf("[Dry Run] Would download file for voucher %s (FileGuid: %s)", voucher.Guid, voucher.FileGuid)
                }
            }
        }
	}

	if !dryRun {
		stateManager.UpdateVouchers(now)
		if err := stateManager.Save(); err != nil {
			return err
		}
	} else {
		log.Printf("[Dry Run] Would update state.vouchers to %s", now)
	}

	return nil
}
