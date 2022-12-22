package core

import (
	"github.com/google/uuid"
)

type DebitHold struct {
	ID          string `json:"id"`
	WalletID    string `json:"walletID"`
	Destination string `json:"destination"`
	Asset       string `json:"asset"`
}

func (h DebitHold) LedgerMetadata(chart *Chart) Metadata {
	return Metadata{
		//nolint:godox
		// TODO: Use defined namespace on ledger
		MetadataKeySpecType:     "wallets.hold",
		MetadataKeyHoldWalletID: h.WalletID,
		MetadataKeyHoldID:       h.ID,
		MetadataKeyAsset:        h.Asset,
		"void_destination": map[string]interface{}{
			"type":  "account",
			"value": chart.GetMainAccount(h.WalletID),
		},
		"destination": map[string]interface{}{
			"type":  "account",
			"value": h.Destination,
		},
	}
}

func NewDebitHold(walletID, destination, asset string) DebitHold {
	return DebitHold{
		ID:          uuid.NewString(),
		WalletID:    walletID,
		Destination: destination,
		Asset:       asset,
	}
}

func DebitHoldFromLedgerAccount(account interface {
	GetMetadata() map[string]any
},
) DebitHold {
	hold := DebitHold{}
	hold.ID = account.GetMetadata()[MetadataKeyHoldID].(string)
	hold.WalletID = account.GetMetadata()[MetadataKeyHoldWalletID].(string)
	hold.Destination = account.GetMetadata()["destination"].(map[string]any)["value"].(string)
	hold.Asset = account.GetMetadata()[MetadataKeyAsset].(string)
	return hold
}
