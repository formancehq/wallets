package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/formancehq/go-libs/v3/service"

	sharedapi "github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/auth"
	sharedhealth "github.com/formancehq/go-libs/v3/health"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/go-chi/chi/v5/middleware"
)

func NewRouter(
	manager *wallet.Manager,
	healthController *sharedhealth.HealthController,
	serviceInfo sharedapi.ServiceInfo,
	authenticator auth.Authenticator,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			handler.ServeHTTP(w, r)
		})
	})

	r.Get("/_healthcheck", healthController.Check)
	r.Get("/_info", sharedapi.InfoHandler(serviceInfo))
	r.Group(func(r chi.Router) {
		r.Use(auth.Middleware(authenticator))
		r.Use(service.OTLPMiddleware("wallets", serviceInfo.Debug))
		r.Use(middleware.AllowContentType("application/json"))

		main := NewMainHandler(manager)

		r.Route("/wallets", func(r chi.Router) {
			r.Get("/", main.listWalletsHandler)
			r.Post("/", main.createWalletHandler)
			r.Route("/{walletID}", func(r chi.Router) {
				r.Get("/summary", main.walletSummaryHandler)
				r.Get("/", main.getWalletHandler)
				r.Patch("/", main.patchWalletHandler)
				r.Post("/debit", main.debitWalletHandler)
				r.Post("/credit", main.creditWalletHandler)
				r.Route("/balances", func(r chi.Router) {
					r.Get("/", main.listBalancesHandler)
					r.Post("/", main.createBalanceHandler)
					r.Get("/{balanceName}", main.getBalanceHandler)
				})
			})
		})
		r.Route("/transactions", func(r chi.Router) {
			r.Get("/", main.listTransactions)
		})
		r.Route("/holds", func(r chi.Router) {
			r.Get("/", main.listHoldsHandler)
			r.Route("/{holdID}", func(r chi.Router) {
				r.Get("/", main.getHoldHandler)
				r.Post("/confirm", main.confirmHoldHandler)
				r.Post("/void", main.voidHoldHandler)
			})
		})
	})

	return r
}
