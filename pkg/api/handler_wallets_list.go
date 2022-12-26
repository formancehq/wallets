package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
)

func (m *MainHandler) ListWalletsHandler(w http.ResponseWriter, r *http.Request) {
	query := wallet.ListQuery[struct{}]{
		Limit:           parseLimit(r),
		PaginationToken: parsePaginationToken(r),
	}
	response, err := m.repository.ListWallets(r.Context(), query)
	if err != nil {
		internalError(w, r, err)
		return
	}

	cursorFromListResponse(w, query, response)
}
