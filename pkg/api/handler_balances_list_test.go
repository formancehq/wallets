package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	sharedapi "github.com/formancehq/go-libs/v3/bun/bunpaginate"

	"github.com/formancehq/go-libs/v3/pointer"

	"github.com/formancehq/go-libs/v3/metadata"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBalancesList(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	var balances []wallet.Balance
	for i := 0; i < 10; i++ {
		balances = append(balances, wallet.NewBalance(uuid.NewString(), nil))
	}
	const pageSize = 2
	numberOfPages := int64(len(balances) / pageSize)

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithListAccounts(func(ctx context.Context, ledger string, query wallet.ListAccountsQuery) (*wallet.AccountsCursorResponseCursor, error) {
			if query.Cursor != "" {
				page, err := strconv.ParseInt(query.Cursor, 10, 64)
				if err != nil {
					panic(err)
				}

				if page >= numberOfPages-1 {
					return &wallet.AccountsCursorResponseCursor{
						Data: make([]wallet.AccountWithVolumesAndBalances, 0),
					}, nil
				}
				hasMore := page < numberOfPages-1
				previous := fmt.Sprint(page - 1)
				next := fmt.Sprint(page + 1)
				accounts := make([]wallet.AccountWithVolumesAndBalances, 0)
				for _, balance := range balances[page*pageSize : (page+1)*pageSize] {
					accounts = append(accounts, wallet.AccountWithVolumesAndBalances{
						Account: wallet.Account{
							Address:  testEnv.Chart().GetBalanceAccount(walletID, balance.Name),
							Metadata: metadataWithExpectingTypesAfterUnmarshalling(balance.LedgerMetadata(walletID)),
						},
					})
				}
				return &wallet.AccountsCursorResponseCursor{
					Data:     accounts,
					PageSize: pageSize,
					HasMore:  hasMore,
					Previous: pointer.For(previous),
					Next:     pointer.For(next),
				}, nil
			}

			require.Equal(t, pageSize, query.Limit)
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, metadata.Metadata{
				wallet.MetadataKeyWalletBalance: wallet.TrueValue,
				wallet.MetadataKeyWalletID:      walletID,
			}, query.Metadata)

			accounts := make([]wallet.AccountWithVolumesAndBalances, 0)
			for _, balance := range balances[:pageSize] {
				accounts = append(accounts, wallet.AccountWithVolumesAndBalances{
					Account: wallet.Account{
						Address:  testEnv.Chart().GetBalanceAccount(walletID, balance.Name),
						Metadata: metadataWithExpectingTypesAfterUnmarshalling(balance.LedgerMetadata(walletID)),
					},
				})
			}
			return &wallet.AccountsCursorResponseCursor{
				PageSize: pageSize,
				HasMore:  true,
				Next:     pointer.For("1"),
				Data:     accounts,
			}, nil
		}),
	)

	req := newRequest(t, http.MethodGet, fmt.Sprintf("/wallets/%s/balances?pageSize=%d", walletID, pageSize), nil)
	rec := httptest.NewRecorder()
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)
	cursor := &sharedapi.Cursor[wallet.Balance]{}
	readCursor(t, rec, cursor)
	require.Len(t, cursor.Data, pageSize)
	require.EqualValues(t, cursor.Data, balances[:pageSize])

	req = newRequest(t, http.MethodGet, fmt.Sprintf("/wallets/%s/balances?cursor=%s", walletID, cursor.Next), nil)
	rec = httptest.NewRecorder()
	testEnv.Router().ServeHTTP(rec, req)
	cursor = &sharedapi.Cursor[wallet.Balance]{}
	readCursor(t, rec, cursor)
	require.Len(t, cursor.Data, pageSize)
	require.EqualValues(t, cursor.Data, balances[pageSize:pageSize*2])
}
