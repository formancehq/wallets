//go:build it

package ledgerv3_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/testing/testservice"
	"github.com/formancehq/go-libs/v5/pkg/transport/httpserver"
	"github.com/formancehq/wallets/cmd"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type smokeResponse struct {
	Data      json.RawMessage `json:"data"`
	ErrorCode string          `json:"errorCode"`
	Cursor    struct {
		Data            []json.RawMessage `json:"data"`
		AccountData     []json.RawMessage `json:"accountData"`
		TransactionData []json.RawMessage `json:"transactionData"`
		Next            string            `json:"next"`
		Previous        string            `json:"previous"`
		PageSize        int               `json:"pageSize"`
	} `json:"cursor"`
}

type smokeClient struct {
	t      *testing.T
	client *http.Client
	base   string
}

func (s smokeClient) call(method, path string, body any, key string, status int) smokeResponse {
	s.t.Helper()
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		require.NoError(s.t, err)
	}
	req, err := http.NewRequestWithContext(s.t.Context(), method, s.base+path, bytes.NewReader(payload))
	require.NoError(s.t, err)
	req.Header.Set("Content-Type", "application/json")
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	resp, err := s.client.Do(req)
	require.NoError(s.t, err)
	defer func() { _ = resp.Body.Close() }()
	var raw bytes.Buffer
	_, err = raw.ReadFrom(resp.Body)
	require.NoError(s.t, err)
	require.Equal(s.t, status, resp.StatusCode, "%s %s: %s", method, path, raw.String())
	var result smokeResponse
	if raw.Len() > 0 {
		require.NoError(s.t, json.Unmarshal(raw.Bytes(), &result))
	}
	return result
}

func decode[T any](t *testing.T, data json.RawMessage) T {
	t.Helper()
	var result T
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

func TestLedgerV3Smoke(t *testing.T) {
	ledgerURL := os.Getenv("WALLETS_LEDGER_V3_URL")
	if ledgerURL == "" {
		t.Skip("set WALLETS_LEDGER_V3_URL to a disposable Ledger v3 instance")
	}
	target, err := url.Parse(ledgerURL)
	require.NoError(t, err)
	client := &http.Client{Timeout: 8 * time.Second}

	t.Run("direct auto detection and native Wallets flow", func(t *testing.T) {
		name := "wallets-smoke-" + uuid.NewString()[:8]
		t.Logf("using disposable Ledger v3 ledger %s", name)
		ledger := smokeClient{t: t, base: ledgerURL, client: client}
		ledger.call(http.MethodGet, "/v3/"+name, nil, "", http.StatusNotFound)

		const cursorKey = "fixture-signing-key-shared-by-wallets"
		wallets := startWallets(t, client, name, "--ledger-url", ledgerURL, "--ledger-api-version", "auto",
			"--ledger-v3-indexed-metadata-keys", "tier", "--ledger-v3-cursor-key", cursorKey)
		id, heldID := walletFlow(t, wallets)
		requirePreparedQueryParity(t, ledger, name, id, heldID)
		accountPath := "/v3/" + name + "/accounts/" + wallet.NewChart("").GetMainBalanceAccount(id)
		account := ledger.call(http.MethodGet, accountPath, nil, "", http.StatusOK)
		require.JSONEq(t, `{"input":1300,"output":600,"balance":700}`,
			string(decode[struct {
				Volumes []struct{ Volumes json.RawMessage }
			}](t, account.Data).Volumes[0].Volumes))
		requireNames(t, ledger.call(http.MethodGet, "/v3/"+name+"/numscripts", nil, "", http.StatusOK).Data,
			"wallets-credit-world", "wallets-confirm-hold")
		requireNames(t, ledger.call(http.MethodGet, "/v3/"+name+"/prepared-queries", nil, "", http.StatusOK).Data,
			"wallets-balances-by-wallet-v1", "wallets-holds-by-wallet-v1", "wallets-hold-funding-v1",
			"wallets-by-name-v1", "wallets-transactions-by-wallet-v1")
		require.Len(t, decode[[]json.RawMessage](t, ledger.call(http.MethodGet, "/v3/"+name+"/indexes", nil, "", http.StatusOK).Data), 10)

		expiredAndMultiSource(t, wallets, id)
		page := wallets.call(http.MethodGet, "/transactions?walletID="+id+"&pageSize=1", nil, "", http.StatusOK).Cursor
		require.NotEmpty(t, page.Next)
		verified := startWallets(t, client, name, "--ledger-url", ledgerURL,
			"--ledger-api-version", "v3", "--ledger-v3-provisioning", "verify",
			"--ledger-v3-indexed-metadata-keys", "tier", "--ledger-v3-cursor-key", cursorKey)
		resumed := verified.call(http.MethodGet, "/transactions?cursor="+url.QueryEscape(page.Next), nil, "", http.StatusOK).Cursor
		require.Len(t, resumed.Data, 1)
		verified.call(http.MethodGet, "/wallets/"+id, nil, "", http.StatusOK)
		verified.call(http.MethodPost, "/wallets/"+id+"/credit", amount(1000), "credit", http.StatusNoContent)
		verified.call(http.MethodPost, "/holds/"+heldID+"/confirm", nil, "confirm", http.StatusNoContent)
		requireAvailable(t, verified, id, 730)
		creditRolloutReplay(t, ledger, name, verified)
	})

	t.Run("gateway mount with client credentials", func(t *testing.T) {
		const token = "wallets-smoke-bearer"
		proxy := httputil.NewSingleHostReverseProxy(target)
		router := http.NewServeMux()
		router.HandleFunc("POST /api/auth/oauth/token", func(w http.ResponseWriter, r *http.Request) {
			if err := r.ParseForm(); err != nil || r.Form.Get("grant_type") != "client_credentials" ||
				r.Form.Get("scope") != "openid ledger:read ledger:write" {
				t.Errorf("unexpected client credentials token request")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = fmt.Fprintf(w, `{"access_token":%q,"token_type":"Bearer","expires_in":3600}`, token)
		})
		router.Handle("/api/ledger/", http.StripPrefix("/api/ledger", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				t.Error("Ledger request is missing gateway authorization")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			proxy.ServeHTTP(w, r)
		})))
		gateway := httptest.NewServer(router)
		t.Cleanup(gateway.Close)

		name := "wallets-gateway-" + uuid.NewString()[:8]
		wallets := startWallets(t, client, name, "--stack-url", gateway.URL, "--ledger-api-version", "auto",
			"--stack-client-id", "wallets-smoke", "--stack-client-secret", "fixture")
		created := wallets.call(http.MethodPost, "/wallets", map[string]string{"name": "gateway"}, "wallet", http.StatusCreated)
		id := decode[struct{ ID string }](t, created.Data).ID
		wallets.call(http.MethodPost, "/wallets/"+id+"/credit", amount(30), "credit", http.StatusNoContent)
		requireAvailable(t, wallets, id, 30)
	})
}

