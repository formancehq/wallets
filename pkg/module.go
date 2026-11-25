package wallet

import (
	"context"
	"net/http"

	"go.uber.org/fx"
)

func Module(ledgerURL string, ledgerName, chartPrefix string) fx.Option {
	return ModuleWithLedgerOptions(context.Background(), ledgerURL, ledgerName, chartPrefix, LedgerOptions{})
}

func ModuleWithLedgerOptions(ctx context.Context, stackURL, ledgerName, chartPrefix string, options LedgerOptions) fx.Option {
	return fx.Module(
		"wallet",
		fx.Provide(func(httpClient *http.Client) (Ledger, error) {
			return selectLedger(ctx, httpClient, stackURL, options)
		}),
		fx.Provide(func() *Chart {
			return NewChart(chartPrefix)
		}),
		fx.Provide(func(ledger Ledger, chart *Chart) *Manager {
			return NewManager(ledgerName, ledger, chart)
		}),
		fx.Invoke(func(manager *Manager, lc fx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: manager.Init,
			})
		}),
	)
}
