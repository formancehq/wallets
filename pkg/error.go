package wallet

import (
	"errors"
	"fmt"
)

var (
	ErrAccountNotFound         = errors.New("account not found")
	ErrWalletNotFound          = errors.New("wallet not found")
	ErrHoldNotFound            = errors.New("hold not found")
	ErrInsufficientFundError   = errors.New("insufficient fund")
	ErrClosedHold              = errors.New("closed hold")
	ErrBalanceAlreadyExists    = errors.New("balance already exists")
	ErrInvalidBalanceName      = errors.New("invalid balance name")
	ErrReservedBalanceName     = errors.New("reserved balance name")
	ErrBalanceNotExists        = errors.New("balance not exists")
	ErrInvalidBalanceSpecified = errors.New("invalid balance specified")
	ErrNegativeAmount          = errors.New("negative amount provided")
	ErrValidation              = errors.New("validation error")
	// ErrNonIdempotentDebit is returned when a debit carries an Idempotency-Key
	// but its source set cannot be resolved deterministically (wildcard sources
	// or sources with an expiry). The ledger enforces idempotency by hashing the
	// whole request body, so such a debit could not be safely replayed: a retry
	// might resolve to a different body and be rejected as a conflict. We reject
	// it up front rather than offer a false idempotency guarantee.
	ErrNonIdempotentDebit = errors.New("debit cannot be made idempotent: wildcard or expiring balances are not allowed with an idempotency key")
	// ErrIdempotencyConflict is returned when an Idempotency-Key is reused with a
	// different request body. Since callers cannot be assumed to be trusted, we
	// surface this as a conflict rather than silently replaying the original
	// resource and hiding the divergent request.
	ErrIdempotencyConflict = errors.New("idempotency key reused with a different request")
)

type GenericOpenAPIError interface {
	Model() any
}

type errInvalidAccountName string

func (e errInvalidAccountName) Error() string {
	return fmt.Sprintf("invalid format for account '%s'", string(e))
}

func (e errInvalidAccountName) Is(err error) bool {
	_, ok := err.(errInvalidAccountName)
	return ok
}

func newErrInvalidAccountName(v string) errInvalidAccountName {
	return errInvalidAccountName(v)
}

var _ error = errInvalidAccountName("")

func IsErrInvalidAccountName(err error) bool {
	return errors.Is(err, errInvalidAccountName(""))
}

type errInvalidAsset string

func (e errInvalidAsset) Error() string {
	return fmt.Sprintf("invalid format for account '%s'", string(e))
}

func (e errInvalidAsset) Is(err error) bool {
	_, ok := err.(errInvalidAsset)
	return ok
}

func newErrInvalidAsset(v string) errInvalidAsset {
	return errInvalidAsset(v)
}

var _ error = errInvalidAsset("")

func IsErrInvalidAsset(err error) bool {
	return errors.Is(err, errInvalidAsset(""))
}
