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

// File represents a file in Dinero's file archive
type File struct {
	FileGuid  string `json:"FileGuid"`
	FileName  string `json:"FileName"`
	CreatedAt string `json:"CreatedAt"`
}

func BackupVouchers(client *dinero.Client, stateManager *state.Manager, outDir string, dryRun bool) error {
	log.Println("Backing up Files...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "files"), 0755); err != nil {
			return err
		}
	}

	lastSync := stateManager.GetLastSyncVouchers()
	now := time.Now().UTC()

	// Fetch files, filtered by upload date and status
	params := url.Values{}
	params.Set("fileStatus", "Used")
	if lastSync != "" && lastSync != "2000-01-01T00:00:00Z" {
		params.Set("uploadedAfter", lastSync)
		log.Printf("Fetching files uploaded after %s", lastSync)
	}

	data, err := client.Get("/v1/{organizationId}/files", params)
	if err != nil {
		return fmt.Errorf("failed to fetch files: %w", err)
	}

	var files []File
	if err := json.Unmarshal(data, &files); err != nil {
		return fmt.Errorf("failed to parse files response: %w", err)
	}

	if len(files) == 0 {
		log.Println("No files found (not updating lastSync).")
		return nil
	}

	log.Printf("Found %d files.", len(files))

	// Download each file
	downloaded := 0
	for _, file := range files {
		filename := file.FileName
		if filename == "" {
			filename = file.FileGuid + ".pdf"
		}
		filePath := filepath.Join(outDir, "files", filename)

		// Skip if file already exists
		if _, err := os.Stat(filePath); err == nil {
			if client.Debug {
				log.Printf("Skipping existing file: %s", filename)
			}
			continue
		}

		if !dryRun {
			stream, err := client.GetStream(fmt.Sprintf("/v1/{organizationId}/files/%s", file.FileGuid))
			if err != nil {
				if client.Debug {
					log.Printf("Failed to download file %s: %v", file.FileGuid, err)
				}
				continue
			}

			outFile, err := os.Create(filePath)
			if err != nil {
				stream.Close()
				log.Printf("Failed to create file %s: %v", filePath, err)
				continue
			}

			_, err = io.Copy(outFile, stream)
			outFile.Close()
			stream.Close()

			if err != nil {
				log.Printf("Failed to write file %s: %v", filePath, err)
			} else {
				downloaded++
				if client.Debug {
					log.Printf("Downloaded: %s", filename)
				}
			}
		} else {
			log.Printf("[Dry Run] Would download: %s", filename)
			downloaded++
		}
	}

	log.Printf("Downloaded %d files.", downloaded)

	if !dryRun {
		stateManager.UpdateVouchers(now.Format(time.RFC3339))
		if err := stateManager.Save(); err != nil {
			return err
		}
	}

	return nil
}
