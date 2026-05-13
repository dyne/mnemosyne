# Mnemosyne

<p align="center">
  <strong>Cryptographic memory archive — verifiable append-only truth</strong><br>
  <sub>Titaness of memory · dyne.org</sub>
</p>

<p align="center">
  <svg width="560" height="160" viewBox="0 0 560 160" xmlns="http://www.w3.org/2000/svg">
    <!-- Background arcs -->
    <defs>
      <linearGradient id="sky" x1="0" y1="0" x2="1" y2="1">
        <stop offset="0%" stop-color="#0a1628"/>
        <stop offset="100%" stop-color="#1a2d4a"/>
      </linearGradient>
      <linearGradient id="gold" x1="0" y1="0" x2="1" y2="0">
        <stop offset="0%" stop-color="#c9a96e"/>
        <stop offset="100%" stop-color="#e0c78a"/>
      </linearGradient>
    </defs>
    <rect width="560" height="160" rx="16" fill="url(#sky)"/>

    <!-- Stars -->
    <circle cx="40" cy="30" r="1.5" fill="#b8c5d6" opacity="0.6"/>
    <circle cx="120" cy="50" r="1" fill="#b8c5d6" opacity="0.4"/>
    <circle cx="200" cy="25" r="1.5" fill="#b8c5d6" opacity="0.5"/>
    <circle cx="340" cy="45" r="1" fill="#b8c5d6" opacity="0.6"/>
    <circle cx="450" cy="30" r="1.5" fill="#b8c5d6" opacity="0.4"/>
    <circle cx="510" cy="55" r="1" fill="#b8c5d6" opacity="0.5"/>
    <circle cx="280" cy="35" r="1.5" fill="#b8c5d6" opacity="0.7"/>

    <!-- Merkle tree -->
    <line x1="280" y1="60" x2="200" y2="100" stroke="url(#gold)" stroke-width="2" opacity="0.6"/>
    <line x1="280" y1="60" x2="360" y2="100" stroke="url(#gold)" stroke-width="2" opacity="0.6"/>
    <line x1="200" y1="100" x2="150" y2="140" stroke="url(#gold)" stroke-width="2" opacity="0.4"/>
    <line x1="200" y1="100" x2="250" y2="140" stroke="url(#gold)" stroke-width="2" opacity="0.4"/>
    <line x1="360" y1="100" x2="310" y2="140" stroke="url(#gold)" stroke-width="2" opacity="0.4"/>
    <line x1="360" y1="100" x2="410" y2="140" stroke="url(#gold)" stroke-width="2" opacity="0.4"/>

    <!-- Root node -->
    <circle cx="280" cy="58" r="12" fill="#c9a96e" opacity="0.9"/>
    <text x="280" y="63" text-anchor="middle" fill="#0a1628" font-family="monospace" font-size="9" font-weight="bold">★</text>

    <!-- Internal nodes -->
    <circle cx="200" cy="98" r="10" fill="#b8c5d6" opacity="0.7"/>
    <circle cx="360" cy="98" r="10" fill="#b8c5d6" opacity="0.7"/>

    <!-- Leaves -->
    <rect x="140" y="132" width="20" height="12" rx="3" fill="#5a8a6a" opacity="0.8"/>
    <rect x="240" y="132" width="20" height="12" rx="3" fill="#6a9a7a" opacity="0.8"/>
    <rect x="300" y="132" width="20" height="12" rx="3" fill="#7aaa8a" opacity="0.8"/>
    <rect x="400" y="132" width="20" height="12" rx="3" fill="#5a8a6a" opacity="0.8"/>

    <!-- Labels -->
    <text x="85" y="115" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="9">memories</text>
    <line x1="130" y1="120" x2="145" y2="132" stroke="#8899aa" stroke-width="0.5" stroke-dasharray="2,2"/>
    <text x="478" y="115" text-anchor="middle" fill="#c9a96e" font-family="sans-serif" font-size="9">constellation</text>
    <line x1="310" y1="55" x2="450" y2="105" stroke="#8899aa" stroke-width="0.5" stroke-dasharray="2,2"/>

    <!-- Title -->
    <text x="280" y="155" text-anchor="middle" fill="#c9a96e" font-family="sans-serif" font-size="10" letter-spacing="4">MNEMOSYNE</text>
  </svg>
</p>

---

## What is Mnemosyne?

Mnemosyne is an **append-only Merkle tree service** — a transparency log for attestations, events, documents, and workflows. Think of it as a cryptographic notary: once something is archived, it can be **proven** to exist, unchanged, at a specific point in time.

