# Architecture

Mnemosyne is a Go-based notarized memory service inspired by the Greek goddess of memory.

## Core Architectural Model

```text
Storage = where operational data lives
Ledger  = where tamper-evident history is recorded
Anchor  = where signed checkpoints / Merkle roots are notarized externally
```

## Repository Layout

```
.
├── cmd/mnemosyne/        # CLI entry point
├── internal/
│   ├── api/              # HTTP API server
│   ├── anchor/           # Anchor backend interface
│   │   └── local/        # Local signature anchor
│   ├── crypto/            # Crypto backend (Zenroom)
│   │   └── zenroom/      # Zenroom executor
│   ├── domain/           # Shared domain types
│   ├── ledger/           # Ledger backend interface
│   │   └── ndjson/       # NDJSON hash-chain ledger
│   ├── merkle/           # Merkle tree orchestrator
│   ├── receipts/         # Receipt bundle export
│   ├── storage/          # Storage backend
│   │   └── sqlite/       # SQLite storage
│   └── verifier/         # Full-chain verification
├── web/                  # Embedded web UI
│   ├── index.html
│   └── static/
│       ├── app.js
│       └── style.css
├── zenflows/             # Zenroom contracts
│   ├── hash.zen
│   ├── keygen.zen
│   ├── merkle_root.zen
│   ├── sign.zen
│   ├── verify_signature.zen
│   ├── proof_generate.lua
│   └── proof_verify.lua
└── docs/                 # Documentation
```

## Key Design Decisions

1. **Crypto boundary is explicit** — All hashing, signing, and proof generation happens in Zenroom. Go code only orchestrates.
2. **Backends are pluggable** — Storage, ledger, and anchor backends implement Go interfaces.
3. **Append-only** — No updates or deletes. Every action is recorded in the ledger.
4. **Portable receipts** — Verification bundles can be exported and verified offline.
5. **Boring and inspectable** — First implementations use SQLite and NDJSON files for transparency.

## Data Flow

```
POST /remember
  → Zenroom: canonicalize + hash payload
  → Storage: save memory
  → Ledger: append MEMORY_RECORDED event

POST /checkpoints
  → Storage: collect pending memories
  → Zenroom: build Merkle tree
  → Storage: save beacon
  → Ledger: append ROOT_SEALED event

POST /anchors
  → Anchor: sign checkpoint hash via Zenroom
  → Ledger: append ANCHOR_CREATED event

POST /verify/full
  → Zenroom: re-hash payload
  → Zenroom: verify Merkle proof
  → Ledger: verify hash chain
  → Anchor: verify signature
  → Return verification checklist
```
