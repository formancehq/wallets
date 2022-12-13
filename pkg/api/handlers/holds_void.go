package handlers

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (m *MainHandler) VoidHoldHandler(w http.ResponseWriter, r *http.Request) {
	err := m.funding.VoidHold(r.Context(), wallet.VoidHold{
		HoldID: chi.URLParam(r, "hold_id"),
	})
	if err != nil {
		render.Status(r, http.StatusUnprocessableEntity)
		render.JSON(w, r, map[string]string{
			// @todo: return a proper error
			"error": err.Error(),
		})
		return
	}

	render.NoContent(w, r)
}
