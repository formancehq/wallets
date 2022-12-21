package core

import (
	"github.com/google/uuid"
)

const (
	MetadataKeySpecType         = "spec/type"
	MetadataKeyWalletID         = "wallets/id"
	MetadataKeyWalletName       = "wallets/name"
	MetadataKeyWalletCustomData = "wallets/custom_data"
	MetadataKeyHoldWalletID     = "holds/wallet_id"
	MetadataKeyHoldID           = "holds/id"

	PrimaryWallet = "wallets.primary"
	HoldWallet    = "wallets.hold"
)

type Wallet struct {
	ID       string              `json:"id"`
	Balances map[string]Monetary `json:"balances"`
	Metadata Metadata            `json:"metadata"`
	Name     string              `json:"name"`
}

func (w Wallet) LedgerMetadata() Metadata {
	return Metadata{
		MetadataKeySpecType:         PrimaryWallet,
		MetadataKeyWalletID:         w.ID,
		MetadataKeyWalletCustomData: map[string]any(w.Metadata),
		MetadataKeyWalletName:       w.Name,
	}
}

func NewWallet(name string, metadata Metadata) Wallet {
	if metadata == nil {
		metadata = Metadata{}
	}
	return Wallet{
		ID:       uuid.NewString(),
		Metadata: metadata,
		Name:     name,
		Balances: make(map[string]Monetary),
	}
}

func WalletFromAccount(account interface {
	GetMetadata() map[string]any
},
) Wallet {
	return Wallet{
		ID:       account.GetMetadata()[MetadataKeyWalletID].(string),
		Metadata: account.GetMetadata()[MetadataKeyWalletCustomData].(map[string]any),
		// @todo: get balances from subaccounts
		Balances: make(map[string]Monetary),
		Name:     account.GetMetadata()[MetadataKeyWalletName].(string),
	}
}
