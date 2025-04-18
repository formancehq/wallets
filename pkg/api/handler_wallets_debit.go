package api

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/formancehq/go-libs/v3/api"

	wallet "github.com/formancehq/wallets/pkg"
	"github.com/go-chi/render"
)

func (m *MainHandler) debitWalletHandler(w http.ResponseWriter, r *http.Request) {
	data := &wallet.DebitRequest{}
	if err := render.Bind(r, data); err != nil {
		badRequest(w, ErrorCodeValidation, err)
		return
	}

	hold, err := m.manager.Debit(r.Context(), api.IdempotencyKeyFromRequest(r), wallet.Debit{
		WalletID:     chi.URLParam(r, "walletID"),
		DebitRequest: *data,
	})
	if err != nil {
		switch {
		case errors.Is(err, wallet.ErrInsufficientFundError):
			badRequest(w, ErrorCodeInsufficientFund, wallet.ErrInsufficientFundError)
		case errors.Is(err, wallet.ErrInvalidBalanceSpecified),
			errors.Is(err, wallet.ErrNegativeAmount),
			wallet.IsErrInvalidAccountName(err),
			wallet.IsErrInvalidAsset(err):
			badRequest(w, ErrorCodeValidation, wallet.ErrInvalidBalanceSpecified)
		default:
			internalError(w, r, err)
		}
		return
	}

	if hold == nil {
		noContent(w)
		return
	}

	created(w, hold)
}