func startWallets(t *testing.T, client *http.Client, ledger string, flags ...string) smokeClient {
	t.Helper()
	args := append([]string{"serve", "--ledger", ledger, "--listen", "127.0.0.1:0"}, flags...)
	service := testservice.New(cmd.NewRootCommand, testservice.WithInstruments(
		testservice.AppendArgsInstrumentation(args...),
		testservice.HTTPServerInstrumentation(),
	))
	require.NoError(t, service.Start(t.Context()))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		require.NoError(t, service.Stop(ctx))
	})
	return smokeClient{t: t, base: httpserver.URL(service.GetContext()), client: client}
}

func requireNames(t *testing.T, data json.RawMessage, names ...string) {
	t.Helper()
	got := decode[[]struct{ Name string }](t, data)
	actual := make([]string, 0, len(got))
	for _, entry := range got {
		actual = append(actual, entry.Name)
	}
	require.ElementsMatch(t, names, actual)
}

func amount(value int) map[string]any {
	return map[string]any{"amount": map[string]any{"amount": value, "asset": "USD/2"}}
}

func hold(t *testing.T, client smokeClient, id string, body map[string]any, key string) string {
	t.Helper()
	body["pending"] = true
	result := client.call(http.MethodPost, "/wallets/"+id+"/debit", body, key, http.StatusCreated)
	return "/holds/" + decode[struct{ ID string }](t, result.Data).ID
}

func requireAvailable(t *testing.T, client smokeClient, id string, expected int) {
	t.Helper()
	result := client.call(http.MethodGet, "/wallets/"+id+"/summary", nil, "", http.StatusOK)
	summary := decode[struct{ AvailableFunds, HoldFunds map[string]int }](t, result.Data)
	require.Equal(t, expected, summary.AvailableFunds["USD/2"])
	require.Zero(t, summary.HoldFunds["USD/2"])
}

