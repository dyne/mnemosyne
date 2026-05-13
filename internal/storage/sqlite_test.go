package storage

import (
	"context"
	"os"
	"testing"

	"github.com/dyne/mnemosyne/internal/domain"
)

func TestSQLiteStore_RememberAndRecall(t *testing.T) {
	path := "/tmp/mnemosyne-test-" + t.Name() + ".db"
	defer func() { _ = os.Remove(path) }()

	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = s.Close() }()

	m, err := s.Remember(context.Background(), map[string]any{"hello": "world"}, "deadbeef", "beacon-1")
	if err != nil {
		t.Fatalf("Remember: %v", err)
	}
	if m.ID == "" {
		t.Error("expected non-empty ID")
	}
	if m.LeafHash != "deadbeef" {
		t.Errorf("expected leaf hash deadbeef, got %s", m.LeafHash)
	}

	recalled, err := s.Recall(context.Background(), m.ID)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}
	if recalled.LeafHash != "deadbeef" {
		t.Errorf("expected leaf hash deadbeef, got %s", recalled.LeafHash)
	}
}

func TestSQLiteStore_AnchorAndLatestBeacon(t *testing.T) {
	path := "/tmp/mnemosyne-test-" + t.Name() + ".db"
	defer func() { _ = os.Remove(path) }()

	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = s.Close() }()

	beacon := &domain.Beacon{
		ID:         "beacon-1",
		Root:       "root-hash",
		ProofCount: 4,
	}
	err = s.AnchorBeacon(context.Background(), beacon)
	if err != nil {
		t.Fatalf("AnchorBeacon: %v", err)
	}

	latest, err := s.LatestBeacon(context.Background())
	if err != nil {
		t.Fatalf("LatestBeacon: %v", err)
	}
	if latest.Root != "root-hash" {
		t.Errorf("expected root-hash, got %s", latest.Root)
	}
}

func TestSQLiteStore_RecallNotFound(t *testing.T) {
	path := "/tmp/mnemosyne-test-" + t.Name() + ".db"
	defer func() { _ = os.Remove(path) }()

	s, err := NewSQLiteStore(path)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	defer func() { _ = s.Close() }()

	_, err = s.Recall(context.Background(), domain.MemoryID("nonexistent"))
	if err != domain.ErrMemoryNotFound {
		t.Errorf("expected ErrMemoryNotFound, got %v", err)
	}
}
