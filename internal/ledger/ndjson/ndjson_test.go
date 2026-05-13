package ndjson

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/dyne/mnemosyne/internal/domain"
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

func newTestLedger(t *testing.T) *Ledger {
	t.Helper()
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "ledger.ndjson")
	contractsDir := "../../../zenflows"

	l, err := New(path, contractsDir, "test-key", zenroom.NewExecutor(bin))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() { _ = l.Close() })
	return l
}

func TestLedger_NewEmpty(t *testing.T) {
	l := newTestLedger(t)

	head, err := l.LatestHead(context.Background())
	if err != nil {
		t.Fatalf("LatestHead: %v", err)
	}
	if head.Seq != 0 {
		t.Errorf("expected seq 0, got %d", head.Seq)
	}
	if head.EventHash != "0x00" {
		t.Errorf("expected hash 0x00, got %s", head.EventHash)
	}
}

func TestLedger_AppendAndGet(t *testing.T) {
	l := newTestLedger(t)

	rec, err := l.Append(context.Background(), domain.EventMemoryRecorded, map[string]string{"memory_id": "mem_001"})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
	if rec.Seq != 1 {
		t.Errorf("expected seq 1, got %d", rec.Seq)
	}
	if rec.EventHash == "" {
		t.Error("expected non-empty event hash")
	}

	event, err := l.GetEvent(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if event.EventType != domain.EventMemoryRecorded {
		t.Errorf("expected MEMORY_RECORDED, got %s", event.EventType)
	}
	if event.PreviousHash != "0x00" {
		t.Errorf("expected previous_hash 0x00, got %s", event.PreviousHash)
	}
}

func TestLedger_AppendChain(t *testing.T) {
	l := newTestLedger(t)

	_, err := l.Append(context.Background(), domain.EventMemoryRecorded, map[string]string{"a": "1"})
	if err != nil {
		t.Fatalf("Append 1: %v", err)
	}

	_, err = l.Append(context.Background(), domain.EventRootSealed, map[string]string{"root": "root_001"})
	if err != nil {
		t.Fatalf("Append 2: %v", err)
	}

	_, err = l.Append(context.Background(), domain.EventCheckpointCreated, map[string]string{"checkpoint": "chk_001"})
	if err != nil {
		t.Fatalf("Append 3: %v", err)
	}

	events, err := l.ListEvents(context.Background(), domain.LedgerListOptions{Limit: 10})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 3 {
		t.Errorf("expected 3 events, got %d", len(events))
	}
}

func TestLedger_VerifyChain(t *testing.T) {
	l := newTestLedger(t)

	l.Append(context.Background(), domain.EventMemoryRecorded, map[string]string{"m": "1"})
	l.Append(context.Background(), domain.EventRootSealed, map[string]string{"root": "r1"})
	l.Append(context.Background(), domain.EventCheckpointCreated, map[string]string{"chk": "c1"})

	verification, err := l.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !verification.Valid {
		t.Errorf("expected valid chain, got invalid: %v", verification.InvalidEvents)
	}
	if verification.TotalEvents != 3 {
		t.Errorf("expected 3 events, got %d", verification.TotalEvents)
	}
}

func TestLedger_TamperDetection(t *testing.T) {
	l := newTestLedger(t)
	l.Append(context.Background(), domain.EventMemoryRecorded, map[string]string{"m": "1"})
	l.Append(context.Background(), domain.EventRootSealed, map[string]string{"root": "r1"})

	// Tamper with the file by appending junk
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatalf("open for tamper: %v", err)
	}
	f.WriteString(`{"seq":3,"event_type":"BAD_EVENT","previous_hash":"invalid","event_hash":"bad"}` + "\n")
	f.Close()

	verification, err := l.Verify(context.Background())
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if verification.Valid {
		t.Error("expected invalid chain after tampering")
	}
}

func TestLedger_Head(t *testing.T) {
	l := newTestLedger(t)

	rec, err := l.Append(context.Background(), domain.EventMemoryRecorded, map[string]string{"m": "1"})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	head, err := l.LatestHead(context.Background())
	if err != nil {
		t.Fatalf("LatestHead: %v", err)
	}
	if head.Seq != 1 {
		t.Errorf("expected head seq 1, got %d", head.Seq)
	}
	if head.EventHash != rec.EventHash {
		t.Errorf("head hash mismatch: %s vs %s", head.EventHash, rec.EventHash)
	}
}

func TestLedger_GetEvent_NotFound(t *testing.T) {
	l := newTestLedger(t)

	_, err := l.GetEvent(context.Background(), 999)
	if err != domain.ErrLedgerEventNotFound {
		t.Errorf("expected ErrLedgerEventNotFound, got %v", err)
	}
}

func TestLedger_ListEvents_Pagination(t *testing.T) {
	l := newTestLedger(t)

	for i := 0; i < 5; i++ {
		l.Append(context.Background(), domain.EventMemoryRecorded, map[string]int{"n": i})
	}

	events, err := l.ListEvents(context.Background(), domain.LedgerListOptions{Limit: 2})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 2 {
		t.Errorf("expected 2 events with limit, got %d", len(events))
	}

	events2, err := l.ListEvents(context.Background(), domain.LedgerListOptions{FromSeq: 3, Limit: 10})
	if err != nil {
		t.Fatalf("ListEvents from 3: %v", err)
	}
	if len(events2) != 3 {
		t.Errorf("expected 3 events from seq 3, got %d", len(events2))
	}
}

func TestLedger_ListEvents_Empty(t *testing.T) {
	l := newTestLedger(t)

	events, err := l.ListEvents(context.Background(), domain.LedgerListOptions{})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}
}

func TestLedger_EventTypes(t *testing.T) {
	l := newTestLedger(t)

	types := []domain.EventType{
		domain.EventMemoryRecorded,
		domain.EventRootSealed,
		domain.EventCheckpointCreated,
		domain.EventAnchorCreated,
	}
	for i, typ := range types {
		rec, err := l.Append(context.Background(), typ, map[string]int{"n": i})
		if err != nil {
			t.Fatalf("Append %s: %v", typ, err)
		}
		if rec.Seq != uint64(i+1) {
			t.Errorf("expected seq %d, got %d", i+1, rec.Seq)
		}
	}
}
