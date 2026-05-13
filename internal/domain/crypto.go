package domain

import "time"

// Signature holds a Zenroom-produced signature.
type Signature struct {
	Type      string `json:"type"`
	Value     string `json:"value"`
	PublicKey string `json:"public_key,omitempty"`
	KeyRef    string `json:"key_ref,omitempty"`
}

// SignatureVerification is the outcome of a signature check.
type SignatureVerification struct {
	Valid   bool   `json:"valid"`
	Details string `json:"details,omitempty"`
}

// Hash is a Zenroom-computed hash value.
type Hash string

// MerkleTreeResult holds a complete Merkle tree.
type MerkleTreeResult struct {
	Root   string     `json:"root"`
	Leaves []string   `json:"leaves"`
	Layers [][]string `json:"layers,omitempty"`
}

// MerkleProof is an inclusion proof for a single leaf.
type MerkleProof struct {
	Leaf      string   `json:"leaf"`
	Root      string   `json:"root"`
	Siblings  []string `json:"siblings"`
	Position  int      `json:"position"`
	LeafCount int      `json:"leaf_count"`
}

// ProofVerification is the outcome of a Merkle proof check.
type ProofVerification struct {
	Valid   bool   `json:"valid"`
	Leaf    string `json:"leaf"`
	Root    string `json:"root"`
	Details string `json:"details,omitempty"`
}

// Root is a sealed Merkle root.
type Root struct {
	RootID    string    `json:"root_id"`
	RootHash  string    `json:"root_hash"`
	LeafCount int       `json:"leaf_count"`
	CreatedAt time.Time `json:"created_at"`
}

// ProofRecord is a stored Merkle proof.
type ProofRecord struct {
	MemoryID  string   `json:"memory_id"`
	RootID    string   `json:"root_id"`
	Leaf      string   `json:"leaf"`
	Root      string   `json:"root"`
	Siblings  []string `json:"siblings"`
	Position  int      `json:"position"`
	LeafCount int      `json:"leaf_count"`
}

// CheckpointRecord is a signed ledger head over a range of events.
type CheckpointRecord struct {
	CheckpointID string    `json:"checkpoint_id"`
	FromSeq      uint64    `json:"from_seq"`
	ToSeq        uint64    `json:"to_seq"`
	LedgerHead   string    `json:"ledger_head"`
	EventCount   int       `json:"event_count"`
	CreatedAt    time.Time `json:"created_at"`
	Signature    Signature `json:"signature"`
}

// CheckpointVerification is the result of verifying a checkpoint.
type CheckpointVerification struct {
	Valid   bool   `json:"valid"`
	Details string `json:"details"`
}
