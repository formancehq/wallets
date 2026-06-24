package api

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/types/time"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/sdkerrors"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func compareJSON(t *testing.T, expected, actual any) {
	data, err := json.Marshal(expected)
	require.NoError(t, err)

	expectedAsMap := make(map[string]any)
	require.NoError(t, json.Unmarshal(data, &expectedAsMap))

	data, err = json.Marshal(actual)
	require.NoError(t, err)

	actualAsMap := make(map[string]any)
	require.NoError(t, json.Unmarshal(data, &actualAsMap))

	require.Equal(t, expectedAsMap, actualAsMap)
}

type testCase struct {
	name                    string
	request                 wallet.DebitRequest
	postTransactionError    error
	expectedPostTransaction func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction
	expectedStatusCode      int
	expectedErrorCode       string
}

var now = time.Now()
var walletDebitTestCases = []testCase{
	{
		name: "nominal",
		request: wallet.DebitRequest{
			Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetMainBalanceAccount(walletID))),
					Vars: map[string]string{
						"destination": wallet.DefaultDebitDest.Identifier,
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}
		},
	},
	{
		name: "using timestamp",
		request: wallet.DebitRequest{
			Amount:    wallet.NewMonetary(big.NewInt(100), "USD"),
			Timestamp: &now,
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetMainBalanceAccount(walletID))),
					Vars: map[string]string{
						"destination": wallet.DefaultDebitDest.Identifier,
						"amount":      "USD 100",
					},
				},
				Metadata:  metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
				Timestamp: &now,
			}
		},
	},
	{
		name: "with custom destination as ledger account",
		request: wallet.DebitRequest{
			Amount:      wallet.NewMonetary(big.NewInt(100), "USD"),
			Destination: wallet.Ptr(wallet.NewLedgerAccountSubject("account1")),
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetMainBalanceAccount(walletID))),
					Vars: map[string]string{
						"destination": "account1",
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}
		},
	},
	{
		name: "with custom destination as wallet",
		request: wallet.DebitRequest{
			Amount:      wallet.NewMonetary(big.NewInt(100), "USD"),
			Destination: wallet.Ptr(wallet.NewWalletSubject("wallet1", "")),
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetMainBalanceAccount(walletID))),
					Vars: map[string]string{
						"destination": testEnv.Chart().GetMainBalanceAccount("wallet1"),
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}
		},
	},
	{
		name: "with insufficient funds",
		request: wallet.DebitRequest{
			Amount: wallet.NewMonetary(big.NewInt(100), "USD"),
		},
		// The Ledger interface (DefaultLedger) translates the SDK's
		// *sdkerrors.V2ErrorResponse into wallet.ErrInsufficientFundError,
		// so the mock returns that domain error directly.
		postTransactionError: wallet.ErrInsufficientFundError,
		expectedStatusCode:   http.StatusBadRequest,
		expectedErrorCode:    string(shared.ErrorsEnumInsufficientFund),
	},
	{
		// Every resolved balance is expired, so the source set is empty.
		// This must surface as INSUFFICIENT_FUND, not a ledger compile 500.
		name: "with only expired balance",
		request: wallet.DebitRequest{
			Amount:   wallet.NewMonetary(big.NewInt(100), "USD"),
			Balances: []string{"coupon3"},
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  string(shared.ErrorsEnumInsufficientFund),
	},
	{
		name: "with debit hold",
		request: wallet.DebitRequest{
			Amount:  wallet.NewMonetary(big.NewInt(100), "USD"),
			Pending: true,
			Metadata: map[string]string{
				"foo": "bar",
			},
			Description: "a first tx",
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{
						testEnv.Chart().GetHoldAccount(h.ID): h.LedgerMetadata(testEnv.Chart()),
					}, testEnv.Chart().GetMainBalanceAccount(walletID))),
					Vars: map[string]string{
						"destination": testEnv.Chart().GetHoldAccount(h.ID),
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(metadata.Metadata{
					"foo": "bar",
				})),
			}
		},
		expectedStatusCode: http.StatusCreated,
	},
	{
		name: "with custom balance as source",
		request: wallet.DebitRequest{
			Amount:   wallet.NewMonetary(big.NewInt(100), "USD"),
			Balances: []string{"secondary"},
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetBalanceAccount(walletID, "secondary"))),
					Vars: map[string]string{
						"destination": "world",
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}
		},
	},
	{
		name: "with wildcard balance as source",
		request: wallet.DebitRequest{
			Amount:   wallet.NewMonetary(big.NewInt(100), "USD"),
			Balances: []string{"*"},
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetBalanceAccount(walletID, "coupon1"), testEnv.Chart().GetBalanceAccount(walletID, "coupon4"), testEnv.Chart().GetBalanceAccount(walletID, "coupon2"), testEnv.Chart().GetBalanceAccount(walletID, "main"))),
					Vars: map[string]string{
						"destination": "world",
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}
		},
	},
	{
		name: "with wildcard plus another source",
		request: wallet.DebitRequest{
			Amount:   wallet.NewMonetary(big.NewInt(100), "USD"),
			Balances: []string{"*", "secondary"},
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetBalanceAccount(walletID, "secondary"))),
					Vars: map[string]string{
						"destination": "world",
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  string(sdkerrors.SchemasErrorCodeValidation),
	},
	{
		name: "with custom balance as destination",
		request: wallet.DebitRequest{
			Amount:      wallet.NewMonetary(big.NewInt(100), "USD"),
			Destination: wallet.Ptr(wallet.NewWalletSubject("wallet1", "secondary")),
		},
		expectedPostTransaction: func(testEnv *testEnv, walletID string, h *wallet.DebitHold) wallet.PostTransaction {
			return wallet.PostTransaction{
				Script: &shared.V2PostTransactionScript{
					Plain: pointer.For(wallet.BuildDebitWalletScript(map[string]map[string]string{}, testEnv.Chart().GetMainBalanceAccount(walletID))),
					Vars: map[string]string{
						"destination": testEnv.Chart().GetBalanceAccount("wallet1", "secondary"),
						"amount":      "USD 100",
					},
				},
				Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.TransactionMetadata(nil)),
			}
		},
	},
}

