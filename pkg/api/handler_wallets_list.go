package api

import (
	"net/http"
)

func (m *MainHandler) ListWalletsHandler(w http.ResponseWriter, r *http.Request) {
	query := readPaginatedRequest[struct{}](r, nil)
	response, err := m.repository.ListWallets(r.Context(), query)
	if err != nil {
		internalError(w, r, err)
		return
	}

	cursorFromListResponse(w, query, response)
}
