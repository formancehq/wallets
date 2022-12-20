package wallet

import (
	"context"

	"github.com/formancehq/go-libs/sharedlogging"
	"github.com/formancehq/wallets/pkg/core"
)

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

type WalletData struct {
	Metadata core.Metadata `json:"metadata"`
}

func (r *Repository) CreateWallet(ctx context.Context, data *WalletData) (*core.Wallet, error) {

	wallet := core.NewWallet(data.Metadata)

	if err := r.client.AddMetadataToAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(wallet.ID),
		wallet.LedgerMetadata(),
	); err != nil {
		sharedlogging.GetLogger(ctx).Error(err)
		return nil, ErrLedgerInternal
	}

	return &wallet, nil
}

func (r *Repository) UpdateWallet(ctx context.Context, id string, data *WalletData) error {
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
		return ErrLedgerInternal
	}

	return nil
}

// @todo: add pagination.
func (r *Repository) ListWallets(ctx context.Context) ([]core.Wallet, error) {
	wallets := []core.Wallet{}

	accounts, err := r.client.ListAccountsWithMetadata(ctx, r.ledgerName, map[string]interface{}{
		core.MetadataKeySpecType: core.PrimaryWallet,
	})
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		wallet := core.Wallet{
			ID:       account.Metadata[core.MetadataKeyWalletID].(string),
			Balances: make(map[string]core.Monetary),
			Metadata: core.Metadata{},
		}
		wallets = append(wallets, wallet)
	}

	return wallets, nil
}

func (r *Repository) GetWallet(ctx context.Context, id string) (*core.Wallet, error) {
	account, err := r.client.GetAccount(
		ctx,
		r.ledgerName,
		r.chart.GetMainAccount(id),
	)
	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, ErrLedgerInternal
	}

	if account.Metadata[core.MetadataKeySpecType] != core.PrimaryWallet {
		return nil, ErrWalletNotFound
	}

	return &core.Wallet{
		ID:       id,
		Metadata: account.Metadata[core.MetadataKeyWalletCustomData].(map[string]any),
		// @todo: get balances from subaccounts
		Balances: make(map[string]core.Monetary),
	}, nil
}

func (r *Repository) ListHolds(ctx context.Context, walletID string) ([]core.DebitHold, error) {
	holds := make([]core.DebitHold, 0)

	filter := core.Metadata{
		core.MetadataKeySpecType:     core.HoldWallet,
		core.MetadataKeyHoldWalletID: walletID,
	}

	accounts, err := r.client.ListAccountsWithMetadata(ctx, r.ledgerName, filter)
	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, err
	}

	for _, account := range accounts {
		holds = append(holds, core.DebitHold{
			ID:          account.Metadata[core.MetadataKeyHoldID].(string),
			WalletID:    account.Metadata[core.MetadataKeyHoldWalletID].(string),
			Destination: account.Metadata["destination"].(map[string]any)["value"].(string),
		})
	}

	return holds, nil
}

func (r *Repository) GetHold(ctx context.Context, id string) (*core.DebitHold, error) {
	account, err := r.client.GetAccount(ctx, r.ledgerName, r.chart.GetHoldAccount(id))
	if err != nil {
		// @todo: log error properly in addition to returning it
		return nil, err
	}

	hold := core.DebitHoldFromLedgerAccount(*account)

	return &hold, nil
}
