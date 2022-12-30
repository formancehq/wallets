package core

import (
	"fmt"
	"strings"

	"github.com/formancehq/go-libs/metadata"
	"github.com/google/uuid"
)

const (
	metadataKeyWalletTransaction   = "wallets"
	metadataKeySpecType            = "wallets/spec/type"
	metadataKeyWalletID            = "wallets/id"
	metadataKeyWalletName          = "wallets/name"
	metadataKeyWalletCustomData    = "wallets/custom_data"
	metadataKeyHoldWalletID        = "wallets/holds/wallet_id"
	metadataKeyHoldAsset           = "wallets/holds/asset"
	metadataKeyHoldID              = "wallets/holds/id"
	metadataKeyHoldVoidDestination = "wallets/holds/void_destination"
	metadataKeyHoldDestination     = "wallets/holds/destination"

	PrimaryWallet = "wallets.primary"
	HoldWallet    = "wallets.hold"
)

func FilterMetadata(name string) string {
	m := metadata.SpecMetadata(name)
	//nolint:gomnd
	parts := strings.SplitN(m, "/", 2)
	return fmt.Sprintf(`"%s"/%s`, parts[0], parts[1])
}

func MetadataKeyWalletTransactionMarker() string {
	return metadata.SpecMetadata(metadataKeyWalletTransaction)
}

func MetadataKeyWalletTransactionMarkerFilter() string {
	return FilterMetadata(metadataKeyWalletTransaction)
}

func MetadataKeyWalletSpecType() string {
	return metadata.SpecMetadata(metadataKeySpecType)
}

func MetadataKeyWalletSpecTypeFilter() string {
	return FilterMetadata(metadataKeySpecType)
}

func MetadataKeyWalletID() string {
	return metadata.SpecMetadata(metadataKeyWalletID)
}

func MetadataKeyWalletIDFilter() string {
	return FilterMetadata(metadataKeyWalletID)
}

func MetadataKeyHoldID() string {
	return metadata.SpecMetadata(metadataKeyHoldID)
}

func MetadataKeyHoldWalletID() string {
	return metadata.SpecMetadata(metadataKeyHoldWalletID)
}

func MetadataKeyHoldWalletIDFilter() string {
	return FilterMetadata(metadataKeyHoldWalletID)
}

func MetadataKeyHoldAsset() string {
	return metadata.SpecMetadata(metadataKeyHoldAsset)
}

func MetadataKeyWalletCustomData() string {
	return metadata.SpecMetadata(metadataKeyWalletCustomData)
}

func MetadataKeyWalletCustomDataFilter(key string) string {
	return FilterMetadata(metadataKeyWalletCustomData) + "." + key
}

func MetadataKeyWalletName() string {
	return metadata.SpecMetadata(metadataKeyWalletName)
}

func MetadataKeyWalletNameFilter() string {
	return FilterMetadata(metadataKeyWalletName)
}

func MetadataKeyHoldVoidDestination() string {
	return metadata.SpecMetadata(metadataKeyHoldVoidDestination)
}

func MetadataKeyHoldDestination() string {
	return metadata.SpecMetadata(metadataKeyHoldDestination)
}

func WalletTransactionBaseMetadata() metadata.Metadata {
	return metadata.Metadata{
		MetadataKeyWalletTransactionMarker(): true,
	}
}

func WalletTransactionBaseMetadataFilter() metadata.Metadata {
	return metadata.Metadata{
		MetadataKeyWalletTransactionMarkerFilter(): true,
	}
}

func IsPrimary(v metadata.Owner) bool {
	return HasMetadata(v, MetadataKeyWalletSpecType(), PrimaryWallet)
}

func IsHold(v metadata.Owner) bool {
	return HasMetadata(v, MetadataKeyWalletSpecType(), HoldWallet)
}

func GetMetadata(v metadata.Owner, key string) any {
	return v.GetMetadata()[key]
}

func HasMetadata(v metadata.Owner, key, value string) bool {
	return GetMetadata(v, key) == value
}

func SpecType(v metadata.Owner) string {
	return GetMetadata(v, MetadataKeyWalletSpecType()).(string)
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
		MetadataKeyWalletSpecType():   PrimaryWallet,
		MetadataKeyWalletID():         w.ID,
		MetadataKeyWalletCustomData(): map[string]any(w.Metadata),
		MetadataKeyWalletName():       w.Name,
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

func WalletFromAccount(account metadata.Owner) Wallet {
	return Wallet{
		ID:       GetMetadata(account, MetadataKeyWalletID()).(string),
		Metadata: GetMetadata(account, MetadataKeyWalletCustomData()).(map[string]any),
		Name:     GetMetadata(account, MetadataKeyWalletName()).(string),
	}
}

func WalletWithBalancesFromAccount(account interface {
	metadata.Owner
	GetBalances() map[string]int32
},
) WalletWithBalances {
	return WalletWithBalances{
		Wallet:   WalletFromAccount(account),
		Balances: account.GetBalances(),
	}
}
