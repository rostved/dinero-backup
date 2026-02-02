package backup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// EntriesToCSV converts entries JSON data to CSV format matching Dinero's export
func EntriesToCSV(jsonData []byte) ([]byte, error) {
	var entries []Entry
	if err := json.Unmarshal(jsonData, &entries); err != nil {
		return nil, fmt.Errorf("failed to parse entries JSON: %w", err)
	}

	// Sort entries by AccountNumber, then by Date
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].AccountNumber != entries[j].AccountNumber {
			return entries[i].AccountNumber < entries[j].AccountNumber
		}
		return entries[i].Date < entries[j].Date
	})

	var buf bytes.Buffer

	// Write UTF-8 BOM for Excel compatibility
	buf.Write([]byte{0xEF, 0xBB, 0xBF})

	// Write header (CRLF line endings for Windows/Excel compatibility)
	buf.WriteString("Konto;Kontonavn;Dato;Bilag;Bilagstype;Tekst;Momstype;Beløb;Saldo\r\n")

	// Track running balance per account
	balances := make(map[int]float64)

	for _, entry := range entries {
		// Update running balance
		balances[entry.AccountNumber] += entry.Amount

		// Format voucher number
		bilag := ""
		if entry.VoucherNumber != nil {
			bilag = fmt.Sprintf("%d", *entry.VoucherNumber)
		}

		// Map voucher type to Danish label
		bilagstype := mapVoucherType(entry.VoucherType, entry.Type)

		// Format amount in Danish format (comma as decimal separator)
		beloeb := formatDanishNumber(entry.Amount)

		// Format running balance
		saldo := formatDanishNumber(balances[entry.AccountNumber])

		// Build CSV line (CRLF line endings for Windows/Excel compatibility)
		line := fmt.Sprintf("%d;%s;%s;%s;%s;%s;%s;%s;%s\r\n",
			entry.AccountNumber,
			entry.AccountName,
			entry.Date,
			bilag,
			bilagstype,
			entry.Description,
			entry.VatType,
			beloeb,
			saldo,
		)
		buf.WriteString(line)
	}

	return buf.Bytes(), nil
}

// mapVoucherType converts the API VoucherType to Danish label matching Dinero's export
func mapVoucherType(voucherType *string, entryType string) string {
	// Primo entries
	if entryType == "Primo" {
		return "---"
	}

	if voucherType == nil {
		return "---"
	}

	switch *voucherType {
	case "Sales":
		return "Salgsfaktura"
	case "Purchases":
		return "Køb"
	case "manuel":
		return "Finansbilag"
	default:
		return *voucherType
	}
}

// formatDanishNumber formats a number in Danish format (comma as decimal, dot as thousands)
func formatDanishNumber(n float64) string {
	// Format with 2 decimal places
	str := fmt.Sprintf("%.2f", n)

	// Split into integer and decimal parts
	parts := strings.Split(str, ".")
	intPart := parts[0]
	decPart := parts[1]

	// Handle negative numbers
	negative := false
	if strings.HasPrefix(intPart, "-") {
		negative = true
		intPart = intPart[1:]
	}

	// Add thousand separators (dots)
	var result strings.Builder
	for i, c := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			result.WriteByte('.')
		}
		result.WriteRune(c)
	}

	// Combine with comma as decimal separator
	formatted := result.String() + "," + decPart

	if negative {
		return "-" + formatted
	}
	return formatted
}
