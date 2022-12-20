package api

import (
	"errors"
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (m *MainHandler) GetWalletHandler(wr http.ResponseWriter, r *http.Request) {
	w, err := m.repository.GetWallet(r.Context(), chi.URLParam(r, "wallet_id"))
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrWalletNotFound):
			notFound(wr)
		default:
			internalError(wr, r, err)
		}
		return
	}

	render.JSON(wr, r, w)
}
