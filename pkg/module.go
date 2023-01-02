package pkg

import (
	sharedapi "github.com/formancehq/go-libs/api"
	"github.com/formancehq/wallets/pkg/api"
	"github.com/formancehq/wallets/pkg/wallet"
	"go.uber.org/fx"
)

func Module(ledgerName, chartPrefix string, serviceInfo sharedapi.ServiceInfo) fx.Option {
	return fx.Module(
		"wallets-core",
		wallet.Module(ledgerName, chartPrefix),
		api.Module(serviceInfo),
	)
}
