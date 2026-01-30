# dinero-backup

A CLI tool to backup data from [Dinero](https://dinero.dk) ERP. Downloads and stores invoices, credit notes, vouchers, accounting entries, and reports locally.

## Installation

Download the latest release for your platform from the [GitHub Releases](https://github.com/rostved/dinero-backup/releases) page.

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
| `--invoices` | Backup invoices |
| `--creditnotes` | Backup credit notes |
| `--entries` | Backup accounting entries |
| `--vouchers` | Backup vouchers |
| `--out-dir` | Output directory (default: `backup`) |
| `--dry-run` | Run without saving files or updating state |
| `--debug` | Enable debug logging |

If no specific type flags are provided, all data types are backed up.

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
