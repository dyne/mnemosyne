package domain

import "errors"

var (
	ErrMemoryNotFound      = errors.New("memory not found")
	ErrBeaconNotFound      = errors.New("beacon not found")
	ErrProofNotAvailable   = errors.New("proof not available")
	ErrWitnessFailed       = errors.New("witness verification failed")
	ErrInvalidPayload      = errors.New("invalid payload")
	ErrTreeEmpty           = errors.New("merkle tree is empty")
	ErrAppendOnly          = errors.New("append-only violation: cannot modify existing memory")
	ErrRootNotFound        = errors.New("root not found")
	ErrCheckpointNotFound  = errors.New("checkpoint not found")
	ErrAnchorNotFound      = errors.New("anchor not found")
	ErrLedgerEventNotFound = errors.New("ledger event not found")
	ErrLedgerVerification  = errors.New("ledger verification failed")
	ErrAnchorVerification  = errors.New("anchor verification failed")
	ErrVerificationFailed  = errors.New("full verification failed")
	ErrNoPendingMemories   = errors.New("no pending memories to seal")
	ErrBackendNotAvailable = errors.New("backend not available")
)
