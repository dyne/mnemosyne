package domain

import "errors"

var (
	ErrMemoryNotFound    = errors.New("memory not found")
	ErrBeaconNotFound    = errors.New("beacon not found")
	ErrProofNotAvailable = errors.New("proof not available")
	ErrWitnessFailed     = errors.New("witness verification failed")
	ErrInvalidPayload    = errors.New("invalid payload")
	ErrTreeEmpty         = errors.New("merkle tree is empty")
	ErrAppendOnly        = errors.New("append-only violation: cannot modify existing memory")
)
