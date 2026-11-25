package wallet

import (
	"context"
	"fmt"
)

const (
	v3HoldReplayKey         = "wallets/holdReplayKey"
	v3HoldReplayFingerprint = "wallets/holdReplayFingerprint"
)

func (m *Manager) holdReplay(ctx context.Context, key, operation string) (fingerprint string, replayed bool, err error) {
	reader, ok := m.client.(interface {
		findHoldReplay(context.Context, string, string, string) (bool, error)
	})
	if key == "" || !ok {
		return "", false, nil
	}

	// Confirm and void must replay even when the original write closed the hold.
	fingerprint = hashIdempotencyKey(operation)
	replayed, err = reader.findHoldReplay(ctx, m.ledgerName, key, fingerprint)
	return fingerprint, replayed, err
}

func (l *ledgerV3) findHoldReplay(ctx context.Context, ledger, key, fingerprint string) (bool, error) {
	replayKey := v3IdempotencyKey(ledger, key)
	page, err := v3List[v3Transaction](ctx, l, "transactions", v3Cursor{
		Ledger: ledger,
		Query:  v3ListQuery{Size: 2, Metadata: map[string]string{v3HoldReplayKey: replayKey}},
	})
	if err != nil {
		return false, fmt.Errorf("look up Ledger v3 hold replay: %w", err)
	}
	if len(page.Data) == 0 {
		return false, nil
	}
	if len(page.Data) != 1 || page.HasMore {
		return false, fmt.Errorf("Ledger v3 returned multiple transactions for one hold replay key")
	}
	saved := page.Data[0].Metadata.strings()
	if saved[v3HoldReplayKey] != replayKey || saved[v3HoldReplayFingerprint] == "" {
		return false, fmt.Errorf("Ledger v3 returned an invalid hold replay marker")
	}
	if saved[v3HoldReplayFingerprint] != fingerprint {
		return false, ErrIdempotencyConflict
	}
	return true, nil
}
