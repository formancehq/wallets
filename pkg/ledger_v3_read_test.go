package wallet

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerV3ReadAccounts(t *testing.T) {
	const address = "wallets:1:main"
	const amount = "9007199254740993"
	const account = `{"address":"wallets:1:main","metadata":{"wallets/spec/type":"wallets.primary","external":true},"volumes":[{"asset":"USD/2","color":"","volumes":{"input":9007199254740993,"output":0}},{"asset":"USD/2","color":"RED","volumes":{"input":9,"output":0}}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/ledger/v3/wallets/accounts" || r.URL.Query().Has("expand") || r.URL.Query().Get("pageSize") != "1" {
			t.Errorf("unexpected Ledger account request %s", r.URL)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.URL.Query().Get("filter") == `{"$match":{"address":"missing"}}` {
			_, _ = fmt.Fprint(w, `{"data":[],"hasMore":false}`)
			return
		}
		switch r.URL.Query().Get("filter") {
		case `{"$match":{"address":"bad-response"}}`:
			_, _ = fmt.Fprint(w, `{}`)
			return
		case `{"$match":{"address":"wallets:1:main"}}`:
			_, _ = fmt.Fprintf(w, `{"data":[%s],"hasMore":false}`, account)
			return
		}
		if r.URL.Query().Get("cursor") == "next-ledger-page" {
			_, _ = fmt.Fprintf(w, `{"data":[%s],"previous":"previous-ledger-page","hasMore":false}`, account)
			return
		}
		_, _ = fmt.Fprintf(w, `{"data":[%s],"next":"next-ledger-page","hasMore":true}`, account)
	}))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL + "/api/ledger")
	require.NoError(t, err)
	key := []byte("test-signing-key")
	adapter := &ledgerV3{base: base, client: srv.Client(), cursorKey: key}
	_, err = adapter.GetAccount(t.Context(), "wallets", "missing")
	require.ErrorIs(t, err, ErrAccountNotFound)
	_, err = adapter.GetAccount(t.Context(), "wallets", "bad-response")
	require.ErrorContains(t, err, "missing its data array")
	require.NotErrorIs(t, err, ErrAccountNotFound)
	got, err := adapter.GetAccount(t.Context(), "wallets", address)
	require.NoError(t, err)
	require.Equal(t, amount, got.Balances["USD/2"].String())
	require.Equal(t, map[string]string{MetadataKeyWalletSpecType: PrimaryWallet}, got.Metadata)

	input := ListAccountsQuery{Limit: 1, ExpandVolumes: true, scope: "wallets",
		Metadata: map[string]string{MetadataKeyWalletSpecType: PrimaryWallet}}
	first, err := adapter.ListAccounts(t.Context(), "wallets", input)
	require.NoError(t, err)
	require.True(t, first.HasMore)
	require.NotNil(t, first.Next)
	input.Cursor = *first.Next
	input.Limit = 15
	input.ExpandVolumes = false
	second, err := adapter.ListAccounts(t.Context(), "wallets", input)
	require.NoError(t, err)
	require.EqualValues(t, 1, second.PageSize)
	require.Equal(t, amount, second.Data[0].Balances["USD/2"].String())
	require.NotNil(t, second.Previous)

	otherBackend := &ledgerV3{base: base.JoinPath("other"), client: srv.Client(), cursorKey: key}
	_, err = otherBackend.ListAccounts(t.Context(), "wallets", input)
	require.ErrorIs(t, err, ErrValidation)

	encoded := strings.TrimPrefix(input.Cursor, v3CursorPrefix)
	payload, signature, _ := strings.Cut(encoded, ".")
	data, err := base64.RawURLEncoding.DecodeString(payload)
	require.NoError(t, err)
	modified := strings.Replace(string(data), "next-ledger-page", "forged-ledger-page", 1)
	forged := input
	forged.Cursor = v3CursorPrefix + base64.RawURLEncoding.EncodeToString([]byte(modified)) + "." + signature
	_, err = adapter.ListAccounts(t.Context(), "wallets", forged)
	require.ErrorIs(t, err, ErrValidation)

	input.scope = "holds"
	_, err = adapter.ListAccounts(t.Context(), "wallets", input)
	require.ErrorIs(t, err, ErrValidation)
}

func TestLedgerV3ReadTransactionPrecision(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/wallets/transactions" || r.URL.Query().Get("filter") != `{"$match":{"address":"wallets:1:main"}}` {
			t.Errorf("unexpected Ledger transaction request %s", r.URL)
		}
		_, _ = fmt.Fprint(w, `{"data":[{"id":9007199254740993,"timestamp":"2026-11-25T11:00:00Z","postings":[{"asset":"USD/2","amount":9007199254740993,"source":"world","destination":"wallets:1:main"}],"postCommitVolumes":{"wallets:1:main":[{"asset":"USD/2","input":9007199254740993,"output":0}]}}]}`)
	}))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	adapter := &ledgerV3{base: base, client: srv.Client()}
	got, err := adapter.ListTransactions(t.Context(), "wallets", ListTransactionsQuery{Limit: 1, Account: "wallets:1:main"})
	require.NoError(t, err)
	require.Len(t, got.Data, 1)
	payload, err := (Transaction{V2Transaction: got.Data[0], Ledger: "wallets"}).MarshalJSON()
	require.NoError(t, err)
	require.Contains(t, string(payload), `"id":9007199254740993`)
	require.Contains(t, string(payload), `"amount":9007199254740993`)
	require.Contains(t, string(payload), `"input":9007199254740993`)
}
