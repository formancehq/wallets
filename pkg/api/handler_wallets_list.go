package api

import (
	"net/http"

	"github.com/go-chi/render"
)

func (m *MainHandler) ListWalletsHandler(w http.ResponseWriter, r *http.Request) {
	wallets, err := m.repository.ListWallets(r.Context())
	if err != nil {
		internalError(w, r, err)
		return
	}

	render.JSON(w, r, wallets)
}
