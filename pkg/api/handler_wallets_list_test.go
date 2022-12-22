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

func TestWalletsList(t *testing.T) {
	t.Parallel()

	req := newRequest(t, http.MethodGet, "/wallets", nil)
	rec := httptest.NewRecorder()

	var wallets []core.Wallet
	for i := 0; i < 3; i++ {
		wallets = append(wallets, core.NewWallet(uuid.NewString(), core.Metadata{}))
	}

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithListAccountsWithMetadata(func(ctx context.Context, name string, m map[string]any) ([]sdk.Account, error) {
			require.Equal(t, testEnv.LedgerName(), name)
			require.Equal(t, map[string]any{
				core.MetadataKeySpecType: core.PrimaryWallet,
			}, m)
			ret := make([]sdk.Account, 0)
			for _, wallet := range wallets {
				ret = append(ret, sdk.Account{
					Address:  testEnv.Chart().GetMainAccount(wallet.ID),
					Metadata: wallet.LedgerMetadata(),
				})
			}
			return ret, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)
	list := make([]core.Wallet, 0)
	readResponse(t, rec, &list)
	require.Len(t, list, len(wallets))
	require.EqualValues(t, list, wallets)
}
