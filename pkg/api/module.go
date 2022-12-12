package api

import (
	"context"
	"fmt"
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
					fmt.Println("Starting API...")
					go func() {
						err := http.ListenAndServe(":8082", router)
						if err != nil {
							return
						}
					}()
					return nil
				},
				OnStop: func(context.Context) error {
					fmt.Println("Stopping API...")
					return nil
				},
			})
		}),
	)
}
