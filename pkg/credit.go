package wallet

import (
	"math/big"
	"net/http"

	"github.com/formancehq/ledger/pkg/assets"

	"github.com/formancehq/go-libs/v5/pkg/types/time"

	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
)

var DefaultCreditSource = NewLedgerAccountSubject("world")

type CreditRequest struct {
	Amount    Monetary          `json:"amount"`
	Metadata  metadata.Metadata `json:"metadata"`
	Sources   Subjects          `json:"sources"`
	Reference string            `json:"reference"`
	Balance   string            `json:"balance"`
	Timestamp *time.Time        `json:"timestamp"`
}

func (c *CreditRequest) Bind(r *http.Request) error {
	return nil
}

func (c CreditRequest) Validate() error {
	if err := c.Sources.Validate(); err != nil {
		return err
	}
	if c.Amount.Amount.Cmp(big.NewInt(0)) < 0 {
		return ErrNegativeAmount
	}
	if !assets.IsValid(c.Amount.Asset) {
		return newErrInvalidAsset(c.Amount.Asset)
	}
	if c.Balance != "" && !balanceNameRegex.MatchString(c.Balance) {
		return newErrInvalidAccountName(c.Balance)
	}

	return nil
}

type Credit struct {
	CreditRequest
	WalletID string `json:"walletID"`
}

// Validate centralizes all credit validation, including the WalletID, so it
// cannot be bypassed by callers that only invoke Validate() (mirrors
// Debit.Validate). The WalletID is used as a chart segment, so it must be a
// single anchored segment with no ':' separator or Numscript metacharacters.
func (c Credit) Validate() error {
	if err := c.CreditRequest.Validate(); err != nil {
		return err
	}
	if !accountSegmentRegexp.MatchString(c.WalletID) {
		return newErrInvalidAccountName(c.WalletID)
	}
	return nil
}

func (c Credit) destinationAccount(chart *Chart) string {
	if c.Balance == "" {
		return chart.GetMainBalanceAccount(c.WalletID)
	}
	return chart.GetBalanceAccount(c.WalletID, c.Balance)
}
