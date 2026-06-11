package api

import (
	"context"
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
		forwardedKeys    []string
		targetedAccounts []string
	)
	testEnv := newTestEnv(
		WithAddMetadataToAccount(func(ctx context.Context, l, a, ik string, m map[string]string) error {
			forwardedKeys = append(forwardedKeys, ik)
			targetedAccounts = append(targetedAccounts, a)
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

	// The Idempotency-Key is forwarded to the ledger and the derived wallet ID
	// (hence the targeted account) is stable across retries, so the retry hits
	// the same account instead of creating a duplicate wallet.
	require.Equal(t, []string{idempotencyKey, idempotencyKey}, forwardedKeys)
	require.Equal(t, first.ID, second.ID)
	require.Equal(t, targetedAccounts[0], targetedAccounts[1])
}
