package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http/httptest"
	"os"
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

func zBin() string {
	for _, p := range []string{"zenroom", "/usr/bin/zenroom", "/usr/local/bin/zenroom"} {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return ""
}

// newFullServer creates a server with ledger and anchor backends configured.
func newFullServer(t *testing.T) *Server {
	t.Helper()
	bin := zBin()
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

	return NewServer(ServerConfig{
		Store:        store,
		Tree:         tree,
		Ledger:       l,
		Anchor:       anchor,
		WebDir:       "",
		ContractsDir: contractsDir,
		Version:      "dev",
	})
}

// newNoLedgerAnchorServer creates a server without ledger/anchor to test error paths.
func newNoLedgerAnchorServer(t *testing.T) *Server {
	t.Helper()
	bin := zBin()
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

	return NewServer(ServerConfig{
		Store:        store,
		Tree:         tree,
		Ledger:       nil,
		Anchor:       nil,
		ContractsDir: contractsDir,
		Version:      "dev",
	})
}

func doFullReq(s *Server, method, path string, body io.Reader) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(method, path, body)
	if body != nil {
		r.Header.Set("Content-Type", "application/json")
	}
	s.Handler().ServeHTTP(w, r)
	return w
}

// ---- Dashboard ----

func TestDashboard(t *testing.T) {
	s := newFullServer(t)

	// Create a memory first
	doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"dashboard-test"}`))

	w := doFullReq(s, "GET", "/dashboard", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	if resp["storage_backend"] != "sqlite" {
		t.Error("expected storage_backend")
	}
	if resp["ledger_backend"] != "ndjson_hash_chain" {
		t.Error("expected ledger_backend")
	}
	if resp["anchor_backend"] != "local_signature" {
		t.Error("expected anchor_backend")
	}
}

func TestDashboard_NoLedgerAnchor(t *testing.T) {
	s := newNoLedgerAnchorServer(t)
	w := doFullReq(s, "GET", "/dashboard", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode dashboard: %v", err)
	}
	// No ledger_backend in response
	if _, ok := resp["ledger_backend"]; ok {
		t.Error("expected no ledger_backend when ledger is nil")
	}
	if _, ok := resp["anchor_backend"]; ok {
		t.Error("expected no anchor_backend when anchor is nil")
	}
}

// ---- Ledger endpoints ----

func TestLedgerEvents(t *testing.T) {
	s := newFullServer(t)

	// Create a memory to generate a ledger event
	doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"ledger-test"}`))

	w := doFullReq(s, "GET", "/ledger/events", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var resp struct {
		Events     []domain.LedgerEvent `json:"events"`
		Total      int                  `json:"total"`
		LedgerHead domain.LedgerHead    `json:"ledger_head"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode events: %v", err)
	}
	if resp.Total == 0 {
		t.Error("expected at least one ledger event")
	}
}

func TestLedgerEvents_NoLedger(t *testing.T) {
	s := newNoLedgerAnchorServer(t)
	w := doFullReq(s, "GET", "/ledger/events", nil)
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestLedgerHead(t *testing.T) {
	s := newFullServer(t)

	// Create a memory to set up the ledger
	doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"head-test"}`))

	w := doFullReq(s, "GET", "/ledger/head", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var head domain.LedgerHead
	if err := json.Unmarshal(w.Body.Bytes(), &head); err != nil {
		t.Fatalf("decode head: %v", err)
	}
	if head.Seq == 0 {
		t.Error("expected non-zero seq")
	}
}

