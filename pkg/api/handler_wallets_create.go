package api

import (
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/go-chi/render"
)

func (m *MainHandler) createWalletHandler(w http.ResponseWriter, r *http.Request) {
	data := &wallet.CreateRequest{}
	if r.ContentLength > 0 {
		if err := render.Bind(r, data); err != nil {
			badRequest(w, ErrorCodeValidation, err)
			return
		}
	}

	createdWallet, err := m.manager.CreateWallet(r.Context(), api.IdempotencyKeyFromRequest(r), data)
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrIdempotencyConflict):
			conflict(w, ErrorCodeConflict, wallet.ErrIdempotencyConflict)
		default:
			internalError(w, r, err)
		}
		return
	}

	created(w, createdWallet)
}
