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

func TestHoldsList(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()

	req := newRequest(t, http.MethodGet, "/wallets/"+walletID+"/holds", nil)
	rec := httptest.NewRecorder()

	holds := make([]core.DebitHold, 0)
	for i := 0; i < 3; i++ {
		holds = append(holds, core.NewDebitHold(walletID, "bank"))
	}

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithListAccountsWithMetadata(func(ctx context.Context, name string, m map[string]any) ([]sdk.Account, error) {
			require.Equal(t, testEnv.LedgerName(), name)
			require.EqualValues(t, core.Metadata{
				core.MetadataKeySpecType:     core.HoldWallet,
				core.MetadataKeyHoldWalletID: walletID,
			}, m)
			ret := make([]sdk.Account, 0)
			for _, hold := range holds {
				ret = append(ret, sdk.Account{
					Address:  testEnv.Chart().GetHoldAccount(hold.ID),
					Metadata: hold.LedgerMetadata(testEnv.Chart()),
				})
			}
			return ret, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)

	list := make([]core.DebitHold, 0)
	readResponse(t, rec, &list)
	require.Len(t, list, 3)
	require.EqualValues(t, holds, list)
}