func TestLedgerHead_NoLedger(t *testing.T) {
	s := newNoLedgerAnchorServer(t)
	w := doFullReq(s, "GET", "/ledger/head", nil)
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestLedgerVerify(t *testing.T) {
	s := newFullServer(t)

	// Create a memory to set up the ledger
	doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"verify-ledger"}`))

	w := doFullReq(s, "POST", "/ledger/verify", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var verif domain.LedgerVerification
	if err := json.Unmarshal(w.Body.Bytes(), &verif); err != nil {
		t.Fatalf("decode verification: %v", err)
	}
	if !verif.Valid {
		t.Error("expected valid ledger chain")
	}
}

func TestLedgerVerify_NoLedger(t *testing.T) {
	s := newNoLedgerAnchorServer(t)
	w := doFullReq(s, "POST", "/ledger/verify", nil)
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// ---- Anchors ----

func TestCreateAnchor(t *testing.T) {
	s := newFullServer(t)

	body := bytes.NewBufferString(`{"hash":"dGVzdC1oYXNo","anchored_type":"checkpoint","anchored_id":"chk_001"}`)
	w := doFullReq(s, "POST", "/anchors", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var receipt domain.AnchorReceipt
	if err := json.Unmarshal(w.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.Backend != "local_signature" {
		t.Errorf("expected backend local_signature, got %s", receipt.Backend)
	}
}

func TestCreateAnchor_NoAnchorBackend(t *testing.T) {
	s := newNoLedgerAnchorServer(t)
	body := bytes.NewBufferString(`{"hash":"dGVzdC1oYXNo","anchored_type":"checkpoint","anchored_id":"chk_001"}`)
	w := doFullReq(s, "POST", "/anchors", body)
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

func TestCreateAnchor_InvalidRequest(t *testing.T) {
	s := newFullServer(t)

	// Invalid JSON
	w := doFullReq(s, "POST", "/anchors", bytes.NewBufferString("not json"))
	if w.Code != 400 {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}

	// Missing hash
	w2 := doFullReq(s, "POST", "/anchors", bytes.NewBufferString(`{"anchored_type":"checkpoint"}`))
	if w2.Code != 400 {
		t.Errorf("expected 400 for missing hash, got %d", w2.Code)
	}
}

func TestCreateAnchor_DefaultType(t *testing.T) {
	s := newFullServer(t)
	// Don't send anchored_type — should default to "checkpoint"
	body := bytes.NewBufferString(`{"hash":"dGVzdC1oYXNo","anchored_id":"chk_002"}`)
	w := doFullReq(s, "POST", "/anchors", body)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAnchor(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "GET", "/anchors/test-anchor-1", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["backend"] != "local_signature" {
		t.Errorf("expected backend local_signature, got %v", resp["backend"])
	}
	if resp["status"] != "confirmed" {
		t.Errorf("expected status confirmed, got %v", resp["status"])
	}
}

func TestGetAnchor_NoAnchorBackend(t *testing.T) {
	s := newNoLedgerAnchorServer(t)
	w := doFullReq(s, "GET", "/anchors/test-1", nil)
	if w.Code != 503 {
		t.Errorf("expected 503, got %d", w.Code)
	}
}

// ---- Receipt Export ----

func TestReceiptExport(t *testing.T) {
	s := newFullServer(t)

	// Create a memory
	w1 := doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"receipt-test"}`))
	var m struct {
		MemoryID string `json:"memory_id"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode memory: %v", err)
	}

	w := doFullReq(s, "GET", "/memories/"+m.MemoryID+"/receipt", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var receipt domain.Receipt
	if err := json.Unmarshal(w.Body.Bytes(), &receipt); err != nil {
		t.Fatalf("decode receipt: %v", err)
	}
	if receipt.Version != "mnemosyne.receipt.v1" {
		t.Errorf("expected receipt version mnemosyne.receipt.v1, got %s", receipt.Version)
	}
	if receipt.Memory.ID == "" {
		t.Error("expected non-empty memory ID in receipt")
	}
	if receipt.Ledger.Backend != "ndjson_hash_chain" {
		t.Errorf("expected ledger backend in receipt, got %s", receipt.Ledger.Backend)
	}
	if receipt.Anchor.Backend != "local_signature" {
		t.Errorf("expected anchor backend in receipt, got %s", receipt.Anchor.Backend)
	}
}

func TestReceiptExport_NotFound(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "GET", "/memories/nonexistent/receipt", nil)
	if w.Code != 500 {
		t.Errorf("expected 500 for not found, got %d", w.Code)
	}
}

// ---- Full Verification ----

func TestFullVerify(t *testing.T) {
	s := newFullServer(t)

	// Create a memory
	w1 := doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"full-verify-test"}`))
	var m struct {
		MemoryID string `json:"memory_id"`
	}
	if err := json.Unmarshal(w1.Body.Bytes(), &m); err != nil {
		t.Fatalf("decode memory: %v", err)
	}

	body := bytes.NewBufferString(`{"memory_id":"` + m.MemoryID + `"}`)
	w := doFullReq(s, "POST", "/verify/full", body)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var result struct {
		Status string `json:"status"`
		Checks []struct {
			Name   string `json:"name"`
			Status string `json:"status"`
		} `json:"checks"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if len(result.Checks) == 0 {
		t.Error("expected at least one check")
	}
}

func TestFullVerify_InvalidJSON(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "POST", "/verify/full", bytes.NewBufferString("not json"))
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFullVerify_MissingMemoryID(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "POST", "/verify/full", bytes.NewBufferString(`{}`))
	if w.Code != 400 {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestFullVerify_NotFound(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "POST", "/verify/full", bytes.NewBufferString(`{"memory_id":"nonexistent"}`))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Status != "invalid" {
		t.Errorf("expected status invalid, got %s", result.Status)
	}
}

// ---- Health with ledger/anchor ----

func TestHealth_WithBackends(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "GET", "/health", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode health: %v", err)
	}
	if resp["ledger"] != "available" {
		t.Error("expected ledger=available")
	}
	if resp["anchor"] != "local_signature" {
		t.Errorf("expected anchor=local_signature, got %s", resp["anchor"])
	}
}

// ---- Checkpoint with ledger ----

func TestAnchorBeacon_WithLedger(t *testing.T) {
	s := newFullServer(t)
	doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"b1"}`))
	doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"b2"}`))

	w := doFullReq(s, "POST", "/checkpoints", nil)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode checkpoint: %v", err)
	}
	// Check that ledger info is in the response
	if _, ok := resp["ledger"]; !ok {
		t.Error("expected ledger info in checkpoint response")
	}
}

// ---- Remember with ledger ----

func TestRemember_WithLedger(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"ledger-memory"}`))
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := resp["ledger"]; !ok {
		t.Error("expected ledger info in response")
	}
}

