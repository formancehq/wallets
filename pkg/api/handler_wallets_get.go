package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (m *MainHandler) GetWalletHandler(wr http.ResponseWriter, r *http.Request) {
	w, err := m.repository.GetWallet(r.Context(), chi.URLParam(r, "wallet_id"))
	if err != nil {
		switch err.Error() {
		case wallet.ErrWalletNotFound.Error():
			render.Status(r, http.StatusNotFound)
		default:
			render.Status(r, http.StatusInternalServerError)
		}
		// @todo: return a proper error from go-libs
		render.JSON(wr, r, map[string]string{
			"error": err.Error(),
		})
		return
	}

	render.JSON(wr, r, w)
}
