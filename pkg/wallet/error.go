package wallet

import (
	"errors"
	"fmt"
)

var (
	ErrWalletNotFound = errors.New("wallet_not_found")
	ErrHoldNotFound   = errors.New("hold_not_found")
)

type MismatchTypeError struct {
	expected, got string
}

func (t MismatchTypeError) Error() string {
	return fmt.Sprintf("unexpected type, got '%s', but '%s' was expected", t.got, t.expected)
}

func NewErrMismatchType(expected, got string) MismatchTypeError {
	return MismatchTypeError{
		expected: expected,
		got:      got,
	}
}

func IsMismatchTypeError(err error) bool {
	return errors.As(err, &MismatchTypeError{})
}