All cryptographic operations are delegated to [**Zenroom**](https://zenroom.org), a deterministic secure language VM. Application code **never** implements hashing, signing, or Merkle proof logic — it only orchestrates.

## Concepts

### Memory — the leaf

A memory is any JSON payload you want to archive. Once stored, it cannot be altered or deleted.

<p align="center">
  <svg width="400" height="100" viewBox="0 0 400 100" xmlns="http://www.w3.org/2000/svg">
    <rect width="400" height="100" rx="12" fill="#faf8f5"/>
    <!-- Document icon -->
    <rect x="50" y="20" width="40" height="50" rx="4" fill="#e8d5b7" stroke="#c9a96e" stroke-width="2"/>
    <line x1="58" y1="32" x2="82" y2="32" stroke="#c9a96e" stroke-width="1.5"/>
    <line x1="58" y1="40" x2="78" y2="40" stroke="#c9a96e" stroke-width="1.5"/>
    <line x1="58" y1="48" x2="82" y2="48" stroke="#c9a96e" stroke-width="1.5"/>
    <!-- Arrow -->
    <line x1="110" y1="45" x2="160" y2="45" stroke="#c9a96e" stroke-width="2" marker-end="url(#arrow-gold)"/>
    <defs><marker id="arrow-gold" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10 Z" fill="#c9a96e"/></marker></defs>
    <!-- Hash badge -->
    <rect x="170" y="20" width="180" height="50" rx="8" fill="#0a1628"/>
    <text x="260" y="38" text-anchor="middle" fill="#b8c5d6" font-family="monospace" font-size="9">payload → Zenroom hash</text>
    <text x="260" y="56" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="10">leaf_hash: uU0nuZN...</text>
    <text x="75" y="88" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="10">{"event":"signed"}</text>
    <text x="260" y="82" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="10">immutable · append-only</text>
  </svg>
</p>

### Beacon — the checkpoint

A beacon anchors the Merkle tree at a point in time. It stores the **root hash** — the cryptographic fingerprint of all memories in the tree. Each beacon links to its parent, forming an unbroken chain of checkpoints.

<p align="center">
  <svg width="480" height="120" viewBox="0 0 480 120" xmlns="http://www.w3.org/2000/svg">
    <!-- Chain of beacons -->
    <rect x="20" y="35" width="100" height="50" rx="10" fill="#1a2d4a" stroke="#c9a96e" stroke-width="2"/>
    <text x="70" y="56" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="9">Beacon #1</text>
    <text x="70" y="72" text-anchor="middle" fill="#b8c5d6" font-family="monospace" font-size="8">root: 1Fu3e...</text>
    <!-- Chain link -->
    <line x1="120" y1="60" x2="155" y2="60" stroke="#c9a96e" stroke-width="2"/>
    <rect x="160" y="35" width="100" height="50" rx="10" fill="#1a2d4a" stroke="#c9a96e" stroke-width="2"/>
    <text x="210" y="56" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="9">Beacon #2</text>
    <text x="210" y="72" text-anchor="middle" fill="#b8c5d6" font-family="monospace" font-size="8">root: KsFrq...</text>
    <!-- Chain link -->
    <line x1="260" y1="60" x2="295" y2="60" stroke="#c9a96e" stroke-width="2"/>
    <rect x="300" y="35" width="100" height="50" rx="10" fill="#1a2d4a" stroke="#c9a96e" stroke-width="2"/>
    <text x="350" y="56" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="9">Beacon #3</text>
    <text x="350" y="72" text-anchor="middle" fill="#b8c5d6" font-family="monospace" font-size="8">root: z/RpD...</text>
    <!-- Parent arrows -->
    <text x="140" y="90" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="7">parent</text>
    <text x="280" y="90" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="7">parent</text>
    <text x="70" y="100" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="9">genesis</text>
    <text x="350" y="100" text-anchor="middle" fill="#c9a96e" font-family="sans-serif" font-size="9">latest</text>
  </svg>
</p>

### Route — the proof

A route is a **Merkle inclusion proof** — a cryptographic path from a single leaf up to the constellation root. It proves that a specific memory exists in the tree without revealing any other memories.

<p align="center">
  <svg width="360" height="180" viewBox="0 0 360 180" xmlns="http://www.w3.org/2000/svg">
    <!-- Tree -->
    <circle cx="180" cy="20" r="8" fill="#c9a96e"/>
    <text x="180" y="15" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="6">★ root</text>

    <line x1="180" y1="28" x2="100" y2="65" stroke="#b8c5d6" stroke-width="1.5"/>
    <line x1="180" y1="28" x2="260" y2="65" stroke="#c9a96e" stroke-width="2.5"/>

    <circle cx="100" cy="68" r="6" fill="#b8c5d6"/>
    <circle cx="260" cy="68" r="6" fill="#c9a96e"/>

    <line x1="100" y1="74" x2="60" y2="110" stroke="#b8c5d6" stroke-width="1"/>
    <line x1="100" y1="74" x2="140" y2="110" stroke="#b8c5d6" stroke-width="1"/>
    <line x1="260" y1="74" x2="220" y2="110" stroke="#c9a96e" stroke-width="2.5"/>
    <line x1="260" y1="74" x2="300" y2="110" stroke="#b8c5d6" stroke-width="1"/>

    <rect x="48" y="112" width="24" height="14" rx="3" fill="#b8c5d6" opacity="0.5"/>
    <rect x="128" y="112" width="24" height="14" rx="3" fill="#b8c5d6" opacity="0.5"/>
    <rect x="208" y="112" width="24" height="14" rx="3" fill="#c9a96e" opacity="0.9"/>
    <rect x="288" y="112" width="24" height="14" rx="3" fill="#b8c5d6" opacity="0.5"/>

    <!-- Highlight: the proven leaf -->
    <rect x="205" y="109" width="30" height="20" rx="4" fill="none" stroke="#c9a96e" stroke-width="2" stroke-dasharray="3,2"/>

    <!-- Labels -->
    <text x="220" y="100" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="7">proven leaf</text>
    <text x="100" y="148" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="9">Proof path: 3 sibling hashes</text>
    <text x="100" y="165" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="9">Recomputed root matches constellation ★</text>
  </svg>
</p>

### Witness — the verification

Witness is the act of verifying a route. Zenroom recomputes the Merkle root from the proof path and checks it against the beacon's root. If it matches, the memory is **authentic**. If a single bit is wrong, it fails.

<p align="center">
  <svg width="360" height="80" viewBox="0 0 360 80" xmlns="http://www.w3.org/2000/svg">
    <rect x="20" y="20" width="120" height="40" rx="8" fill="#1a2d4a" stroke="#c9a96e" stroke-width="2"/>
    <text x="80" y="45" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="10">route + beacon</text>
    <!-- Arrow -->
    <line x1="145" y1="40" x2="180" y2="40" stroke="#c9a96e" stroke-width="2"/>
    <circle cx="195" cy="40" r="20" fill="#0a1628" stroke="#5a8a6a" stroke-width="2.5"/>
    <text x="195" y="45" text-anchor="middle" fill="#7eb77f" font-family="monospace" font-size="12">✓</text>
    <!-- Or fail -->
    <line x1="220" y1="40" x2="260" y2="40" stroke="#c9a96e" stroke-width="2" stroke-dasharray="4,2"/>
    <circle cx="275" cy="40" r="20" fill="#0a1628" stroke="#c4746e" stroke-width="2.5" opacity="0.4"/>
    <text x="275" y="45" text-anchor="middle" fill="#c4746e" font-family="monospace" font-size="12" opacity="0.4">✗</text>
    <text x="195" y="72" text-anchor="middle" fill="#7eb77f" font-family="sans-serif" font-size="9">valid</text>
    <text x="275" y="72" text-anchor="middle" fill="#c4746e" font-family="sans-serif" font-size="9">tampered</text>
  </svg>
</p>

## Architecture

<p align="center">
  <svg width="560" height="200" viewBox="0 0 560 200" xmlns="http://www.w3.org/2000/svg">
    <rect width="560" height="200" rx="12" fill="#faf8f5"/>
    <!-- User -->
    <circle cx="70" cy="100" r="22" fill="#e8d5b7" stroke="#c9a96e" stroke-width="2"/>
    <text x="70" y="105" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="14">U</text>
    <text x="70" y="140" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="9">Client</text>

    <!-- API -->
    <rect x="130" y="78" width="100" height="44" rx="8" fill="#0a1628" stroke="#c9a96e" stroke-width="1.5"/>
    <text x="180" y="102" text-anchor="middle" fill="#c9a96e" font-family="monospace" font-size="8">Go API</text>
    <text x="180" y="114" text-anchor="middle" fill="#b8c5d6" font-family="monospace" font-size="6">net/http</text>

    <!-- SQLite -->
    <rect x="270" y="78" width="80" height="44" rx="8" fill="#1a2d4a" stroke="#5a8a6a" stroke-width="1.5"/>
    <text x="310" y="102" text-anchor="middle" fill="#7eb77f" font-family="monospace" font-size="8">SQLite</text>
    <text x="310" y="114" text-anchor="middle" fill="#b8c5d6" font-family="monospace" font-size="6">append-only</text>

    <!-- Zenroom -->
    <rect x="390" y="78" width="120" height="44" rx="8" fill="#2d1a3a" stroke="#9a5aaa" stroke-width="1.5"/>
    <text x="450" y="102" text-anchor="middle" fill="#c9a0da" font-family="monospace" font-size="8">Zenroom VM</text>
    <text x="450" y="114" text-anchor="middle" fill="#b8c5d6" font-family="monospace" font-size="6">crypto boundary</text>

    <!-- Arrows -->
    <line x1="92" y1="100" x2="128" y2="100" stroke="#c9a96e" stroke-width="1.5" marker-end="url(#a2)"/>
    <line x1="230" y1="100" x2="268" y2="100" stroke="#8899aa" stroke-width="1" marker-end="url(#a3)"/>
    <line x1="350" y1="100" x2="388" y2="100" stroke="#9a5aaa" stroke-width="1.5" marker-end="url(#a4)"/>

    <defs>
      <marker id="a2" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10 Z" fill="#c9a96e"/></marker>
      <marker id="a3" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10 Z" fill="#8899aa"/></marker>
      <marker id="a4" viewBox="0 0 10 10" refX="8" refY="5" markerWidth="6" markerHeight="6" orient="auto"><path d="M0,0 L10,5 L0,10 Z" fill="#9a5aaa"/></marker>
    </defs>

    <!-- Labels -->
    <text x="180" y="170" text-anchor="middle" fill="#8899aa" font-family="sans-serif" font-size="8">orchestration only</text>
    <text x="450" y="170" text-anchor="middle" fill="#c9a0da" font-family="sans-serif" font-size="8">hashing · proofs · signing</text>
    <text x="180" y="185" text-anchor="middle" fill="#8899aa" font-family="monospace" font-size="7">never implements crypto</text>
    <text x="450" y="185" text-anchor="middle" fill="#c9a0da" font-family="monospace" font-size="7">all crypto through contracts</text>
  </svg>
</p>

**The cryptographic boundary is absolute.** Go code only orchestrates — calling Zenroom for every hash, every Merkle root, every proof, every signature. There is no `sha256.Sum()` anywhere in the codebase.

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

| Verb | Path | Concept |
|------|------|---------|
| `POST` | `/memories` | Remember — archive a memory |
| `GET` | `/memories/{id}` | Recall — retrieve a memory |
| `POST` | `/checkpoints` | Anchor — seal a beacon |
| `POST` | `/beacons/{id}/extend` | Extend — add a leaf to a beacon |
| `GET` | `/beacons/{id}` | Inspect — view beacon details |
| `GET` | `/beacons/{id}/memories` | Leaves — list memories in a beacon |
| `GET` | `/proofs/{id}` | Route — generate inclusion proof |
| `POST` | `/verify` | Witness — verify a proof |
| `GET` | `/contracts` | Audit — list Zenroom contracts |
| `GET` | `/contracts/{name}` | Source — read contract source |
| `GET` | `/health` | Pulse — health check |
| `GET` | `/version` | Version — build version |
| `GET` | `/docs` | Reference — Swagger UI |
| `GET` | `/openapi.json` | Spec — OpenAPI 3.0 |

Full interactive documentation at `/docs`.

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

## Cryptographic contracts

Every cryptographic operation is a **versioned Zenroom contract** in `zenflows/`:

| Contract | Language | Purpose |
|----------|----------|---------|
| `hash.zen` | Zencode | Deterministic SHA256 hashing |
| `merkle_root.zen` | Zencode | Merkle tree root from data array |
| `proof_generate.lua` | Lua | Generate inclusion proof |
| `proof_verify.lua` | Lua | Verify inclusion proof |
| `sign.zen` | Zencode | ECDSA signature generation |

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
- **SBOM + provenance** — every release includes SPDX SBOM, cosign signatures, and SLSA build attestation

## License

AGPL-3.0 — Dyne.org foundation

<p align="center">
  <sub>Mnemosyne — titaness of memory, mother of the Muses.<br>She who remembers everything. She who can prove it.</sub>
</p>
