package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletsCreate(t *testing.T) {
	t.Parallel()

	createWalletRequest := wallet.CreateRequest{
		PatchRequest: wallet.PatchRequest{
			Metadata: metadata.Metadata{
				"foo": "bar",
			},
		},
		Name: uuid.NewString(),
	}

	req := newRequest(t, http.MethodPost, "/wallets", createWalletRequest)
	rec := httptest.NewRecorder()

	var (
		ledger  string
		account string
		md      map[string]string
	)
	testEnv := newTestEnv(
		WithAddMetadataToAccount(func(ctx context.Context, l, a, ik string, m map[string]string) error {
			ledger = l
			account = a
			md = m
			return nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	wallet := &wallet.Wallet{}
	readResponse(t, rec, wallet)
	require.Equal(t, testEnv.LedgerName(), ledger)
	require.Equal(t, account, testEnv.Chart().GetMainBalanceAccount(wallet.ID))
	require.Equal(t, wallet.LedgerMetadata(), md)
	require.Equal(t, wallet.Metadata, createWalletRequest.Metadata)
	require.Equal(t, wallet.Name, createWalletRequest.Name)
}

func TestWalletsCreateIdempotency(t *testing.T) {
	t.Parallel()

	const idempotencyKey = "create-wallet-key-1"

	// A real idempotent retry replays the *same* request body, so the payload
	// is fixed across both calls (a regenerated name would be a different body
	// with the same key, which the ledger treats as a conflict, not a replay).
	request := wallet.CreateRequest{Name: "savings-account"}

	var (
		created         bool
		forwardedKeys   []string
		targetAccount   string
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
		WithAddMetadataToAccount(func(ctx context.Context, l, a, ik string, m map[string]string) error {
			forwardedKeys = append(forwardedKeys, ik)
			targetAccount = a
			appliedMetadata = m
			created = true
			return nil
		}),
	)

	create := func() *wallet.Wallet {
		req := newRequest(t, http.MethodPost, "/wallets", request)
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
		w := &wallet.Wallet{}
		readResponse(t, rec, w)
		return w
	}

	first := create()
	second := create()

	// The first create writes once (forwarding the key); the retry under the
	// same key finds the persisted wallet and replays it rather than re-sending
	// a body whose CreatedAt would have drifted to time.Now() and been rejected
	// by the ledger's body-hash idempotency as a conflict.
	require.Equal(t, []string{idempotencyKey}, forwardedKeys)
	require.Equal(t, targetAccount, testEnv.Chart().GetMainBalanceAccount(first.ID))
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.Name, second.Name)
	require.Equal(t, first.CreatedAt, second.CreatedAt)
}

func TestWalletsCreateConcurrentReplaysOnWriteRejection(t *testing.T) {
	t.Parallel()

	const idempotencyKey = "create-wallet-concurrent-key"
	request := wallet.CreateRequest{Name: "savings-account"}

	// Model two concurrent creates under the same key: both existence checks
	// miss (the account is not yet visible), so both reach the ledger. The first
	// commits; the second submits a body with a different CreatedAt, which the
	// ledger rejects. The rejection may surface as VALIDATION (key replayed with
	// a different body hash) or CONFLICT (simultaneous insert), so the manager
	// does not classify it: it re-checks existence and replays the persisted
	// wallet because the request was in fact identical.
	var (
		getCalls        int
		addCalls        int
		appliedMetadata map[string]string
	)
	testEnv := newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			getCalls++
			// The first two reads are the racing pre-write existence checks.
			if getCalls <= 2 {
				return nil, wallet.ErrAccountNotFound
			}
			return &wallet.AccountWithVolumesAndBalances{
				Account: wallet.Account{
					Address:  account,
					Metadata: metadataWithExpectingTypesAfterUnmarshalling(appliedMetadata),
				},
			}, nil
		}),
		WithAddMetadataToAccount(func(ctx context.Context, l, a, ik string, m map[string]string) error {
			addCalls++
			if addCalls == 1 {
				appliedMetadata = m
				return nil
			}
			// The ledger's body-hash mismatch under a reused key surfaces as a
			// VALIDATION error (not CONFLICT) in the common case.
			return errors.New("ledger: idempotency key reused with a different request (VALIDATION)")
		}),
	)

	create := func() *wallet.Wallet {
		req := newRequest(t, http.MethodPost, "/wallets", request)
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
		w := &wallet.Wallet{}
		readResponse(t, rec, w)
		return w
	}

	first := create()
	second := create()

	// Both attempts tried to write (the second was rejected by the ledger), and
	// the second resolved to a replay of the persisted wallet rather than a 500.
	require.Equal(t, 2, addCalls)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, first.Name, second.Name)
	require.Equal(t, first.CreatedAt, second.CreatedAt)
}

