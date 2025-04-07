package api

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/go-libs/v3/metadata"
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
		WithListTransactions(func(ctx context.Context, ledger string, query wallet.ListTransactionsQuery) (*shared.V2TransactionsCursorResponseCursor, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)

			return &shared.V2TransactionsCursorResponseCursor{
				Data: []shared.V2Transaction{{
					Postings: []shared.V2Posting{
						{
							Source:      testEnv.Chart().GetBalanceAccount(walletID, "secondary"),
							Destination: testEnv.Chart().GetHoldAccount(hold.ID),
							Amount:      big.NewInt(100),
							Asset:       "USD",
						},
						{
							Source:      testEnv.Chart().GetMainBalanceAccount(walletID),
							Destination: testEnv.Chart().GetHoldAccount(hold.ID),
							Amount:      big.NewInt(100),
							Asset:       "USD",
						},
					},
				}},
			}, nil
		}),
		WithCreateTransaction(func(ctx context.Context, name, ik string, script wallet.PostTransaction) (*shared.V2Transaction, error) {
			compareJSON(t, wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: wallet.BuildCancelHoldScript("USD",
						shared.V2Posting{
							Source:      testEnv.Chart().GetBalanceAccount(walletID, "secondary"),
							Destination: testEnv.Chart().GetHoldAccount(hold.ID),
							Amount:      big.NewInt(100),
							Asset:       "USD",
						},
						shared.V2Posting{
							Source:      testEnv.Chart().GetMainBalanceAccount(walletID),
							Destination: testEnv.Chart().GetHoldAccount(hold.ID),
							Amount:      big.NewInt(100),
							Asset:       "USD",
						},
					),
					Vars: map[string]string{
						"hold": testEnv.Chart().GetHoldAccount(hold.ID),
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
