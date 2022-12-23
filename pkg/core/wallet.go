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
	MetadataKeyAsset            = "holds/asset"
	MetadataKeyHoldID           = "holds/id"

	PrimaryWallet = "wallets.primary"
	HoldWallet    = "wallets.hold"
)

type Wallet struct {
	ID       string   `json:"id"`
	Metadata Metadata `json:"metadata"`
	Name     string   `json:"name"`
}

type WalletWithBalances struct {
	Wallet
	Balances map[string]int32 `json:"balances"`
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
	}
}

func WalletFromAccount(account interface {
	GetMetadata() map[string]any
},
) Wallet {
	return Wallet{
		ID:       account.GetMetadata()[MetadataKeyWalletID].(string),
		Metadata: account.GetMetadata()[MetadataKeyWalletCustomData].(map[string]any),
		Name:     account.GetMetadata()[MetadataKeyWalletName].(string),
	}
}

func WalletWithBalancesFromAccount(account interface {
	GetMetadata() map[string]any
	GetBalances() map[string]int32
},
) WalletWithBalances {
	return WalletWithBalances{
		Wallet:   WalletFromAccount(account),
		Balances: account.GetBalances(),
	}
}
