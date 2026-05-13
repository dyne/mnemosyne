package ndjson

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/dyne/mnemosyne/internal/domain"
	"github.com/dyne/mnemosyne/internal/zenroom"
)

// Ledger is the NDJSON hash-chain ledger backend.
// Events are written one per line with each event linking to the previous via hash.
type Ledger struct {
	mu           sync.Mutex
	path         string
	contractsDir string
	executor     *zenroom.Executor
	keyRef       string
	keypair      map[string]any
	head         domain.LedgerHead
}

// New opens (or creates) an NDJSON ledger at the given path.
func New(path, contractsDir, keyRef string, executor *zenroom.Executor) (*Ledger, error) {
	l := &Ledger{
		path:         path,
		contractsDir: contractsDir,
		executor:     executor,
		keyRef:       keyRef,
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("ledger mkdir: %w", err)
	}

	// Load or generate keypair
	kp, err := l.loadOrGenerateKeypair()
	if err != nil {
		return nil, fmt.Errorf("keypair: %w", err)
	}
	l.keypair = kp

	// Verify chain on open — this sets l.head
	if _, err := l.Verify(context.Background()); err != nil {
		return nil, fmt.Errorf("ledger verify on open: %w", err)
	}

	return l, nil
}

// Append writes a signed event to the ledger.
func (l *Ledger) Append(ctx context.Context, typ domain.EventType, payload any) (domain.LedgerReceipt, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now().UTC()

	// Compute the event body (without signature)
	body := map[string]any{
		"seq":           l.head.Seq + 1,
		"event_type":    string(typ),
		"payload":       payload,
		"previous_hash": l.head.EventHash,
		"created_at":    now.Format(time.RFC3339),
	}

	// Hash the canonical body via Zenroom
	eventHash, err := l.hashBody(ctx, body)
	if err != nil {
		return domain.LedgerReceipt{}, fmt.Errorf("hash event body: %w", err)
	}

	// Sign the body via Zenroom
	sig, err := l.signBody(ctx, body)
	if err != nil {
		return domain.LedgerReceipt{}, fmt.Errorf("sign event: %w", err)
	}

	event := domain.LedgerEvent{
		Seq:          l.head.Seq + 1,
		EventType:    typ,
		Payload:      payload,
		PreviousHash: l.head.EventHash,
		EventHash:    eventHash,
		CreatedAt:    now,
		Signature:    sig,
	}

	line, err := json.Marshal(event)
	if err != nil {
		return domain.LedgerReceipt{}, fmt.Errorf("marshal event: %w", err)
	}

	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return domain.LedgerReceipt{}, fmt.Errorf("open ledger: %w", err)
	}
	defer f.Close()

	if _, err := fmt.Fprintln(f, string(line)); err != nil {
		return domain.LedgerReceipt{}, fmt.Errorf("write ledger: %w", err)
	}

	l.head = domain.LedgerHead{
		Seq:       event.Seq,
		EventHash: eventHash,
		UpdatedAt: now,
	}

	return domain.LedgerReceipt{
		Seq:       event.Seq,
		EventHash: eventHash,
		Head:      eventHash,
	}, nil
}

// GetEvent returns a single ledger event by sequence number.
func (l *Ledger) GetEvent(ctx context.Context, seq uint64) (domain.LedgerEvent, error) {
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return domain.LedgerEvent{}, domain.ErrLedgerEventNotFound
		}
		return domain.LedgerEvent{}, fmt.Errorf("open ledger: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lineNum uint64
	for scanner.Scan() {
		lineNum++
		if lineNum == seq {
			var e domain.LedgerEvent
			if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
				return domain.LedgerEvent{}, fmt.Errorf("unmarshal event %d: %w", seq, err)
			}
			return e, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return domain.LedgerEvent{}, fmt.Errorf("scan ledger: %w", err)
	}
	return domain.LedgerEvent{}, domain.ErrLedgerEventNotFound
}

// ListEvents returns events from the ledger with pagination.
func (l *Ledger) ListEvents(ctx context.Context, opts domain.LedgerListOptions) ([]domain.LedgerEvent, error) {
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open ledger: %w", err)
	}
	defer f.Close()

	limit := opts.Limit
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	var events []domain.LedgerEvent
	scanner := bufio.NewScanner(f)
	var lineNum uint64
	for scanner.Scan() {
		lineNum++
		if lineNum < opts.FromSeq {
			continue
		}
		var e domain.LedgerEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return nil, fmt.Errorf("unmarshal event %d: %w", lineNum, err)
		}
		events = append(events, e)
		if len(events) >= limit {
			break
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan ledger: %w", err)
	}
	return events, nil
}

