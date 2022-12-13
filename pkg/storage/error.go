package storage

import "errors"

var (
	InternalLedgerError = errors.New("internal_ledger_error")
	WalletNotFound      = errors.New("wallet_not_found")
)
