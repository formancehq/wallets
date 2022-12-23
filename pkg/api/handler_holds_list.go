package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
)

func (m *MainHandler) ListHoldsHandler(w http.ResponseWriter, r *http.Request) {
	query := wallet.ListQuery[wallet.ListHolds]{
		Payload: wallet.ListHolds{
			WalletID: chi.URLParam(r, "wallet_id"),
		},
		Limit:           parseLimit(r),
		PaginationToken: parsePaginationToken(r),
	}

	holds, err := m.repository.ListHolds(r.Context(), query)
	if err != nil {
		internalError(w, r, err)
		return
	}

	cursorFromListResponse(w, query, holds)
}
