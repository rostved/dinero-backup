package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/rostved/dinero-backup/backup"
	"github.com/rostved/dinero-backup/dinero"
	"github.com/rostved/dinero-backup/state"
	"github.com/spf13/cobra"
)

var (
	backupCmd = &cobra.Command{
		Use:   "dinero-backup",
		Short: "CLI tool to backup Dinero ERP data",
		Run:   runBackup,
	}

	// Flags
	dryRun      bool
	debug       bool
	outDir      string
	reports     bool
	invoices    bool
	creditNotes bool
	entries     bool
	vouchers    bool
)

func init() {
	backupCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run without saving files or updating state")
	backupCmd.Flags().BoolVar(&debug, "debug", false, "Enable debug logging")
	backupCmd.Flags().StringVar(&outDir, "out-dir", "backup", "Output directory for backup files")

	backupCmd.Flags().BoolVar(&reports, "reports", false, "Backup reports")
	backupCmd.Flags().BoolVar(&invoices, "invoices", false, "Backup invoices")
	backupCmd.Flags().BoolVar(&creditNotes, "creditnotes", false, "Backup credit notes")
	backupCmd.Flags().BoolVar(&entries, "entries", false, "Backup entries")
	backupCmd.Flags().BoolVar(&vouchers, "vouchers", false, "Backup vouchers")
}

func main() {
	if err := backupCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func runBackup(cmd *cobra.Command, args []string) {
	if debug {
		log.Println("Debug mode enabled")
	}

	err := godotenv.Load()
	if err != nil && debug {
		log.Println("Error loading .env file (optional)")
	}

	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	apiKey := os.Getenv("API_KEY")
	orgID := os.Getenv("ORG_ID")

	// Check Env Vars
	if clientID == "" || clientSecret == "" || apiKey == "" || orgID == "" {
		log.Fatal("Missing environment variables. Required: CLIENT_ID, CLIENT_SECRET, API_KEY, ORG_ID")
	}

	client := dinero.NewClient(clientID, clientSecret, apiKey, orgID)
	client.SetDebug(debug)

	stateManager := state.NewManager("state.json")
	if err := stateManager.Load(); err != nil {
		log.Printf("Could not load state (starting fresh?): %v", err)
	}

	log.Printf("Starting backup to %s...", outDir)
	if dryRun {
		log.Println("DRY RUN MODE: No files will be written, state will not be updated.")
	}

	// Determine what to backup
	all := !reports && !invoices && !creditNotes && !entries && !vouchers
	runReports := all || reports
	runInvoices := all || invoices
	runCreditNotes := all || creditNotes
	runEntries := all || entries
	runVouchers := all || vouchers

	if runReports {
		if err := backup.BackupReports(client, outDir, dryRun); err != nil {
			log.Printf("Error backing up reports: %v", err)
		}
	}

	if runInvoices {
		if err := backup.BackupInvoices(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up invoices: %v", err)
		}
	}

	if runCreditNotes {
		if err := backup.BackupCreditNotes(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up credit notes: %v", err)
		}
	}

	if runEntries {
		if err := backup.BackupEntries(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up entries: %v", err)
		}
	}

	if runVouchers {
		if err := backup.BackupVouchers(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up vouchers: %v", err)
		}
	}

	log.Println("Backup completed successfully.")
}
