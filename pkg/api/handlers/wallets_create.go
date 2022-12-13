package handlers

import (
	"net/http"

	"github.com/go-chi/render"
)

func (m *MainHandler) CreateWalletHandler(w http.ResponseWriter, r *http.Request) {
	wallet, err := m.repository.CreateWallet(r.Context())
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
