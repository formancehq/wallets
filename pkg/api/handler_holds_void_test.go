package api

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/go-libs/v2/metadata"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHoldsVoid(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	hold := wallet.NewDebitHold(walletID, wallet.NewLedgerAccountSubject("bank"), "USD", "", metadata.Metadata{})

	req := newRequest(t, http.MethodPost, "/holds/"+hold.ID+"/void", nil)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, testEnv.Chart().GetHoldAccount(hold.ID), account)

			return &wallet.AccountWithVolumesAndBalances{
				Account: wallet.Account{
					Address:  testEnv.Chart().GetHoldAccount(hold.ID),
					Metadata: metadataWithExpectingTypesAfterUnmarshalling(hold.LedgerMetadata(testEnv.Chart())),
				},
				Balances: map[string]*big.Int{
					"USD": big.NewInt(100),
				},
				Volumes: map[string]shared.V2Volume{
					"USD": {
						Input: big.NewInt(100),
					},
				},
			}, nil
		}),
		WithCreateTransaction(func(ctx context.Context, name, ik string, script wallet.PostTransaction) (*shared.V2Transaction, error) {
			compareJSON(t, wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: wallet.BuildCancelHoldScript("USD"),
					Vars: map[string]interface{}{
						"hold": testEnv.Chart().GetHoldAccount(hold.ID),
						"dest": testEnv.Chart().GetMainBalanceAccount(hold.WalletID),
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}, script)
			return &shared.V2Transaction{}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
}
