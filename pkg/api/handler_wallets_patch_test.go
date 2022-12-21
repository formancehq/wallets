package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/stretchr/testify/require"
)

func TestWalletsPatch(t *testing.T) {
	t.Parallel()

	wallet := core.NewWallet(core.Metadata{
		"foo": "bar",
	})
	patchWalletRequest := PatchWalletRequest{
		Metadata: map[string]interface{}{
			"role": "admin",
			"foo":  "baz",
		},
	}

	req := newRequest(t, http.MethodPatch, "/wallets/"+wallet.ID, patchWalletRequest)
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
		WithAddMetadataToAccount(func(ctx context.Context, ledger, account string, metadata core.Metadata) error {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, testEnv.Chart().GetMainAccount(wallet.ID), account)
			require.EqualValues(t, core.Metadata{
				core.MetadataKeyWalletCustomData: core.Metadata{
					"role": "admin",
					"foo":  "baz",
				},
			}, metadata)
			return nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
}