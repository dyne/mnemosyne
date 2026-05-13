# Verification

Verification is the process of walking the entire trust chain to confirm a memory's integrity.

## Verification Path

The full verification checks:

```text
1. memory_hash      — Memory payload canonicalizes to the expected leaf hash
2. merkle_inclusion — Leaf is included in the Merkle root
3. ledger_event     — Root was recorded in a ROOT_SEALED ledger event
4. ledger_chain     — Ledger hash chain is intact
5. checkpoint       — Ledger event is covered by a checkpoint
6. anchor           — Checkpoint is notarized by an anchor
```

## Verification Result

```json
{
  "status": "valid",
  "checks": [
    {
      "name": "memory_hash",
      "status": "ok",
      "details": "Memory payload matches leaf hash."
    },
    {
      "name": "merkle_inclusion",
      "status": "ok",
      "details": "Leaf is included in beacon ..."
    },
    {
      "name": "ledger_event",
      "status": "ok",
      "details": "Memory recorded at ledger event #1"
    },
    {
      "name": "ledger_chain",
      "status": "ok",
      "details": "Ledger hash chain is intact"
    },
    {
      "name": "anchor",
      "status": "ok",
      "details": "Anchor backend local_signature is available"
    }
  ]
}
```

## Check Statuses

| Status | Meaning |
|--------|---------|
| ok | Check passed |
| failed | Check failed — trust chain broken |
| warning | Non-critical issue |
| error | Check could not be performed |
| skipped | Check not applicable |

## Tamper Detection

- Changing a memory payload → `memory_hash` fails
- Modifying the Merkle tree → `merkle_inclusion` fails
- Editing a ledger event → `ledger_chain` fails
- Missing checkpoint → `checkpoint` skipped
- Tampered anchor → `anchor` fails
