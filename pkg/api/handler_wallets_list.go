package api

import (
	"net/http"

	wallet "github.com/formancehq/wallets/pkg"
)

func (m *MainHandler) listWalletsHandler(w http.ResponseWriter, r *http.Request) {
	query, err := readPaginatedRequest[wallet.ListWallets](r, func(r *http.Request) wallet.ListWallets {
		return wallet.ListWallets{
			Metadata:       getQueryMap(r.URL.Query(), "metadata"),
			Name:           r.URL.Query().Get("name"),
			ExpandBalances: r.URL.Query().Get("expand") == "balances",
		}
	})
	if err != nil {
		badRequest(w, ErrorCodeValidation, err)
		return
	}
	response, err := m.manager.ListWallets(r.Context(), query)
	if err != nil {
		internalError(w, r, err)
		return
	}

	cursorFromListResponse(w, query, response)
}
