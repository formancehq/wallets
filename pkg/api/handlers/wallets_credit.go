package handlers

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type CreditWalletRequest struct {
	Amount core.Monetary `json:"amount"`
}

func (c *CreditWalletRequest) Bind(r *http.Request) error {
	return nil
}

func (m *MainHandler) CreditWalletHandler(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "wallet_id")
	data := &CreditWalletRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": err.Error(),
		})
		return
	}

	credit := wallet.Credit{
		WalletID: id,
		Amount:   data.Amount,
	}

	err := m.funding.Credit(r.Context(), credit)
	if err != nil {
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, map[string]interface{}{
			// @todo: return a proper error
			"error": err.Error(),
		})
		return
	}

	render.NoContent(w, r)
}
