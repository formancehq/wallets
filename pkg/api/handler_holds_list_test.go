package api

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/go-libs/sharedapi"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestHoldsList(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()

	holds := make([]core.DebitHold, 0)
	for i := 0; i < 10; i++ {
		holds = append(holds, core.NewDebitHold(walletID, "bank", "USD"))
	}
	const pageSize = 2
	numberOfPages := int64(len(holds) / pageSize)

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithListAccounts(func(ctx context.Context, ledger string, query wallet.ListAccountQuery) (*sdk.ListAccounts200ResponseCursor, error) {
			if query.PaginationToken != "" {
				page, err := strconv.ParseInt(query.PaginationToken, 10, 64)
				if err != nil {
					panic(err)
				}

				if page >= numberOfPages-1 {
					return &sdk.ListAccounts200ResponseCursor{}, nil
				}
				hasMore := page < numberOfPages-1
				previous := fmt.Sprint(page - 1)
				next := fmt.Sprint(page + 1)
				accounts := make([]sdk.Account, 0)
				for _, hold := range holds[page*pageSize : (page+1)*pageSize] {
					accounts = append(accounts, sdk.Account{
						Address:  testEnv.Chart().GetMainAccount(hold.ID),
						Metadata: hold.LedgerMetadata(testEnv.Chart()),
					})
				}
				return &sdk.ListAccounts200ResponseCursor{
					PageSize: pageSize,
					HasMore:  &hasMore,
					Previous: &previous,
					Next:     &next,
					Data:     accounts,
				}, nil
			}

			require.Equal(t, pageSize, query.Limit)
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.EqualValues(t, core.Metadata{
				core.MetadataKeySpecType:     core.HoldWallet,
				core.MetadataKeyHoldWalletID: walletID,
			}, query.Metadata)

			hasMore := true
			next := "1"
			accounts := make([]sdk.Account, 0)
			for _, wallet := range holds[:pageSize] {
				accounts = append(accounts, sdk.Account{
					Address:  testEnv.Chart().GetMainAccount(wallet.ID),
					Metadata: wallet.LedgerMetadata(testEnv.Chart()),
				})
			}
			return &sdk.ListAccounts200ResponseCursor{
				PageSize: pageSize,
				HasMore:  &hasMore,
				Next:     &next,
				Data:     accounts,
			}, nil
		}),
	)
	req := newRequest(t, http.MethodGet, fmt.Sprintf("/wallets/%s/holds?limit=%d", walletID, pageSize), nil)
	rec := httptest.NewRecorder()
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Result().StatusCode)
	cursor := &sharedapi.Cursor[core.DebitHold]{}
	readCursor(t, rec, cursor)
	require.Len(t, cursor.Data, pageSize)
	require.EqualValues(t, holds[:pageSize], cursor.Data)

	req = newRequest(t, http.MethodGet, fmt.Sprintf("/wallets/%s/holds?cursor=%s", walletID, cursor.Next), nil)
	rec = httptest.NewRecorder()
	testEnv.Router().ServeHTTP(rec, req)

	cursor = &sharedapi.Cursor[core.DebitHold]{}
	readCursor(t, rec, cursor)
	require.Len(t, cursor.Data, pageSize)
	require.EqualValues(t, holds[pageSize:pageSize*2], cursor.Data)
}
