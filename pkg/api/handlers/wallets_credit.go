package handlers

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (m *MainHandler) CreditWalletHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "wallet_id")

	err := m.funding.Credit(r.Context(), wallet.Credit{
		WalletID: id,
		Amount: core.Monetary{
			// @todo: parse amount from request
			Amount: core.NewMonetaryInt(100),
			Asset:  "USD/2",
		},
	})
	if err != nil {
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, map[string]interface{}{
			// @todo: return a proper error
			"error": err.Error(),
		})
		return
	}

	render.Status(r, http.StatusNoContent)
}
