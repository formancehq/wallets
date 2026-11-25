package wallet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerV3PreparedHistoryPaging(t *testing.T) {
	const address = "wallets:1:main"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Parameters map[string]string
			Reverse    bool
			PageSize   int
			Cursor     string
		}
		err := json.NewDecoder(r.Body).Decode(&body)
		if err != nil || r.Method != http.MethodPost ||
			r.URL.Path != "/v3/test/prepared-queries/wallets-transactions-by-wallet-v1/execute" ||
			body.Parameters["walletAddress"] != address || !body.Reverse || body.PageSize != 1 {
			t.Errorf("unexpected prepared-query request %s, %+v, %v", r.URL, body, err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		switch body.Cursor {
		case "", "previous":
			_, _ = fmt.Fprint(w, `{"cursor":{"transactionData":[{"id":2,"timestamp":"2026-11-25T11:00:00Z"}],"next":"next","hasMore":true}}`)
		case "next":
			_, _ = fmt.Fprint(w, `{"cursor":{"transactionData":[{"id":1,"timestamp":"2026-11-25T11:00:00Z"}],"previous":"previous","hasMore":false}}`)
		default:
			t.Errorf("unexpected cursor %q", body.Cursor)
		}
	}))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	adapter := &ledgerV3{base: base, client: srv.Client()}
	input := ListTransactionsQuery{Limit: 1, Account: address, Metadata: TransactionBaseMetadataFilter()}
	first, err := adapter.ListTransactions(t.Context(), "test", input)
	require.NoError(t, err)
	require.Equal(t, "2", first.Data[0].ID.String())
	second, err := adapter.ListTransactions(t.Context(), "test", ListTransactionsQuery{Cursor: *first.Next})
	require.NoError(t, err)
	require.Equal(t, "1", second.Data[0].ID.String())
	previous, err := adapter.ListTransactions(t.Context(), "test", ListTransactionsQuery{Cursor: *second.Previous})
	require.NoError(t, err)
	require.Equal(t, "2", previous.Data[0].ID.String())
}

func TestLedgerV3PreparedQueryFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body, message string
		status              int
	}{
		{"missing index", `{"errorCode":"INDEX_NOT_FOUND","errorMessage":"transaction address index missing"}`, "INDEX_NOT_FOUND", http.StatusNotFound},
		{"malformed response", `{"data":[]}`, "missing its cursor", http.StatusOK},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = fmt.Fprint(w, tc.body)
			}))
			t.Cleanup(srv.Close)
			base, err := url.Parse(srv.URL)
			require.NoError(t, err)
			adapter := &ledgerV3{base: base, client: srv.Client()}
			_, err = adapter.ListTransactions(t.Context(), "test", ListTransactionsQuery{
				Account: "wallets:1:main", Metadata: TransactionBaseMetadataFilter(),
			})
			require.ErrorContains(t, err, tc.message)
			require.NotErrorIs(t, err, ErrValidation)
		})
	}
}
