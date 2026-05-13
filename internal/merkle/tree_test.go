package merkle

import (
	"context"
	"os/exec"
	"testing"

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

func TestTree_CreateRoot(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}

	// Use in-memory store for tests
	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")

	root, err := tree.CreateRoot(context.Background(), []string{"data1", "data2", "data3", "data4"})
	if err != nil {
		t.Fatalf("CreateRoot: %v", err)
	}
	// Known test vector from Zenroom suite
	expected := "1Fu3eBfOGVlDihcanmfcZj45yuy4Z3/SrSq1iupzD/Q="
	if root != expected {
		t.Errorf("expected root %s, got %s", expected, root)
	}
}

func TestTree_GenerateAndWitnessRoute(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}

	tree := NewTree(zenroom.NewExecutor(bin), nil, "../../zenflows")
	leaves := []string{"data1", "data2", "data3", "data4"}

	// Generate proof for leaf at position 1
	route, err := tree.GenerateRoute(context.Background(), leaves, 1)
	if err != nil {
		t.Fatalf("GenerateRoute: %v", err)
	}
	if len(route.Path) == 0 {
		t.Error("expected non-empty proof path")
	}
	t.Logf("root: %s", route.Root)
	t.Logf("proof path length: %d", len(route.Path))

	// Verify the proof
	result, err := tree.Witness(context.Background(), route, "data1", 1, len(leaves))
	if err != nil {
		t.Fatalf("Witness: %v", err)
	}
	if !result.Valid {
		t.Error("proof verification failed")
	}
}

func TestTree_CreateRootEmpty(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found in PATH")
	}

	tree := NewTree(zenroom.NewExecutor(bin), nil, ".")
	_, err := tree.CreateRoot(context.Background(), []string{})
	if err == nil {
		t.Error("expected error for empty leaves")
	}
}
