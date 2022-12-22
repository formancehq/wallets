package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHoldsGet(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	hold := core.NewDebitHold(walletID, "bank", "USD")

	req := newRequest(t, http.MethodGet, "/wallets/"+walletID+"/holds/"+hold.ID, nil)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*sdk.AccountWithVolumesAndBalances, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, testEnv.Chart().GetHoldAccount(hold.ID), account)
			balances := map[string]int32{
				"USD": 100,
			}
			return &sdk.AccountWithVolumesAndBalances{
				Address:  testEnv.Chart().GetHoldAccount(hold.ID),
				Metadata: hold.LedgerMetadata(testEnv.Chart()),
				Balances: &balances,
			}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)

	ret := core.DebitHold{}
	readResponse(t, rec, &ret)
	require.EqualValues(t, hold, ret)
}
