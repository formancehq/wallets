package api

import (
	"net/http"

	wallet "github.com/formancehq/wallets/pkg"
)

func (m *MainHandler) listTransactions(w http.ResponseWriter, r *http.Request) {
	query, err := readPaginatedRequest[wallet.ListTransactions](r, func(r *http.Request) wallet.ListTransactions {
		return wallet.ListTransactions{
			WalletID: r.URL.Query().Get("walletID"),
		}
	})
	if err != nil {
		badRequest(w, ErrorCodeValidation, err)
		return
	}
	transactions, err := m.manager.ListTransactions(r.Context(), query)
	if err != nil {
		internalError(w, r, err)
		return
	}

	cursorFromListResponse(w, query, transactions)
}
