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

// VoucherResponse represents Dinero's paginated voucher response
type VoucherResponse struct {
	Collection []Voucher `json:"Collection"`
	Pagination any       `json:"Pagination"`
}
