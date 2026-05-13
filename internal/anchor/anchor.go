package anchor

import (
	"context"

	"github.com/dyne/mnemosyne/internal/domain"
)

// Backend is where roots or checkpoints are notarized externally.
type Backend interface {
	Name() string
	Anchor(ctx context.Context, hash string, anchoredType, anchoredID string) (domain.AnchorReceipt, error)
	VerifyAnchor(ctx context.Context, receipt domain.AnchorReceipt) (domain.AnchorVerification, error)
}
