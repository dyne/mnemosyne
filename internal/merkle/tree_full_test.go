package merkle

import (
	"context"
	"os/exec"
	"testing"

	"github.com/dyne/mnemosyne/internal/domain"
	"github.com/dyne/mnemosyne/internal/zenroom"
)

func zb() string {
	for _, p := range []string{"zenroom", "/usr/bin/zenroom", "/usr/local/bin/zenroom"} {
		if _, err := exec.LookPath(p); err == nil {
			return p
		}
	}
	return ""
}

func TestHashPayload(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")

	hash, err := tree.HashPayload(context.Background(), "hello world")
	if err != nil {
		t.Fatalf("HashPayload: %v", err)
	}
	if hash == "" {
		t.Error("expected non-empty hash")
	}

	// Same input should produce same hash (deterministic)
	hash2, _ := tree.HashPayload(context.Background(), "hello world")
	if hash != hash2 {
		t.Error("hash should be deterministic")
	}
}

func TestCreateRoot_SingleLeaf(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")

	root, err := tree.CreateRoot(context.Background(), []string{"solo"})
	if err != nil {
		t.Fatalf("CreateRoot: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}

func TestWitness_InvalidPosition(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")

	_, err := tree.GenerateRoute(context.Background(), []string{"a", "b"}, 0)
	if err == nil {
		t.Error("expected error for position 0")
	}

	_, err = tree.GenerateRoute(context.Background(), []string{"a", "b"}, 3)
	if err == nil {
		t.Error("expected error for out-of-range position")
	}
}

func TestCreateRoot_ContractNotFound(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "/nonexistent")

	_, err := tree.HashPayload(context.Background(), "test")
	if err == nil {
		t.Error("expected error for missing contract")
	}
}

func TestWitness_ContractNotFound(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "/nonexistent")

	route := &domain.Route{Leaf: "x", Root: "y", Path: []string{"a"}}
	_, err := tree.Witness(context.Background(), route, "x", 1, 2)
	if err == nil {
		t.Error("expected error for missing contract")
	}
}

func TestNewTree(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, ".")
	if tree == nil {
		t.Error("expected non-nil tree")
	}
}

func TestCreateRoot_Deterministic(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")

	leaves := []string{"a", "b", "c"}
	r1, _ := tree.CreateRoot(context.Background(), leaves)
	r2, _ := tree.CreateRoot(context.Background(), leaves)
	if r1 != r2 {
		t.Error("Merkle root should be deterministic")
	}
}

func TestWitness_TamperedProof(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")

	leaves := []string{"alpha", "beta", "gamma", "delta"}
	route, err := tree.GenerateRoute(context.Background(), leaves, 2)
	if err != nil {
		t.Fatalf("GenerateRoute: %v", err)
	}

	// Tamper with a proof path element
	tampered := &domain.Route{
		Leaf: route.Leaf,
		Root: route.Root,
		Path: make([]string, len(route.Path)),
	}
	copy(tampered.Path, route.Path)
	tampered.Path[0] = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

	result, err := tree.Witness(context.Background(), tampered, route.Leaf, 2, len(leaves))
	if err != nil {
		t.Fatalf("Witness: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid for tampered proof")
	}
}

func TestCreateTreeWithProof(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")

	root, err := tree.CreateTreeWithProof(context.Background(), []string{"a", "b"}, "beacon-1")
	if err != nil {
		t.Fatalf("CreateTreeWithProof: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}

func TestCreateRoot_ErrorPath(t *testing.T) {
	bin := zb()
	if bin == "" {
		t.Skip("zenroom not found")
	}
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")
	// Single leaf should work fine
	root, err := tree.CreateRoot(context.Background(), []string{"x"})
	if err != nil {
		t.Fatalf("CreateRoot: %v", err)
	}
	if root == "" {
		t.Error("expected non-empty root")
	}
}
