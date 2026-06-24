package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/types/time"

	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func ptr[V any](v V) *V {
	return &v
}

type balanceCreateTestCase struct {
	name               string
	request            wallet.CreateBalance
	expectedStatusCode int
	expectedErrorCode  string
}

var balanceCreateTestCases = []balanceCreateTestCase{
	{
		name: "nominal",
		request: wallet.CreateBalance{
			Name: uuid.NewString(),
		},
	},
	{
		name: "with invalid name",
		request: wallet.CreateBalance{
			Name: "!!!!!!!",
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
	{
		name: "with reserved name",
		request: wallet.CreateBalance{
			Name: wallet.MainBalance,
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
	{
		name: "with expiration",
		request: wallet.CreateBalance{
			Name:      wallet.MainBalance,
			ExpiresAt: ptr(time.Now().Add(10 * time.Second)),
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
}

func TestBalancesCreateForwardsIdempotencyKey(t *testing.T) {
	t.Parallel()

	const idempotencyKey = "create-balance-key-1"
	walletID := uuid.NewString()

	var forwardedKey string
	testEnv := newTestEnv(
		WithAddMetadataToAccount(func(ctx context.Context, ledger, account, ik string, metadata map[string]string) error {
			forwardedKey = ik
			return nil
		}),
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			return nil, wallet.ErrAccountNotFound
		}),
	)

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/balances", wallet.CreateBalance{Name: uuid.NewString()})
	req.Header.Set("Idempotency-Key", idempotencyKey)
	rec := httptest.NewRecorder()
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	require.Equal(t, idempotencyKey, forwardedKey)
}

func TestBalancesCreateIdempotentReplay(t *testing.T) {
	t.Parallel()

	const idempotencyKey = "create-balance-key-replay"
	walletID := uuid.NewString()
	const balanceName = "savings"

	var (
		created         bool
		addCalls        int
		appliedMetadata map[string]string
	)
	testEnv := newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			if created {
				return &wallet.AccountWithVolumesAndBalances{
					Account: wallet.Account{
						Address:  account,
						Metadata: metadataWithExpectingTypesAfterUnmarshalling(appliedMetadata),
					},
				}, nil
			}
			return nil, wallet.ErrAccountNotFound
		}),
		WithAddMetadataToAccount(func(ctx context.Context, ledger, account, ik string, md map[string]string) error {
			addCalls++
			appliedMetadata = md
			created = true
			return nil
		}),
	)

	create := func() *httptest.ResponseRecorder {
		req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/balances", wallet.CreateBalance{Name: balanceName})
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		return rec
	}

	first := create()
	require.Equal(t, http.StatusCreated, first.Result().StatusCode)

	// The retry under the same key replays the existing balance (201) instead
	// of failing with 400 ALREADY_EXISTS, and does not re-write metadata.
	second := create()
	require.Equal(t, http.StatusCreated, second.Result().StatusCode)
	require.Equal(t, 1, addCalls)

	b1, b2 := &wallet.Balance{}, &wallet.Balance{}
	readResponse(t, first, b1)
	readResponse(t, second, b2)
	require.Equal(t, balanceName, b1.Name)
	require.Equal(t, b1.Name, b2.Name)
}

func TestBalancesCreateWithDifferentIdempotencyKeyDoesNotReplay(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	const balanceName = "savings"

	var (
		created         bool
		addCalls        int
		appliedMetadata map[string]string
	)
	testEnv := newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			if created {
				return &wallet.AccountWithVolumesAndBalances{
					Account: wallet.Account{
						Address:  account,
						Metadata: metadataWithExpectingTypesAfterUnmarshalling(appliedMetadata),
					},
				}, nil
			}
			return nil, wallet.ErrAccountNotFound
		}),
		WithAddMetadataToAccount(func(ctx context.Context, ledger, account, ik string, md map[string]string) error {
			addCalls++
			appliedMetadata = md
			created = true
			return nil
		}),
	)

	create := func(idempotencyKey string) *httptest.ResponseRecorder {
		req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/balances", wallet.CreateBalance{Name: balanceName})
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		return rec
	}

	first := create("create-balance-key-1")
	require.Equal(t, http.StatusCreated, first.Result().StatusCode)

	second := create("create-balance-key-2")
	require.Equal(t, http.StatusBadRequest, second.Result().StatusCode)
	require.Equal(t, 1, addCalls)
}

