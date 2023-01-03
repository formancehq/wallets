package wallet

import (
	"net/http"

	"github.com/formancehq/go-libs/metadata"
	"github.com/google/uuid"
)

type ListWallets struct {
	Metadata metadata.Metadata
	Name     string
}

type PatchRequest struct {
	Metadata metadata.Metadata `json:"metadata"`
}

func (c *PatchRequest) Bind(r *http.Request) error {
	return nil
}

type CreateRequest struct {
	PatchRequest
	Name string `json:"name"`
}

func (c *CreateRequest) Bind(r *http.Request) error {
	return nil
}

type Wallet struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Metadata metadata.Metadata `json:"metadata"`
}

type WithBalances struct {
	Wallet
	Balances map[string]int32 `json:"balances"`
}

func (w Wallet) LedgerMetadata() metadata.Metadata {
	return metadata.Metadata{
		MetadataKeyWalletSpecType:   PrimaryWallet,
		MetadataKeyWalletName:       w.Name,
		MetadataKeyWalletCustomData: map[string]any(w.Metadata),
		MetadataKeyWalletID:         w.ID,
		MetadataKeyWalletBalance:    TrueValue,
		MetadataKeyBalanceName:      MainBalance,
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

func FromAccount(account metadata.Owner) Wallet {
	return Wallet{
		ID:       GetMetadata(account, MetadataKeyWalletID).(string),
		Name:     GetMetadata(account, MetadataKeyWalletName).(string),
		Metadata: GetMetadata(account, MetadataKeyWalletCustomData).(map[string]any),
	}
}

func WithBalancesFromAccount(account interface {
	metadata.Owner
	GetBalances() map[string]int32
},
) WithBalances {
	return WithBalances{
		Wallet:   FromAccount(account),
		Balances: account.GetBalances(),
	}
}
