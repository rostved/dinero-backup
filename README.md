# dinero-backup

A CLI tool to backup data from [Dinero](https://dinero.dk) ERP. Downloads and stores invoices, credit notes, vouchers (bilag), accounting entries (posteringer), and reports locally.

## Installation

Download the latest release for your platform from the [GitHub Releases](https://github.com/rostved/dinero-backup/releases) page.

### macOS

macOS may block the binary because it's not signed. Remove the quarantine attribute to run it:

```bash
xattr -d com.apple.quarantine ./dinero-backup-macos-*
chmod +x ./dinero-backup-macos-*
```

## Configuration

### 1. Obtain API credentials

Request a personal API key from Dinero by following their documentation:
https://developer.dinero.dk/documentation/personal-integration/

You will need:
- **Client ID** and **Client Secret** (provided by Dinero)
- **API Key** (your personal integration key)
- **Organization ID** (your Dinero organization ID)

### 2. Set environment variables

Create a `.env` file in the same directory as the binary, or export the variables directly:

```bash
CLIENT_ID=your_client_id
CLIENT_SECRET=your_client_secret
API_KEY=your_api_key
ORG_ID=your_organization_id
OUT_DIR=./my-backup  # Optional, defaults to "output"
```

### 3. Run the backup

```bash
# Backup everything
./dinero-backup

# Backup specific data types
./dinero-backup --invoices --creditnotes

# Preview without writing files
./dinero-backup --dry-run

# Enable debug logging
./dinero-backup --debug

# Specify output directory
./dinero-backup --out-dir ./my-backup
```

### Available flags

| Flag | Description |
|------|-------------|
| `--reports` | Backup reports |
| `--invoices` | Backup invoices (includes PDFs) |
| `--creditnotes` | Backup credit notes |
| `--entries` | Backup accounting entries |
| `--vouchers` | Backup voucher files |
| `--csv` | Export entries in CSV format (in addition to JSON) |
| `--out-dir` | Output directory (default: `output`, or `OUT_DIR` env var) |
| `--dry-run` | Run without saving files or updating state |
| `--debug` | Enable debug logging |

If no specific type flags are provided, all data types are backed up.

## How it works

### Entries backup

Entries are exported as full accounting years (January 1 to December 31):

- **First run**: Uses `/entries` endpoint which includes primo (opening balance) values
- **Subsequent runs**: Uses `/entries/changes` endpoint for efficient incremental updates
- Changes are merged into existing year files to preserve primo values

Files are saved as `entries_YYYY.json` (and `entries_YYYY.csv` with `--csv` flag).

### Incremental backups

The tool tracks sync state in `state.json` to enable incremental backups. Only new or changed data is fetched on subsequent runs.

---

## Development

### Prerequisites

- Go 1.21 or later

### Build

```bash
go build -o dist/dinero-backup .
```

### Run from source

```bash
go run .
```
