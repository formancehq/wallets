package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/fx"
)

func Module() fx.Option {
	return fx.Module(
		"api",
		fx.Provide(NewRouter),
		fx.Invoke(func(lc fx.Lifecycle, router *chi.Mux) {
			lc.Append(fx.Hook{
				OnStart: func(context context.Context) error {
					go func() {
						err := http.ListenAndServe(":8080", router)
						if err != nil && errors.Is(err, http.ErrServerClosed) {
							panic(err)
						}
					}()
					return nil
				},
			})
		}),
	)
}
