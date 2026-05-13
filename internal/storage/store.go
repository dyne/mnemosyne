package storage

import (
	"context"
	"time"

	"github.com/dyne/mnemosyne/internal/domain"
)

// Store is the persistence interface for Mnemosyne.
// All methods are append-only by design — updates and deletes are forbidden.
type Store interface {
	// Remember stores a new memory. Returns the created memory with its ID set.
	Remember(ctx context.Context, payload any, leafHash string, beaconID string) (*domain.Memory, error)

	// Recall retrieves a memory by ID.
	Recall(ctx context.Context, id domain.MemoryID) (*domain.Memory, error)

	// MemoriesByBeacon returns all memories anchored to a given beacon.
	MemoriesByBeacon(ctx context.Context, beaconID domain.BeaconID) ([]*domain.Memory, error)

	// AnchorBeacon persists a new beacon (checkpoint).
	AnchorBeacon(ctx context.Context, beacon *domain.Beacon) error

	// LatestBeacon returns the most recent beacon.
	LatestBeacon(ctx context.Context) (*domain.Beacon, error)

	// BeaconByID returns a beacon by its ID.
	BeaconByID(ctx context.Context, id domain.BeaconID) (*domain.Beacon, error)

	// UpdateBeaconID sets the beacon_id for all memories that currently have oldBeaconID.
	UpdateBeaconID(ctx context.Context, oldBeaconID, newBeaconID string) error

	// SaveTreeNode stores a Merkle tree node hash.
	SaveTreeNode(ctx context.Context, beaconID string, index int, hash string) error

	// TreeNodesByBeacon returns all tree nodes for a beacon, ordered by index.
	TreeNodesByBeacon(ctx context.Context, beaconID string) ([]TreeNode, error)

	// Close releases any resources held by the store.
	Close() error
}

// TreeNode is a single node in a Merkle tree persisted for proof generation.
type TreeNode struct {
	Index int
	Hash  string
}

// NewMemoryID generates a unique memory ID based on the current time.
// NOT cryptographic — identity is established by the leaf hash from Zenroom.
func NewMemoryID() domain.MemoryID {
	return domain.MemoryID(time.Now().UTC().Format("20060102T150405.000000Z"))
}

// NewBeaconID generates a unique beacon ID.
func NewBeaconID() domain.BeaconID {
	return domain.BeaconID(time.Now().UTC().Format("20060102T150405.000000Z"))
}
