package api

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/types/time"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletsCredit(t *testing.T) {
	t.Parallel()
	now := time.Now()

	type testCase struct {
		name                    string
		request                 wallet.CreditRequest
		postTransactionResult   shared.V2Transaction
		expectedPostTransaction func(testEnv *testEnv, walletID string) wallet.PostTransaction
		expectedStatusCode      int
		expectedErrorCode       string
	}
	testCases := []testCase{
		{
			name: "nominal",
			request: wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
				Metadata: metadata.Metadata{
					"foo": "bar",
				},
			},
			expectedPostTransaction: func(testEnv *testEnv, walletID string) wallet.PostTransaction {
				return wallet.PostTransaction{
					Script: &shared.V2PostTransactionScript{
						Plain: pointer.For(wallet.BuildCreditWalletScript("world")),
						Vars: map[string]string{
							"destination": testEnv.chart.GetMainBalanceAccount(walletID),
							"amount":      "USD 100",
						},
					},
					Metadata: wallet.TransactionMetadata(metadata.Metadata{
						"foo": "bar",
					}),
				}
			},
		},
		{
			name: "with source list",
			request: wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
				Sources: []wallet.Subject{
					wallet.NewLedgerAccountSubject("emitter1"),
					wallet.NewWalletSubject("wallet1", ""),
				},
			},
			expectedPostTransaction: func(testEnv *testEnv, walletID string) wallet.PostTransaction {
				return wallet.PostTransaction{
					Script: &shared.V2PostTransactionScript{
						Plain: pointer.For(wallet.BuildCreditWalletScript(
							"emitter1",
							testEnv.Chart().GetMainBalanceAccount("wallet1"),
						)),
						Vars: map[string]string{
							"destination": testEnv.chart.GetMainBalanceAccount(walletID),
							"amount":      "USD 100",
						},
					},
					Metadata: wallet.TransactionMetadata(nil),
				}
			},
		},
		{
			name: "with secondary balance from source",
			request: wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
				Sources: []wallet.Subject{
					wallet.NewWalletSubject("emitter1", "secondary"),
				},
			},
			expectedPostTransaction: func(testEnv *testEnv, walletID string) wallet.PostTransaction {
				return wallet.PostTransaction{
					Script: &shared.V2PostTransactionScript{
						Plain: pointer.For(wallet.BuildCreditWalletScript(
							testEnv.Chart().GetBalanceAccount("emitter1", "secondary"),
						)),
						Vars: map[string]string{
							"destination": testEnv.Chart().GetMainBalanceAccount(walletID),
							"amount":      "USD 100",
						},
					},
					Metadata: wallet.TransactionMetadata(nil),
				}
			},
		},
		{
			name: "with wallet source containing numscript injection in balance",
			request: wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
				Sources: []wallet.Subject{
					wallet.NewWalletSubject("emitter1", "secondary\n@world"),
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  ErrorCodeValidation,
		},
		{
			name: "with wallet source spanning multiple account segments",
			request: wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
				Sources: []wallet.Subject{
					wallet.NewWalletSubject("emitter1", "balance:injected"),
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  ErrorCodeValidation,
		},
		{
			// Dashes are allowed in balance names (they still alias under
			// Address.String(); see chart.go), so a dashed wallet source resolves.
			name: "with dashed balance in wallet source",
			request: wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
				Sources: []wallet.Subject{
					wallet.NewWalletSubject("emitter1", "foo-bar"),
				},
			},
			expectedPostTransaction: func(testEnv *testEnv, walletID string) wallet.PostTransaction {
				return wallet.PostTransaction{
					Script: &shared.V2PostTransactionScript{
						Plain: pointer.For(wallet.BuildCreditWalletScript(
							testEnv.Chart().GetBalanceAccount("emitter1", "foo-bar"),
						)),
						Vars: map[string]string{
							"destination": testEnv.Chart().GetMainBalanceAccount(walletID),
							"amount":      "USD 100",
						},
					},
					Metadata: wallet.TransactionMetadata(nil),
				}
			},
		},
		{
			name: "with wallet source containing numscript injection in identifier",
			request: wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
				Sources: []wallet.Subject{
					wallet.NewWalletSubject("emitter1 @world", ""),
				},
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  ErrorCodeValidation,
		},
		{
			// Dashes are allowed in balance names (still alias under
			// Address.String(); see chart.go), so a dashed destination resolves.
			name: "with dashed destination balance",
			request: wallet.CreditRequest{
				Amount:  wallet.NewMonetary(big.NewInt(100), "USD"),
				Balance: "foo-bar",
			},
			expectedPostTransaction: func(testEnv *testEnv, walletID string) wallet.PostTransaction {
				return wallet.PostTransaction{
					Script: &shared.V2PostTransactionScript{
						Plain: pointer.For(wallet.BuildCreditWalletScript("world")),
						Vars: map[string]string{
							"destination": testEnv.Chart().GetBalanceAccount(walletID, "foo-bar"),
							"amount":      "USD 100",
						},
					},
					Metadata: wallet.TransactionMetadata(nil),
				}
			},
		},
		{
			name: "with secondary balance as destination",
			request: wallet.CreditRequest{
				Amount:  wallet.NewMonetary(big.NewInt(100), "USD"),
				Balance: "secondary",
			},
			expectedPostTransaction: func(testEnv *testEnv, walletID string) wallet.PostTransaction {
				return wallet.PostTransaction{
					Script: &shared.V2PostTransactionScript{
						Plain: pointer.For(wallet.BuildCreditWalletScript("world")),
						Vars: map[string]string{
							"destination": testEnv.Chart().GetBalanceAccount(walletID, "secondary"),
							"amount":      "USD 100",
						},
					},
					Metadata: wallet.TransactionMetadata(nil),
				}
			},
		},
		{
			name: "with not existing secondary balance as destination",
			request: wallet.CreditRequest{
				Amount:  wallet.NewMonetary(big.NewInt(100), "USD"),
				Balance: "not_existing",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedErrorCode:  ErrorCodeValidation,
		},
		{
			name: "with specified timestamp",
			request: wallet.CreditRequest{
				Amount:    wallet.NewMonetary(big.NewInt(100), "USD"),
				Timestamp: &now,
			},
			expectedPostTransaction: func(testEnv *testEnv, walletID string) wallet.PostTransaction {
				return wallet.PostTransaction{
					Script: &shared.V2PostTransactionScript{
						Plain: pointer.For(wallet.BuildCreditWalletScript("world")),
						Vars: map[string]string{
							"destination": testEnv.chart.GetMainBalanceAccount(walletID),
							"amount":      "USD 100",
						},
					},
					Metadata:  wallet.TransactionMetadata(nil),
					Timestamp: &now,
				}
			},
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			walletID := uuid.NewString()
			secondaryBalance := wallet.NewBalance("secondary", nil)
			dashedBalance := wallet.NewBalance("foo-bar", nil)

			req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/credit", testCase.request)
			rec := httptest.NewRecorder()

			var (
				testEnv         *testEnv
				postTransaction wallet.PostTransaction
			)
			testEnv = newTestEnv(
				WithCreateTransaction(func(ctx context.Context, ledger, ik string, p wallet.PostTransaction) (*shared.V2Transaction, error) {
					require.Equal(t, testEnv.LedgerName(), ledger)
					postTransaction = p
					return &testCase.postTransactionResult, nil
				}),
				WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
					for _, b := range []wallet.Balance{secondaryBalance, dashedBalance} {
						if testEnv.Chart().GetBalanceAccount(walletID, b.Name) == account {
							return &wallet.AccountWithVolumesAndBalances{
								Account: wallet.Account{
									Address:  account,
									Metadata: metadataWithExpectingTypesAfterUnmarshalling(b.LedgerMetadata(walletID)),
								},
							}, nil
						}
					}
					return &wallet.AccountWithVolumesAndBalances{
						Account: wallet.Account{
							Metadata: map[string]string{},
						},
					}, nil
				}),
			)
			testEnv.Router().ServeHTTP(rec, req)

			expectedStatusCode := testCase.expectedStatusCode
			if expectedStatusCode == 0 {
				expectedStatusCode = http.StatusNoContent
			}

			require.Equal(t, expectedStatusCode, rec.Result().StatusCode)
			if expectedStatusCode == http.StatusNoContent {
				if testCase.expectedPostTransaction != nil {
					expectedScript := testCase.expectedPostTransaction(testEnv, walletID)
					require.Equal(t, expectedScript, postTransaction)
				}
			} else {
				errorResponse := readErrorResponse(t, rec)
				require.Equal(t, ErrorCodeValidation, errorResponse.ErrorCode)
			}
		})
	}
}

// TestWalletsCreditRejectsInvalidWalletID guards the WalletID supplied via the
// URL path: a value spanning multiple account segments or carrying Numscript
// tokens must be rejected before any ledger transaction is created.
func TestWalletsCreditRejectsInvalidWalletID(t *testing.T) {
	t.Parallel()

	for _, walletID := range []string{
		"wallet:injected",
		"wallet\n@world",
	} {
		walletID := walletID
		t.Run(walletID, func(t *testing.T) {
			t.Parallel()

			req := newRequest(t, http.MethodPost, "/wallets/"+url.PathEscape(walletID)+"/credit", wallet.CreditRequest{
				Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
			})
			rec := httptest.NewRecorder()

			var (
				testEnv            *testEnv
				createdTransaction bool
			)
			testEnv = newTestEnv(
				WithCreateTransaction(func(ctx context.Context, ledger, ik string, p wallet.PostTransaction) (*shared.V2Transaction, error) {
					createdTransaction = true
					return &shared.V2Transaction{}, nil
				}),
			)
			testEnv.Router().ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
			errorResponse := readErrorResponse(t, rec)
			require.Equal(t, ErrorCodeValidation, errorResponse.ErrorCode)
			require.False(t, createdTransaction)
		})
	}
}
