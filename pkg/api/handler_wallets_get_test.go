package api

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v3/metadata"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletsGet(t *testing.T) {
	t.Parallel()

	w := wallet.NewWallet(uuid.NewString(), "default", metadata.Metadata{})
	balances := map[string]*big.Int{
		"USD": big.NewInt(100),
	}

	req := newRequest(t, http.MethodGet, "/wallets/"+w.ID, nil)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, testEnv.Chart().GetMainBalanceAccount(w.ID), account)

			return &wallet.AccountWithVolumesAndBalances{
				Account: wallet.Account{
					Address:  account,
					Metadata: metadataWithExpectingTypesAfterUnmarshalling(w.LedgerMetadata()),
				},
				Balances: balances,
			}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)
	walletWithBalances := wallet.Wallet{}
	readResponse(t, rec, &walletWithBalances)
	cp := w
	cp.Balances = balances
	require.Equal(t, cp, walletWithBalances)
}
