package ledger

import (
	"context"

	"github.com/dyne/mnemosyne/internal/domain"
)

// Backend is the tamper-evident ledger interface.
type Backend interface {
	Append(ctx context.Context, typ domain.EventType, payload any) (domain.LedgerReceipt, error)
	GetEvent(ctx context.Context, seq uint64) (domain.LedgerEvent, error)
	ListEvents(ctx context.Context, opts domain.LedgerListOptions) ([]domain.LedgerEvent, error)
	LatestHead(ctx context.Context) (domain.LedgerHead, error)
	Verify(ctx context.Context) (domain.LedgerVerification, error)
	Close() error
}
