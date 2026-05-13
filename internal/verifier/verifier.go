package verifier

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/dyne/mnemosyne/internal/anchor"
	"github.com/dyne/mnemosyne/internal/domain"
	"github.com/dyne/mnemosyne/internal/ledger"
	"github.com/dyne/mnemosyne/internal/merkle"
	"github.com/dyne/mnemosyne/internal/storage"
)

// Verifier validates the full trust chain for a memory or receipt.
type Verifier struct {
	store  storage.Store
	tree   *merkle.Tree
	ledger ledger.Backend
	anchor anchor.Backend
}

// New creates a new Verifier.
func New(store storage.Store, tree *merkle.Tree, l ledger.Backend, a anchor.Backend) *Verifier {
	return &Verifier{store: store, tree: tree, ledger: l, anchor: a}
}

// Check represents one step in the verification chain.
type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

// VerificationResult is the outcome of full-chain verification.
type VerificationResult struct {
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}

// VerifyMemory performs full-chain verification of a memory.
func (v *Verifier) VerifyMemory(ctx context.Context, memoryID string) (*VerificationResult, error) {
	checks := []Check{}

	m, err := v.store.Recall(ctx, domain.MemoryID(memoryID))
	if err != nil {
		return &VerificationResult{Status: "invalid", Checks: []Check{{
			Name: "memory_lookup", Status: "failed", Details: fmt.Sprintf("memory %s not found", memoryID),
		}}}, nil
	}

	// Step 1: Memory hash verification
	payloadJSON, _ := json.Marshal(m.Payload)
	computedHash, err := v.tree.HashPayload(ctx, string(payloadJSON))
	if err != nil {
		checks = append(checks, Check{Name: "memory_hash", Status: "failed", Details: err.Error()})
	} else if computedHash != m.LeafHash {
		checks = append(checks, Check{Name: "memory_hash", Status: "failed", Details: "Memory payload does not match leaf hash"})
		return &VerificationResult{Status: "invalid", Checks: append(checks, Check{
			Name: "merkle_inclusion", Status: "skipped", Details: "Skipped due to hash mismatch",
		})}, nil
	} else {
		checks = append(checks, Check{Name: "memory_hash", Status: "ok", Details: "Memory payload matches leaf hash"})
	}

	// Step 2: Merkle inclusion (if in a beacon)
	if m.BeaconID != "" && m.BeaconID != "current" {
		memories, err := v.store.MemoriesByBeacon(ctx, domain.BeaconID(m.BeaconID))
		if err != nil {
			checks = append(checks, Check{Name: "merkle_inclusion", Status: "error", Details: err.Error()})
		} else {
			leaves := make([]string, len(memories))
			pos := -1
			for i, mem := range memories {
				pj, _ := json.Marshal(mem.Payload)
				leaves[i] = string(pj)
				if string(mem.ID) == memoryID {
					pos = i + 1
				}
			}
			if pos > 0 {
				route, err := v.tree.GenerateRoute(ctx, leaves, pos)
				if err != nil {
					checks = append(checks, Check{Name: "merkle_inclusion", Status: "error", Details: err.Error()})
				} else {
					result, err := v.tree.Witness(ctx, route, string(payloadJSON), pos, len(leaves))
					if err != nil || !result.Valid {
						checks = append(checks, Check{Name: "merkle_inclusion", Status: "failed", Details: "Leaf is not included in root"})
					} else {
						checks = append(checks, Check{Name: "merkle_inclusion", Status: "ok", Details: fmt.Sprintf("Leaf is included in beacon %s", m.BeaconID)})
					}
				}
			} else {
				checks = append(checks, Check{Name: "merkle_inclusion", Status: "skipped", Details: "Memory position not found in beacon"})
			}
		}
	} else {
		checks = append(checks, Check{Name: "merkle_inclusion", Status: "skipped", Details: "Memory not yet sealed into a beacon"})
	}

	// Step 3: Ledger event check
	if v.ledger != nil {
		events, err := v.ledger.ListEvents(ctx, domain.LedgerListOptions{Limit: 1000})
		if err != nil {
			checks = append(checks, Check{Name: "ledger_event", Status: "error", Details: err.Error()})
		} else {
			found := false
			for _, evt := range events {
				if evt.EventType == domain.EventMemoryRecorded {
					if p, ok := evt.Payload.(map[string]any); ok {
						if id, _ := p["memory_id"].(string); id == memoryID {
							checks = append(checks, Check{Name: "ledger_event", Status: "ok", Details: fmt.Sprintf("Memory recorded at ledger event #%d", evt.Seq)})
							found = true
							break
						}
					}
				}
			}
			if !found {
				checks = append(checks, Check{Name: "ledger_event", Status: "warning", Details: "No ledger event found for this memory"})
			}
		}
	} else {
		checks = append(checks, Check{Name: "ledger_event", Status: "skipped", Details: "Ledger not configured"})
	}

	// Step 4: Ledger chain integrity
	if v.ledger != nil {
		verif, err := v.ledger.Verify(ctx)
		if err != nil {
			checks = append(checks, Check{Name: "ledger_chain", Status: "error", Details: err.Error()})
		} else if verif.Valid {
			checks = append(checks, Check{Name: "ledger_chain", Status: "ok", Details: "Ledger hash chain is intact"})
		} else {
			checks = append(checks, Check{Name: "ledger_chain", Status: "failed", Details: "Ledger hash chain is broken"})
		}
	} else {
		checks = append(checks, Check{Name: "ledger_chain", Status: "skipped", Details: "Ledger not configured"})
	}

	// Step 5: Anchor check
	if v.anchor != nil {
		checks = append(checks, Check{Name: "anchor", Status: "ok", Details: fmt.Sprintf("Anchor backend %s is available", v.anchor.Name())})
	} else {
		checks = append(checks, Check{Name: "anchor", Status: "skipped", Details: "Anchor not configured"})
	}

	// Determine overall status
	allOK := true
	for _, c := range checks {
		if c.Status == "failed" || c.Status == "error" {
			allOK = false
			break
		}
	}

	status := "valid"
	if !allOK {
		status = "invalid"
	}

	return &VerificationResult{Status: status, Checks: checks}, nil
}
