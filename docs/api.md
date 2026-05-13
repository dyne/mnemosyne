# API Reference

Mnemosyne exposes a REST API. Full OpenAPI specification is available at `/openapi.json` and the Swagger UI at `/docs`.

## Endpoints

### System

| Method | Path | Description |
|--------|------|-------------|
| GET | `/health` | Health check |
| GET | `/version` | Version info |
| GET | `/dashboard` | Dashboard stats |

### Memories

| Method | Path | Description |
|--------|------|-------------|
| POST | `/memories` | Remember a new memory |
| GET | `/memories/{id}` | Recall a memory |
| GET | `/memories/{id}/receipt` | Export receipt bundle |

### Roots / Beacons

| Method | Path | Description |
|--------|------|-------------|
| POST | `/checkpoints` | Seal pending memories into a beacon |
| GET | `/beacons/{id}` | Get beacon details |
| GET | `/beacons/{id}/memories` | List memories in a beacon |
| POST | `/beacons/{id}/extend` | Extend a beacon with a new leaf |

### Proofs

| Method | Path | Description |
|--------|------|-------------|
| GET | `/proofs/{memory_id}` | Generate Merkle inclusion proof |
| POST | `/verify` | Verify a Merkle proof |
| POST | `/verify/full` | Full trust-chain verification |

### Ledger

| Method | Path | Description |
|--------|------|-------------|
| GET | `/ledger/events` | List ledger events |
| GET | `/ledger/head` | Get current ledger head |
| POST | `/ledger/verify` | Verify ledger chain integrity |

### Anchors

| Method | Path | Description |
|--------|------|-------------|
| POST | `/anchors` | Create an anchor |
| GET | `/anchors/{id}` | Get anchor receipt |

### Contracts

| Method | Path | Description |
|--------|------|-------------|
| GET | `/contracts` | List Zenroom contracts |
| GET | `/contracts/{name}` | Get contract source |

### Documentation

| Method | Path | Description |
|--------|------|-------------|
| GET | `/docs` | Swagger UI |
| GET | `/openapi.json` | OpenAPI 3.0 specification |
| GET | `/` | Web UI |
