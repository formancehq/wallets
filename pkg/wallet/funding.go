package wallet

import (
	"context"
	"strings"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet/numscript"
	"github.com/pkg/errors"
)

const (
	DefaultCreditSource = "world"
	DefaultDebitDest    = "world"
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
	WalletID    string        `json:"walletID"`
	Amount      core.Monetary `json:"amount"`
	Destination string        `json:"destination"`
	Reference   string        `json:"reference"`
	Pending     bool          `json:"pending"`
}

type ConfirmHold struct {
	HoldID    string `json:"holdID"`
	Amount    core.Monetary
	Reference string
}

type VoidHold struct {
	HoldID string `json:"holdID"`
}

type Credit struct {
	WalletID  string        `json:"walletID"`
	Source    string        `json:"source"`
	Amount    core.Monetary `json:"amount"`
	Reference string        `json:"reference"`
}

func (s *FundingService) Debit(ctx context.Context, debit Debit) (*core.DebitHold, error) {
	dest := DefaultDebitDest
	if debit.Destination != "" {
		dest = debit.Destination
	}

	var hold *core.DebitHold
	if debit.Pending {
		newHold := core.NewDebitHold(debit.WalletID, dest)
		hold = &newHold

		holdAccount := s.chart.GetHoldAccount(hold.ID)
		if err := s.client.AddMetadataToAccount(ctx, s.ledgerName, holdAccount,
			newHold.LedgerMetadata(s.chart)); err != nil {
			return nil, errors.Wrap(err, "adding metadata to account")
		}

		dest = holdAccount
	}

	transaction := sdk.TransactionData{
		Postings: []sdk.Posting{
			{
				// @todo: upgrade this to proper int after sdk is updated
				Amount:      int32(debit.Amount.Amount.Uint64()),
				Asset:       debit.Amount.Asset,
				Source:      s.chart.GetMainAccount(debit.WalletID),
				Destination: dest,
			},
		},
	}

	if debit.Reference != "" {
		transaction.Reference = &debit.Reference
	}

	if err := s.client.CreateTransaction(ctx, s.ledgerName, transaction); err != nil {
		return nil, errors.Wrap(err, "creating transaction")
	}

	return hold, nil
}

func (s *FundingService) ConfirmHold(ctx context.Context, debit ConfirmHold) error {
	holdAccount := s.chart.GetHoldAccount(debit.HoldID)

	account, err := s.client.GetAccount(ctx, s.ledgerName, holdAccount)
	if err != nil {
		return errors.Wrap(err, "getting account")
	}

	mType, ok := account.Metadata[core.MetadataKeySpecType]
	if !ok {
		return ErrHoldNotFound
	}

	if mType, ok := mType.(string); !ok || mType != core.HoldWallet {
		return NewErrMismatchType(core.HoldWallet, mType)
	}

	var asset string
	for key := range *account.Balances {
		asset = key
		break
	}

	if err := s.client.RunScript(
		ctx,
		s.ledgerName,
		sdk.Script{
			Plain: strings.ReplaceAll(numscript.ConfirmHold, "ASSET", asset),
			Vars: map[string]interface{}{
				"hold": s.chart.GetHoldAccount(debit.HoldID),
			},
		},
	); err != nil {
		return errors.Wrap(err, "running script")
	}

	return nil
}

func (s *FundingService) VoidHold(ctx context.Context, void VoidHold) error {
	account, err := s.client.GetAccount(ctx, s.ledgerName, s.chart.GetHoldAccount(void.HoldID))
	if err != nil {
		return errors.Wrap(err, "getting account")
	}

	var asset string
	for key := range *account.Balances {
		asset = key
		break
	}

	if err := s.client.RunScript(
		ctx,
		s.ledgerName,
		sdk.Script{
			Plain: strings.ReplaceAll(numscript.CancelHold, "ASSET", asset),
			Vars: map[string]interface{}{
				"hold": s.chart.GetHoldAccount(void.HoldID),
			},
		},
	); err != nil {
		return errors.Wrap(err, "running script")
	}

	return nil
}

func (s *FundingService) Credit(ctx context.Context, credit Credit) error {
	source := DefaultCreditSource
	if credit.Source != "" {
		source = credit.Source
	}

	transaction := sdk.TransactionData{
		Postings: []sdk.Posting{
			{
				// @todo: upgrade this to proper int after sdk is updated
				Amount:      int32(credit.Amount.Amount.Uint64()),
				Asset:       credit.Amount.Asset,
				Source:      source,
				Destination: s.chart.GetMainAccount(credit.WalletID),
			},
		},
	}

	if credit.Reference != "" {
		transaction.Reference = &credit.Reference
	}

	if err := s.client.CreateTransaction(ctx, s.ledgerName, transaction); err != nil {
		return errors.Wrap(err, "creating transaction")
	}

	return nil
}
