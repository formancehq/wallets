package wallet

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLedgerV3ProvisioningRejectsDrift(t *testing.T) {
	plan, err := newV3Provisioning(NewChart("wallet-prefix"), "verify", []string{"tier", "tier"})
	require.NoError(t, err)
	require.Len(t, plan.accountTypes, 2)
	require.Len(t, plan.indexes(), 10)
	require.Equal(t, "walletprefix:wallets:{walletId}:{balanceName}", plan.accountTypes["wallets-balance"].Pattern)

	adapter := &ledgerV3{provision: plan}
	require.ErrorContains(t, adapter.reconcileDefinitions(t.Context(), "wallets", v3LedgerInfo{}), "missing Wallets definitions")
	drift := v3LedgerInfo{AccountTypes: map[string]v3AccountType{
		"wallets-hold": {Name: "wallets-hold", Pattern: "elsewhere:holds:{holdId}", Persistence: "NORMAL"},
	}}
	require.ErrorContains(t, adapter.reconcileDefinitions(t.Context(), "wallets", drift), "differs from the Wallets definition")
	drift.AccountTypes = nil
	drift.MetadataSchema.Accounts = map[string]v3FieldInfo{MetadataKeyWalletSpecType: {Type: "bool"}}
	require.ErrorContains(t, adapter.reconcileDefinitions(t.Context(), "wallets", drift), "want \"string\"")
	require.ErrorIs(t, adapter.validateCustomFilters(map[string]string{MetadataKeyWalletCustomDataPrefix + "unknown": "gold"}), ErrValidation)
}

func TestLedgerV3ProvisioningWaitsForIndex(t *testing.T) {
	var checks, replicaChecks atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const path = "/v3/wallets/indexes/metadata:TARGET_TYPE_ACCOUNT:wallets%2Fname/status"
		if r.Method == http.MethodGet && r.URL.EscapedPath() == "/replica"+path {
			if replicaChecks.Add(1) == 1 {
				w.WriteHeader(http.StatusNotFound)
				_, _ = fmt.Fprint(w, `{"errorCode":"INDEX_NOT_FOUND"}`)
				return
			}
			_, _ = fmt.Fprint(w, `{"data":{"currentVersion":1}}`)
			return
		}
		if r.Method != http.MethodGet || r.URL.EscapedPath() != path {
			t.Errorf("unexpected index request: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if checks.Add(1) == 1 {
			_, _ = fmt.Fprint(w, `{"data":{"currentVersion":0,"pendingVersion":1}}`)
			return
		}
		_, _ = fmt.Fprint(w, `{"data":{"currentVersion":1}}`)
	}))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	adapter := &ledgerV3{client: srv.Client(), base: base, provision: v3Provisioning{verifyOnly: true, indexStatusURLs: []*url.URL{base.JoinPath("replica")}}}
	require.NoError(t, adapter.ensureIndex(t.Context(), "wallets", "metadata:TARGET_TYPE_ACCOUNT:wallets/name"))
	require.EqualValues(t, 2, checks.Load())
	require.EqualValues(t, 2, replicaChecks.Load())
}

func TestLedgerV3ProvisioningLeavesConcurrentlyDeclaredFieldIntact(t *testing.T) {
	var reads atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v3/wallets" {
			t.Errorf("unexpected write to a concurrently declared field: %s %s", r.Method, r.URL)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		reads.Add(1)
		_, _ = fmt.Fprintf(w, `{"data":{"name":"wallets","metadataSchema":{"accountFields":{%q:{"type":"string"}}}}}`, MetadataKeyWalletName)
	}))
	t.Cleanup(srv.Close)
	base, err := url.Parse(srv.URL)
	require.NoError(t, err)
	adapter := &ledgerV3{client: srv.Client(), base: base, provision: v3Provisioning{
		fields: []v3MetadataField{{Target: "account", Key: MetadataKeyWalletName, Type: "string"}},
	}}

	require.NoError(t, adapter.reconcileDefinitions(t.Context(), "wallets", v3LedgerInfo{Name: "wallets"}))
	require.EqualValues(t, 2, reads.Load())
}
