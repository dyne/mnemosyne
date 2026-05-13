package local

import (
	"context"
	"fmt"
	"os/exec"
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

func TestAnchor_New(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}

	a, err := New("../../../zenflows", "test-anchor", zenroom.NewExecutor(bin))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if a.Name() != "local_signature" {
		t.Errorf("expected local_signature, got %s", a.Name())
	}
}

func TestAnchor_Anchor(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}

	a, err := New("../../../zenflows", "test-anchor", zenroom.NewExecutor(bin))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	receipt, err := a.Anchor(context.Background(), "0xtest", "checkpoint", "chk_001")
	if err != nil {
		t.Fatalf("Anchor: %v", err)
	}
	if receipt.Backend != "local_signature" {
		t.Errorf("expected local_signature, got %s", receipt.Backend)
	}
	if receipt.AnchoredType != "checkpoint" {
		t.Errorf("expected checkpoint, got %s", receipt.AnchoredType)
	}
	if receipt.AnchoredHash != "0xtest" {
		t.Errorf("expected 0xtest, got %s", receipt.AnchoredHash)
	}
	if receipt.Status != "confirmed" {
		t.Errorf("expected confirmed, got %s", receipt.Status)
	}
	if receipt.Signature == nil || receipt.Signature.Value == "" {
		t.Error("expected non-empty signature")
	}
	if receipt.AnchorID == "" {
		t.Error("expected non-empty anchor ID")
	}
}

func TestAnchor_VerifyAnchor(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}

	a, err := New("../../../zenflows", "test-anchor", zenroom.NewExecutor(bin))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	receipt, err := a.Anchor(context.Background(), "0xtest", "root", "root_001")
	if err != nil {
		t.Fatalf("Anchor: %v", err)
	}

	verification, err := a.VerifyAnchor(context.Background(), receipt)
	if err != nil {
		t.Fatalf("VerifyAnchor: %v", err)
	}
	if !verification.Valid {
		t.Error("expected valid anchor verification")
	}
}

func TestAnchor_VerifyAnchor_Tampered(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}

	a, err := New("../../../zenflows", "test-anchor", zenroom.NewExecutor(bin))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	receipt, err := a.Anchor(context.Background(), "0xtest", "root", "root_001")
	if err != nil {
		t.Fatalf("Anchor: %v", err)
	}

	// Tamper with the hash
	tampered := receipt
	tampered.AnchoredHash = "0xtampered"

	verification, err := a.VerifyAnchor(context.Background(), tampered)
	if err != nil {
		t.Fatalf("VerifyAnchor: %v", err)
	}
	if verification.Valid {
		t.Error("expected invalid anchor verification after tampering")
	}
}

func TestAnchor_VerifyAnchor_NoSignature(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}

	a, err := New("../../../zenflows", "test-anchor", zenroom.NewExecutor(bin))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	receipt := domain.AnchorReceipt{
		AnchorID:     "anc_test",
		Backend:      "local_signature",
		AnchoredHash: "0xabc",
		Status:       "pending",
	}

	verification, err := a.VerifyAnchor(context.Background(), receipt)
	if err != nil {
		t.Fatalf("VerifyAnchor: %v", err)
	}
	if verification.Valid {
		t.Error("expected invalid when no signature")
	}
}

func TestAnchor_Name(t *testing.T) {
	a := &Anchor{}
	if a.Name() != "local_signature" {
		t.Errorf("expected local_signature, got %s", a.Name())
	}
}

func TestAnchor_Multiple(t *testing.T) {
	bin := zenroomBin()
	if bin == "" {
		t.Skip("zenroom not found")
	}

	a, err := New("../../../zenflows", "test-anchor", zenroom.NewExecutor(bin))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for i := 0; i < 5; i++ {
		hash := fmt.Sprintf("0xhash_%d", i)
		receipt, err := a.Anchor(context.Background(), hash, "checkpoint", fmt.Sprintf("chk_%d", i))
		if err != nil {
			t.Fatalf("Anchor %d: %v", i, err)
		}

		verification, err := a.VerifyAnchor(context.Background(), receipt)
		if err != nil {
			t.Fatalf("VerifyAnchor %d: %v", i, err)
		}
		if !verification.Valid {
			t.Errorf("expected valid for %d", i)
		}
	}
}
