package api

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHoldsGet(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	hold := wallet.NewDebitHold(walletID, wallet.NewLedgerAccountSubject("bank"),
		"USD", "", metadata.Metadata{})

	req := newRequest(t, http.MethodGet, "/holds/"+hold.ID, nil)
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
					"USD": big.NewInt(50),
				},
				Volumes: map[string]shared.V2Volume{
					"USD": {
						Input: big.NewInt(100),
					},
				},
			}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)

	ret := wallet.ExpandedDebitHold{}
	readResponse(t, rec, &ret)
	require.EqualValues(t, wallet.ExpandedDebitHold{
		DebitHold:      hold,
		OriginalAmount: big.NewInt(100),
		Remaining:      big.NewInt(50),
	}, ret)
}

func TestHoldsGetNotFound(t *testing.T) {
	t.Parallel()

	req := newRequest(t, http.MethodGet, "/holds/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()

	testEnv := newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			return nil, wallet.ErrAccountNotFound
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Result().StatusCode)
}

func TestHoldsGetWrongAccountType(t *testing.T) {
	t.Parallel()

	// A non-hold account (no hold metadata) must not be returned as a hold.
	req := newRequest(t, http.MethodGet, "/holds/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()

	testEnv := newTestEnv(
		WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
			return &wallet.AccountWithVolumesAndBalances{
				Account: wallet.Account{Metadata: map[string]string{}},
			}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNotFound, rec.Result().StatusCode)
}
