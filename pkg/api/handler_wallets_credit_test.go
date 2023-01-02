package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/go-libs/metadata"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletsCredit(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	creditWalletRequest := wallet.CreditWalletRequest{
		Amount: core.Monetary{
			Amount: core.NewMonetaryInt(100),
			Asset:  "USD",
		},
		Metadata: map[string]interface{}{
			"foo": "bar",
		},
	}

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/credit", creditWalletRequest)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithRunScript(func(ctx context.Context, ledger string, script sdk.Script) (*sdk.ScriptResult, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, sdk.Script{
				Plain: wallet.BuildCreditWalletScript("world"),
				Vars: map[string]interface{}{
					"destination": testEnv.chart.GetMainAccount(walletID),
					"amount": map[string]any{
						"amount": uint64(100),
						"asset":  "USD",
					},
				},
				Metadata: core.WalletTransactionBaseMetadata().Merge(metadata.Metadata{
					core.MetadataKeyWalletCustomData: metadata.Metadata{
						"foo": "bar",
					},
				}),
			}, script)
			return &sdk.ScriptResult{}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
}

func TestWalletsCreditWithSourceList(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	creditWalletRequest := wallet.CreditWalletRequest{
		Amount: core.Monetary{
			Amount: core.NewMonetaryInt(100),
			Asset:  "USD",
		},
		Sources: []wallet.Subject{{
			Type:       wallet.SourceTypeLedgerAccount,
			Identifier: "emitter1",
		}, {
			Type:       wallet.SourceTypeWallet,
			Identifier: "wallet1",
		}},
	}

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/credit", creditWalletRequest)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithRunScript(func(ctx context.Context, ledger string, script sdk.Script) (*sdk.ScriptResult, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, sdk.Script{
				Plain: wallet.BuildCreditWalletScript(
					creditWalletRequest.Sources[0].Identifier,
					testEnv.Chart().GetMainAccount(creditWalletRequest.Sources[1].Identifier),
				),
				Vars: map[string]interface{}{
					"destination": testEnv.chart.GetMainAccount(walletID),
					"amount": map[string]any{
						"amount": uint64(100),
						"asset":  "USD",
					},
				},
				Metadata: core.WalletTransactionBaseMetadata().Merge(metadata.Metadata{
					core.MetadataKeyWalletCustomData: metadata.Metadata{},
				}),
			}, script)
			return &sdk.ScriptResult{}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
}
