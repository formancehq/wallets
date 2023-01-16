package wallet

import (
	"net/http"
	"regexp"

	"github.com/formancehq/go-libs/metadata"
	"github.com/go-chi/chi/v5"
)

var balanceNameRegex = regexp.MustCompile("[0-9A-Za-z_-]+")

type CreateBalance struct {
	WalletID string `json:"walletID"`
	Name     string `json:"name"`
}

func (c *CreateBalance) Validate() error {
	if !balanceNameRegex.MatchString(c.Name) {
		return ErrInvalidBalanceName
	}
	if c.Name == MainBalance {
		return ErrReservedBalanceName
	}
	return nil
}

func (c *CreateBalance) Bind(r *http.Request) error {
	c.WalletID = chi.URLParam(r, "walletID")
	return nil
}

type Balance struct {
	Name   string           `json:"name,omitempty"`
	Assets map[string]int32 `json:"assets,omitempty"`
}

func (b Balance) LedgerMetadata(walletID string) metadata.Metadata {
	return metadata.Metadata{
		MetadataKeyWalletID:      walletID,
		MetadataKeyWalletBalance: TrueValue,
		MetadataKeyBalanceName:   b.Name,
		MetadataKeyBalanceAsset:  b.Assets,
	}
}

func NewBalance(name string) Balance {
	return Balance{
		Name: name,
	}
}

func BalanceFromAccount(account Account) Balance {
	balance := Balance{}
	assets, ok := account.GetMetadata()[MetadataKeyBalanceAsset].(map[string]int32)
	if ok {
		balance.Assets = assets
	}
	name, ok := account.GetMetadata()[MetadataKeyBalanceName].(string)
	if ok {
		balance.Name = name
	}
	return balance
}

type ExpandedBalance struct {
	Balance
	Assets map[string]int32 `json:"assets"`
}

func ExpandedBalanceFromAccount(account interface {
	Account
	GetBalances() map[string]int32
},
) ExpandedBalance {
	expandedBalance := ExpandedBalance{
		Balance: BalanceFromAccount(account),
		Assets:  account.GetBalances(),
	}
	return expandedBalance
}
