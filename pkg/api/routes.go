package api

import (
	"github.com/formancehq/wallets/pkg/api/handlers"
	"github.com/formancehq/wallets/pkg/storage"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	funding *wallet.FundingService,
	repository *storage.Repository,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.Logger)

	main := handlers.NewMainHandler(funding, repository)

	r.Route("/wallets", func(r chi.Router) {
		r.Get("/", main.ListWalletsHandler)
		r.Post("/", main.CreateWalletHandler)
		r.Route("/{wallet_id}", func(r chi.Router) {
			r.Get("/", main.GetWalletHandler)
			r.Patch("/", main.PatchWalletHandler)
			r.Post("/debit", main.DebitWalletHandler)
			r.Post("/credit", main.CreditWalletHandler)
			r.Route("/holds", func(r chi.Router) {
				r.Get("/", main.ListHoldsHandler)
				r.Route("/{hold_id}", func(r chi.Router) {
					r.Get("/", main.GetHoldHandler)
					r.Post("/confirm", main.ConfirmHoldHandler)
					r.Post("/void", main.VoidHoldHandler)
				})
			})
		})
	})

	return r
}
