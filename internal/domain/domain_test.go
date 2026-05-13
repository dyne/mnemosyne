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
