package verifier

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dyne/mnemosyne/internal/anchor/local"
	"github.com/dyne/mnemosyne/internal/domain"
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

func newTestVerifierNoBackends(t *testing.T) *Verifier {
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

	return New(store, tree, nil, nil)
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

func TestVerifyMemory_AfterAnchor(t *testing.T) {
	v := newTestVerifier(t)
	store := v.store

	// Create and anchor memories
	var memIDs []string
	for i := 0; i < 3; i++ {
		payload := json.RawMessage(`{"idx":` + string(rune('0'+i)) + `}`)
		hash, _ := v.tree.HashPayload(context.Background(), string(payload))
		mem, err := store.Remember(context.Background(), payload, hash, "current")
		if err != nil {
			t.Fatalf("Remember %d: %v", i, err)
		}
		memIDs = append(memIDs, string(mem.ID))
	}

	leaves := []string{`{"idx":0}`, `{"idx":1}`, `{"idx":2}`}
	root, err := v.tree.CreateRoot(context.Background(), leaves)
	if err != nil {
		t.Fatalf("CreateRoot: %v", err)
	}

	beaconID := storage.NewBeaconID()
	beacon := &domain.Beacon{
		ID:             beaconID,
		Root:           root,
		ParentBeaconID: "",
		ProofCount:     len(leaves),
	}
	if err := store.AnchorBeacon(context.Background(), beacon); err != nil {
		t.Fatalf("AnchorBeacon: %v", err)
	}
	if err := store.UpdateBeaconID(context.Background(), "current", string(beaconID)); err != nil {
		t.Fatalf("UpdateBeaconID: %v", err)
	}

	// Verify anchored memory
	result, err := v.VerifyMemory(context.Background(), memIDs[1])
	if err != nil {
		t.Fatalf("VerifyMemory: %v", err)
	}
	if result.Status != "valid" {
		t.Errorf("expected status valid, got %s", result.Status)
	}
	// Should have merkle_inclusion check not skipped
	foundMerkle := false
	for _, c := range result.Checks {
		if c.Name == "merkle_inclusion" {
			foundMerkle = true
			if c.Status == "skipped" {
				t.Error("merkle_inclusion should not be skipped for anchored memory")
			}
		}
	}
	if !foundMerkle {
		t.Error("expected merkle_inclusion check")
	}
}

func TestVerifyMemory_NoLedgerAnchor(t *testing.T) {
	v := newTestVerifierNoBackends(t)
	store := v.store

	payload := json.RawMessage(`{"note":"solo"}`)
	hash, _ := v.tree.HashPayload(context.Background(), string(payload))
	mem, err := store.Remember(context.Background(), payload, hash, "current")
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}

	result, err := v.VerifyMemory(context.Background(), string(mem.ID))
	if err != nil {
		t.Fatalf("VerifyMemory: %v", err)
	}
	// Ledger and anchor should be skipped
	for _, c := range result.Checks {
		if (c.Name == "ledger_event" || c.Name == "ledger_chain" || c.Name == "anchor") && c.Status != "skipped" {
			t.Errorf("expected %s to be skipped, got %s", c.Name, c.Status)
		}
	}
}

func TestVerifyMemory_TamperedHash(t *testing.T) {
	v := newTestVerifier(t)
	store := v.store

	payload := json.RawMessage(`{"note":"tamper-test"}`)
	// Store with wrong hash
	mem, err := store.Remember(context.Background(), payload, "wrong-hash", "current")
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}

	result, err := v.VerifyMemory(context.Background(), string(mem.ID))
	if err != nil {
		t.Fatalf("VerifyMemory: %v", err)
	}
	if result.Status != "invalid" {
		t.Errorf("expected status invalid for tampered hash, got %s", result.Status)
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
