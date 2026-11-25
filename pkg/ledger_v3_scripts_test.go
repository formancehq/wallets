package wallet

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/stretchr/testify/require"
)

func TestLedgerV3SavedScriptReplaysEarlierInlineRequest(t *testing.T) {
	var requests []v3TransactionRequest
	var keys []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request v3TransactionRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode transaction: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		requests = append(requests, request)
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		if request.ScriptReference != nil {
			w.WriteHeader(http.StatusConflict)
			_, _ = fmt.Fprint(w, `{"errorCode":"IDEMPOTENCY_KEY_CONFLICT"}`)
			return
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"data":{"transaction":{"id":7,"timestamp":"2026-11-25T11:00:00Z","postings":[]}}}`)
	}))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	adapter := &ledgerV3{client: srv.Client(), base: base}
	inline := &shared.V2PostTransactionScript{
		Plain: pointer.For(BuildCreditWalletScript("world")),
		Vars:  map[string]string{"destination": "wallets:1:main", "amount": "USD/2 10"},
	}
	result, err := adapter.CreateTransaction(t.Context(), "wallets", "credit",
		PostTransaction{Script: inline, libraryScript: scriptDefaultCredit})
	require.NoError(t, err)
	require.Equal(t, "7", result.ID.String())
	require.Equal(t, []string{v3IdempotencyKey("wallets", "credit"), v3IdempotencyKey("wallets", "credit")}, keys)
	require.Len(t, requests, 2)
	require.Equal(t, &v3ScriptReference{Name: scriptDefaultCredit, Version: walletScriptVersion, Vars: inline.Vars}, requests[0].ScriptReference)
	require.Equal(t, inline, requests[1].Script)
	require.Nil(t, requests[1].ScriptReference)
}
