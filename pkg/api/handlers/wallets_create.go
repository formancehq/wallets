package handlers

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/storage"
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
	if err := render.Bind(r, data); err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, map[string]string{
			"error": err.Error(),
		})
		return
	}

	wallet, err := m.repository.CreateWallet(r.Context(), &storage.WalletData{
		Metadata: data.Metadata,
	})
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, map[string]string{
			// @todo: return a proper error
			"error": err.Error(),
		})
		return
	}

	render.JSON(w, r, wallet)
}