// LatestHead returns the current ledger head.
func (l *Ledger) LatestHead(ctx context.Context) (domain.LedgerHead, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.head, nil
}

// Verify checks the entire hash chain.
func (l *Ledger) Verify(ctx context.Context) (domain.LedgerVerification, error) {
	f, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			// Fresh ledger — no events yet
			l.head = domain.LedgerHead{
				Seq:       0,
				EventHash: "0x00",
				UpdatedAt: time.Now().UTC(),
			}
			return domain.LedgerVerification{
				Valid:       true,
				TotalEvents: 0,
				FirstHash:   "0x00",
				LastHash:    "0x00",
			}, nil
		}
		return domain.LedgerVerification{}, fmt.Errorf("open ledger: %w", err)
	}
	defer f.Close()

	var events []domain.LedgerEvent
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		var e domain.LedgerEvent
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			return domain.LedgerVerification{}, fmt.Errorf("unmarshal event: %w", err)
		}
		events = append(events, e)
	}
	if err := scanner.Err(); err != nil {
		return domain.LedgerVerification{}, fmt.Errorf("scan ledger: %w", err)
	}

	result := domain.LedgerVerification{
		Valid:       true,
		TotalEvents: uint64(len(events)),
	}

	if len(events) > 0 {
		result.FirstHash = events[0].EventHash
		result.LastHash = events[len(events)-1].EventHash

		prevHash := "0x00"
		for i, e := range events {
			if e.PreviousHash != prevHash {
				result.Valid = false
				result.InvalidEvents = append(result.InvalidEvents, e.Seq)
			}

			// Recompute and verify event hash
			body := map[string]any{
				"seq":           e.Seq,
				"event_type":    string(e.EventType),
				"payload":       e.Payload,
				"previous_hash": e.PreviousHash,
				"created_at":    e.CreatedAt.Format(time.RFC3339),
			}
			computedHash, err := l.hashBody(ctx, body)
			if err != nil {
				result.Valid = false
				result.InvalidEvents = append(result.InvalidEvents, e.Seq)
			} else if computedHash != e.EventHash {
				result.Valid = false
				result.InvalidEvents = append(result.InvalidEvents, e.Seq)
			}

			// Verify signature
			sigVerify, err := l.verifySignature(ctx, e)
			if err != nil || !sigVerify.Valid {
				result.Valid = false
				result.InvalidEvents = append(result.InvalidEvents, e.Seq)
			}

			_ = i
			prevHash = e.EventHash
		}
	}

	// Update head
	if len(events) > 0 {
		last := events[len(events)-1]
		l.head = domain.LedgerHead{
			Seq:       last.Seq,
			EventHash: last.EventHash,
			UpdatedAt: last.CreatedAt,
		}
	} else {
		l.head = domain.LedgerHead{
			Seq:       0,
			EventHash: "0x00",
			UpdatedAt: time.Now().UTC(),
		}
	}

	return result, nil
}

// Close releases resources.
func (l *Ledger) Close() error { return nil }

// loadOrGenerateKeypair loads an existing keypair or generates a new one via Zenroom.
func (l *Ledger) loadOrGenerateKeypair() (map[string]any, error) {
	keyPath := filepath.Join(filepath.Dir(l.path), "ledger-keypair.json")

	data, err := os.ReadFile(keyPath)
	if err == nil {
		var kp map[string]any
		if err := json.Unmarshal(data, &kp); err != nil {
			return nil, fmt.Errorf("unmarshal keypair: %w", err)
		}
		return kp, nil
	}

	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read keypair: %w", err)
	}

	// Generate new keypair via Zenroom
	script, err := os.ReadFile(l.contractsDir + "/keygen.zen")
	if err != nil {
		return nil, fmt.Errorf("load keygen contract: %w", err)
	}

	result, err := l.executor.Run(script, nil, nil)
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

	// Build keypair from generated hash
	kp := map[string]any{
		"private_key": keyHash,
		"public_key":  keyHash,
	}

	// Wrap for Zenroom consumption
	wrapped := map[string]any{"keypair": kp}

	// Persist keypair
	raw, _ := json.MarshalIndent(wrapped, "", "  ")
	if err := os.WriteFile(keyPath, raw, 0600); err != nil {
		return nil, fmt.Errorf("write keypair: %w", err)
	}

	return wrapped, nil
}

