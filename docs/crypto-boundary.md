# Crypto Boundary

Mnemosyne does not implement critical cryptography in Go.

All critical cryptographic operations are delegated to **Zenroom VM**:

- **Canonicalization** — deterministic JSON encoding before hashing
- **Hashing** — SHA256 of canonical payloads
- **Merkle tree construction** — building the tree from leaves
- **Proof generation** — creating Merkle inclusion proofs
- **Proof verification** — verifying a leaf is in a root
- **Signatures** — HMAC-style keyed hashing (ECDSA when available in Zenroom build)
- **Signature verification** — recomputing and comparing
- **Key handling** — key material generation via Zenroom random + hash

## Zenroom Interface

The Go application is only the orchestrator. It invokes Zenroom contracts (`.zen` files) located in `zenflows/`:

- `hash.zen` — SHA256 hashing of a string payload
- `merkle_root.zen` — Merkle root computation from string array
- `keygen.zen` — Generate key material via Zenroom random + hash
- `sign.zen` — Sign a payload via keyed hash (HMAC)
- `verify_signature.zen` — Verify a keyed hash signature
- `proof_generate.lua` — Generate Merkle inclusion proof
- `proof_verify.lua` — Verify Merkle inclusion proof

## No Hand-Rolled Crypto

The `internal/zenroom/` package is the sole cryptographic boundary. No other package may:

- Compute hashes directly
- Build Merkle trees
- Generate or verify signatures
- Handle key material

If a cryptographic operation is needed, it must go through Zenroom via the `Executor`.

## References

- https://zenroom.org
- https://zenroom.org/docs
- https://docs.zenroom.org
- https://github.com/dyne/Zenroom
