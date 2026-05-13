-- Generate a Merkle inclusion proof for a leaf at a given position.
-- Input (DATA): {"leaves": ["leaf1", "leaf2", "leaf3", "leaf4"], "position": 2}
-- Output: {"root": "...", "proof": ["...", "..."], "leaf": "leaf2", "position": 2}
-- Proof elements are base64-encoded for JSON transport.

local MT = require('crypto_merkle')

local data = JSON.decode(DATA)
local leaves = data.leaves
local pos = data.position

if not leaves or not pos then
    error("missing 'leaves' or 'position' in DATA")
end

local tree = MT.create_merkle_tree(leaves)
local proof_raw = MT.generate_proof(tree, pos)
local root = tree[1]

-- Encode proof elements as base64 for safe JSON transport
local proof_encoded = {}
for i, v in ipairs(proof_raw) do
    proof_encoded[i] = OCTET.to_base64(v)
end

local result = {
    root = OCTET.to_base64(root),
    proof = proof_encoded,
    leaf = leaves[pos],
    position = pos
}

print(JSON.encode(result))
