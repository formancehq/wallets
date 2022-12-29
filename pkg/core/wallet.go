package core

import (
	"github.com/formancehq/go-libs/metadata"
	"github.com/google/uuid"
)

const (
	MetadataKeyWalletTransaction = "wallets"
	MetadataKeySpecType          = "spec/type"
	MetadataKeyWalletID          = "wallets/id"
	MetadataKeyWalletName        = "wallets/name"
	MetadataKeyWalletCustomData  = "wallets/custom_data"
	MetadataKeyHoldWalletID      = "holds/wallet_id"
	MetadataKeyAsset             = "holds/asset"
	MetadataKeyHoldID            = "holds/id"

	PrimaryWallet = "wallets.primary"
	HoldWallet    = "wallets.hold"
)

func WalletTransactionBaseMetadata() metadata.Metadata {
	return metadata.Metadata{
		MetadataKeyWalletTransaction: true,
	}
}

type Wallet struct {
	ID       string            `json:"id"`
	Metadata metadata.Metadata `json:"metadata"`
	Name     string            `json:"name"`
}

type WalletWithBalances struct {
	Wallet
	Balances map[string]int32 `json:"balances"`
}

func (w Wallet) LedgerMetadata() metadata.Metadata {
	return metadata.Metadata{
		MetadataKeySpecType:         PrimaryWallet,
		MetadataKeyWalletID:         w.ID,
		MetadataKeyWalletCustomData: map[string]any(w.Metadata),
		MetadataKeyWalletName:       w.Name,
	}
}

func NewWallet(name string, m metadata.Metadata) Wallet {
	if m == nil {
		m = metadata.Metadata{}
	}
	return Wallet{
		ID:       uuid.NewString(),
		Metadata: m,
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
