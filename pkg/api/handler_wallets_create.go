package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/render"
)

type CreateWalletRequest struct {
	Metadata core.Metadata `json:"metadata"`
}

func (c *CreateWalletRequest) Bind(r *http.Request) error {
	return nil
}

func (m *MainHandler) CreateWalletHandler(w http.ResponseWriter, r *http.Request) {
	data := &CreateWalletRequest{}
	if r.ContentLength > 0 {
		if err := render.Bind(r, data); err != nil {
			badRequest(w, err)
			return
		}
	}

	wallet, err := m.repository.CreateWallet(r.Context(), &wallet.Data{
		Metadata: data.Metadata,
	})
	if err != nil {
		internalError(w, r, err)
		return
	}

	ok(w, wallet)
}