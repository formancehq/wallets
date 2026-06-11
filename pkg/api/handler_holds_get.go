package api

import (
	"errors"
	"net/http"

	wallet "github.com/formancehq/wallets/pkg"
	"github.com/go-chi/chi/v5"
)

func (m *MainHandler) getHoldHandler(w http.ResponseWriter, r *http.Request) {
	hold, err := m.manager.GetHold(r.Context(), chi.URLParam(r, "holdID"))
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrHoldNotFound):
			notFound(w)
		default:
			internalError(w, r, err)
		}
		return
	}

	ok(w, hold)
}
