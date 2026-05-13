-- Verify a Merkle inclusion proof.
-- Input (DATA): {"proof": ["base64hash1","base64hash2"], "leaf":"leaf-data", "root":"base64root", "position":2, "leaf_count":4}
-- Output: {"valid": true/false}
-- Handles invalid base64 and other errors gracefully by returning valid=false.

local MT = require('crypto_merkle')

local data = JSON.decode(DATA)
local valid = false
local ok, err = pcall(function()
    local proof_raw = {}
    for i, v in ipairs(data.proof) do
        proof_raw[i] = OCTET.from_base64(v)
    end
    local root_raw = OCTET.from_base64(data.root)
    valid = MT.verify_proof(proof_raw, data.position, root_raw, data.leaf_count, "sha256")
end)

if not ok then
    valid = false
end

print(JSON.encode({
    valid = valid,
    leaf = data.leaf,
    root = data.root
}))
