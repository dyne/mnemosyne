package domain

import "time"

// AnchorReceipt records that a checkpoint or root was notarized.
type AnchorReceipt struct {
	AnchorID       string            `json:"anchor_id"`
	Backend        string            `json:"backend"`
	AnchoredType   string            `json:"anchored_type"`
	AnchoredID     string            `json:"anchored_id"`
	AnchoredHash   string            `json:"anchored_hash"`
	Status         string            `json:"status"`
	CreatedAt      time.Time         `json:"created_at"`
	Signature      *Signature        `json:"signature,omitempty"`
	ExternalRef    string            `json:"external_reference,omitempty"`
	RawReceiptPath string            `json:"raw_receipt_path,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// AnchorVerification is the result of checking an anchor receipt.
type AnchorVerification struct {
	Valid   bool   `json:"valid"`
	Details string `json:"details"`
}

// AnchorListOptions controls pagination for listing anchors.
type AnchorListOptions struct {
	Backend string
	Limit   int
}
