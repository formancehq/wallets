package api

import (
	"github.com/formancehq/wallets/pkg/api/handlers"
	"github.com/formancehq/wallets/pkg/storage"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
)

func NewRouter(
	funding *wallet.FundingService,
	repository *storage.Repository,
) *chi.Mux {
	r := chi.NewRouter()
	main := handlers.NewMainHandler(funding, repository)
	r.Route("/wallets", func(r chi.Router) {
		r.Get("/", main.ListWalletsHandler)
		r.Post("/", main.CreateWalletHandler)
		r.Route("/{wallet_id}", func(r chi.Router) {
			r.Get("/", main.GetWalletHandler)
			r.Post("/debit", main.DebitWalletHandler)
			r.Post("/credit", main.CreditWalletHandler)

			r.Route("/holds", func(r chi.Router) {
				r.Post("/", main.GetHoldHandler)
				r.Route("/{hold_id}", func(r chi.Router) {
					r.Get("/", main.ListHoldsHandler)
					r.Post("/confirm", main.ConfirmHoldHandler)
					r.Post("/void", main.VoidHoldHandler)
				})
			})
		})
	})

	return r
}
