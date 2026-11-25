package wallet

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfirmHoldPreservesRemainingAmountPrecision(t *testing.T) {
	balance := new(big.Int).Lsh(big.NewInt(1), 64)
	hold := ExpandedDebitHold{Remaining: balance}
	require.False(t, hold.IsClosed())

	full, err := (ConfirmHold{}).resolveAmount(hold)
	require.NoError(t, err)
	require.Equal(t, balance, full)

	part, err := (ConfirmHold{Amount: big.NewInt(30)}).resolveAmount(hold)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(30), part)

	_, err = (ConfirmHold{Amount: big.NewInt(-1)}).resolveAmount(hold)
	require.ErrorIs(t, err, ErrNegativeAmount)
	_, err = (ConfirmHold{Amount: new(big.Int).Add(balance, big.NewInt(1))}).resolveAmount(hold)
	require.ErrorIs(t, err, ErrInsufficientFundError)
}
