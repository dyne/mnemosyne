package domain

// Route is an inclusion proof — a cryptographic path from a leaf (memory)
// to a root (constellation), verifying that the memory is part of the tree.
type Route struct {
	Leaf string   `json:"leaf"`
	Root string   `json:"root"`
	Path []string `json:"path"`
}

// WitnessResult is the outcome of verifying an inclusion proof.
type WitnessResult struct {
	Valid    bool   `json:"valid"`
	Leaf     string `json:"leaf"`
	Root     string `json:"root"`
	MemoryID string `json:"memory_id,omitempty"`
}
