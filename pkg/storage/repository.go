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

type WalletData struct {
	Metadata core.Metadata `json:"metadata"`
}

func (r *Repository) CreateWallet(ctx context.Context, data *WalletData) (*core.Wallet, error) {
	id := uuid.NewString()

	meta := core.Metadata{
		"spec/type":  "wallets.primary",
		"wallets/id": id,
	}

	custom := core.Metadata{}
	for k, v := range data.Metadata {
		custom[k] = v
	}
	meta["wallets/custom_data"] = custom

	_, err := r.client.AccountsApi.AddMetadataToAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(id),
	).RequestBody(meta).Execute()
	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, InternalLedgerError
	}

	return &core.Wallet{
		ID:       id,
		Metadata: custom,
		Balances: make(map[string]core.Monetary),
	}, nil
}

// @todo: add pagination
func (r *Repository) ListWallets(ctx context.Context) ([]core.Wallet, error) {
	wallets := []core.Wallet{}

	res, _, err := r.client.AccountsApi.ListAccounts(ctx, r.ledgerName).Metadata(map[string]interface{}{
		"spec/type": "wallets.primary",
	}).Execute()
	if err != nil {
		return nil, err
	}

	for _, account := range res.Cursor.Data {
		wallet := core.Wallet{
			ID:       account.Metadata["wallets/id"].(string),
			Balances: make(map[string]core.Monetary),
			Metadata: core.Metadata{},
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

func (r *Repository) GetWallet(ctx context.Context, id string) (*core.Wallet, error) {
	wallet := &core.Wallet{
		ID:       id,
		Metadata: core.Metadata{},
		// @todo: get balances from subaccounts
		Balances: make(map[string]core.Monetary),
	}

	res, _, err := r.client.AccountsApi.GetAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(id),
	).Execute()
	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, InternalLedgerError
	}

	if res.Data.Metadata["spec/type"] != "wallets.primary" {
		return nil, WalletNotFound
	}

	for k, v := range res.Data.Metadata {
		if k != "wallets/custom_data" {
			continue
		}
		for k2, v2 := range v.(map[string]interface{}) {
			wallet.Metadata[k2] = v2
		}
	}

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
