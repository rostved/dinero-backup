package backup

// PaginatedResponse represents Dinero's paginated API response format
type PaginatedResponse struct {
	Collection []any `json:"Collection"`
	Pagination any   `json:"Pagination"`
}

// Invoice represents a Dinero invoice
type Invoice struct {
	Guid   string `json:"Guid"`
	Number int    `json:"Number"`
	Status string `json:"Status"`
}

// InvoiceResponse represents Dinero's paginated invoice response
type InvoiceResponse struct {
	Collection []Invoice `json:"Collection"`
	Pagination any       `json:"Pagination"`
}

// Entry represents an accounting entry with voucher reference
type Entry struct {
	AccountNumber int     `json:"AccountNumber"`
	AccountName   string  `json:"AccountName"`
	Date          string  `json:"Date"`
	VoucherNumber *int    `json:"VoucherNumber"`
	VoucherType   *string `json:"VoucherType"`
	Description   string  `json:"Description"`
	VatType       string  `json:"VatType"`
	VatCode       string  `json:"VatCode"`
	Amount        float64 `json:"Amount"`
	EntryGuid     string  `json:"EntryGuid"`
	ContactGuid   *string `json:"ContactGuid"`
	Type          string  `json:"Type"`
}

// PurchaseVoucher represents a purchase voucher with file reference
type PurchaseVoucher struct {
	Guid     string `json:"Guid"`
	FileGuid string `json:"FileGuid"`
	Number   int    `json:"Number"`
}

// AccountingYear represents a Dinero accounting year
type AccountingYear struct {
	FromDate  string `json:"FromDate"`
	DateStart string `json:"dateStart"`
	ToDate    string `json:"ToDate"`
	DateEnd   string `json:"dateEnd"`
	Name      string `json:"name"`
}
