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
	RootID    string    `json:"root_id,omitempty"`
	CreatedAt time.Time `json:"inserted_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// Recollection is a stored leaf representation.
type Recollection struct {
	MemoryID  MemoryID  `json:"memory_id"`
	LeafHash  string    `json:"leaf_hash"`
	RootID    string    `json:"root_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// MemoryMetadata holds metadata updates for an existing memory.
type MemoryMetadata struct {
	Tags       []string          `json:"tags,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	References []string          `json:"references,omitempty"`
}
