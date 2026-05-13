# Mnemosyne

<p align="center">
  <strong>Cryptographic memory archive — verifiable append-only truth</strong><br>
  <sub>Titaness of memory · dyne.org</sub>
</p>

<p align="center">
  <img src="assets/hero.svg" alt="Mnemosyne — Merkle tree constellation">
</p>

---

## What is Mnemosyne?

Mnemosyne is an **append-only Merkle tree service** — a transparency log for attestations, events, documents, and workflows. Think of it as a cryptographic notary: once something is archived, it can be **proven** to exist, unchanged, at a specific point in time.

All cryptographic operations are delegated to [**Zenroom**](https://zenroom.org), a deterministic secure language VM. Application code **never** implements hashing, signing, or Merkle proof logic — it only orchestrates.

## The three-layer model

Mnemosyne separates operational persistence, tamper evidence, and external notarization into three distinct layers:

```
Storage = where operational data lives
Ledger  = where tamper-evident history is recorded
Anchor  = where signed checkpoints / Merkle roots are notarized externally
```

| Layer | Role | Trust model | Backend |
|-------|------|-------------|---------|
| **Storage** | Holds memories, roots, proofs, receipts | Not trusted — verifiable against ledger | SQLite (pluggable) |
| **Ledger** | Append-only signed event log with hash-chain linking | Tamper-evident | NDJSON hash chain (pluggable) |
| **Anchor** | External notarization of checkpoints | Proof of existence | Local Zenroom signatures (pluggable) |

All three layers have **pluggable backends** — see [Backends](docs/backends.md) for the full matrix of implemented and planned backends.

## Concepts

### Memory — the leaf

A memory is any JSON payload you want to archive. Once stored, it cannot be altered or deleted.

<p align="center">
  <img src="assets/memory.svg" alt="Memory — JSON payload hashed to leaf">
</p>

### Beacon — the checkpoint

A beacon anchors the Merkle tree at a point in time. It stores the **root hash** — the cryptographic fingerprint of all memories in the tree. Each beacon links to its parent, forming an unbroken chain of checkpoints.

<p align="center">
  <img src="assets/beacon.svg" alt="Beacon — chain of Merkle root checkpoints">
</p>

### Route — the proof

A route is a **Merkle inclusion proof** — a cryptographic path from a single leaf up to the constellation root. It proves that a specific memory exists in the tree without revealing any other memories.

<p align="center">
  <img src="assets/route.svg" alt="Route — Merkle inclusion proof path">
</p>

### Witness — the verification

Witness is the act of verifying a route. Zenroom recomputes the Merkle root from the proof path and checks it against the beacon's root. If it matches, the memory is **authentic**. If a single bit is wrong, it fails.

<p align="center">
  <img src="assets/witness.svg" alt="Witness — verify or reject a proof">
</p>

### Receipt — the portable proof

A receipt is a portable verification bundle containing everything needed to verify a memory offline — the memory itself, its Merkle proof, the ledger event, the checkpoint, and the anchor receipt. Receipts can be exported via API or CLI and verified without the running server.

### Full verification

Full verification walks the entire trust chain end to end: memory payload → Zenroom hash → Merkle leaf → Merkle proof → Merkle root → ledger event → ledger hash chain → checkpoint → anchor receipt. Each step returns pass/fail/warning, making tampering immediately visible.

## Architecture

<p align="center">
  <img src="assets/architecture.svg" alt="Architecture — Go API, SQLite, Zenroom VM">
</p>

**The cryptographic boundary is absolute.** Go code only orchestrates — calling Zenroom for every hash, every Merkle root, every proof, every signature. There is no `sha256.Sum()` anywhere in the codebase.

See [Architecture](docs/architecture.md) for repository layout, data flow diagrams, and key design decisions.

## Quick start

```bash
# Clone and start
git clone https://github.com/dyne/mnemosyne.git
cd mnemosyne
task run              # starts on :8546

# Or with Docker
task docker:up        # server on :8546
task docker:tunnel    # server + Cloudflare tunnel
```

Open `http://localhost:8546` — you'll see the maritime observatory UI.

## API

| Method | Path | Description |
|--------|------|-------------|
| **System** | | |
| `GET` | `/health` | Health check |
| `GET` | `/version` | Version info |
| `GET` | `/dashboard` | Dashboard stats |
| **Memories** | | |
| `POST` | `/memories` | Remember — archive a memory |
| `GET` | `/memories/{id}` | Recall — retrieve a memory |
| `GET` | `/memories/{id}/receipt` | Export — portable receipt bundle |
| **Beacons** | | |
| `POST` | `/checkpoints` | Anchor — seal a beacon |
| `POST` | `/beacons/{id}/extend` | Extend — add a leaf to a beacon |
| `GET` | `/beacons/{id}` | Inspect — view beacon details |
| `GET` | `/beacons/{id}/memories` | Leaves — list memories in a beacon |
| **Proofs** | | |
| `GET` | `/proofs/{memory_id}` | Route — generate inclusion proof |
| `POST` | `/verify` | Witness — verify a proof |
| `POST` | `/verify/full` | Full trust-chain verification |
| **Ledger** | | |
| `GET` | `/ledger/events` | List ledger events |
| `GET` | `/ledger/head` | Current ledger head |
| `POST` | `/ledger/verify` | Verify ledger chain integrity |
| **Anchors** | | |
| `POST` | `/anchors` | Create an anchor receipt |
| `GET` | `/anchors/{id}` | Get anchor receipt |
| **Contracts** | | |
| `GET` | `/contracts` | List Zenroom contracts |
| `GET` | `/contracts/{name}` | Read contract source |
| **Docs** | | |
| `GET` | `/docs` | Swagger UI |
| `GET` | `/openapi.json` | OpenAPI 3.0 specification |

Full interactive documentation at `/docs`.

## Documentation

| Document | Description |
|----------|-------------|
| [API Reference](docs/api.md) | Complete endpoint reference |
| [Architecture](docs/architecture.md) | Repository layout and data flows |
| [Storage / Ledger / Anchor](docs/storage-ledger-anchor.md) | The three-layer trust model |
| [Backends](docs/backends.md) | Pluggable backend matrix and interfaces |
| [Crypto Boundary](docs/crypto-boundary.md) | Zenroom delegation contract |
| [Receipts](docs/receipts.md) | Portable verification bundle format |
| [Verification](docs/verification.md) | Full trust-chain verification walk |

## Vocabulary

| Technical term | Mnemosyne name | Why |
|----------------|----------------|-----|
| Leaf | **Memory** | Something remembered |
| Merkle root | **Constellation** | A pattern of connected stars |
| Checkpoint | **Beacon** | A signal anchoring time |
| Inclusion proof | **Route** | A verifiable path |
| Verification | **Witness** | Bearing testimony to truth |
| Append | **Remember** | Committing to memory |
| Merkle tree | **Tree of memories** | Rooted in truth |
| Operational store | **Storage** | Where data lives — not trusted |
| Event log | **Ledger** | Tamper-evident history |
| Notarization | **Anchor** | External proof of existence |
| Proof bundle | **Receipt** | Portable offline verification |

## Cryptographic contracts

Every cryptographic operation is a **versioned Zenroom contract** in `zenflows/`:

| Contract | Language | Purpose |
|----------|----------|---------|
| `hash.zen` | Zencode | Deterministic SHA256 hashing |
| `merkle_root.zen` | Zencode | Merkle tree root from data array |
| `keygen.zen` | Zencode | Generate key material via Zenroom random + hash |
| `sign.zen` | Zencode | Sign payload via HMAC (keyed hash) |
| `verify_signature.zen` | Zencode | Verify signature via keyed hash |
| `proof_generate.lua` | Lua | Generate inclusion proof |
| `proof_verify.lua` | Lua | Verify inclusion proof |

All contracts are auditable at runtime — visit `/contracts` or click **Contracts** in the UI to read the source with syntax highlighting.

## Design

Mnemosyne draws visual inspiration from:

- **Maritime observatories** — brass instruments, nautical charts, starlight navigation
- **Ancient archives** — parchment, sealed records, immutable ledgers
- **Constellations** — branching Merkle paths forming star patterns across the sky

The color palette: deep navy (`#0a1628`), parchment (`#f4e4c1`), brass (`#c9a96e`), starlight silver (`#b8c5d6`), dark slate (`#2d3a4a`).

## Security

- **No crypto in application code** — all hashing, signing, and Merkle operations are delegated to Zenroom VM
- **Append-only** — memories can be created and retrieved; there is no update or delete path
- **Immutability** — once a beacon anchors a tree, its root becomes a permanent cryptographic checkpoint
- **Deterministic builds** — `CGO_ENABLED=0`, pure Go, reproducible binaries
- **Transparent contracts** — all cryptographic logic is in human-readable Zencode/Lua scripts served at `/contracts`
- **Tamper-evident ledger** — every action is recorded as a hash-chain-linked signed event; any modification breaks the chain
- **Portable receipts** — proof bundles can be exported and verified offline without the running server
- **SBOM + provenance** — every release includes SPDX SBOM, cosign signatures, and SLSA build attestation

## License

AGPL-3.0 — Dyne.org foundation

<p align="center">
  <sub>Mnemosyne — titaness of memory, mother of the Muses.<br>She who remembers everything. She who can prove it.</sub>
</p>
