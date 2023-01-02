package wallet

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/go-libs/metadata"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/pkg/errors"
)

var (
	DefaultCreditSource = Subject{
		Type:       SourceTypeLedgerAccount,
		Identifier: "world",
	}
	DefaultDebitDest = "world"
)

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

type Debit struct {
	WalletID    string            `json:"walletID"`
	Amount      core.Monetary     `json:"amount"`
	Destination string            `json:"destination"`
	Reference   string            `json:"reference"`
	Pending     bool              `json:"pending"`
	Metadata    metadata.Metadata `json:"metadata"`
	Description string            `json:"description"`
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

const (
	SourceTypeLedgerAccount string = "ACCOUNT"
	SourceTypeWallet        string = "WALLET"
)

type Subject struct {
	Type       string `json:"type"`
	Identifier string `json:"identifier"`
}

func (s Subject) resolveAccount(chart *core.Chart) string {
	switch s.Type {
	case SourceTypeLedgerAccount:
		return s.Identifier
	case SourceTypeWallet:
		return chart.GetMainAccount(s.Identifier)
	}
	panic("unknown type")
}

func (s Subject) Validate() error {
	if s.Type != SourceTypeWallet && s.Type != SourceTypeLedgerAccount {
		return fmt.Errorf("unknown source type: %s", s.Type)
	}
	return nil
}

type Subjects []Subject

func (subjects Subjects) resolveAccounts(chart *core.Chart) []string {
	if len(subjects) == 0 {
		subjects = []Subject{DefaultCreditSource}
	}
	resolvedSources := make([]string, 0)
	for _, source := range subjects {
		resolvedSources = append(resolvedSources, source.resolveAccount(chart))
	}
	return resolvedSources
}

func (subjects Subjects) Validate() error {
	for _, source := range subjects {
		if err := source.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type CreditWalletRequest struct {
	Amount    core.Monetary     `json:"amount"`
	Metadata  metadata.Metadata `json:"metadata"`
	Sources   Subjects          `json:"sources"`
	Reference string            `json:"reference"`
}

func (c *CreditWalletRequest) Bind(r *http.Request) error {
	return nil
}

func (c CreditWalletRequest) Validate() error {
	if err := c.Sources.Validate(); err != nil {
		return err
	}
	return nil
}

type Credit struct {
	CreditWalletRequest
	WalletID string `json:"walletID"`
}

func (s *FundingService) Debit(ctx context.Context, debit Debit) (*core.DebitHold, error) {
	dest := DefaultDebitDest
	if debit.Destination != "" {
		dest = debit.Destination
	}

	var hold *core.DebitHold
	if debit.Pending {
		md := debit.Metadata
		if md == nil {
			md = metadata.Metadata{}
		}
		newHold := core.NewDebitHold(debit.WalletID, dest, debit.Amount.Asset, debit.Description, md)
		hold = &newHold

		holdAccount := s.chart.GetHoldAccount(hold.ID)
		if err := s.client.AddMetadataToAccount(ctx, s.ledgerName, holdAccount,
			newHold.LedgerMetadata(s.chart)); err != nil {
			return nil, errors.Wrap(err, "adding metadata to account")
		}

		dest = holdAccount
	}

	customMetadata := debit.Metadata
	if customMetadata == nil {
		customMetadata = metadata.Metadata{}
	}

	script := sdk.Script{
		Plain: BuildDebitWalletScript(),
		Vars: map[string]interface{}{
			"source":      s.chart.GetMainAccount(debit.WalletID),
			"destination": dest,
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
