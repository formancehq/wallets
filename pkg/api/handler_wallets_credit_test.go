package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/go-libs/metadata"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletsCredit(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		request        wallet.CreditRequest
		scriptResult   sdk.ScriptResult
		expectedScript func(testEnv *testEnv, walletID string) sdk.Script
	}
	testCases := []testCase{
		{
			name: "nominal",
			request: wallet.CreditRequest{
				Amount: core.Monetary{
					Amount: core.NewMonetaryInt(100),
					Asset:  "USD",
				},
				Metadata: map[string]interface{}{
					"foo": "bar",
				},
			},
			scriptResult: sdk.ScriptResult{},
			expectedScript: func(testEnv *testEnv, walletID string) sdk.Script {
				return sdk.Script{
					Plain: wallet.BuildCreditWalletScript("world"),
					Vars: map[string]interface{}{
						"destination": testEnv.chart.GetMainAccount(walletID),
						"amount": map[string]any{
							"amount": uint64(100),
							"asset":  "USD",
						},
					},
					Metadata: core.WalletTransactionBaseMetadata().Merge(metadata.Metadata{
						core.MetadataKeyWalletCustomData: metadata.Metadata{
							"foo": "bar",
						},
					}),
				}
			},
		},
		{
			name: "with source list",
			request: wallet.CreditRequest{
				Amount: core.Monetary{
					Amount: core.NewMonetaryInt(100),
					Asset:  "USD",
				},
				Sources: []wallet.Subject{{
					Type:       wallet.SubjectTypeLedgerAccount,
					Identifier: "emitter1",
				}, {
					Type:       wallet.SubjectTypeWallet,
					Identifier: "wallet1",
				}},
			},
			scriptResult: sdk.ScriptResult{},
			expectedScript: func(testEnv *testEnv, walletID string) sdk.Script {
				return sdk.Script{
					Plain: wallet.BuildCreditWalletScript(
						"emitter1",
						testEnv.Chart().GetMainAccount("wallet1"),
					),
					Vars: map[string]interface{}{
						"destination": testEnv.chart.GetMainAccount(walletID),
						"amount": map[string]any{
							"amount": uint64(100),
							"asset":  "USD",
						},
					},
					Metadata: core.WalletTransactionBaseMetadata().Merge(metadata.Metadata{
						core.MetadataKeyWalletCustomData: metadata.Metadata{},
					}),
				}
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			walletID := uuid.NewString()

			req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/credit", testCase.request)
			rec := httptest.NewRecorder()

			var (
				testEnv        *testEnv
				executedScript sdk.Script
			)
			testEnv = newTestEnv(
				WithRunScript(func(ctx context.Context, ledger string, script sdk.Script) (*sdk.ScriptResult, error) {
					require.Equal(t, testEnv.LedgerName(), ledger)
					executedScript = script
					return &testCase.scriptResult, nil
				}),
			)
			testEnv.Router().ServeHTTP(rec, req)

			require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
			if testCase.expectedScript != nil {
				expectedScript := testCase.expectedScript(testEnv, walletID)
				require.Equal(t, expectedScript, executedScript)
			}
		})
	}
}
