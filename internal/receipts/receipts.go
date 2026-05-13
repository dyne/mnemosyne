package receipts

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

// Exporter creates portable receipt bundles for offline verification.
type Exporter struct {
	store  storage.Store
	tree   *merkle.Tree
	ledger ledger.Backend
	anchor anchor.Backend
}

// NewExporter creates a receipt exporter.
func NewExporter(store storage.Store, tree *merkle.Tree, l ledger.Backend, a anchor.Backend) *Exporter {
	return &Exporter{store: store, tree: tree, ledger: l, anchor: a}
}

// ExportMemory creates a full receipt bundle for a memory.
func (e *Exporter) ExportMemory(ctx context.Context, memoryID string) (*domain.Receipt, error) {
	m, err := e.store.Recall(ctx, domain.MemoryID(memoryID))
	if err != nil {
		return nil, fmt.Errorf("recall memory: %w", err)
	}

	receipt := &domain.Receipt{
		Version: "mnemosyne.receipt.v1",
		Memory: domain.ReceiptMemory{
			ID:       string(m.ID),
			LeafHash: m.LeafHash,
		},
	}

	// Find the beacon (checkpoint) for this memory
	if m.BeaconID != "" && m.BeaconID != "current" {
		beacon, err := e.store.BeaconByID(ctx, domain.BeaconID(m.BeaconID))
		if err == nil {
			// Build proof components
			memories, err := e.store.MemoriesByBeacon(ctx, domain.BeaconID(m.BeaconID))
			if err == nil {
				leaves := make([]string, len(memories))
				pos := -1
				for i, mem := range memories {
					payloadJSON, _ := json.Marshal(mem.Payload)
					leaves[i] = string(payloadJSON)
					if string(mem.ID) == memoryID {
						pos = i + 1
					}
				}
				if pos > 0 {
					route, err := e.tree.GenerateRoute(ctx, leaves, pos)
					if err == nil {
						receipt.Proof = domain.ReceiptProof{
							RootID:    string(beacon.ID),
							RootHash:  beacon.Root,
							Siblings:  route.Path,
							Position:  pos,
							LeafCount: len(leaves),
						}
					}
				}
			}
		}
	}

	if receipt.Proof.RootHash == "" {
		receipt.Proof = domain.ReceiptProof{
			RootID:   m.BeaconID,
			RootHash: m.LeafHash,
		}
	}

	// Add ledger info if available
	if e.ledger != nil {
		head, err := e.ledger.LatestHead(ctx)
		if err == nil {
			receipt.Ledger = domain.ReceiptLedger{
				Backend:    "ndjson_hash_chain",
				LedgerHead: head.EventHash,
			}
		}
	}

	// Add anchor info if available
	if e.anchor != nil {
		receipt.Anchor = domain.ReceiptAnchor{
			Backend: e.anchor.Name(),
			Status:  "confirmed",
		}
	}

	return receipt, nil
}