func TestWalletsDebit(t *testing.T) {
	t.Parallel()
	for _, testCase := range walletDebitTestCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			walletID := uuid.NewString()

			req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", testCase.request)
			rec := httptest.NewRecorder()

			var (
				testEnv         *testEnv
				chart           *wallet.Chart
				ledgerName      string
				postTransaction wallet.PostTransaction
			)
			testEnv = newTestEnv(
				WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
					switch account {
					case chart.GetMainBalanceAccount(walletID):
						return &wallet.AccountWithVolumesAndBalances{
							Account: wallet.Account{
								Address: chart.GetMainBalanceAccount(walletID),
								Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
									Name: "main",
								}.LedgerMetadata(walletID)),
							},
						}, nil
					case chart.GetBalanceAccount(walletID, "coupon1"):
						return &wallet.AccountWithVolumesAndBalances{
							Account: wallet.Account{
								Address: chart.GetBalanceAccount(walletID, "coupon1"),
								Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
									Name:      "coupon1",
									ExpiresAt: ptr(time.Now().Add(5 * time.Second)),
								}.LedgerMetadata(walletID)),
							},
						}, nil
					case chart.GetBalanceAccount(walletID, "coupon2"):
						return &wallet.AccountWithVolumesAndBalances{
							Account: wallet.Account{
								Address: chart.GetBalanceAccount(walletID, "coupon2"),
								Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
									Name:     "coupon2",
									Priority: 10,
								}.LedgerMetadata(walletID)),
							},
						}, nil
					case chart.GetBalanceAccount(walletID, "coupon3"):
						return &wallet.AccountWithVolumesAndBalances{
							Account: wallet.Account{
								Address: chart.GetBalanceAccount(walletID, "coupon3"),
								Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
									Name:      "coupon3",
									ExpiresAt: ptr(time.Now().Add(-time.Minute)),
								}.LedgerMetadata(walletID)),
							},
						}, nil
					case chart.GetBalanceAccount(walletID, "coupon4"):
						return &wallet.AccountWithVolumesAndBalances{
							Account: wallet.Account{
								Address: chart.GetBalanceAccount(walletID, "coupon4"),
								Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
									Name: "coupon4",
								}.LedgerMetadata(walletID)),
							},
						}, nil
					case chart.GetBalanceAccount(walletID, "secondary"):
						return &wallet.AccountWithVolumesAndBalances{
							Account: wallet.Account{
								Address: chart.GetBalanceAccount(walletID, "secondary"),
								Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
									Name: "secondary",
								}.LedgerMetadata(walletID)),
							},
						}, nil
					default:
						return nil, errors.New("unexpected account: " + account)
					}
				}),
				WithListAccounts(func(ctx context.Context, ledger string, query wallet.ListAccountsQuery) (*wallet.AccountsCursorResponseCursor, error) {
					require.Equal(t, ledgerName, ledger)
					require.Equal(t, query.Metadata, wallet.BalancesMetadataFilter(walletID))

					return &wallet.AccountsCursorResponseCursor{
						Data: []wallet.AccountWithVolumesAndBalances{
							{
								Account: wallet.Account{
									Address: chart.GetBalanceAccount(walletID, "coupon2"),
									Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
										Name:     "coupon2",
										Priority: 10,
									}.LedgerMetadata(walletID)),
								},
							},
							{
								Account: wallet.Account{
									Address: chart.GetBalanceAccount(walletID, "coupon1"),
									Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
										Name:      "coupon1",
										ExpiresAt: ptr(time.Now().Add(5 * time.Second)),
									}.LedgerMetadata(walletID)),
								},
							},
							{
								Account: wallet.Account{
									Address: chart.GetBalanceAccount(walletID, "coupon3"),
									Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
										Name:      "coupon3",
										ExpiresAt: ptr(time.Now().Add(-time.Minute)),
									}.LedgerMetadata(walletID)),
								},
							},
							{
								Account: wallet.Account{
									Address: chart.GetBalanceAccount(walletID, "coupon4"),
									Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
										Name: "coupon4",
									}.LedgerMetadata(walletID)),
								},
							},
							{
								Account: wallet.Account{
									Address: chart.GetBalanceAccount(walletID, "main"),
									Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
										Name: "main",
									}.LedgerMetadata(walletID)),
								},
							},
						},
					}, nil
				}),
				WithCreateTransaction(func(ctx context.Context, ledger, ik string, p wallet.PostTransaction) (*shared.V2Transaction, error) {
					require.Equal(t, ledgerName, ledger)
					postTransaction = p
					if testCase.postTransactionError != nil {
						return nil, testCase.postTransactionError
					}
					//nolint:nilnil
					return nil, nil
				}),
			)
			chart = testEnv.Chart()
			ledgerName = testEnv.LedgerName()
			testEnv.Router().ServeHTTP(rec, req)

			expectedStatusCode := testCase.expectedStatusCode
			if expectedStatusCode == 0 {
				expectedStatusCode = http.StatusNoContent
			}
			require.Equal(t, expectedStatusCode, rec.Result().StatusCode)

			var hold *wallet.DebitHold
			switch expectedStatusCode {
			case http.StatusCreated:
				hold = &wallet.DebitHold{}
				readResponse(t, rec, hold)
			case http.StatusNoContent:
			default:
				errorResponse := readErrorResponse(t, rec)
				require.Equal(t, testCase.expectedErrorCode, errorResponse.ErrorCode)
				return
			}

			if testCase.expectedPostTransaction != nil {
				expectedPostTransaction := testCase.expectedPostTransaction(testEnv, walletID, hold)
				compareJSON(t, expectedPostTransaction, postTransaction)
			}

			if testCase.request.Pending {
				require.Equal(t, walletID, hold.WalletID)
				require.Equal(t, testCase.request.Amount.Asset, hold.Asset)
			}
		})
	}
}

