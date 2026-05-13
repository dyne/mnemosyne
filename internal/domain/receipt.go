package domain

// Receipt is a portable verification bundle that contains everything needed
// to verify a memory offline.
type Receipt struct {
	Version    string            `json:"receipt_version"`
	Memory     ReceiptMemory     `json:"memory"`
	Proof      ReceiptProof      `json:"proof"`
	Ledger     ReceiptLedger     `json:"ledger"`
	Checkpoint ReceiptCheckpoint `json:"checkpoint"`
	Anchor     ReceiptAnchor     `json:"anchor"`
}

// ReceiptMemory is the memory portion of a receipt.
type ReceiptMemory struct {
	ID       string `json:"id"`
	LeafHash string `json:"leaf_hash"`
}

// ReceiptProof is the Merkle proof portion of a receipt.
type ReceiptProof struct {
	RootID    string   `json:"root_id"`
	RootHash  string   `json:"root_hash"`
	Siblings  []string `json:"siblings"`
	Position  int      `json:"position"`
	LeafCount int      `json:"leaf_count"`
}

// ReceiptLedger is the ledger portion of a receipt.
type ReceiptLedger struct {
	Backend    string `json:"backend"`
	EventSeq   uint64 `json:"event_seq"`
	EventHash  string `json:"event_hash"`
	LedgerHead string `json:"ledger_head"`
}

// ReceiptCheckpoint is the checkpoint portion of a receipt.
type ReceiptCheckpoint struct {
	ID         string    `json:"id"`
	FromSeq    uint64    `json:"from_seq"`
	ToSeq      uint64    `json:"to_seq"`
	LedgerHead string    `json:"ledger_head"`
	Signature  Signature `json:"signature"`
}

// ReceiptAnchor is the anchor portion of a receipt.
type ReceiptAnchor struct {
	Backend string        `json:"backend"`
	Status  string        `json:"status"`
	Receipt AnchorReceipt `json:"receipt"`
}
