package api

import (
	"errors"
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
)

func (m *MainHandler) ConfirmHoldHandler(w http.ResponseWriter, r *http.Request) {
	err := m.funding.ConfirmHold(r.Context(), wallet.ConfirmHold{
		HoldID: chi.URLParam(r, "hold_id"),
	})
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrHoldNotFound):
			notFound(w)
		default:
			internalError(w, r, err)
		}
		return
	}

	noContent(w)
}
