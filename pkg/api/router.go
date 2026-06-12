package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-chi/chi/v5"

	"github.com/formancehq/go-libs/v5/pkg/transport/httpserver"
	"github.com/formancehq/go-libs/v5/pkg/audit/httpaudit"

	sharedapi "github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/go-libs/v5/pkg/authn/jwt"
	sharedhealth "github.com/formancehq/go-libs/v5/pkg/service/health"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/go-chi/chi/v5/middleware"
)

// maxRequestBodyBytes caps the size of request bodies the service will read.
// The JSON payloads handled here are small; this protects the service (and the
// audit middleware, which buffers the whole body) from memory-exhaustion DoS.
const maxRequestBodyBytes = 1 << 20 // 1 MiB

// limitRequestBody reads the request body up to maxRequestBodyBytes and rejects
// anything larger with 413. It buffers the (bounded) body and resets r.Body so
// the audit middleware and the handlers can still read it.
//
// We do this instead of http.MaxBytesReader because the audit middleware reads
// the body with io.ReadAll *before* the handler and turns any non-EOF error
// (including MaxBytesReader's overflow error) into a 500 — so an oversized body
// would surface as "500 failed to read request body" instead of 413.
func limitRequestBody(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			buf, err := io.ReadAll(io.LimitReader(r.Body, maxRequestBodyBytes+1))
			_ = r.Body.Close()
			if err != nil {
				w.WriteHeader(http.StatusBadRequest)
				_ = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
					ErrorCode:    ErrorCodeValidation,
					ErrorMessage: "failed to read request body",
				})
				return
			}
			if int64(len(buf)) > maxRequestBodyBytes {
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_ = json.NewEncoder(w).Encode(sharedapi.ErrorResponse{
					ErrorCode:    "REQUEST_TOO_LARGE",
					ErrorMessage: "request body too large",
				})
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(buf))
			r.ContentLength = int64(len(buf))
		}
		handler.ServeHTTP(w, r)
	})
}

func NewRouter(
	manager *wallet.Manager,
	healthController *sharedhealth.HealthController,
	serviceInfo sharedapi.ServiceInfo,
	authenticator jwt.Authenticator,
	publisher message.Publisher,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.Recoverer)
	r.Use(func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			handler.ServeHTTP(w, r)
		})
	})
	r.Use(limitRequestBody)
	r.Use(httpaudit.Middleware(publisher, "audit-events", "wallets", nil))

	r.Get("/_healthcheck", healthController.Check)
	r.Get("/_info", sharedapi.InfoHandler(serviceInfo))
	r.Group(func(r chi.Router) {
		r.Use(jwt.Middleware(authenticator))
		r.Use(httpserver.OTLPMiddleware("wallets", serviceInfo.Debug))
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
