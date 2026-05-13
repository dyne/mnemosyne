package domain

import "time"

// BeaconID identifies a checkpoint (constellation root).
type BeaconID string

// Beacon is a signed checkpoint anchoring the Merkle tree at a point in time.
type Beacon struct {
	ID             BeaconID  `json:"beacon_id"`
	Root           string    `json:"root"`
	ParentBeaconID string    `json:"parent_beacon_id,omitempty"`
	SignedRoot     string    `json:"signed_root"`
	ProofCount     int       `json:"proof_count"`
	CreatedAt      time.Time `json:"created_at"`
}
