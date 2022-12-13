package handlers

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (m *MainHandler) DebitWalletHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "wallet_id")

	debit := wallet.Debit{
		WalletID: id,
		// @todo: parse amount from request
		Amount: core.Monetary{
			Asset:  "USD/2",
			Amount: core.NewMonetaryInt(100),
		},
		// @todo: parse pending from request
		Pending: true,
	}

	hold, err := m.funding.Debit(r.Context(), debit)

	if err != nil {
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, map[string]interface{}{
			// @todo: return a proper error
			"error": err.Error(),
		})
		return
	}

	if hold == nil {
		render.Status(r, http.StatusNoContent)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, map[string]interface{}{
		"hold": hold,
	})
}
