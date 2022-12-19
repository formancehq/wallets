package storage

import "errors"

var (
	ErrLedgerInternal = errors.New("internal_ledger_error")
	ErrWalletNotFound = errors.New("wallet_not_found")
)
