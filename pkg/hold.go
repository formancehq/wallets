package wallet

import (
	"math/big"

	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
	"github.com/google/uuid"
)

type DebitHold struct {
	ID          string            `json:"id"`
	WalletID    string            `json:"walletID"`
	Destination Subject           `json:"destination"`
	Asset       string            `json:"asset"`
	Metadata    metadata.Metadata `json:"metadata"`
	Description string            `json:"description"`
}

func (h DebitHold) LedgerMetadata(chart *Chart) map[string]string {

	return metadata.Metadata{
		MetadataKeyWalletSpecType: HoldWallet,
		MetadataKeyHoldWalletID:   h.WalletID,
		MetadataKeyHoldID:         h.ID,
		MetadataKeyHoldAsset:      h.Asset,
		MetadataKeyHoldVoidDestination: metadata.MarshalValue(map[string]any{
			"type":  "account",
			"value": chart.GetMainBalanceAccount(h.WalletID),
		}),
		MetadataKeyHoldDestination: metadata.MarshalValue(map[string]any{
			"type":  "account",
			"value": h.Destination.getAccount(chart),
		}),
		MetadataKeyWalletHoldDescription: h.Description,
		MetadataKeyHoldSubject: metadata.MarshalValue(map[string]any{
			"type":       h.Destination.Type,
			"identifier": h.Destination.Identifier,
			"balance":    h.Destination.Balance,
		}),
	}.Merge(EncodeCustomMetadata(h.Metadata))
}

func NewDebitHold(walletID string, destination Subject, asset, description string, md metadata.Metadata) DebitHold {
	return DebitHold{
		ID:          uuid.NewString(),
		WalletID:    walletID,
		Destination: destination,
		Asset:       asset,
		Metadata:    md,
		Description: description,
	}
}

func DebitHoldFromLedgerAccount(account AccountWithVolumesAndBalances) DebitHold {
	destination := metadata.UnmarshalValue[metadata.Metadata](account.GetMetadata()[MetadataKeyHoldSubject])

	hold := DebitHold{}
	hold.ID = account.GetMetadata()[MetadataKeyHoldID]
	hold.WalletID = account.GetMetadata()[MetadataKeyHoldWalletID]
	hold.Destination = Subject{
		Type:       destination["type"],
		Identifier: destination["identifier"],
		Balance:    destination["balance"],
	}
	hold.Asset = account.GetMetadata()[MetadataKeyHoldAsset]
	hold.Description = account.GetMetadata()[MetadataKeyWalletHoldDescription]
	hold.Metadata = ExtractCustomMetadata(account)

	return hold
}

type ExpandedDebitHold struct {
	DebitHold
	OriginalAmount *big.Int `json:"originalAmount"`
	Remaining      *big.Int `json:"remaining"`
}

func (h ExpandedDebitHold) IsClosed() bool {
	return h.Remaining.Sign() == 0
}

func ExpandedDebitHoldFromLedgerAccount(account AccountWithVolumesAndBalances) ExpandedDebitHold {
	hold := ExpandedDebitHold{
		DebitHold: DebitHoldFromLedgerAccount(account),
	}
	hold.OriginalAmount = account.GetVolumes()[hold.Asset].Input
	hold.Remaining = account.GetBalances()[hold.Asset]
	return hold
}

type ConfirmHold struct {
	HoldID    string `json:"holdID"`
	Amount    *big.Int
	Reference string
	Final     bool
}

func (c ConfirmHold) resolveAmount(hold ExpandedDebitHold) (*big.Int, error) {
	if c.Amount == nil || c.Amount.Sign() == 0 {
		return hold.Remaining, nil
	}
	if c.Amount.Sign() < 0 {
		return nil, ErrNegativeAmount
	}
	if c.Amount.Cmp(hold.Remaining) > 0 {
		return nil, ErrInsufficientFundError
	}
	return c.Amount, nil
}

type VoidHold struct {
	HoldID string `json:"holdID"`
}
