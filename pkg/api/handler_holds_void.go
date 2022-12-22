package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
)

func (m *MainHandler) VoidHoldHandler(w http.ResponseWriter, r *http.Request) {
	err := m.funding.VoidHold(r.Context(), wallet.VoidHold{
		HoldID: chi.URLParam(r, "hold_id"),
	})
	if err != nil {
		internalError(w, r, err)
		return
	}

	noContent(w)
}
