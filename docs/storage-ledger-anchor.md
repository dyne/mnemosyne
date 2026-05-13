# Storage / Ledger / Anchor

Mnemosyne separates operational persistence, tamper evidence, and external notarization into three distinct layers.

## The Three Layers

```
Storage = where operational data lives
Ledger  = where tamper-evident history is recorded
Anchor  = where signed checkpoints / Merkle roots are notarized externally
```

**Storage is not trust.** The SQLite store holds memories, roots, proofs, and receipts for operational use. It may be changed by an administrator, so integrity is always verified against the ledger and anchors.

**Ledger is tamper evidence.** The NDJSON hash-chain ledger records every action as an append-only signed event. Each event links to the previous via cryptographic hash. The full chain can be verified at any time through the API or CLI.

**Anchor is proof of existence.** The local signature anchor notarizes checkpoints (or roots) by signing them via Zenroom. This creates a portable receipt that can be verified offline.

**Verification connects all three.** The full verification chain walks from memory payload → Zenroom hash → Merkle leaf → Merkle proof → Merkle root → ledger event → checkpoint signature → anchor receipt.

## Trust Chain Diagram

```
Memory payload
    ↓ Zenroom canonicalization + hash
Leaf hash
    ↓ Zenroom Merkle proof
Merkle root
    ↓ signed ledger event
Ledger hash chain
    ↓ signed checkpoint
Checkpoint
    ↓ anchor backend
Notarization receipt
```

## Key Principles

1. Do not silently trust SQLite — verify against ledger
2. Do not silently trust the NDJSON file — verify the hash chain
3. Always verify hashes/signatures through Zenroom
4. Make verification explicit in API, CLI, and UI
5. Make receipts portable — verifiable without the running server
