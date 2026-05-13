package domain

import "time"

// MemoryID is a unique identifier for a memory entry.
type MemoryID string

// Memory represents an immutable record stored in the archive.
type Memory struct {
	ID        MemoryID  `json:"memory_id"`
	Payload   any       `json:"payload"`
	LeafHash  string    `json:"leaf_hash"`
	BeaconID  string    `json:"beacon_id"`
	CreatedAt time.Time `json:"inserted_at"`
}