func TestWalletsCreateIdempotencyKeyConflict(t *testing.T) {
	t.Parallel()

	const idempotencyKey = "create-wallet-conflict-key"

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
		WithAddMetadataToAccount(func(ctx context.Context, l, a, ik string, m map[string]string) error {
			addCalls++
			appliedMetadata = m
			created = true
			return nil
		}),
	)

	create := func(name string) *httptest.ResponseRecorder {
		req := newRequest(t, http.MethodPost, "/wallets", wallet.CreateRequest{Name: name})
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		return rec
	}

	require.Equal(t, http.StatusCreated, create("first-name").Result().StatusCode)

	// Reusing the same key with a different body is reported as a 409 conflict
	// rather than silently replaying the original wallet, and no second write is
	// sent to the ledger.
	second := create("different-name")
	require.Equal(t, http.StatusConflict, second.Result().StatusCode)
	require.Equal(t, ErrorCodeConflict, readErrorResponse(t, second).ErrorCode)
	require.Equal(t, 1, addCalls)
}

func TestWalletsCreateIdempotentReplayAfterPatch(t *testing.T) {
	t.Parallel()

	const idempotencyKey = "create-wallet-patched-key"
	request := wallet.CreateRequest{
		PatchRequest: wallet.PatchRequest{Metadata: metadata.Metadata{"foo": "bar"}},
		Name:         "savings-account",
	}

	var (
		created         bool
		addCalls        int
		appliedMetadata map[string]string
	)
	testEnv := newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			if !created {
				return nil, wallet.ErrAccountNotFound
			}
			// Simulate a post-create patch: the wallet's live custom metadata has
			// changed, but the stored create-request fingerprint is immutable.
			patched := map[string]string{}
			for k, v := range appliedMetadata {
				patched[k] = v
			}
			patched[wallet.MetadataKeyWalletCustomDataPrefix+"foo"] = "patched-value"
			return &wallet.AccountWithVolumesAndBalances{
				Account: wallet.Account{
					Address:  account,
					Metadata: metadataWithExpectingTypesAfterUnmarshalling(patched),
				},
			}, nil
		}),
		WithAddMetadataToAccount(func(ctx context.Context, l, a, ik string, m map[string]string) error {
			addCalls++
			appliedMetadata = m
			created = true
			return nil
		}),
	)

	create := func() *httptest.ResponseRecorder {
		req := newRequest(t, http.MethodPost, "/wallets", request)
		req.Header.Set("Idempotency-Key", idempotencyKey)
		rec := httptest.NewRecorder()
		testEnv.Router().ServeHTTP(rec, req)
		return rec
	}

	require.Equal(t, http.StatusCreated, create().Result().StatusCode)

	// Retrying the original create after the wallet was patched must still replay,
	// not 409: the match is against the immutable create fingerprint, not the
	// now-changed live metadata.
	second := create()
	require.Equal(t, http.StatusCreated, second.Result().StatusCode)
	require.Equal(t, 1, addCalls)
}
