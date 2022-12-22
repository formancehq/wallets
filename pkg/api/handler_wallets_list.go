package api

import (
	"net/http"
)

func (m *MainHandler) ListWalletsHandler(w http.ResponseWriter, r *http.Request) {
	wallets, err := m.repository.ListWallets(r.Context())
	if err != nil {
		internalError(w, r, err)
		return
	}

	ok(w, wallets)
}
