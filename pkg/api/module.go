package api

import (
	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/go-libs/v5/pkg/fx/servicefx"
	"github.com/formancehq/go-libs/v5/pkg/fx/transportfx"
	"github.com/formancehq/go-libs/v5/pkg/transport/httpserver"
	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

func Module(serviceInfo sharedapi.ServiceInfo, listen string) fx.Option {
	return fx.Module(
		"api",
		fx.Provide(NewRouter),
		fx.Supply(serviceInfo),
		servicefx.HealthModule(),
		fx.Invoke(func(lc fx.Lifecycle, router *chi.Mux) {
			lc.Append(transportfx.FXHook(httpserver.NewHook(router, httpserver.WithAddress(listen))))
		}),
	)
}