func TestBalancesCreateConcurrentCreatePreservesReplayForAllKeys(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	const balanceName = "savings"

	// Model the documented concurrent first-create race: two creates with
	// different keys both pass the existence check (GetAccount not-found) before
	// either writes, then both write. The ledger merges account metadata
	// additively, so the mock accumulates every write into one map. With per-key
	// replay markers both keys' markers survive the merge, so either caller can
	// still replay; a single shared marker field would have been overwritten by
	// the later write, breaking replay for the earlier caller.
	var (
		getCalls int
		addCalls int
		merged   = map[string]string{}
	)
	testEnv := newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			getCalls++
			// The first two checks (the racing creates) see no account yet.
			if getCalls <= 2 {
				return nil, wallet.ErrAccountNotFound
			}
			return &wallet.AccountWithVolumesAndBalances{
				Account: wallet.Account{
					Address:  account,
					Metadata: metadataWithExpectingTypesAfterUnmarshalling(merged),
				},
			}, nil
		}),
		WithAddMetadataToAccount(func(ctx context.Context, ledger, account, ik string, md map[string]string) error {
			addCalls++
			for k, v := range md {
				merged[k] = v
			}
			return nil
		}),
	)

	create := func(idempotencyKey string) int {
		req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/balances", wallet.CreateBalance{Name: balanceName})
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		return rec.Result().StatusCode
	}

	// Two racing first-time creates with different keys both succeed and both
	// write — the race the PR documents.
	require.Equal(t, http.StatusCreated, create("key-A"))
	require.Equal(t, http.StatusCreated, create("key-B"))
	require.Equal(t, 2, addCalls)

	// Both keys can now replay: neither marker was clobbered by the other.
	require.Equal(t, http.StatusCreated, create("key-A"))
	require.Equal(t, http.StatusCreated, create("key-B"))
	require.Equal(t, 2, addCalls, "replays must not write to the ledger again")
}

func TestBalancesCreate(t *testing.T) {
	t.Parallel()

	for _, testCase := range balanceCreateTestCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			walletID := uuid.NewString()
			req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/balances", testCase.request)
			rec := httptest.NewRecorder()

			var (
				targetedLedger  string
				targetedAccount string
				appliedMetadata map[string]string
			)
			testEnv := newTestEnv(
				WithAddMetadataToAccount(func(ctx context.Context, ledger, account, ik string, metadata map[string]string) error {
					targetedLedger = ledger
					targetedAccount = account
					appliedMetadata = metadata
					return nil
				}),
				WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
					return &wallet.AccountWithVolumesAndBalances{}, nil
				}),
			)
			testEnv.Router().ServeHTTP(rec, req)

			expectedStatusCode := testCase.expectedStatusCode
			if expectedStatusCode == 0 {
				expectedStatusCode = http.StatusCreated
			}
			require.Equal(t, expectedStatusCode, rec.Result().StatusCode)

			if expectedStatusCode == http.StatusCreated {
				balance := &wallet.Balance{}
				readResponse(t, rec, balance)
				require.Equal(t, testEnv.LedgerName(), targetedLedger)
				require.Equal(t, targetedAccount, testEnv.Chart().GetBalanceAccount(walletID, balance.Name))
				require.Equal(t, balance.LedgerMetadata(walletID), appliedMetadata)
				require.Equal(t, balance.Name, testCase.request.Name)
			} else {
				errorResponse := readErrorResponse(t, rec)
				require.Equal(t, testCase.expectedErrorCode, errorResponse.ErrorCode)
			}
		})
	}
}
