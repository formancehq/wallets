package wallet

import (
	"context"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/pkg/errors"
)

type ListResponse[T any] struct {
	Data           []T
	Next, Previous string
	HasMore        bool
}

func newListResponse[T any](cursor *sdk.ListAccounts200ResponseCursor, mapper func(account sdk.Account) T) *ListResponse[T] {
	ret := make([]T, 0)
	for _, item := range cursor.Data {
		ret = append(ret, mapper(item))
	}

	return &ListResponse[T]{
		Data: ret,
		Next: func() string {
			if cursor.Next == nil {
				return ""
			}
			return *cursor.Next
		}(),
		Previous: func() string {
			if cursor.Previous == nil {
				return ""
			}
			return *cursor.Previous
		}(),
		HasMore: *cursor.HasMore,
	}
}

type ListQuery[T any] struct {
	Payload         T
	Limit           int
	PaginationToken string
}

type ListHolds struct {
	WalletID string
}

type Repository struct {
	ledgerName string
	chart      *core.Chart
	client     Ledger
}

func NewRepository(
	ledgerName string,
	client Ledger,
	chart *core.Chart,
) *Repository {
	return &Repository{
		ledgerName: ledgerName,
		chart:      chart,
		client:     client,
	}
}

type Data struct {
	Metadata core.Metadata `json:"metadata"`
	Name     string        `json:"name"`
}

func (r *Repository) CreateWallet(ctx context.Context, data *Data) (*core.Wallet, error) {
	wallet := core.NewWallet(data.Name, data.Metadata)

	if err := r.client.AddMetadataToAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(wallet.ID),
		wallet.LedgerMetadata(),
	); err != nil {
		return nil, errors.Wrap(err, "adding metadata to account")
	}

	return &wallet, nil
}

func (r *Repository) UpdateWallet(ctx context.Context, id string, data *Data) error {
	meta := core.Metadata{}
	custom := core.Metadata{}

	account, err := r.client.GetAccount(ctx, r.ledgerName, r.chart.GetMainAccount(id))
	if err != nil {
		return ErrWalletNotFound
	}
	if account.Metadata[core.MetadataKeySpecType] != core.PrimaryWallet {
		return ErrWalletNotFound
	}

	for k, v := range account.Metadata {
		if k != core.MetadataKeyWalletCustomData {
			continue
		}
		for k, v := range v.(map[string]interface{}) {
			custom[k] = v
		}
	}
	for k, v := range data.Metadata {
		custom[k] = v
	}
	meta[core.MetadataKeyWalletCustomData] = custom

	if err := r.client.AddMetadataToAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(id),
		meta,
	); err != nil {
		return errors.Wrap(err, "adding metadata to account")
	}

	return nil
}

func (r *Repository) ListWallets(ctx context.Context, query ListQuery[struct{}]) (*ListResponse[core.Wallet], error) {
	var (
		response *sdk.ListAccounts200ResponseCursor
		err      error
	)
	if query.PaginationToken == "" {
		response, err = r.client.ListAccounts(ctx, r.ledgerName, ListAccountQuery{
			Limit: query.Limit,
			Metadata: map[string]interface{}{
				core.MetadataKeySpecType: core.PrimaryWallet,
			},
		})
	} else {
		response, err = r.client.ListAccounts(ctx, r.ledgerName, ListAccountQuery{
			PaginationToken: query.PaginationToken,
		})
	}
	if err != nil {
		return nil, err
	}

	return newListResponse(response, func(account sdk.Account) core.Wallet {
		return core.WalletFromAccount(&account)
	}), nil
}

func (r *Repository) GetWallet(ctx context.Context, id string) (*core.WalletWithBalances, error) {
	account, err := r.client.GetAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(id),
	)
	if err != nil {
		return nil, errors.Wrap(err, "getting account")
	}

	if account.Metadata[core.MetadataKeySpecType] != core.PrimaryWallet {
		return nil, ErrWalletNotFound
	}

	w := core.WalletWithBalancesFromAccount(account)

	return &w, nil
}

func (r *Repository) ListHolds(ctx context.Context, query ListQuery[ListHolds]) (*ListResponse[core.DebitHold], error) {
	var (
		response *sdk.ListAccounts200ResponseCursor
		err      error
	)
	if query.PaginationToken == "" {
		response, err = r.client.ListAccounts(ctx, r.ledgerName, ListAccountQuery{
			Limit: query.Limit,
			Metadata: core.Metadata{
				core.MetadataKeySpecType:     core.HoldWallet,
				core.MetadataKeyHoldWalletID: query.Payload.WalletID,
			},
		})
	} else {
		response, err = r.client.ListAccounts(ctx, r.ledgerName, ListAccountQuery{
			PaginationToken: query.PaginationToken,
		})
	}
	if err != nil {
		return nil, errors.Wrap(err, "listing accounts")
	}

	return newListResponse(response, func(account sdk.Account) core.DebitHold {
		return core.DebitHoldFromLedgerAccount(&account)
	}), nil
}

func (r *Repository) GetHold(ctx context.Context, id string) (*core.DebitHold, error) {
	account, err := r.client.GetAccount(ctx, r.ledgerName, r.chart.GetHoldAccount(id))
	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, err
	}

	hold := core.DebitHoldFromLedgerAccount(account)

	return &hold, nil
}
