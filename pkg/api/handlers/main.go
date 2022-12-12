package handlers

import (
	"github.com/formancehq/wallets/pkg/storage"
	"github.com/formancehq/wallets/pkg/wallet"
)

type MainHandler struct {
	funding    *wallet.FundingService
	repository *storage.Repository
}

func NewMainHandler(
	funding *wallet.FundingService,
	repository *storage.Repository,
) *MainHandler {
	return &MainHandler{
		funding:    funding,
		repository: repository,
	}
}
