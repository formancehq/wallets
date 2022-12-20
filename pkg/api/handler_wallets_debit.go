package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type DebitWalletRequest struct {
	Amount  core.Monetary `json:"amount"`
	Pending bool          `json:"pending"`
}

func (c *DebitWalletRequest) Bind(r *http.Request) error {
	return nil
}

func (m *MainHandler) DebitWalletHandler(w http.ResponseWriter, r *http.Request) {
	data := &DebitWalletRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": err.Error(),
		})
		return
	}

	id := chi.URLParam(r, "wallet_id")

	debit := wallet.Debit{
		WalletID: id,
		Amount:   data.Amount,
		Pending:  data.Pending,
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
		render.NoContent(w, r)
		return
	}

	render.Status(r, http.StatusCreated)
	render.JSON(w, r, hold)
}
