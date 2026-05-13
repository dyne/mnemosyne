# Receipts

A receipt is a portable verification bundle. It contains everything needed to verify a memory offline — without the running Mnemosyne server.

## Receipt Format

```json
{
  "receipt_version": "mnemosyne.receipt.v1",
  "memory": {
    "id": "mem_...",
    "leaf_hash": "0x..."
  },
  "proof": {
    "root_id": "root_...",
    "root_hash": "0x...",
    "siblings": [],
    "position": 1,
    "leaf_count": 4
  },
  "ledger": {
    "backend": "ndjson_hash_chain",
    "event_seq": 29,
    "event_hash": "0x...",
    "ledger_head": "0x..."
  },
  "checkpoint": {
    "id": "chk_...",
    "from_seq": 1,
    "to_seq": 29,
    "ledger_head": "0x...",
    "signature": {}
  },
  "anchor": {
    "backend": "local_signature",
    "status": "confirmed",
    "receipt": {}
  }
}
```

## Export

Receipts can be exported via:

- **API**: `GET /memories/{id}/receipt`
- **CLI**: `mnemosyne receipt export --memory mem_...`

## Verification

A receipt can be verified by checking:

1. Memory payload canonicalizes to the expected leaf hash
2. Leaf is included in the Merkle root (using the proof siblings)
3. Merkle root is recorded in a ROOT_SEALED ledger event
4. ROOT_SEALED event is part of the ledger hash chain
5. Ledger head is included in a checkpoint
6. Checkpoint signature is valid
7. Anchor receipt is valid

Verification can be done via:

- **API**: `POST /verify/full`
- **CLI**: `mnemosyne verify --receipt receipt.json`
