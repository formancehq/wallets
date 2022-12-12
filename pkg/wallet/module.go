package wallet

import (
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/storage"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"wallet",
		fx.Provide(fx.Annotate(
			NewFundingService,
			fx.ParamTags(`name:"ledger-name"`),
		)),
		fx.Provide(fx.Annotate(
			storage.NewRepository,
			fx.ParamTags(`name:"ledger-name"`),
		)),
		fx.Provide(fx.Annotate(
			func(prefix string) *core.Chart {
				return core.NewChart(prefix)
			},
			fx.ParamTags(`name:"chart-prefix"`),
		)),
		// @todo: replace this with configureable value
		fx.Provide(
			fx.Annotate(func() string {
				return "wallets-001"
			}, fx.ResultTags(`name:"ledger-name"`)),
		),
		// @todo: replace this with configureable value
		fx.Provide(
			fx.Annotate(func() string {
				return ""
			}, fx.ResultTags(`name:"chart-prefix"`)),
		),
	)
}
