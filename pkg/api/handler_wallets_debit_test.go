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

func TestWalletsDebit(t *testing.T) {
	t.Parallel()

	type testCase struct {
		name           string
		request        wallet.DebitRequest
		scriptResult   sdk.ScriptResult
		expectedScript func(testEnv *testEnv, walletID string, h *core.DebitHold) sdk.Script
	}
	testCases := []testCase{
		{
			name: "nominal",
			request: wallet.DebitRequest{
				Amount: core.Monetary{
					Amount: core.NewMonetaryInt(100),
					Asset:  "USD",
				},
			},
			expectedScript: func(testEnv *testEnv, walletID string, h *core.DebitHold) sdk.Script {
				return sdk.Script{
					Plain: wallet.BuildDebitWalletScript(),
					Vars: map[string]interface{}{
						"source":      testEnv.Chart().GetMainAccount(walletID),
						"destination": wallet.DefaultDebitDest.Identifier,
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
		{
			name: "with custom destination as ledger account",
			request: wallet.DebitRequest{
				Amount: core.Monetary{
					Amount: core.NewMonetaryInt(100),
					Asset:  "USD",
				},
				Destination: &wallet.Subject{
					Type:       wallet.SubjectTypeLedgerAccount,
					Identifier: "account1",
				},
			},
			expectedScript: func(testEnv *testEnv, walletID string, h *core.DebitHold) sdk.Script {
				return sdk.Script{
					Plain: wallet.BuildDebitWalletScript(),
					Vars: map[string]interface{}{
						"source":      testEnv.Chart().GetMainAccount(walletID),
						"destination": "account1",
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
		{
			name: "with custom destination as wallet",
			request: wallet.DebitRequest{
				Amount: core.Monetary{
					Amount: core.NewMonetaryInt(100),
					Asset:  "USD",
				},
				Destination: &wallet.Subject{
					Type:       wallet.SubjectTypeWallet,
					Identifier: "wallet1",
				},
			},
			expectedScript: func(testEnv *testEnv, walletID string, h *core.DebitHold) sdk.Script {
				return sdk.Script{
					Plain: wallet.BuildDebitWalletScript(),
					Vars: map[string]interface{}{
						"source":      testEnv.Chart().GetMainAccount(walletID),
						"destination": testEnv.Chart().GetMainAccount("wallet1"),
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
		{
			name: "with insufficient funds",
			request: wallet.DebitRequest{
				Amount: core.Monetary{
					Amount: core.NewMonetaryInt(100),
					Asset:  "USD",
				},
			},
			scriptResult: sdk.ScriptResult{
				ErrorCode: func() *string {
					ret := string(sdk.INSUFFICIENT_FUND)
					return &ret
				}(),
			},
		},
		{
			name: "with debit hold",
			request: wallet.DebitRequest{
				Amount: core.Monetary{
					Amount: core.NewMonetaryInt(100),
					Asset:  "USD",
				},
				Pending: true,
				Metadata: map[string]any{
					"foo": "bar",
				},
				Description: "a first tx",
			},
			expectedScript: func(testEnv *testEnv, walletID string, h *core.DebitHold) sdk.Script {
				return sdk.Script{
					Plain: wallet.BuildDebitWalletScript(),
					Vars: map[string]interface{}{
						"source":      testEnv.Chart().GetMainAccount(walletID),
						"destination": testEnv.Chart().GetHoldAccount(h.ID),
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
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			walletID := uuid.NewString()

			req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", testCase.request)
			rec := httptest.NewRecorder()

			var (
				testEnv             *testEnv
				executedScript      sdk.Script
				holdAccount         string
				holdAccountMetadata metadata.Metadata
			)
			testEnv = newTestEnv(
				WithAddMetadataToAccount(func(ctx context.Context, ledger, account string, m metadata.Metadata) error {
					require.Equal(t, testEnv.LedgerName(), ledger)
					holdAccount = account
					holdAccountMetadata = m
					return nil
				}),
				WithRunScript(func(ctx context.Context, ledger string, script sdk.Script) (*sdk.ScriptResult, error) {
					require.Equal(t, testEnv.LedgerName(), ledger)
					executedScript = script
					return &testCase.scriptResult, nil
				}),
			)
			testEnv.Router().ServeHTTP(rec, req)

			hold := &core.DebitHold{}
			switch {
			case testCase.request.Pending:
				require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
				readResponse(t, rec, hold)
			case !testCase.request.Pending && testCase.scriptResult.ErrorCode == nil:
				require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
			case !testCase.request.Pending && testCase.scriptResult.ErrorCode != nil:
				require.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
				errorResponse := readErrorResponse(t, rec)
				require.Equal(t, *testCase.scriptResult.ErrorCode, errorResponse.ErrorCode)
			}

			if testCase.expectedScript != nil {
				expectedScript := testCase.expectedScript(testEnv, walletID, hold)
				require.Equal(t, expectedScript, executedScript)
			}

			if testCase.request.Pending {
				require.Equal(t, testEnv.Chart().GetHoldAccount(hold.ID), holdAccount)
				require.Equal(t, walletID, hold.WalletID)
				require.Equal(t, testCase.request.Amount.Asset, hold.Asset)
				require.Equal(t, hold.LedgerMetadata(testEnv.Chart()), holdAccountMetadata)
			}
		})
	}
}
