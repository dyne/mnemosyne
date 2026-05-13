package local

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dyne/mnemosyne/internal/domain"
	"github.com/dyne/mnemosyne/internal/zenroom"
)

// Anchor is a local-signature anchor backend.
// It signs checkpoint hashes via Zenroom and stores receipts locally.
type Anchor struct {
	contractsDir string
	executor     *zenroom.Executor
	keyRef       string
	keypair      map[string]any
}

// New creates a local-signature anchor backend.
func New(contractsDir, keyRef string, executor *zenroom.Executor) (*Anchor, error) {
	a := &Anchor{
		contractsDir: contractsDir,
		executor:     executor,
		keyRef:       keyRef,
	}

	kp, err := a.loadOrGenerateKeypair()
	if err != nil {
		return nil, fmt.Errorf("anchor keypair: %w", err)
	}
	a.keypair = kp

	return a, nil
}

// Name returns the backend identifier.
func (a *Anchor) Name() string { return "local_signature" }

// Anchor signs a hash and returns a receipt.
func (a *Anchor) Anchor(ctx context.Context, hash string, anchoredType, anchoredID string) (domain.AnchorReceipt, error) {
	now := time.Now().UTC()

	script, err := os.ReadFile(a.contractsDir + "/sign.zen")
	if err != nil {
		return domain.AnchorReceipt{}, fmt.Errorf("load sign contract: %w", err)
	}

	keys, _ := json.Marshal(a.keypair)
	data, _ := json.Marshal(map[string]any{"payload": hash})

	result, err := a.executor.Run(script, keys, data)
	if err != nil {
		return domain.AnchorReceipt{}, fmt.Errorf("zenroom sign: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return domain.AnchorReceipt{}, err
	}

	sigValue, ok := m["hash"].(string)
	if !ok {
		return domain.AnchorReceipt{}, fmt.Errorf("hash not found in sign output")
	}

	publicKey := ""
	if kp, ok := a.keypair["keypair"].(map[string]any); ok {
		if pk, ok := kp["public_key"].(string); ok {
			publicKey = pk
		}
	}

	anchorID := fmt.Sprintf("anc_%s", now.Format("20060102T150405.000000Z"))

	return domain.AnchorReceipt{
		AnchorID:     anchorID,
		Backend:      a.Name(),
		AnchoredType: anchoredType,
		AnchoredID:   anchoredID,
		AnchoredHash: hash,
		Status:       "confirmed",
		CreatedAt:    now,
		Signature: &domain.Signature{
			Type:      "zenroom-hmac",
			Value:     sigValue,
			PublicKey: publicKey,
			KeyRef:    a.keyRef,
		},
	}, nil
}

// VerifyAnchor checks an anchor receipt by recomputing the signature.
func (a *Anchor) VerifyAnchor(ctx context.Context, receipt domain.AnchorReceipt) (domain.AnchorVerification, error) {
	if receipt.Signature == nil || receipt.Signature.Value == "" {
		return domain.AnchorVerification{Valid: false, Details: "no signature in receipt"}, nil
	}

	script, err := os.ReadFile(a.contractsDir + "/verify_signature.zen")
	if err != nil {
		return domain.AnchorVerification{}, fmt.Errorf("load verify contract: %w", err)
	}

	keys, _ := json.Marshal(map[string]any{
		"keypair": map[string]any{"public_key": receipt.Signature.PublicKey},
	})
	data, _ := json.Marshal(map[string]any{
		"payload":   receipt.AnchoredHash,
		"signature": receipt.Signature.Value,
	})

	result, err := a.executor.Run(script, keys, data)
	if err != nil {
		return domain.AnchorVerification{Valid: false, Details: err.Error()}, nil
	}

	m, err := result.OutputMap()
	if err != nil {
		return domain.AnchorVerification{Valid: false, Details: err.Error()}, nil
	}

	computedHash, ok := m["hash"].(string)
	if !ok {
		return domain.AnchorVerification{Valid: false, Details: "hash not found in verify output"}, nil
	}

	valid := computedHash == receipt.Signature.Value
	details := "anchor verified via Zenroom HMAC"
	if !valid {
		details = "signature mismatch"
	}

	return domain.AnchorVerification{Valid: valid, Details: details}, nil
}

func (a *Anchor) loadOrGenerateKeypair() (map[string]any, error) {
	// Reuse the same keygen approach as the ledger
	script, err := os.ReadFile(a.contractsDir + "/keygen.zen")
	if err != nil {
		return nil, fmt.Errorf("load keygen contract: %w", err)
	}

	keyPath := filepath.Join(a.contractsDir, "..", "anchor-keypair.json")

	data, err := os.ReadFile(keyPath)
	if err == nil {
		var wrapped map[string]any
		if err := json.Unmarshal(data, &wrapped); err != nil {
			return nil, fmt.Errorf("unmarshal keypair: %w", err)
		}
		return wrapped, nil
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read keypair: %w", err)
	}

	result, err := a.executor.Run(script, nil, nil)
	if err != nil {
		return nil, fmt.Errorf("zenroom keygen: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return nil, err
	}

	keyHash, ok := m["hash"].(string)
	if !ok {
		return nil, fmt.Errorf("hash not found in keygen output")
	}

	kp := map[string]any{
		"private_key": keyHash,
		"public_key":  keyHash,
	}

	wrapped := map[string]any{"keypair": kp}

	raw, _ := json.MarshalIndent(wrapped, "", "  ")
	if err := os.WriteFile(keyPath, raw, 0600); err != nil {
		return nil, fmt.Errorf("write keypair: %w", err)
	}

	return wrapped, nil
}
