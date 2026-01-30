package backup

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/rostved/dinero-backup/dinero"
)

type AccountingYear struct {
	DateEnd string `json:"dateEnd"`
	ToDate  string `json:"ToDate"` // PascalCase fallback
    Name string `json:"name"`
}

func BackupReports(client *dinero.Client, outDir string, dryRun bool) error {
	log.Println("Backing up Reports...")

	if !dryRun {
		if err := os.MkdirAll(filepath.Join(outDir, "reports"), 0755); err != nil {
			return err
		}
	} else {
        log.Printf("[Dry Run] Would ensure directory matches: %s", filepath.Join(outDir, "reports"))
    }

    // Fetch accounting years
	data, err := client.Get("/v1/{organizationId}/accountingyears", nil)
	if err != nil {
		return fmt.Errorf("failed to fetch accounting years: %w", err)
	}

	var accountingYears []AccountingYear
	if err := json.Unmarshal(data, &accountingYears); err != nil {
		return err
	}

	for _, str := range []string{"balance", "result", "saldo"} {
		for _, accYear := range accountingYears {
            endDate := accYear.ToDate
            if endDate == "" {
                endDate = accYear.DateEnd
            }
            
            // Year logic
            year := accYear.Name
            
            if year == "" && endDate != "" {
                 t, err := time.Parse("2006-01-02", endDate)
                 if err == nil {
                     year = strconv.Itoa(t.Year())
                 }
            }
            
            if year == "" {
                if client.Debug {
                    log.Printf("Skipping accounting year with no name or end date: %+v", accYear)
                }
                continue
            }

			filename := filepath.Join(outDir, "reports", fmt.Sprintf("%s_%s.json", year, str))

			if !dryRun {
				reportData, err := client.Get(fmt.Sprintf("/v1/{organizationId}/%s/reports/%s", year, str), nil)
				if err != nil {
					// Handle 404 gracefully? Log it.
                    if client.Debug {
					    log.Printf("Error fetching %s for %s: %v", str, year, err)
                    }
					continue
				}

				if err := os.WriteFile(filename, reportData, 0644); err != nil {
					return err
				}
                if client.Debug {
                    log.Printf("Saved %s", filename)
                }
			} else {
				log.Printf("[Dry Run] Would save report: %s", filename)
			}
		}
	}
	return nil
}
