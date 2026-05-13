package domain

// EventType labels every ledger entry.
type EventType string

const (
	EventMemoryRecorded        EventType = "MEMORY_RECORDED"
	EventMemoryUpdatedMetadata EventType = "MEMORY_UPDATED_METADATA"
	EventRootSealed            EventType = "ROOT_SEALED"
	EventProofGenerated        EventType = "PROOF_GENERATED"
	EventCheckpointCreated     EventType = "CHECKPOINT_CREATED"
	EventAnchorCreated         EventType = "ANCHOR_CREATED"
	EventAnchorConfirmed       EventType = "ANCHOR_CONFIRMED"
	EventVerifyRequested       EventType = "VERIFY_REQUESTED"
	EventVerifyOK              EventType = "VERIFY_OK"
	EventVerifyFailed          EventType = "VERIFY_FAILED"
)
