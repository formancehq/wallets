package handlers

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/render"
)

func (m *MainHandler) DebitWalletHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("wallet_id")

	err := m.funding.Debit(r.Context(), wallet.Debit{
		WalletID: id,
		// @todo: parse amount from request
		Amount: core.Monetary{},
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