func TestWalletsDebitPendingIdempotency(t *testing.T) {
	t.Parallel()

	const idempotencyKey = "debit-pending-key-1"
	walletID := uuid.NewString()

	var bodies []wallet.PostTransaction
	testEnv := newTestEnv(
		WithCreateTransaction(func(ctx context.Context, ledger, ik string, p wallet.PostTransaction) (*shared.V2Transaction, error) {
			require.Equal(t, idempotencyKey, ik)
			bodies = append(bodies, p)
			//nolint:nilnil
			return nil, nil
		}),
	)

	debit := func() *wallet.DebitHold {
		req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", wallet.DebitRequest{
			Amount:  wallet.NewMonetary(big.NewInt(100), "USD"),
			Pending: true,
		})
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
		hold := &wallet.DebitHold{}
		readResponse(t, rec, hold)
		return hold
	}

	// With a stable (explicit, non-expiring) source set, retrying a pending
	// debit under the same Idempotency-Key yields the same hold ID *and* a
	// byte-identical ledger request body — so the ledger sees a genuine replay,
	// not a conflict, and no duplicate hold is created.
	require.Equal(t, debit().ID, debit().ID)
	require.Len(t, bodies, 2)
	require.Equal(t, bodies[0], bodies[1])
}

func TestWalletsDebitWithIdempotencyKeyRejectsNonReplayableSources(t *testing.T) {
	t.Parallel()

	// A debit body resolved from live ledger state cannot be replayed
	// byte-for-byte: an expiring balance can cross its expiry boundary and a
	// wildcard set can change between attempts, so the ledger (which hashes the
	// body to enforce idempotency) would reject the retry as a conflict. We
	// therefore refuse such debits up front when an Idempotency-Key is present
	// rather than offer a false idempotency guarantee.
	for _, tc := range []struct {
		name     string
		balances []string
	}{
		{name: "expiring balance", balances: []string{"promo"}},
		{name: "wildcard balance", balances: []string{"*"}},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			const idempotencyKey = "debit-non-replayable-key-1"
			walletID := uuid.NewString()

			var (
				chart   *wallet.Chart
				created bool
			)
			testEnv := newTestEnv(
				WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
					return &wallet.AccountWithVolumesAndBalances{
						Account: wallet.Account{
							Address: account,
							Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
								Name:      "promo",
								ExpiresAt: ptr(time.Now().Add(time.Hour)),
							}.LedgerMetadata(walletID)),
						},
					}, nil
				}),
				WithListAccounts(func(ctx context.Context, ledger string, query wallet.ListAccountsQuery) (*wallet.AccountsCursorResponseCursor, error) {
					return &wallet.AccountsCursorResponseCursor{
						Data: []wallet.AccountWithVolumesAndBalances{
							{
								Account: wallet.Account{
									Address: chart.GetBalanceAccount(walletID, "main"),
									Metadata: metadataWithExpectingTypesAfterUnmarshalling(wallet.Balance{
										Name: "main",
									}.LedgerMetadata(walletID)),
								},
							},
						},
					}, nil
				}),
				WithCreateTransaction(func(ctx context.Context, ledger, ik string, p wallet.PostTransaction) (*shared.V2Transaction, error) {
					created = true
					//nolint:nilnil
					return nil, nil
				}),
			)
			chart = testEnv.Chart()

			req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", wallet.DebitRequest{
				Amount:   wallet.NewMonetary(big.NewInt(100), "USD"),
				Pending:  true,
				Balances: tc.balances,
			})
			req.Header.Set("Idempotency-Key", idempotencyKey)
			rec := httptest.NewRecorder()
			testEnv.Router().ServeHTTP(rec, req)

			require.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
			errorResponse := readErrorResponse(t, rec)
			require.Equal(t, ErrorCodeValidation, errorResponse.ErrorCode)
			require.False(t, created, "no transaction should be submitted to the ledger")
		})
	}
}
