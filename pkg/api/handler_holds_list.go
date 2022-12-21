package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func (m *MainHandler) ListHoldsHandler(w http.ResponseWriter, r *http.Request) {
	holds, err := m.repository.ListHolds(r.Context(), chi.URLParam(r, "wallet_id"))
	if err != nil {
		internalError(w, r, err)
		return
	}

	ok(w, holds)
}