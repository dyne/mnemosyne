package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

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

func newTestServer(t *testing.T) *Server {
	t.Helper()
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}

	dbPath := "/tmp/mnemosyne-test-" + t.Name() + ".db"
	t.Cleanup(func() { os.Remove(dbPath) })

	store, err := storage.NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	t.Cleanup(func() { store.Close() })

	executor := zenroom.NewExecutor(bin)
	tree := merkle.NewTree(executor, store, "../../zenflows")

	return NewServer(store, tree, "", "../../zenflows", "dev")
}

func TestRememberAndRecall(t *testing.T) {
	srv := newTestServer(t)

	// Create memory
	body := bytes.NewBufferString(`{"payload":{"greeting":"hello mnemosyne"}}`)
	req := httptest.NewRequest("POST", "/memories", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var m struct {
		MemoryID string `json:"memory_id"`
		LeafHash string `json:"leaf_hash"`
	}
	json.Unmarshal(w.Body.Bytes(), &m)
	if m.MemoryID == "" {
		t.Error("expected non-empty memory_id")
	}
	if m.LeafHash == "" {
		t.Error("expected non-empty leaf_hash")
	}
	t.Logf("memory_id: %s, leaf_hash: %s", m.MemoryID, m.LeafHash)

	// Recall memory
	req2 := httptest.NewRequest("GET", "/memories/"+m.MemoryID, nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w2.Code)
	}
}

func TestProofGenerationAndVerification(t *testing.T) {
	srv := newTestServer(t)

	// Create 4 memories
	payloads := []string{`"alpha"`, `"beta"`, `"gamma"`, `"delta"`}
	var memID string
	for i, p := range payloads {
		body := bytes.NewBufferString(`{"payload":` + p + `}`)
		req := httptest.NewRequest("POST", "/memories", body)
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusCreated {
			t.Fatalf("memory %d: expected 201, got %d", i, w.Code)
		}
		var m struct {
			MemoryID string `json:"memory_id"`
		}
		json.Unmarshal(w.Body.Bytes(), &m)
		if i == 3 {
			memID = m.MemoryID
		}
	}

	// Generate proof
	req := httptest.NewRequest("GET", "/proofs/"+memID, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("proof generation: expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var proof map[string]any
	json.Unmarshal(w.Body.Bytes(), &proof)
	t.Logf("proof: %v", proof)

	if len(proof["path"].([]any)) == 0 {
		t.Error("expected non-empty proof path")
	}

	// Verify proof
	verifyBody, _ := json.Marshal(proof)
	req2 := httptest.NewRequest("POST", "/verify", bytes.NewBuffer(verifyBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("verification: expected 200, got %d", w2.Code)
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	json.Unmarshal(w2.Body.Bytes(), &result)
	if !result.Valid {
		t.Error("expected valid proof")
	}
}

func TestRecallNotFound(t *testing.T) {
	srv := newTestServer(t)

	req := httptest.NewRequest("GET", "/memories/nonexistent", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestVerifyInvalidProof(t *testing.T) {
	srv := newTestServer(t)

	body := bytes.NewBufferString(`{"leaf":"x","root":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","path":["AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=","AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="],"position":1,"leaf_count":2}`)
	req := httptest.NewRequest("POST", "/verify", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result struct {
		Valid bool `json:"valid"`
	}
	json.Unmarshal(w.Body.Bytes(), &result)
	if result.Valid {
		t.Error("expected invalid proof")
	}
}

func TestCreateMemoryNoPayload(t *testing.T) {
	srv := newTestServer(t)

	body := bytes.NewBufferString(`{}`)
	req := httptest.NewRequest("POST", "/memories", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}
