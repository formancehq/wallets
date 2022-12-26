package api

import (
	"errors"
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type PatchWalletRequest struct {
	Metadata core.Metadata `json:"metadata"`
}

func (c *PatchWalletRequest) Bind(r *http.Request) error {
	return nil
}

func (m *MainHandler) PatchWalletHandler(w http.ResponseWriter, r *http.Request) {
	data := &PatchWalletRequest{}
	if err := render.Bind(r, data); err != nil {
		badRequest(w, ErrorCodeValidation, err)
		return
	}

	err := m.repository.UpdateWallet(r.Context(), chi.URLParam(r, "wallet_id"), &wallet.Data{
		Metadata: data.Metadata,
	})
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrWalletNotFound):
			notFound(w)
		default:
			internalError(w, r, err)
		}
		return
	}

	noContent(w)
}
