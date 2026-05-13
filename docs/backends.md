# Backends

Mnemosyne uses a pluggable backend architecture with three families.

## Backend Matrix

### Storage Backends

| Backend | Status | Description |
|---------|--------|-------------|
| sqlite | implemented | SQLite operational store via modernc.org/sqlite |
| postgres | planned | PostgreSQL for multi-node deployments |
| filesystem | planned | File-based storage for simplicity |
| s3 | planned | Object storage for cloud deployments |

### Ledger Backends

| Backend | Status | Description |
|---------|--------|-------------|
| ndjson_hash_chain | implemented | Append-only NDJSON file with hash-chain linking |
| sqlite_event_log | planned | SQLite-based event log |
| git_signed_commits | planned | Git repository with signed commits |
| s3_object_lock | planned | S3 with object lock for immutability |
| transparency_log | planned | Certificate Transparency-style log |

### Anchor Backends

| Backend | Status | Description |
|---------|--------|-------------|
| local_signature | implemented | Local Zenroom-based signature notarization |
| opentimestamps | planned | OpenTimestamps-compatible anchoring |
| bitcoin | planned | Bitcoin transaction anchoring |
| ethereum | planned | Ethereum smart contract anchoring |
| transparency_checkpoint | planned | Generic transparency checkpoint |

## Backend Interfaces

Each backend family has a Go interface defined in `internal/domain/backends.go`:

```go
type StorageBackend interface { ... }
type LedgerBackend interface { ... }
type AnchorBackend interface { ... }
```

New backends implement the corresponding interface and are plugged in via configuration.
