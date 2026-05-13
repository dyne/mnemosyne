package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/dyne/mnemosyne/internal/domain"
	_ "modernc.org/sqlite"
)

// SQLiteStore implements Store using SQLite.
type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("sqlite open: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite ping: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("sqlite migrate: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS memories (
			id TEXT PRIMARY KEY,
			payload TEXT NOT NULL,
			leaf_hash TEXT NOT NULL,
			beacon_id TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS beacons (
			id TEXT PRIMARY KEY,
			root TEXT NOT NULL,
			parent_beacon_id TEXT NOT NULL DEFAULT '',
			signed_root TEXT NOT NULL DEFAULT '',
			proof_count INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS tree_nodes (
			beacon_id TEXT NOT NULL,
			idx INTEGER NOT NULL,
			hash TEXT NOT NULL,
			PRIMARY KEY (beacon_id, idx)
		);
		CREATE INDEX IF NOT EXISTS idx_memories_beacon ON memories(beacon_id);
	`)
	return err
}

func (s *SQLiteStore) Remember(ctx context.Context, payload any, leafHash string, beaconID string) (*domain.Memory, error) {
	id := NewMemoryID()
	m := &domain.Memory{
		ID:        id,
		Payload:   payload,
		LeafHash:  leafHash,
		BeaconID:  beaconID,
		CreatedAt: ctxTime(ctx),
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO memories (id, payload, leaf_hash, beacon_id, created_at) VALUES (?, ?, ?, ?, ?)`,
		string(m.ID), string(payloadJSON), m.LeafHash, m.BeaconID, m.CreatedAt.UTC().Format(timeFormat),
	)
	if err != nil {
		return nil, fmt.Errorf("insert memory: %w", err)
	}
	return m, nil
}

func (s *SQLiteStore) Recall(ctx context.Context, id domain.MemoryID) (*domain.Memory, error) {
	var m domain.Memory
	var payloadJSON, createdAt string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, payload, leaf_hash, beacon_id, created_at FROM memories WHERE id = ?`,
		string(id),
	).Scan(&m.ID, &payloadJSON, &m.LeafHash, &m.BeaconID, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrMemoryNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query memory: %w", err)
	}

	if err := json.Unmarshal([]byte(payloadJSON), &m.Payload); err != nil {
		return nil, fmt.Errorf("unmarshal payload: %w", err)
	}
	m.CreatedAt, _ = parseTime(createdAt)
	return &m, nil
}

func (s *SQLiteStore) MemoriesByBeacon(ctx context.Context, beaconID domain.BeaconID) ([]*domain.Memory, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, payload, leaf_hash, beacon_id, created_at FROM memories WHERE beacon_id = ? ORDER BY created_at`,
		string(beaconID),
	)
	if err != nil {
		return nil, fmt.Errorf("query memories: %w", err)
	}
	defer rows.Close()

	var memories []*domain.Memory
	for rows.Next() {
		var m domain.Memory
		var payloadJSON, createdAt string
		if err := rows.Scan(&m.ID, &payloadJSON, &m.LeafHash, &m.BeaconID, &createdAt); err != nil {
			return nil, fmt.Errorf("scan memory: %w", err)
		}
		json.Unmarshal([]byte(payloadJSON), &m.Payload)
		m.CreatedAt, _ = parseTime(createdAt)
		memories = append(memories, &m)
	}
	return memories, rows.Err()
}

func (s *SQLiteStore) UpdateBeaconID(ctx context.Context, oldBeaconID, newBeaconID string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE memories SET beacon_id = ? WHERE beacon_id = ?`,
		newBeaconID, oldBeaconID,
	)
	return err
}

func (s *SQLiteStore) AnchorBeacon(ctx context.Context, beacon *domain.Beacon) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO beacons (id, root, parent_beacon_id, signed_root, proof_count, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
		string(beacon.ID), beacon.Root, beacon.ParentBeaconID, beacon.SignedRoot, beacon.ProofCount, beacon.CreatedAt.UTC().Format(timeFormat),
	)
	if err != nil {
		return fmt.Errorf("insert beacon: %w", err)
	}
	return nil
}

func (s *SQLiteStore) LatestBeacon(ctx context.Context) (*domain.Beacon, error) {
	var b domain.Beacon
	var createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, root, parent_beacon_id, signed_root, proof_count, created_at FROM beacons ORDER BY created_at DESC LIMIT 1`,
	).Scan(&b.ID, &b.Root, &b.ParentBeaconID, &b.SignedRoot, &b.ProofCount, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrBeaconNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query latest beacon: %w", err)
	}
	b.CreatedAt, _ = parseTime(createdAt)
	return &b, nil
}

func (s *SQLiteStore) BeaconByID(ctx context.Context, id domain.BeaconID) (*domain.Beacon, error) {
	var b domain.Beacon
	var createdAt string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, root, parent_beacon_id, signed_root, proof_count, created_at FROM beacons WHERE id = ?`,
		string(id),
	).Scan(&b.ID, &b.Root, &b.ParentBeaconID, &b.SignedRoot, &b.ProofCount, &createdAt)
	if err == sql.ErrNoRows {
		return nil, domain.ErrBeaconNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query beacon: %w", err)
	}
	b.CreatedAt, _ = parseTime(createdAt)
	return &b, nil
}

func (s *SQLiteStore) SaveTreeNode(ctx context.Context, beaconID string, index int, hash string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT OR REPLACE INTO tree_nodes (beacon_id, idx, hash) VALUES (?, ?, ?)`,
		beaconID, index, hash,
	)
	return err
}

func (s *SQLiteStore) TreeNodesByBeacon(ctx context.Context, beaconID string) ([]TreeNode, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT idx, hash FROM tree_nodes WHERE beacon_id = ? ORDER BY idx`,
		beaconID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []TreeNode
	for rows.Next() {
		var n TreeNode
		if err := rows.Scan(&n.Index, &n.Hash); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

const timeFormat = "2006-01-02T15:04:05.000000Z"

func ctxTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value("time").(time.Time); ok {
		return t
	}
	return time.Now().UTC()
}

func parseTime(s string) (time.Time, error) {
	return time.Parse(timeFormat, s)
}
