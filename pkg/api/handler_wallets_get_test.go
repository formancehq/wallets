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

func TestWalletsGet(t *testing.T) {
	t.Parallel()

	wallet := core.NewWallet(uuid.NewString(), core.Metadata{})

	req := newRequest(t, http.MethodGet, "/wallets/"+wallet.ID, nil)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*sdk.AccountWithVolumesAndBalances, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, testEnv.Chart().GetMainAccount(wallet.ID), account)
			return &sdk.AccountWithVolumesAndBalances{
				Address:  account,
				Metadata: wallet.LedgerMetadata(),
			}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)
}