func walletFlow(t *testing.T, client smokeClient) (walletID, holdID string) {
	t.Helper()
	input := map[string]any{"name": "smoke", "metadata": map[string]string{"tier": "gold"}}
	created := client.call(http.MethodPost, "/wallets", input, "create", http.StatusCreated)
	replay := client.call(http.MethodPost, "/wallets", input, "create", http.StatusCreated)
	require.JSONEq(t, string(created.Data), string(replay.Data))
	id := decode[struct{ ID string }](t, created.Data).ID
	require.NotEmpty(t, id)
	path := "/wallets/" + id
	client.call(http.MethodPost, path+"/balances", map[string]string{"name": "secondary"}, "balance", http.StatusCreated)
	client.call(http.MethodPost, path+"/credit", amount(1000), "credit", http.StatusNoContent)
	client.call(http.MethodPost, path+"/credit", amount(1000), "credit", http.StatusNoContent)
	conflict := client.call(http.MethodPost, path+"/credit", amount(1001), "credit", http.StatusConflict)
	require.Equal(t, "CONFLICT", conflict.ErrorCode)
	client.call(http.MethodPost, path+"/debit", amount(100), "debit", http.StatusNoContent)
	confirmed := hold(t, client, id, amount(200), "hold-confirm")
	before := client.call(http.MethodGet, confirmed, nil, "", http.StatusOK)
	require.Equal(t, 200, decode[struct{ Remaining int }](t, before.Data).Remaining)
	client.call(http.MethodPost, confirmed+"/confirm", nil, "confirm", http.StatusNoContent)
	client.call(http.MethodPost, confirmed+"/confirm", nil, "confirm", http.StatusNoContent)
	reconfirm := client.call(http.MethodPost, confirmed+"/confirm", map[string]int{"amount": 1}, "confirm", http.StatusConflict)
	require.Equal(t, "CONFLICT", reconfirm.ErrorCode)
	voided := hold(t, client, id, amount(300), "hold-void")
	client.call(http.MethodPost, voided+"/void", nil, "void", http.StatusNoContent)
	client.call(http.MethodPost, voided+"/void", nil, "void", http.StatusNoContent)
	revoid := client.call(http.MethodPost, voided+"/void", nil, "confirm", http.StatusConflict)
	require.Equal(t, "CONFLICT", revoid.ErrorCode)
	failure := client.call(http.MethodPost, path+"/debit", amount(9999), "insufficient", http.StatusBadRequest)
	require.Equal(t, "INSUFFICIENT_FUND", failure.ErrorCode)
	requireAvailable(t, client, id, 700)
	require.Len(t, client.call(http.MethodGet, path+"/balances", nil, "", http.StatusOK).Cursor.Data, 2)
	require.Len(t, client.call(http.MethodGet, "/wallets?name=smoke", nil, "", http.StatusOK).Cursor.Data, 1)
	require.Len(t, client.call(http.MethodGet, "/wallets?metadata%5Btier%5D=gold", nil, "", http.StatusOK).Cursor.Data, 1)
	client.call(http.MethodGet, "/wallets?metadata%5Bunknown%5D=gold", nil, "", http.StatusBadRequest)

	first := client.call(http.MethodGet, "/transactions?walletID="+id+"&pageSize=1", nil, "", http.StatusOK).Cursor
	require.Len(t, first.Data, 1)
	require.NotEmpty(t, first.Next)
	second := client.call(http.MethodGet, "/transactions?cursor="+url.QueryEscape(first.Next), nil, "", http.StatusOK).Cursor
	require.Len(t, second.Data, 1)
	require.Equal(t, 1, second.PageSize)
	require.NotEmpty(t, second.Previous)
	require.Greater(t, decode[struct{ ID int }](t, first.Data[0]).ID, decode[struct{ ID int }](t, second.Data[0]).ID)
	back := client.call(http.MethodGet, "/transactions?cursor="+url.QueryEscape(second.Previous), nil, "", http.StatusOK).Cursor
	require.Len(t, back.Data, 1)
	require.JSONEq(t, string(first.Data[0]), string(back.Data[0]))
	client.call(http.MethodGet, "/transactions?cursor=invalid", nil, "", http.StatusBadRequest)
	return id, confirmed[len("/holds/"):]
}

func expiredAndMultiSource(t *testing.T, client smokeClient, id string) {
	t.Helper()
	path := "/wallets/" + id
	client.call(http.MethodPost, path+"/balances", map[string]any{
		"name": "expired", "expiresAt": time.Now().Add(-time.Hour).UTC(), "priority": 9,
	}, "", http.StatusCreated)
	for balance, funds := range map[string]int{"secondary": 80, "expired": 900} {
		credit := amount(funds)
		credit["balance"] = balance
		client.call(http.MethodPost, path+"/credit", credit, "credit-"+balance, http.StatusNoContent)
	}
	debit := amount(120)
	debit["balances"] = []string{"secondary", "main"}
	multi := hold(t, client, id, debit, "multi-source")
	secondaryPath := path + "/balances/secondary"
	depleted := client.call(http.MethodGet, secondaryPath, nil, "", http.StatusOK)
	require.Zero(t, decode[struct{ Assets map[string]int }](t, depleted.Data).Assets["USD/2"])
	client.call(http.MethodPost, multi+"/void", nil, "multi-source-void", http.StatusNoContent)
	repaid := client.call(http.MethodGet, secondaryPath, nil, "", http.StatusOK)
	require.Equal(t, 80, decode[struct{ Assets map[string]int }](t, repaid.Data).Assets["USD/2"])

	wildcard := amount(110)
	wildcard["balances"] = []string{"*"}
	wildcard["destination"] = map[string]string{"type": "ACCOUNT", "identifier": "external:merchant"}
	pending := hold(t, client, id, wildcard, "")
	client.call(http.MethodPost, pending+"/confirm", map[string]any{"amount": 30}, "partial", http.StatusNoContent)
	client.call(http.MethodPost, pending+"/confirm", map[string]any{"amount": 20, "final": true}, "final", http.StatusNoContent)
	requireAvailable(t, client, id, 730)
}

