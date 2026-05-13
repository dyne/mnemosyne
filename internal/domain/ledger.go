package domain

import "time"

// LedgerEvent is a single entry in the append-only ledger.
// event_hash is computed by Zenroom over the canonical body excluding signature.
type LedgerEvent struct {
	Seq          uint64    `json:"seq"`
	EventType    EventType `json:"event_type"`
	Payload      any       `json:"payload"`
	PreviousHash string    `json:"previous_hash"`
	EventHash    string    `json:"event_hash"`
	CreatedAt    time.Time `json:"created_at"`
	Signature    Signature `json:"signature"`
}

// LedgerReceipt is returned after a successful append.
type LedgerReceipt struct {
	Seq       uint64 `json:"seq"`
	EventHash string `json:"event_hash"`
	Head      string `json:"ledger_head"`
}

// LedgerHead is the latest state of the ledger.
type LedgerHead struct {
	Seq       uint64    `json:"seq"`
	EventHash string    `json:"event_hash"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LedgerVerification is the outcome of a full chain check.
type LedgerVerification struct {
	Valid         bool     `json:"valid"`
	TotalEvents   uint64   `json:"total_events"`
	FirstHash     string   `json:"first_hash"`
	LastHash      string   `json:"last_hash"`
	InvalidEvents []uint64 `json:"invalid_events,omitempty"`
}

// LedgerListOptions controls pagination for listing events.
// (Re-exported alias for the interface.)
