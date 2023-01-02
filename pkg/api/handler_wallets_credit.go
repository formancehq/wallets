package api

import (
	"net/http"

	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

const (
	ErrorCodeInternal         = "INTERNAL"
	ErrorCodeInsufficientFund = "INSUFFICIENT_FUND"
	ErrorCodeValidation       = "VALIDATION"
	ErrorCodeClosedHold       = "HOLD_CLOSED"
)

func (m *MainHandler) CreditWalletHandler(w http.ResponseWriter, r *http.Request) {
	data := &wallet.CreditRequest{}
	if err := render.Bind(r, data); err != nil {
		badRequest(w, ErrorCodeValidation, err)
		return
	}

	id := chi.URLParam(r, "walletID")
	credit := wallet.Credit{
		WalletID:      id,
		CreditRequest: *data,
	}

	err := m.funding.Credit(r.Context(), credit)
	if err != nil {
		internalError(w, r, err)
		return
	}

	noContent(w)
}