func requirePreparedQueryParity(t *testing.T, ledger smokeClient, name, walletID, holdID string) {
	t.Helper()
	chart := wallet.NewChart("")
	walletAddress, holdAddress := chart.GetMainBalanceAccount(walletID), chart.GetHoldAccount(holdID)
	match := func(field, value string) map[string]any {
		return map[string]any{"$match": map[string]string{field: value}}
	}
	meta := func(key, value string) map[string]any {
		return match("metadata["+key+"]", value)
	}
	and := func(parts ...map[string]any) map[string]any {
		return map[string]any{"$and": parts}
	}
	for _, query := range []struct {
		name, collection, parameter, value string
		filter                             map[string]any
	}{
		{"wallets-balances-by-wallet-v1", "accounts", "walletId", walletID,
			and(meta(wallet.MetadataKeyWalletBalance, wallet.TrueValue), meta(wallet.MetadataKeyWalletID, walletID))},
		{"wallets-holds-by-wallet-v1", "accounts", "walletId", walletID, meta(wallet.MetadataKeyHoldWalletID, walletID)},
		{"wallets-by-name-v1", "accounts", "name", "smoke",
			and(meta(wallet.MetadataKeyWalletSpecType, wallet.PrimaryWallet), meta(wallet.MetadataKeyWalletName, "smoke"))},
		{"wallets-hold-funding-v1", "transactions", "holdAddress", holdAddress, match("destination", holdAddress)},
		{"wallets-transactions-by-wallet-v1", "transactions", "walletAddress", walletAddress,
			and(meta(wallet.MetadataKeyWalletTransaction, wallet.TrueValue), match("address", walletAddress))},
	} {
		filter, err := json.Marshal(query.filter)
		require.NoError(t, err)
		base := "/v3/" + name
		direct := ledger.call(http.MethodGet, base+"/"+query.collection+"?filter="+url.QueryEscape(string(filter)), nil, "", http.StatusOK)
		prepared := ledger.call(http.MethodPost, base+"/prepared-queries/"+query.name+"/execute", map[string]any{
			"parameters": map[string]string{query.parameter: query.value},
			"pageSize":   15,
			"reverse":    query.collection == "transactions",
		}, "", http.StatusOK).Cursor
		directRows := decode[[]smokeQueryRow](t, direct.Data)
		require.NotEmpty(t, directRows, query.name)
		rows := prepared.AccountData
		if query.collection == "transactions" {
			rows = prepared.TransactionData
		}
		preparedRows := make([]smokeQueryRow, 0, len(rows))
		for _, row := range rows {
			preparedRows = append(preparedRows, decode[smokeQueryRow](t, row))
		}
		require.ElementsMatch(t, directRows, preparedRows, query.name)
	}
}

type smokeQueryRow struct {
	ID      json.Number
	Address string
}

func creditRolloutReplay(t *testing.T, ledger smokeClient, name string, wallets smokeClient) {
	t.Helper()
	created := wallets.call(http.MethodPost, "/wallets", map[string]string{"name": "inline rollout"}, "", http.StatusCreated)
	id := decode[struct{ ID string }](t, created.Data).ID
	key := fmt.Sprintf("wallets-v3-%x", sha256.Sum256([]byte(name+"\x00inline-before-rollout")))
	ledger.call(http.MethodPost, "/v3/"+name+"/transactions", map[string]any{
		"metadata":  wallet.TransactionMetadata(nil),
		"reference": "inline-rollout",
		"script": map[string]any{
			"plain": wallet.BuildCreditWalletScript("world"),
			"vars": map[string]string{
				"destination": wallet.NewChart("").GetMainBalanceAccount(id),
				"amount":      "USD/2 10",
			},
		},
	}, key, http.StatusCreated)
	path := "/wallets/" + id + "/credit"
	credit := amount(10)
	credit["reference"] = "inline-rollout"
	wallets.call(http.MethodPost, path, credit, "inline-before-rollout", http.StatusNoContent)
	duplicate := wallets.call(http.MethodPost, path, credit, "different-key", http.StatusConflict)
	require.Equal(t, "CONFLICT", duplicate.ErrorCode)
	requireAvailable(t, wallets, id, 10)
}
