# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.3.0] - 2026-02-04

### Added
- Contacts backup from `/v2/contacts` endpoint with pagination and incremental sync

### Fixed
- 31-day chunking for `/entries/changes` endpoint (API only allows 31 days per request)
- Preserve entry order when merging changes (diffs now show actual changes, not reordering)
- Proper error reporting when entry backup fails (no longer reports success on failure)
- Only update lastSync timestamps when API returns data (handles unstable endpoints)

### Changed
- Simplified voucher type mapping to always return "Finansbilag" for manuel entries

## [0.2.0] - 2026-02-01

### Added
- CLI subcommands: `run`, `state`, `test-connection`
- CSV export for entries with `--csv` flag
- Full accounting year exports (01-01 to 12-31) with primo values
- State tracking for initialized entry years
- `OUT_DIR` environment variable support
- Linux builds (amd64, arm64)

### Changed
- State file now stored in output directory
- First run uses `/entries` (includes primo), subsequent runs use `/entries/changes`

### Fixed
- Tilde expansion for paths (`~/path` now works)

## [0.1.1] - 2026-01-31

### Fixed
- Invoice PDF downloads for booked invoices
- Vouchers endpoint changed to `/files`

## [0.1] - 2026-01-30

- Initial release
