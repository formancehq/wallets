package pkg

import (
	"github.com/formancehq/wallets/pkg/api"
	"github.com/formancehq/wallets/pkg/client"
	"github.com/formancehq/wallets/pkg/wallet"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"wallets-core",
		fx.Provide(client.NewStackClient),
		wallet.Module(),
		api.Module(),
	)
}
