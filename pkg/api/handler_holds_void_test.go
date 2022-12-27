package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet/numscript"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHoldsVoid(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	hold := core.NewDebitHold(walletID, "bank", "USD")

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/holds/"+hold.ID+"/void", nil)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*sdk.AccountWithVolumesAndBalances, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, testEnv.Chart().GetHoldAccount(hold.ID), account)

			return &sdk.AccountWithVolumesAndBalances{
				Address:  testEnv.Chart().GetHoldAccount(hold.ID),
				Metadata: hold.LedgerMetadata(testEnv.Chart()),
			}, nil
		}),
		WithRunScript(func(ctx context.Context, name string, script sdk.Script) (*sdk.ScriptResult, error) {
			require.Equal(t, sdk.Script{
				Plain: strings.ReplaceAll(numscript.CancelHold, "ASSET", "USD"),
				Vars: map[string]interface{}{
					"hold": testEnv.Chart().GetHoldAccount(hold.ID),
				},
				Metadata: core.WalletTransactionBaseMetadata(),
			}, script)
			return &sdk.ScriptResult{}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
}
