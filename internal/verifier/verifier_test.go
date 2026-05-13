package verifier

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

func newTestVerifier(t *testing.T) *Verifier {
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

	return New(store, tree, l, anchor)
}

func TestVerifyMemory_NotFound(t *testing.T) {
	v := newTestVerifier(t)
	result, err := v.VerifyMemory(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("VerifyMemory: %v", err)
	}
	if result.Status != "invalid" {
		t.Errorf("expected status invalid, got %s", result.Status)
	}
}

func TestVerifyMemory_NoBeacon(t *testing.T) {
	v := newTestVerifier(t)
	store := v.store

	payload := json.RawMessage(`{"note":"test"}`)
	hash, _ := v.tree.HashPayload(context.Background(), string(payload))

	mem, err := store.Remember(context.Background(), payload, hash, "current")
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}

	result, err := v.VerifyMemory(context.Background(), string(mem.ID))
	if err != nil {
		t.Fatalf("VerifyMemory: %v", err)
	}
	if result.Status != "valid" {
		t.Errorf("expected status valid, got %s", result.Status)
	}
	if len(result.Checks) == 0 {
		t.Error("expected at least one check")
	}
}

func TestVerifyMemory_WithLedgerAnchor(t *testing.T) {
	v := newTestVerifier(t)
	store := v.store

	payload := json.RawMessage(`{"note":"test"}`)
	hash, _ := v.tree.HashPayload(context.Background(), string(payload))

	mem, err := store.Remember(context.Background(), payload, hash, "current")
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}

	result, err := v.VerifyMemory(context.Background(), string(mem.ID))
	if err != nil {
		t.Fatalf("VerifyMemory: %v", err)
	}
	foundAnchor := false
	foundLedgerEvent := false
	for _, c := range result.Checks {
		if c.Name == "anchor" && c.Status == "ok" {
			foundAnchor = true
		}
		if c.Name == "ledger_event" {
			foundLedgerEvent = true
		}
	}
	if !foundAnchor {
		t.Error("expected anchor check")
	}
	if !foundLedgerEvent {
		t.Error("expected ledger_event check")
	}
	if result.Status != "valid" {
		t.Errorf("expected status valid, got %s", result.Status)
	}
}

func TestNewVerifier(t *testing.T) {
	v := newTestVerifier(t)
	if v == nil {
		t.Fatal("expected non-nil verifier")
	}
	if v.store == nil {
		t.Error("expected store to be set")
	}
	if v.tree == nil {
		t.Error("expected tree to be set")
	}
}
