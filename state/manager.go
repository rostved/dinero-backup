package state

import (
	"encoding/json"
	"os"

)

type LastSync struct {
	Reports     string `json:"reports"`
	Invoices    string `json:"invoices"`
	CreditNotes string `json:"creditNotes"`
	Entries     string `json:"entries"`
	Vouchers    string `json:"vouchers"`
}

type State struct {
	LastSync LastSync `json:"lastSync"`
}

type Manager struct {
	Path  string
	State State
}

var DefaultState = State{
	LastSync: LastSync{
		Reports:     "2000-01-01T00:00:00.000Z",
		Invoices:    "2000-01-01T00:00:00.000Z",
		CreditNotes: "2000-01-01T00:00:00.000Z",
		Entries:     "2000-01-01T00:00:00.000Z",
		Vouchers:    "2000-01-01T00:00:00.000Z",
	},
}

func NewManager(path string) *Manager {
	return &Manager{
		Path:  path,
		State: DefaultState,
	}
}

func (m *Manager) Load() error {
	if _, err := os.Stat(m.Path); os.IsNotExist(err) {
		m.State = DefaultState
		return nil
	}

	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, &m.State); err != nil {
		return err
	}
	return nil
}

func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.State, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.Path, data, 0644)
}

func (m *Manager) UpdateInvoices(timestamp string) {
	m.State.LastSync.Invoices = timestamp
}

func (m *Manager) UpdateCreditNotes(timestamp string) {
	m.State.LastSync.CreditNotes = timestamp
}

func (m *Manager) UpdateEntries(timestamp string) {
	m.State.LastSync.Entries = timestamp
}

func (m *Manager) UpdateVouchers(timestamp string) {
	m.State.LastSync.Vouchers = timestamp
}

func (m *Manager) GetLastSyncInvoices() string {
    return m.State.LastSync.Invoices
}

func (m *Manager) GetLastSyncCreditNotes() string {
    return m.State.LastSync.CreditNotes
}

func (m *Manager) GetLastSyncEntries() string {
    return m.State.LastSync.Entries
}

func (m *Manager) GetLastSyncVouchers() string {
    return m.State.LastSync.Vouchers
}
