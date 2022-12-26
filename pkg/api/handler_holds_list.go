package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
)

func (m *MainHandler) ListHoldsHandler(w http.ResponseWriter, r *http.Request) {
	query := readPaginatedRequest(r, func(r *http.Request) wallet.ListHolds {
		return wallet.ListHolds{
			WalletID: chi.URLParam(r, "wallet_id"),
		}
	})

	holds, err := m.repository.ListHolds(r.Context(), query)
	if err != nil {
		internalError(w, r, err)
		return
	}

	cursorFromListResponse(w, query, holds)
}
