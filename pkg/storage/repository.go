package storage

import (
	"context"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/google/uuid"
)

type Repository struct {
	ledgerName string
	chart      *core.Chart
	client     *sdk.APIClient
}

func NewRepository(
	ledgerName string,
	client *sdk.APIClient,
	chart *core.Chart,
) *Repository {
	return &Repository{
		ledgerName: ledgerName,
		chart:      chart,
		client:     client,
	}
}

func (r *Repository) CreateWallet(ctx context.Context) (*core.Wallet, error) {
	id := uuid.NewString()

	_, err := r.client.AccountsApi.AddMetadataToAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(id),
	).RequestBody(map[string]interface{}{
		"spec/type": "wallets.wallet",
	}).Execute()

	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, InternalLedgerError
	}

	return &core.Wallet{
		ID: id,
	}, nil
}

func (r *Repository) ListWallets(ctx context.Context) ([]core.Wallet, error) {
	wallets := []core.Wallet{}

	res, _, err := r.client.AccountsApi.ListAccounts(ctx, r.ledgerName).Metadata(map[string]interface{}{
		"spec/type": "wallets.wallet",
	}).Execute()

	if err != nil {
		return nil, err
	}

	for _, account := range res.Cursor.Data {
		wallet := core.Wallet{
			ID: account.Address,
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

func (r *Repository) GetWallet(ctx context.Context, id string) (*core.Wallet, error) {
	wallet := &core.Wallet{}

	res, _, err := r.client.AccountsApi.GetAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(id),
	).Execute()

	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, InternalLedgerError
	}

	wallet.ID = res.Data.Address

	return wallet, nil
}

func (r *Repository) ListHolds(ctx context.Context, walletID string) ([]core.Hold, error) {
	holds := []core.Hold{}

	res, _, err := r.client.AccountsApi.ListAccounts(ctx, r.ledgerName).Metadata(map[string]interface{}{
		"spec/type": "wallets.hold",
	}).Execute()

	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, err
	}

	for _, account := range res.Cursor.Data {
		hold := core.Hold{
			ID:       account.Address,
			WalletID: account.Metadata["wallet"].(string),
		}
		holds = append(holds, hold)
	}

	return holds, nil
}

func (r *Repository) GetHold(ctx context.Context, id string) (*core.Hold, error) {
	hold := &core.Hold{}

	res, _, err := r.client.AccountsApi.GetAccount(
		ctx,
		r.ledgerName,
		r.chart.GetHoldAccount(id),
	).Execute()

	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, err
	}

	hold.ID = res.Data.Address
	hold.WalletID = res.Data.Metadata["wallet"].(string)

	return hold, nil
}
