package handlers

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func (m *MainHandler) GetWalletHandler(w http.ResponseWriter, r *http.Request) {
	wallet, err := m.repository.GetWallet(r.Context(), chi.URLParam(r, "wallet_id"))
	if err != nil {
		switch err.Error() {
		case storage.ErrWalletNotFound.Error():
			render.Status(r, http.StatusNotFound)
		default:
			render.Status(r, http.StatusInternalServerError)
		}
		// @todo: return a proper error from go-libs
		render.JSON(w, r, map[string]string{
			"error": err.Error(),
		})
		return
	}

	render.JSON(w, r, wallet)
}
