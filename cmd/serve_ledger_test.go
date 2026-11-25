package cmd

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLedgerClientBoundsTokenAcquisition(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/oauth/token", func(_ http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("read OAuth request: %v", err)
			return
		}
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	})
	mux.HandleFunc("GET /api/ledger/_info", func(w http.ResponseWriter, r *http.Request) {
		t.Error("Ledger request reached the server without a token")
		w.WriteHeader(http.StatusUnauthorized)
	})
	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.CloseClientConnections()
		server.Close()
	})

	client, err := getHTTPClient(t.Context(), "client", "secret", server.URL, false, 30*time.Millisecond)
	require.NoError(t, err)
	require.Zero(t, client.Timeout)
	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, server.URL+"/api/ledger/_info", nil)
	require.NoError(t, err)
	started := time.Now()
	_, err = client.Do(req)
	var timeout net.Error
	require.ErrorAs(t, err, &timeout)
	require.True(t, timeout.Timeout())
	require.Less(t, time.Since(started), time.Second)
}
