package wallet

import (
	"context"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/pkg/errors"
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

// @todo: add pagination.
func (r *Repository) ListWallets(ctx context.Context) ([]core.Wallet, error) {
	wallets := make([]core.Wallet, 0)

	accounts, err := r.client.ListAccountsWithMetadata(ctx, r.ledgerName, map[string]interface{}{
		core.MetadataKeySpecType: core.PrimaryWallet,
	})
	if err != nil {
		return nil, err
	}

	for _, account := range accounts {
		//nolint:scopelint
		wallets = append(wallets, core.WalletFromAccount(&account))
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
		return nil, errors.Wrap(err, "getting account")
	}

	if account.Metadata[core.MetadataKeySpecType] != core.PrimaryWallet {
		return nil, ErrWalletNotFound
	}

	w := core.WalletFromAccount(account)

	return &w, nil
}

func (r *Repository) ListHolds(ctx context.Context, walletID string) ([]core.DebitHold, error) {
	holds := make([]core.DebitHold, 0)

	filter := core.Metadata{
		core.MetadataKeySpecType:     core.HoldWallet,
		core.MetadataKeyHoldWalletID: walletID,
	}

	accounts, err := r.client.ListAccountsWithMetadata(ctx, r.ledgerName, filter)
	if err != nil {
		return nil, errors.Wrap(err, "listing accounts")
	}

	for _, account := range accounts {
		account := account // Create a scoped variable
		holds = append(holds, core.DebitHoldFromLedgerAccount(&account))
	}

	return holds, nil
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
