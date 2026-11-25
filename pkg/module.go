package wallet

import (
	"context"
	"fmt"
	"net/http"

	"go.uber.org/fx"
)

func Module(ledgerURL string, ledgerName, chartPrefix string) fx.Option {
	return ModuleWithLedgerOptions(context.Background(), ledgerURL, ledgerName, chartPrefix, LedgerOptions{})
}

func ModuleWithLedgerOptions(ctx context.Context, stackURL, ledgerName, chartPrefix string, options LedgerOptions) fx.Option {
	return fx.Module(
		"wallet",
		fx.Provide(func(httpClient *http.Client, chart *Chart) (Ledger, error) {
			ledger, err := selectLedger(ctx, httpClient, stackURL, options)
			if err != nil {
				return nil, err
			}
			if v3, ok := ledger.(*ledgerV3); ok {
				plan, err := newV3Provisioning(chart, options.V3Provisioning, options.IndexedMetadataKeys)
				if err != nil {
					return nil, err
				}
				for _, address := range options.V3IndexStatusURLs {
					base, err := parseLedgerURL(address)
					if err != nil {
						return nil, fmt.Errorf("Ledger v3 index status URL: %w", err)
					}
					plan.indexStatusURLs = append(plan.indexStatusURLs, base)
				}
				v3.provision = plan
			}
			return ledger, nil
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
