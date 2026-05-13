package domain

import (
	"testing"
)

func TestMemoryTypes(t *testing.T) {
	id := MemoryID("test-id")
	if string(id) != "test-id" {
		t.Errorf("MemoryID mismatch: got %q", id)
	}

	bid := BeaconID("beacon-1")
	if string(bid) != "beacon-1" {
		t.Errorf("BeaconID mismatch: got %q", bid)
	}
}

func TestRouteTypes(t *testing.T) {
	r := &Route{Leaf: "leaf1", Root: "root1", Path: []string{"a", "b"}}
	if r.Leaf != "leaf1" || r.Root != "root1" || len(r.Path) != 2 {
		t.Error("Route fields mismatch")
	}
}

func TestWitnessResult(t *testing.T) {
	wr := &WitnessResult{Valid: true, Leaf: "l", Root: "r", MemoryID: "m"}
	if !wr.Valid || wr.Leaf != "l" || wr.Root != "r" || wr.MemoryID != "m" {
		t.Error("WitnessResult fields mismatch")
	}
}

func TestBeacon(t *testing.T) {
	b := &Beacon{ID: BeaconID("b1"), Root: "r", SignedRoot: "sr", ProofCount: 4}
	if b.ProofCount != 4 || b.SignedRoot != "sr" {
		t.Error("Beacon fields mismatch")
	}
}

func TestErrors(t *testing.T) {
	errors := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrMemoryNotFound", ErrMemoryNotFound, "memory not found"},
		{"ErrBeaconNotFound", ErrBeaconNotFound, "beacon not found"},
		{"ErrProofNotAvailable", ErrProofNotAvailable, "proof not available"},
		{"ErrWitnessFailed", ErrWitnessFailed, "witness verification failed"},
		{"ErrInvalidPayload", ErrInvalidPayload, "invalid payload"},
		{"ErrTreeEmpty", ErrTreeEmpty, "merkle tree is empty"},
		{"ErrAppendOnly", ErrAppendOnly, "append-only violation: cannot modify existing memory"},
	}
	for _, tc := range errors {
		if tc.err.Error() != tc.msg {
			t.Errorf("%s: expected %q, got %q", tc.name, tc.msg, tc.err.Error())
		}
	}
}

func TestMemoryStruct(t *testing.T) {
	m := &Memory{ID: MemoryID("m1"), LeafHash: "deadbeef", BeaconID: "b1"}
	if m.LeafHash != "deadbeef" || m.BeaconID != "b1" {
		t.Error("Memory fields mismatch")
	}
}

func TestLedgerEventTypes(t *testing.T) {
	events := []EventType{
		EventMemoryRecorded,
		EventRootSealed,
		EventCheckpointCreated,
		EventAnchorCreated,
		EventAnchorConfirmed,
		EventVerifyRequested,
		EventVerifyOK,
		EventVerifyFailed,
	}
	names := make(map[EventType]bool)
	for _, e := range events {
		if string(e) == "" {
			t.Error("empty event type")
		}
		if names[e] {
			t.Errorf("duplicate event type: %s", e)
		}
		names[e] = true
	}
}

func TestLedgerEvent(t *testing.T) {
	e := LedgerEvent{
		Seq:          1,
		EventType:    EventMemoryRecorded,
		PreviousHash: "0x00",
		EventHash:    "0xabc",
	}
	if e.Seq != 1 || e.EventHash != "0xabc" {
		t.Error("LedgerEvent fields mismatch")
	}
}

func TestAnchorReceipt(t *testing.T) {
	a := AnchorReceipt{
		AnchorID:     "anc_001",
		Backend:      "local_signature",
		AnchoredType: "checkpoint",
		Status:       "confirmed",
	}
	if a.Backend != "local_signature" || a.Status != "confirmed" {
		t.Error("AnchorReceipt fields mismatch")
	}
}

func TestReceipt(t *testing.T) {
	r := Receipt{
		Version: "mnemosyne.receipt.v1",
		Memory:  ReceiptMemory{ID: "mem_001", LeafHash: "0xabc"},
	}
	if r.Version != "mnemosyne.receipt.v1" || r.Memory.ID != "mem_001" {
		t.Error("Receipt fields mismatch")
	}
}

func TestCheckpointRecord(t *testing.T) {
	c := CheckpointRecord{
		CheckpointID: "chk_001",
		FromSeq:      1,
		ToSeq:        29,
		EventCount:   29,
	}
	if c.EventCount != 29 || c.FromSeq != 1 {
		t.Error("CheckpointRecord fields mismatch")
	}
}

func TestRootTypes(t *testing.T) {
	r := Root{RootID: "root_001", RootHash: "0xabc", LeafCount: 128}
	if r.LeafCount != 128 || r.RootHash != "0xabc" {
		t.Error("Root fields mismatch")
	}
}

func TestSignatureTypes(t *testing.T) {
	s := Signature{Type: "zenroom", Value: "sig123", PublicKey: "pub456"}
	if s.Type != "zenroom" || s.Value != "sig123" || s.PublicKey != "pub456" {
		t.Error("Signature fields mismatch")
	}
}

func TestMerkleProofTypes(t *testing.T) {
	p := MerkleProof{Leaf: "l", Root: "r", Position: 1, LeafCount: 4}
	if p.Position != 1 || p.LeafCount != 4 {
		t.Error("MerkleProof fields mismatch")
	}
}

func TestNewErrors(t *testing.T) {
	cases := []struct {
		name string
		err  error
		msg  string
	}{
		{"ErrRootNotFound", ErrRootNotFound, "root not found"},
		{"ErrCheckpointNotFound", ErrCheckpointNotFound, "checkpoint not found"},
		{"ErrAnchorNotFound", ErrAnchorNotFound, "anchor not found"},
		{"ErrLedgerEventNotFound", ErrLedgerEventNotFound, "ledger event not found"},
		{"ErrLedgerVerification", ErrLedgerVerification, "ledger verification failed"},
		{"ErrAnchorVerification", ErrAnchorVerification, "anchor verification failed"},
		{"ErrVerificationFailed", ErrVerificationFailed, "full verification failed"},
		{"ErrNoPendingMemories", ErrNoPendingMemories, "no pending memories to seal"},
		{"ErrBackendNotAvailable", ErrBackendNotAvailable, "backend not available"},
	}
	for _, tc := range cases {
		if tc.err.Error() != tc.msg {
			t.Errorf("%s: expected %q, got %q", tc.name, tc.msg, tc.err.Error())
		}
	}
}