// hashBody hashes a canonical event body via Zenroom.
func (l *Ledger) hashBody(ctx context.Context, body any) (string, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return "", fmt.Errorf("marshal body: %w", err)
	}

	script, err := os.ReadFile(l.contractsDir + "/hash.zen")
	if err != nil {
		return "", fmt.Errorf("load hash contract: %w", err)
	}

	input, _ := json.Marshal(map[string]any{"input": string(data)})
	result, err := l.executor.Run(script, nil, input)
	if err != nil {
		return "", fmt.Errorf("zenroom hash: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return "", err
	}
	h, ok := m["hash"].(string)
	if !ok {
		return "", fmt.Errorf("hash not found in zenroom output")
	}
	return h, nil
}

// signBody signs a canonical event body via Zenroom (hash-based HMAC).
func (l *Ledger) signBody(ctx context.Context, body any) (domain.Signature, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return domain.Signature{}, fmt.Errorf("marshal body: %w", err)
	}

	script, err := os.ReadFile(l.contractsDir + "/sign.zen")
	if err != nil {
		return domain.Signature{}, fmt.Errorf("load sign contract: %w", err)
	}

	keys, _ := json.Marshal(l.keypair)
	data, _ := json.Marshal(map[string]any{"payload": string(payload)})

	result, err := l.executor.Run(script, keys, data)
	if err != nil {
		return domain.Signature{}, fmt.Errorf("zenroom sign: %w", err)
	}

	m, err := result.OutputMap()
	if err != nil {
		return domain.Signature{}, err
	}

	sigValue, ok := m["hash"].(string)
	if !ok {
		return domain.Signature{}, fmt.Errorf("hash not found in sign output")
	}

	publicKey := ""
	if kp, ok := l.keypair["keypair"].(map[string]any); ok {
		if pk, ok := kp["public_key"].(string); ok {
			publicKey = pk
		}
	}

	return domain.Signature{
		Type:      "zenroom-hmac",
		Value:     sigValue,
		PublicKey: publicKey,
		KeyRef:    l.keyRef,
	}, nil
}

// verifySignature checks an event signature by recomputing the expected hash.
func (l *Ledger) verifySignature(ctx context.Context, e domain.LedgerEvent) (domain.SignatureVerification, error) {
	if e.Signature.Value == "" || e.Signature.PublicKey == "" {
		return domain.SignatureVerification{Valid: true}, nil
	}

	// Rebuild the canonical body
	payload, err := json.Marshal(map[string]any{
		"seq":           e.Seq,
		"event_type":    string(e.EventType),
		"payload":       e.Payload,
		"previous_hash": e.PreviousHash,
		"created_at":    e.CreatedAt.Format(time.RFC3339),
	})
	if err != nil {
		return domain.SignatureVerification{}, fmt.Errorf("marshal payload: %w", err)
	}

	script, err := os.ReadFile(l.contractsDir + "/verify_signature.zen")
	if err != nil {
		return domain.SignatureVerification{}, fmt.Errorf("load verify_signature contract: %w", err)
	}

	keys, _ := json.Marshal(map[string]any{
		"keypair": map[string]any{"public_key": e.Signature.PublicKey},
	})
	data, _ := json.Marshal(map[string]any{
		"payload":   string(payload),
		"signature": e.Signature.Value,
	})

	result, err := l.executor.Run(script, keys, data)
	if err != nil {
		return domain.SignatureVerification{Valid: false, Details: err.Error()}, nil
	}

	m, err := result.OutputMap()
	if err != nil {
		return domain.SignatureVerification{Valid: false, Details: err.Error()}, nil
	}

	computedHash, ok := m["hash"].(string)
	if !ok {
		return domain.SignatureVerification{Valid: false, Details: "hash not found in verify output"}, nil
	}

	valid := computedHash == e.Signature.Value
	details := "signature verified via Zenroom HMAC"
	if !valid {
		details = fmt.Sprintf("signature mismatch: computed=%s stored=%s", computedHash[:16], e.Signature.Value[:16])
	}

	return domain.SignatureVerification{
		Valid:   valid,
		Details: details,
	}, nil
}
