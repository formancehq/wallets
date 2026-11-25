package wallet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerBackendSelection(t *testing.T) {
	const v3 = `{"version":"3.0.0-rc.3","protocolVersion":"40"}`
	const v2 = `{"data":{"server":"ledger","version":"v2.4.0"}}`
	for _, tc := range []struct {
		name, configured, mount, route, body string
		status                               int
		expected                             Ledger
		message                              string
	}{
		{name: "default uses v2", expected: &DefaultLedger{}},
		{name: "explicit v2", configured: "v2", mount: "/ledger", expected: &DefaultLedger{}},
		{name: "explicit v3", configured: "v3", expected: &ledgerV3{}},
		{name: "v3 direct", configured: "auto", route: "/_info", body: v3, expected: &ledgerV3{}},
		{name: "v3 gateway", configured: "auto", mount: "/api/ledger", route: "/_info", body: v3, expected: &ledgerV3{}},
		{name: "v2 direct", configured: "auto", route: "/_info", body: v2, expected: &DefaultLedger{}},
		{name: "v2 gateway legacy alias", configured: "auto", mount: "/api/ledger", route: "/_/info", body: v2, expected: &DefaultLedger{}},
		{name: "versioned v2 route", configured: "auto", route: "/v2/_info", body: v2, expected: &DefaultLedger{}},
		{name: "unsupported major", configured: "auto", route: "/_info", body: `{"version":"4.0.0","protocolVersion":"50"}`, message: "unrecognized"},
		{name: "non-Ledger info", configured: "auto", route: "/_info", body: `{"version":"3.0.0"}`, message: "unrecognized"},
		{name: "authorization failure", configured: "auto", route: "/_info", status: 401, message: "401"},
		{name: "server failure", configured: "auto", route: "/_info", status: 503, message: "503"},
		{name: "route mismatch", configured: "auto", message: "no Ledger version endpoint"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if tc.configured != "auto" || r.Method != http.MethodGet {
					t.Errorf("unexpected Ledger request %s %s", r.Method, r.URL)
				}
				if tc.route == "" || r.URL.Path != tc.mount+tc.route {
					http.NotFound(w, r)
					return
				}
				if tc.status != 0 {
					w.WriteHeader(tc.status)
				}
				_, _ = fmt.Fprint(w, tc.body)
			}))
			t.Cleanup(srv.Close)

			options := LedgerOptions{APIVersion: tc.configured}
			if tc.configured != "" {
				options.URL = srv.URL + tc.mount
			}
			got, err := selectLedger(t.Context(), srv.Client(), srv.URL, options)
			if tc.message != "" {
				require.ErrorContains(t, err, tc.message)
				return
			}
			require.NoError(t, err)
			require.IsType(t, tc.expected, got)
		})
	}
}

func TestLedgerV2ServiceURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/ledger/_info":
			_, _ = fmt.Fprint(w, `{"data":{"server":"ledger","version":"2.4.0"}}`)
		case "/ledger/v2/test":
			_, _ = fmt.Fprint(w, `{"data":{"name":"test","bucket":"test","addedAt":"2026-11-25T00:00:00Z"}}`)
		default:
			t.Errorf("unexpected v2 path %s", r.URL)
			http.NotFound(w, r)
			return
		}
	}))
	t.Cleanup(srv.Close)
	got, err := selectLedger(t.Context(), srv.Client(), "", LedgerOptions{APIVersion: "auto", URL: srv.URL + "/ledger"})
	require.NoError(t, err)
	require.NoError(t, got.EnsureLedgerExists(t.Context(), "test"))
}

func TestLedgerV3CursorKey(t *testing.T) {
	options := LedgerOptions{APIVersion: "v3", URL: "http://localhost:9000"}
	first, err := selectLedger(t.Context(), http.DefaultClient, "", options)
	require.NoError(t, err)
	second, err := selectLedger(t.Context(), http.DefaultClient, "", options)
	require.NoError(t, err)
	require.Len(t, first.(*ledgerV3).cursorKey, 32)
	require.NotEqual(t, first.(*ledgerV3).cursorKey, second.(*ledgerV3).cursorKey)

	options.V3CursorKey = "too-short"
	_, err = selectLedger(t.Context(), http.DefaultClient, "", options)
	require.ErrorContains(t, err, "at least 32 bytes")

	options.V3CursorKey = "the-same-signing-key-on-all-replicas"
	shared, err := selectLedger(t.Context(), http.DefaultClient, "", options)
	require.NoError(t, err)
	require.Equal(t, []byte(options.V3CursorKey), shared.(*ledgerV3).cursorKey)
}
