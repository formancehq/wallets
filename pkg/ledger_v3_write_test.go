package wallet

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/stretchr/testify/require"
)

func TestLedgerV3ReplaysTheSameWriteAfterLostResponse(t *testing.T) {
	const hold = "wallets:holds:abc"
	const subject = `{"type":"ACCOUNT","identifier":"world"}`
	var firstBody, firstKey string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request: %v", err)
		}
		key := r.Header.Get("Idempotency-Key")
		if r.Method != http.MethodPost || r.URL.Path != "/v3/wallets-a/transactions" || key == "" {
			t.Errorf("unexpected Ledger write %s %s", r.Method, r.URL)
		}
		if firstKey == "" {
			firstKey, firstBody = key, string(data)
			conn, _, err := w.(http.Hijacker).Hijack()
			if err != nil {
				t.Errorf("drop response: %v", err)
				return
			}
			_ = conn.Close()
			return
		}
		if key != firstKey || string(data) != firstBody {
			t.Error("retry changed the key or request")
		}
		w.WriteHeader(http.StatusCreated)
		_, _ = fmt.Fprint(w, `{"data":{"transaction":{"id":7,"timestamp":"2026-11-25T11:00:00Z","postings":[]}}}`)
	}))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	adapter := &ledgerV3{client: srv.Client(), base: base}

	request := PostTransaction{
		Script: &shared.V2PostTransactionScript{Plain: pointer.For("v2 metadata script"),
			Vars: map[string]string{"destination": hold, "amount": "USD/2 10"}},
		accountMetadata: map[string]map[string]string{hold: {MetadataKeyHoldSubject: subject}},
		debitSources:    []string{"wallets:123:main"},
	}
	_, err = adapter.CreateTransaction(t.Context(), "wallets-a", "credit", request)
	require.Error(t, err)
	result, err := adapter.CreateTransaction(t.Context(), "wallets-a", "credit", request)
	require.NoError(t, err)
	require.Equal(t, "7", result.ID.String())
	require.NotEqual(t, v3IdempotencyKey("wallets-b", "credit"), firstKey)
	require.Empty(t, v3IdempotencyKey("wallets-a", ""))

	var sent v3TransactionRequest
	require.NoError(t, json.Unmarshal([]byte(firstBody), &sent))
	require.Equal(t, subject, sent.AccountMetadata[hold][MetadataKeyHoldSubject])
	require.True(t, strings.Contains(*sent.Script.Plain, "@wallets:123:main"))
	require.False(t, strings.Contains(*sent.Script.Plain, "set_account_meta"))
}

func TestLedgerV3WriteErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		status int
		code   string
		err    error
	}{
		{http.StatusBadRequest, "VALIDATION", ErrValidation},
		{http.StatusBadRequest, "TRANSACTION_REFERENCE_TOO_LONG", ErrValidation},
		{http.StatusBadRequest, "INSUFFICIENT_FUNDS", ErrInsufficientFundError},
		{http.StatusConflict, "TRANSACTION_REFERENCE_CONFLICT", ErrTransactionReferenceConflict},
		{http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT", ErrIdempotencyConflict},
	} {
		t.Run(tc.code, func(t *testing.T) {
			require.ErrorIs(t, v3WriteError(&ledgerV3Error{Status: tc.status, Code: tc.code}), tc.err)
		})
	}
}
