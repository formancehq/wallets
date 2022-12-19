package wallet

import (
	"context"
	"log"
	"strings"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet/numscript"
	"github.com/google/uuid"
)

const (
	DefaultCreditSource = "world"
	DefaultDebitDest    = "world"
)

type FundingService struct {
	client     *sdk.APIClient
	chart      *core.Chart
	ledgerName string
}

func NewFundingService(
	ledgerName string,
	client *sdk.APIClient,
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

func (s *FundingService) Debit(ctx context.Context, debit Debit) (*core.Hold, error) {
	dest := DefaultDebitDest
	if debit.Destination != "" {
		dest = debit.Destination
	}

	var hold *core.Hold
	if debit.Pending {
		hold = &core.Hold{
			ID:       uuid.NewString(),
			WalletID: debit.WalletID,
		}
		holdAccount := s.chart.GetHoldAccount(hold.ID)
		_, err := s.client.AccountsApi.
			AddMetadataToAccount(ctx, s.ledgerName, holdAccount).
			RequestBody(map[string]interface{}{
				//nolint:godox
				// TODO: Use defined namespace on ledger
				"spec/type":       "wallets.hold",
				"holds/wallet_id": debit.WalletID,
				"void_destination": map[string]interface{}{
					"type":  "account",
					"value": s.chart.GetMainAccount(debit.WalletID),
				},
				"destination": map[string]interface{}{
					"type":  "account",
					"value": dest,
				},
			}).
			Execute()
		if err != nil {
			// @todo: log error properly in addition to returning it
			log.Println(err)
			return nil, ErrLedgerInternal
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

	_, _, err := s.client.TransactionsApi.
		CreateTransaction(ctx, s.ledgerName).
		TransactionData(transaction).
		Execute()
	if err != nil {
		// @todo: log error properly in addition to returning it
		log.Println(err)
		return nil, ErrLedgerInternal
	}

	return hold, nil
}

func (s *FundingService) ConfirmHold(ctx context.Context, debit ConfirmHold) error {
	holdAccount := s.chart.GetHoldAccount(debit.HoldID)

	res, _, err := s.client.AccountsApi.
		GetAccount(ctx, s.ledgerName, holdAccount).
		Execute()
	if err != nil {
		// @todo: log error properly in addition to returning it
		log.Println(err)
		return ErrLedgerInternal
	}

	if res.Data.Metadata["spec/type"] != "wallets.hold" {
		// @todo: log error properly in addition to returning it
		return ErrLedgerInternal
	}

	var asset string
	for key := range *res.Data.Balances {
		asset = key
		break
	}

	script := strings.ReplaceAll(numscript.ConfirmHold, "ASSET", asset)

	_, _, err = s.client.ScriptApi.RunScript(
		ctx,
		s.ledgerName,
	).Script(sdk.Script{
		Plain: script,
		Vars: map[string]interface{}{
			"hold": s.chart.GetHoldAccount(debit.HoldID),
		},
	}).Execute()
	if err != nil {
		// @todo: log error properly in addition to returning it
		log.Println(err)
		return ErrLedgerInternal
	}

	return nil
}

func (s *FundingService) VoidHold(ctx context.Context, void VoidHold) error {
	res, _, err := s.client.AccountsApi.
		GetAccount(ctx, s.ledgerName, s.chart.GetHoldAccount(void.HoldID)).
		Execute()
	if err != nil {
		// @todo: log error properly in addition to returning it
		log.Println(err)
		return ErrLedgerInternal
	}

	var asset string
	for key := range *res.Data.Balances {
		asset = key
		break
	}

	script := strings.ReplaceAll(numscript.CancelHold, "ASSET", asset)

	_, _, err = s.client.ScriptApi.RunScript(
		ctx,
		s.ledgerName,
	).Script(sdk.Script{
		Plain: script,
		Vars: map[string]interface{}{
			"hold": s.chart.GetHoldAccount(void.HoldID),
		},
	}).Execute()

	if err != nil {
		// @todo: log error properly in addition to returning it
		log.Println(err)
		return ErrLedgerInternal
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

	_, _, err := s.client.TransactionsApi.
		CreateTransaction(ctx, s.ledgerName).
		TransactionData(transaction).
		Execute()
	if err != nil {
		// @todo: log error properly in addition to returning it
		return ErrLedgerInternal
	}

	return nil
}
