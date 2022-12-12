package handlers

import (
	"net/http"

	"github.com/go-chi/render"
)

func (m *MainHandler) ListWalletsHandler(w http.ResponseWriter, r *http.Request) {
	wallets, err := m.repository.ListWallets(r.Context())

	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{"error": err.Error()})
		return
	}

	render.JSON(w, r, wallets)
}
