package wallet

import (
	"context"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
)

type ListAccountQuery struct {
	PaginationToken string
	Limit           int
	Metadata        map[string]any
}

type Ledger interface {
	AddMetadataToAccount(ctx context.Context, ledger, account string, metadata core.Metadata) error
	GetAccount(ctx context.Context, ledger, account string) (*sdk.AccountWithVolumesAndBalances, error)
	ListAccounts(ctx context.Context, ledger string, query ListAccountQuery) (*sdk.ListAccounts200ResponseCursor, error)
	CreateTransaction(ctx context.Context, ledger string, transaction sdk.TransactionData) error
	RunScript(ctx context.Context, ledger string, script sdk.Script) error
}

type DefaultLedger struct {
	client *sdk.APIClient
}

func (d DefaultLedger) AddMetadataToAccount(ctx context.Context, ledger, account string, metadata core.Metadata) error {
	_, err := d.client.AccountsApi.AddMetadataToAccount(ctx, ledger, account).RequestBody(metadata).Execute()
	return err
}

func (d DefaultLedger) GetAccount(ctx context.Context, ledger, account string) (*sdk.AccountWithVolumesAndBalances, error) {
	ret, _, err := d.client.AccountsApi.GetAccount(ctx, ledger, account).Execute()
	return &ret.Data, err
}

func (d DefaultLedger) ListAccounts(ctx context.Context, ledger string, query ListAccountQuery) (*sdk.ListAccounts200ResponseCursor, error) {
	var (
		ret *sdk.ListAccounts200Response
		err error
	)
	if query.PaginationToken == "" {
		ret, _, err = d.client.AccountsApi.ListAccounts(ctx, ledger).
			Metadata(query.Metadata).
			PageSize(int32(query.Limit)).
			Execute()
	} else {
		ret, _, err = d.client.AccountsApi.ListAccounts(ctx, ledger).
			PaginationToken(query.PaginationToken).
			Execute()
	}
	if err != nil {
		return nil, err
	}

	return &ret.Cursor, nil
}

func (d DefaultLedger) CreateTransaction(ctx context.Context, ledger string, transaction sdk.TransactionData) error {
	_, _, err := d.client.TransactionsApi.CreateTransaction(ctx, ledger).TransactionData(transaction).Execute()
	return err
}

func (d DefaultLedger) RunScript(ctx context.Context, ledger string, script sdk.Script) error {
	_, _, err := d.client.ScriptApi.RunScript(ctx, ledger).Script(script).Execute()
	return err
}

var _ Ledger = &DefaultLedger{}

func NewDefaultLedger(client *sdk.APIClient) *DefaultLedger {
	return &DefaultLedger{
		client: client,
	}
}
