package wallet

import (
	"context"
	"net/http"
	"strings"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/go-libs/metadata"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/pkg/errors"
)

var (
	DefaultCreditSource = Subject{
		Type:       SubjectTypeLedgerAccount,
		Identifier: "world",
	}
	DefaultDebitDest = Subject{
		Type:       SubjectTypeLedgerAccount,
		Identifier: "world",
	}
)

type DebitRequest struct {
	Amount      core.Monetary     `json:"amount"`
	Pending     bool              `json:"pending"`
	Metadata    metadata.Metadata `json:"metadata"`
	Description string            `json:"description"`
	Reference   string            `json:"reference"`
	Destination *Subject          `json:"destination"`
}

func (c *DebitRequest) Bind(r *http.Request) error {
	return nil
}

type Debit struct {
	DebitRequest
	WalletID string `json:"walletID"`
}

type ConfirmHold struct {
	HoldID    string `json:"holdID"`
	Amount    core.MonetaryInt
	Reference string
	Final     bool
}

type VoidHold struct {
	HoldID string `json:"holdID"`
}

type CreditRequest struct {
	Amount    core.Monetary     `json:"amount"`
	Metadata  metadata.Metadata `json:"metadata"`
	Sources   Subjects          `json:"sources"`
	Reference string            `json:"reference"`
}

func (c *CreditRequest) Bind(r *http.Request) error {
	return nil
}

func (c CreditRequest) Validate() error {
	if err := c.Sources.Validate(); err != nil {
		return err
	}
	return nil
}

type Credit struct {
	CreditRequest
	WalletID string `json:"walletID"`
}

type FundingService struct {
	client     Ledger
	chart      *core.Chart
	ledgerName string
}

func NewFundingService(
	ledgerName string,
	client Ledger,
	chart *core.Chart,
) *FundingService {
	return &FundingService{
		client:     client,
		chart:      chart,
		ledgerName: ledgerName,
	}
}

func (s *FundingService) Debit(ctx context.Context, debit Debit) (*core.DebitHold, error) {
	dest := DefaultDebitDest
	if debit.Destination != nil {
		dest = *debit.Destination
	}

	var hold *core.DebitHold
	if debit.Pending {
		md := debit.Metadata
		if md == nil {
			md = metadata.Metadata{}
		}
		newHold := core.NewDebitHold(debit.WalletID, dest.resolveAccount(s.chart),
			debit.Amount.Asset, debit.Description, md)
		hold = &newHold

		holdAccount := s.chart.GetHoldAccount(hold.ID)
		if err := s.client.AddMetadataToAccount(ctx, s.ledgerName, holdAccount,
			newHold.LedgerMetadata(s.chart)); err != nil {
			return nil, errors.Wrap(err, "adding metadata to account")
		}

		dest = Subject{
			Type:       SubjectTypeLedgerAccount,
			Identifier: holdAccount,
		}
	}

	customMetadata := debit.Metadata
	if customMetadata == nil {
		customMetadata = metadata.Metadata{}
	}

	script := sdk.Script{
		Plain: BuildDebitWalletScript(),
		Vars: map[string]interface{}{
			"source":      s.chart.GetMainAccount(debit.WalletID),
			"destination": dest.resolveAccount(s.chart),
			"amount": map[string]any{
				// @todo: upgrade this to proper int after sdk is updated
				"amount": debit.Amount.Amount.Uint64(),
				"asset":  debit.Amount.Asset,
			},
		},
		Metadata: core.WalletTransactionBaseMetadata().Merge(metadata.Metadata{
			core.MetadataKeyWalletCustomData: customMetadata,
		}),
		//nolint:godox
		// TODO: Add set account metadata for hold when released on ledger (v1.9)
	}
	if debit.Reference != "" {
		script.Reference = &debit.Reference
	}

	return hold, s.runScript(ctx, script)
}

func (s *FundingService) runScript(ctx context.Context, script sdk.Script) error {
	ret, err := s.client.RunScript(ctx, s.ledgerName, script)
	if err != nil {
		return err
	}
	if ret.ErrorCode == nil {
		return nil
	}
	if *ret.ErrorCode == string(sdk.INSUFFICIENT_FUND) {
		return ErrInsufficientFundError
	}
	if ret.ErrorMessage != nil {
		return errors.New(*ret.ErrorMessage)
	}
	return errors.New(*ret.ErrorCode)
}

func (s *FundingService) ConfirmHold(ctx context.Context, debit ConfirmHold) error {
	holdAccount := s.chart.GetHoldAccount(debit.HoldID)

	account, err := s.client.GetAccount(ctx, s.ledgerName, holdAccount)
	if err != nil {
		return errors.Wrap(err, "getting account")
	}

	if !core.IsHold(account) {
		return newErrMismatchType(core.HoldWallet, core.SpecType(account))
	}

	hold := core.ExpandedDebitHoldFromLedgerAccount(account)

	if hold.Remaining.Uint64() == 0 {
		return ErrClosedHold
	}

	amount := hold.Remaining.Uint64()
	if debit.Amount.Uint64() != 0 {
		if debit.Amount.Uint64() > amount {
			return ErrInsufficientFundError
		}
		amount = debit.Amount.Uint64()
	}

	return s.runScript(
		ctx,
		sdk.Script{
			Plain: BuildConfirmHoldScript(debit.Final, hold.Asset),
			Vars: map[string]interface{}{
				"hold": s.chart.GetHoldAccount(debit.HoldID),
				"amount": map[string]any{
					"amount": amount,
					"asset":  hold.Asset,
				},
			},
			Metadata: core.WalletTransactionBaseMetadata(),
		},
	)
}

func (s *FundingService) VoidHold(ctx context.Context, void VoidHold) error {
	account, err := s.client.GetAccount(ctx, s.ledgerName, s.chart.GetHoldAccount(void.HoldID))
	if err != nil {
		return errors.Wrap(err, "getting account")
	}

	hold := core.ExpandedDebitHoldFromLedgerAccount(account)
	if hold.Remaining.Uint64() == 0 {
		return ErrClosedHold
	}

	return s.runScript(ctx, sdk.Script{
		Plain: strings.ReplaceAll(CancelHoldScript, "ASSET", hold.Asset),
		Vars: map[string]interface{}{
			"hold": s.chart.GetHoldAccount(void.HoldID),
		},
		Metadata: core.WalletTransactionBaseMetadata(),
	})
}

func (s *FundingService) Credit(ctx context.Context, credit Credit) error {
	if credit.Metadata == nil {
		credit.Metadata = metadata.Metadata{}
	}
	if err := credit.Validate(); err != nil {
		return err
	}
	script := sdk.Script{
		Plain: BuildCreditWalletScript(credit.Sources.resolveAccounts(s.chart)...),
		Vars: map[string]interface{}{
			"destination": s.chart.GetMainAccount(credit.WalletID),
			"amount": map[string]any{
				// @todo: upgrade this to proper int after sdk is updated
				"amount": credit.Amount.Amount.Uint64(),
				"asset":  credit.Amount.Asset,
			},
		},
		Metadata: core.WalletTransactionBaseMetadata().Merge(metadata.Metadata{
			core.MetadataKeyWalletCustomData: credit.Metadata,
		}),
	}
	if credit.Reference != "" {
		script.Reference = &credit.Reference
	}

	return s.runScript(ctx, script)
}
