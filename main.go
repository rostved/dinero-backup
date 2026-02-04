package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rostved/dinero-backup/backup"
	"github.com/rostved/dinero-backup/dinero"
	"github.com/rostved/dinero-backup/state"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	outDir string
	debug  bool

	// Run command flags
	dryRun      bool
	csvOutput   bool
	reports     bool
	invoices    bool
	creditNotes bool
	entries     bool
	vouchers    bool
	contacts    bool
)

var rootCmd = &cobra.Command{
	Use:   "dinero-backup",
	Short: "CLI tool to backup Dinero ERP data",
}

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the backup",
	Run:   runBackup,
}

var stateCmd = &cobra.Command{
	Use:   "state",
	Short: "Display current backup state",
	Run:   showState,
}

var testConnectionCmd = &cobra.Command{
	Use:   "test-connection",
	Short: "Test API connection and credentials",
	Run:   testConnection,
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVar(&outDir, "out-dir", "output", "Output directory for backup files")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "Enable debug logging")

	// Run command flags
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Run without saving files or updating state")
	runCmd.Flags().BoolVar(&csvOutput, "csv", false, "Output entries in CSV format instead of JSON")
	runCmd.Flags().BoolVar(&reports, "reports", false, "Backup reports")
	runCmd.Flags().BoolVar(&invoices, "invoices", false, "Backup invoices")
	runCmd.Flags().BoolVar(&creditNotes, "creditnotes", false, "Backup credit notes")
	runCmd.Flags().BoolVar(&entries, "entries", false, "Backup entries")
	runCmd.Flags().BoolVar(&vouchers, "vouchers", false, "Backup vouchers")
	runCmd.Flags().BoolVar(&contacts, "contacts", false, "Backup contacts")

	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(stateCmd)
	rootCmd.AddCommand(testConnectionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}

func loadEnvAndOutDir(cmd *cobra.Command) {
	if debug {
		log.Println("Debug mode enabled")
	}

	err := godotenv.Load()
	if err != nil && debug {
		log.Println("Error loading .env file (optional)")
	}

	// Use OUT_DIR env var if --out-dir flag wasn't explicitly set
	if !cmd.Flags().Changed("out-dir") && !rootCmd.PersistentFlags().Changed("out-dir") {
		if envOutDir := os.Getenv("OUT_DIR"); envOutDir != "" {
			outDir = envOutDir
		}
	}

	// Expand tilde in path
	outDir = expandTilde(outDir)
}

func getAPIClient() (*dinero.Client, error) {
	clientID := os.Getenv("CLIENT_ID")
	clientSecret := os.Getenv("CLIENT_SECRET")
	apiKey := os.Getenv("API_KEY")
	orgID := os.Getenv("ORG_ID")

	if clientID == "" || clientSecret == "" || apiKey == "" || orgID == "" {
		return nil, fmt.Errorf("missing environment variables. Required: CLIENT_ID, CLIENT_SECRET, API_KEY, ORG_ID")
	}

	client := dinero.NewClient(clientID, clientSecret, apiKey, orgID)
	client.SetDebug(debug)
	return client, nil
}

func showState(cmd *cobra.Command, args []string) {
	loadEnvAndOutDir(cmd)

	statePath := filepath.Join(outDir, "state.json")
	stateManager := state.NewManager(statePath)

	if err := stateManager.Load(); err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("No state file found at %s\n", statePath)
			fmt.Println("Run 'dinero-backup run' to create an initial backup.")
			return
		}
		log.Fatalf("Error loading state: %v", err)
	}

	fmt.Printf("State file: %s\n\n", statePath)
	fmt.Println("Last sync times:")
	fmt.Printf("  Reports:      %s\n", stateManager.State.LastSync.Reports)
	fmt.Printf("  Invoices:     %s\n", stateManager.State.LastSync.Invoices)
	fmt.Printf("  Credit Notes: %s\n", stateManager.State.LastSync.CreditNotes)
	fmt.Printf("  Entries:      %s\n", stateManager.State.LastSync.Entries)
	fmt.Printf("  Vouchers:     %s\n", stateManager.State.LastSync.Vouchers)
	fmt.Printf("  Contacts:     %s\n", stateManager.State.LastSync.Contacts)

	if len(stateManager.State.EntriesInitializedYears) > 0 {
		years := make([]int, len(stateManager.State.EntriesInitializedYears))
		copy(years, stateManager.State.EntriesInitializedYears)
		sort.Ints(years)
		fmt.Printf("\nEntries initialized for years: %v\n", years)
	}
}

func testConnection(cmd *cobra.Command, args []string) {
	loadEnvAndOutDir(cmd)

	client, err := getAPIClient()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Testing API connection...")

	years, err := backup.GetAccountingYears(client)
	if err != nil {
		log.Fatalf("Connection failed: %v", err)
	}

	fmt.Println("Connection successful!")
	fmt.Printf("Found %d accounting year(s):\n", len(years))
	for _, year := range years {
		fmt.Printf("  - %d\n", year.Year())
	}
}

func runBackup(cmd *cobra.Command, args []string) {
	loadEnvAndOutDir(cmd)

	client, err := getAPIClient()
	if err != nil {
		log.Fatal(err)
	}

	// Ensure output directory exists before creating state file there
	if err := os.MkdirAll(outDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	stateManager := state.NewManager(filepath.Join(outDir, "state.json"))
	if err := stateManager.Load(); err != nil {
		log.Printf("Could not load state (starting fresh?): %v", err)
	}

	log.Printf("Starting backup to %s...", outDir)
	if dryRun {
		log.Println("DRY RUN MODE: No files will be written, state will not be updated.")
	}

	// Determine what to backup
	all := !reports && !invoices && !creditNotes && !entries && !vouchers && !contacts
	runReports := all || reports
	runInvoices := all || invoices
	runCreditNotes := all || creditNotes
	runEntries := all || entries
	runVouchers := all || vouchers
	runContacts := all || contacts

	var hasErrors bool

	if runReports {
		if err := backup.BackupReports(client, outDir, dryRun); err != nil {
			log.Printf("Error backing up reports: %v", err)
			hasErrors = true
		}
	}

	if runInvoices {
		if err := backup.BackupInvoices(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up invoices: %v", err)
			hasErrors = true
		}
	}

	if runCreditNotes {
		if err := backup.BackupCreditNotes(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up credit notes: %v", err)
			hasErrors = true
		}
	}

	if runEntries {
		if err := backup.BackupEntries(client, stateManager, outDir, dryRun, csvOutput); err != nil {
			log.Printf("Error backing up entries: %v", err)
			hasErrors = true
		}
	}

	if runVouchers {
		if err := backup.BackupVouchers(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up vouchers: %v", err)
			hasErrors = true
		}
	}

	if runContacts {
		if err := backup.BackupContacts(client, stateManager, outDir, dryRun); err != nil {
			log.Printf("Error backing up contacts: %v", err)
			hasErrors = true
		}
	}

	if hasErrors {
		log.Println("Backup completed with errors.")
		os.Exit(1)
	}
	log.Println("Backup completed successfully.")
}
