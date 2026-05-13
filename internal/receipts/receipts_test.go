package receipts

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dyne/mnemosyne/internal/anchor/local"
	"github.com/dyne/mnemosyne/internal/ledger/ndjson"
	"github.com/dyne/mnemosyne/internal/merkle"
	"github.com/dyne/mnemosyne/internal/storage"
	"github.com/dyne/mnemosyne/internal/zenroom"
)

func zenroomBin() string {
	for _, p := range []string{"zenroom", "/usr/bin/zenroom", "/usr/local/bin/zenroom"} {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return ""
}

func newTestExporter(t *testing.T) *Exporter {
	t.Helper()
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}

	tmpDir := t.TempDir()
	contractsDir := "../../zenflows"

	store, err := storage.NewSQLiteStore(filepath.Join(tmpDir, "test.db"))
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	executor := zenroom.NewExecutor(bin)
	tree := merkle.NewTree(executor, store, contractsDir)

	ledgerPath := filepath.Join(tmpDir, "ledger.ndjson")
	l, err := ndjson.New(ledgerPath, contractsDir, "test-key", executor)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}

	anchor, err := local.New(contractsDir, "test-key", executor)
	if err != nil {
		t.Fatalf("anchor: %v", err)
	}

	return NewExporter(store, tree, l, anchor)
}

func TestExportMemory_NotFound(t *testing.T) {
	e := newTestExporter(t)
	_, err := e.ExportMemory(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent memory")
	}
}

func TestExportMemory_NoBeacon(t *testing.T) {
	e := newTestExporter(t)
	store := e.store

	payload := json.RawMessage(`{"note":"hello"}`)
	hash, _ := e.tree.HashPayload(context.Background(), string(payload))

	mem, err := store.Remember(context.Background(), payload, hash, "current")
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}

	receipt, err := e.ExportMemory(context.Background(), string(mem.ID))
	if err != nil {
		t.Fatalf("ExportMemory: %v", err)
	}
	if receipt.Version != "mnemosyne.receipt.v1" {
		t.Errorf("expected receipt version mnemosyne.receipt.v1, got %s", receipt.Version)
	}
	if receipt.Proof.RootHash != hash {
		t.Errorf("expected root hash %s, got %s", hash, receipt.Proof.RootHash)
	}
}

func TestExportMemory_WithLedgerAnchor(t *testing.T) {
	e := newTestExporter(t)
	store := e.store

	payload := json.RawMessage(`{"note":"hello"}`)
	hash, _ := e.tree.HashPayload(context.Background(), string(payload))

	mem, err := store.Remember(context.Background(), payload, hash, "current")
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}

	receipt, err := e.ExportMemory(context.Background(), string(mem.ID))
	if err != nil {
		t.Fatalf("ExportMemory: %v", err)
	}
	if receipt.Ledger.Backend != "ndjson_hash_chain" {
		t.Errorf("expected ledger backend ndjson_hash_chain, got %s", receipt.Ledger.Backend)
	}
	if receipt.Anchor.Backend != "local_signature" {
		t.Errorf("expected anchor backend local_signature, got %s", receipt.Anchor.Backend)
	}
}

func TestNewExporter(t *testing.T) {
	e := newTestExporter(t)
	if e == nil {
		t.Fatal("expected non-nil exporter")
	}
	if e.store == nil {
		t.Error("expected store to be set")
	}
	if e.tree == nil {
		t.Error("expected tree to be set")
	}
}
