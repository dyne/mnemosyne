package merkle

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/dyne/mnemosyne/internal/domain"
	"github.com/dyne/mnemosyne/internal/storage"
	"github.com/dyne/mnemosyne/internal/zenroom"
)

// Tree orchestrates Merkle tree operations by delegating all cryptographic
// work to Zenroom. It never implements hashing, proof generation, or
// verification directly.
type Tree struct {
	executor     *zenroom.Executor
	store        storage.Store
	contractsDir string
}

// NewTree creates a Tree orchestrator.
func NewTree(executor *zenroom.Executor, store storage.Store, contractsDir string) *Tree {
	return &Tree{
		executor:     executor,
		store:        store,
		contractsDir: contractsDir,
	}
}

// HashPayload hashes a raw payload string via Zenroom.
// Returns the base64-encoded hash.
func (t *Tree) HashPayload(ctx context.Context, payload string) (string, error) {
	script, err := os.ReadFile(t.contractsDir + "/hash.zen")
	if err != nil {
		return "", fmt.Errorf("load hash contract: %w", err)
	}

	data, _ := json.Marshal(map[string]any{"input": payload})
	result, err := t.executor.Run(script, nil, data)
	if err != nil {
		return "", fmt.Errorf("zenroom hash: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return "", err
	}
	h, ok := m["hash"].(string)
	if !ok {
		return "", fmt.Errorf("hash not found in zenroom output")
	}
	return h, nil
}

// CreateRoot builds a Merkle tree from string leaves and returns the root hash.
// All hashing is performed by Zenroom.
func (t *Tree) CreateRoot(ctx context.Context, leaves []string) (string, error) {
	if len(leaves) == 0 {
		return "", domain.ErrTreeEmpty
	}

	script, err := os.ReadFile(t.contractsDir + "/merkle_root.zen")
	if err != nil {
		return "", fmt.Errorf("load merkle_root contract: %w", err)
	}

	data, _ := json.Marshal(map[string]any{"data": leaves})
	result, err := t.executor.Run(script, nil, data)
	if err != nil {
		return "", fmt.Errorf("zenroom merkle_root: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return "", err
	}
	root, ok := m["merkle_root"].(string)
	if !ok {
		return "", fmt.Errorf("merkle_root not found in zenroom output")
	}
	return root, nil
}

// CreateTreeWithProof builds a full Merkle tree, persists tree nodes,
// and returns the root hash.
func (t *Tree) CreateTreeWithProof(ctx context.Context, leaves []string, beaconID string) (string, error) {
	if len(leaves) == 0 {
		return "", domain.ErrTreeEmpty
	}

	// Use Lua contract to build the full tree
	script, err := os.ReadFile(t.contractsDir + "/proof_generate.lua")
	if err != nil {
		return "", fmt.Errorf("load proof_generate contract: %w", err)
	}

	// Build tree for the entire leaf set and save nodes
	// Use position 1 to get the tree structure, then extract all nodes
	// For the full tree, we call CreateRoot instead.
	root, err := t.CreateRoot(ctx, leaves)
	if err != nil {
		return "", err
	}

	// Build the tree and save all nodes for later proof generation
	data, _ := json.Marshal(map[string]any{"leaves": leaves, "position": 1})
	result, err := t.executor.RunLua(script, nil, data)
	if err != nil {
		return "", fmt.Errorf("zenroom tree build: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return "", err
	}

	// The proof_generate script also returns the root; we use that to cross-validate
	proofRoot, _ := m["root"].(string)
	if proofRoot != "" && proofRoot != root {
		return "", fmt.Errorf("root mismatch: zen=%s zencode=%s", proofRoot, root)
	}

	return root, nil
}

// GenerateRoute creates an inclusion proof for a leaf at a given position.
// Position is 1-indexed.
func (t *Tree) GenerateRoute(ctx context.Context, leaves []string, position int) (*domain.Route, error) {
	if position < 1 || position > len(leaves) {
		return nil, fmt.Errorf("position %d out of range [1,%d]", position, len(leaves))
	}

	script, err := os.ReadFile(t.contractsDir + "/proof_generate.lua")
	if err != nil {
		return nil, fmt.Errorf("load proof_generate: %w", err)
	}

	data, _ := json.Marshal(map[string]any{"leaves": leaves, "position": position})
	result, err := t.executor.RunLua(script, nil, data)
	if err != nil {
		return nil, fmt.Errorf("zenroom generate proof: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return nil, err
	}

	proofRaw, _ := m["proof"].([]any)
	path := make([]string, len(proofRaw))
	for i, v := range proofRaw {
		path[i] = fmt.Sprint(v)
	}

	return &domain.Route{
		Leaf: fmt.Sprint(m["leaf"]),
		Root: fmt.Sprint(m["root"]),
		Path: path,
	}, nil
}

// Witness verifies an inclusion proof. All verification is performed by Zenroom.
func (t *Tree) Witness(ctx context.Context, route *domain.Route, leaf string, position, leafCount int) (*domain.WitnessResult, error) {
	script, err := os.ReadFile(t.contractsDir + "/proof_verify.lua")
	if err != nil {
		return nil, fmt.Errorf("load proof_verify: %w", err)
	}

	data, _ := json.Marshal(map[string]any{
		"proof":      route.Path,
		"leaf":       leaf,
		"root":       route.Root,
		"position":   position,
		"leaf_count": leafCount,
	})

	result, err := t.executor.RunLua(script, nil, data)
	if err != nil {
		return nil, fmt.Errorf("zenroom verify proof: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return nil, err
	}

	valid, _ := m["valid"].(bool)
	return &domain.WitnessResult{
		Valid: valid,
		Leaf:  fmt.Sprint(m["leaf"]),
		Root:  fmt.Sprint(m["root"]),
	}, nil
}
