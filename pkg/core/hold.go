package core

import (
	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/google/uuid"
)

type DebitHold struct {
	ID          string `json:"id"`
	WalletID    string `json:"walletID"`
	Destination string `json:"destination"`
	Asset string `json:"asset"`
}

func (h DebitHold) LedgerMetadata(chart *Chart) Metadata {
	return Metadata{
		//nolint:godox
		// TODO: Use defined namespace on ledger
		MetadataKeySpecType:     "wallets.hold",
		MetadataKeyHoldWalletID: h.WalletID,
		MetadataKeyHoldID:       h.ID,
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

func NewDebitHold(walletID, destination string) DebitHold {
	return DebitHold{
		ID:          uuid.NewString(),
		WalletID:    walletID,
		Destination: destination,
	}
}

func DebitHoldFromLedgerAccount(account sdk.AccountWithVolumesAndBalances) DebitHold {
	hold := DebitHold{}
	hold.ID = account.Metadata[MetadataKeyHoldID].(string)
	hold.WalletID = account.Metadata[MetadataKeyHoldWalletID].(string)
	hold.Destination = account.Metadata["destination"].(map[string]any)["value"].(string)
	return hold
}
