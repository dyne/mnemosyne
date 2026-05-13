package domain

import "context"

// StorageBackend is the operational persistence interface.
// Storage is useful but not trusted alone — integrity is verified against the ledger and anchors.
type StorageBackend interface {
	SaveMemory(ctx context.Context, memory Memory) error
	GetMemory(ctx context.Context, id string) (Memory, error)

	SaveRoot(ctx context.Context, root Root) error
	GetRoot(ctx context.Context, id string) (Root, error)

	SaveProof(ctx context.Context, proof ProofRecord) error
	GetProof(ctx context.Context, memoryID string) (ProofRecord, error)

	SaveCheckpoint(ctx context.Context, checkpoint CheckpointRecord) error
	GetCheckpoint(ctx context.Context, id string) (CheckpointRecord, error)
	ListCheckpoints(ctx context.Context) ([]CheckpointRecord, error)

	SaveAnchor(ctx context.Context, anchor AnchorReceipt) error
	GetAnchor(ctx context.Context, id string) (AnchorReceipt, error)
	ListAnchors(ctx context.Context) ([]AnchorReceipt, error)

	Stats(ctx context.Context) (StorageStats, error)
	Close() error
}

// StorageStats holds aggregate numbers for the dashboard.
type StorageStats struct {
	TotalMemories    int `json:"total_memories"`
	TotalRoots       int `json:"total_roots"`
	TotalCheckpoints int `json:"total_checkpoints"`
	TotalAnchors     int `json:"total_anchors"`
}

// LedgerBackend is the tamper-evident event history.
// Append-only, no updates, no deletes. Every event links to the previous via hash.
type LedgerBackend interface {
	Append(ctx context.Context, event LedgerEvent) (LedgerReceipt, error)
	GetEvent(ctx context.Context, seq uint64) (LedgerEvent, error)
	ListEvents(ctx context.Context, opts LedgerListOptions) ([]LedgerEvent, error)
	LatestHead(ctx context.Context) (LedgerHead, error)
	Verify(ctx context.Context) (LedgerVerification, error)
	Close() error
}

// LedgerListOptions controls pagination for ListEvents.
type LedgerListOptions struct {
	FromSeq uint64
	Limit   int
}

// AnchorBackend is where roots or checkpoints are notarized externally.
type AnchorBackend interface {
	Name() string
	Anchor(ctx context.Context, hash string, metadata map[string]string) (AnchorReceipt, error)
	VerifyAnchor(ctx context.Context, receipt AnchorReceipt) (AnchorVerification, error)
}

// CryptoBackend defines the cryptographic boundary.
// All implementations must delegate to Zenroom — no hand-rolled crypto in Go.
type CryptoBackend interface {
	Canonicalize(ctx context.Context, input any) ([]byte, error)
	Hash(ctx context.Context, input []byte) ([]byte, error)

	BuildMerkleTree(ctx context.Context, leaves [][]byte) (MerkleTreeResult, error)
	GenerateProof(ctx context.Context, tree MerkleTreeResult, leafIndex int) (MerkleProof, error)
	VerifyProof(ctx context.Context, proof MerkleProof) (ProofVerification, error)

	Sign(ctx context.Context, payload []byte, keyRef string) (Signature, error)
	VerifySignature(ctx context.Context, payload []byte, sig Signature) (SignatureVerification, error)
}
