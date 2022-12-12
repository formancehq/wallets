package handlers

import (
	"net/http"

	"github.com/go-chi/render"
)

func (m *MainHandler) GetWalletHandler(w http.ResponseWriter, r *http.Request) {
	wallet, err := m.repository.GetWallet(r.Context(), r.URL.Query().Get("wallet_id"))
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			// @todo: return a proper error
			"error": err.Error(),
		})
		return
	}

	render.JSON(w, r, wallet)
}
