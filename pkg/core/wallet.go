package core

import (
	"github.com/google/uuid"
)

const (
	MetadataKeySpecType         = "spec/type"
	MetadataKeyWalletID         = "wallets/id"
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
}

func (w Wallet) LedgerMetadata() Metadata {
	return Metadata{
		MetadataKeySpecType:         PrimaryWallet,
		MetadataKeyWalletID:         w.ID,
		MetadataKeyWalletCustomData: map[string]any(w.Metadata),
	}
}

func NewWallet(metadata Metadata) Wallet {
	if metadata == nil {
		metadata = Metadata{}
	}
	return Wallet{
		ID:       uuid.NewString(),
		Metadata: metadata,
		Balances: make(map[string]Monetary),
	}
}