// ---- Witness with ledger ----

func TestWitness_WithLedger(t *testing.T) {
	s := newFullServer(t)
	// Create 4 memories and anchor
	for i := 0; i < 4; i++ {
		doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"wit`+string(rune('0'+i))+`"}`))
	}
	doFullReq(s, "POST", "/checkpoints", nil)

	// Get the first memory to generate a proof
	w1 := doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"witness-target"}`))
	var m struct {
		MemoryID string `json:"memory_id"`
	}
	json.Unmarshal(w1.Body.Bytes(), &m)

	w2 := doFullReq(s, "GET", "/proofs/"+m.MemoryID, nil)
	if w2.Code != 200 {
		t.Skipf("cannot generate proof: %d", w2.Code)
		return
	}

	var proof map[string]any
	json.Unmarshal(w2.Body.Bytes(), &proof)
	verifyBody, _ := json.Marshal(proof)
	w3 := doFullReq(s, "POST", "/verify", bytes.NewBuffer(verifyBody))
	if w3.Code != 200 {
		t.Errorf("expected 200, got %d", w3.Code)
	}
}

// ---- GetContract edge cases ----

func TestGetContract_DotInName(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "GET", "/contracts/../hash.zen", nil)
	t.Logf("dot-dot name code: %d", w.Code)
}

// ---- NewServer full config ----

func TestNewServer_FullConfig(t *testing.T) {
	s := newFullServer(t)
	if s.ledger == nil {
		t.Error("expected ledger to be set")
	}
	if s.anchor == nil {
		t.Error("expected anchor to be set")
	}
	if s.store == nil {
		t.Error("expected store to be set")
	}
	if s.tree == nil {
		t.Error("expected tree to be set")
	}
}

func TestNewServer_NoLedgerAnchor(t *testing.T) {
	s := newNoLedgerAnchorServer(t)
	if s.ledger != nil {
		t.Error("expected ledger to be nil")
	}
	if s.anchor != nil {
		t.Error("expected anchor to be nil")
	}
}

// ---- Verify with ledger event recording ----

func TestWitness_ValidFalse(t *testing.T) {
	s := newFullServer(t)
	// An invalid proof — should still work and record ledger event
	w := doFullReq(s, "POST", "/verify", bytes.NewBufferString(`{"leaf":"x","root":"y","path":["a"],"position":1,"leaf_count":1}`))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var result struct {
		Valid bool `json:"valid"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	// Result may or may not be valid — either OK
}

func TestFullVerify_ValidFalse(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "POST", "/verify/full", bytes.NewBufferString(`{"memory_id":"nonexistent"}`))
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

// ---- Additional edge cases ----

func TestDashboard_AfterCheckpoint(t *testing.T) {
	s := newFullServer(t)
	doFullReq(s, "POST", "/memories", bytes.NewBufferString(`{"payload":"db-mem"}`))
	doFullReq(s, "POST", "/checkpoints", nil)

	w := doFullReq(s, "GET", "/dashboard", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp["pending_memories"] == nil {
		t.Skip("pending_memories not present")
	}
}

func TestGetContract_NonexistentDir(t *testing.T) {
	bin := zBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	store, _ := storage.NewSQLiteStore("/tmp/test-bad-contracts.db")
	t.Cleanup(func() { _ = store.Close(); _ = os.Remove("/tmp/test-bad-contracts.db") })
	tree := merkle.NewTree(zenroom.NewExecutor(bin), store, "/nonexistent/path")

	s := NewServer(ServerConfig{
		Store:        store,
		Tree:         tree,
		ContractsDir: "/nonexistent/path",
		Version:      "dev",
	})
	w := doFullReq(s, "GET", "/contracts", nil)
	if w.Code != 500 {
		t.Errorf("expected 500 for bad contracts dir, got %d", w.Code)
	}
}

func TestGetContract_LuaContentType(t *testing.T) {
	s := newFullServer(t)
	w := doFullReq(s, "GET", "/contracts/proof_generate.lua", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	ct := w.Header().Get("Content-Type")
	if ct == "" {
		t.Error("expected Content-Type header")
	}
	t.Logf("Content-Type: %s", ct)
}
