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
	"github.com/formancehq/wallets/pkg/wallet/numscript"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestWalletsDebit(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	debitWalletRequest := DebitWalletRequest{
		Amount: core.Monetary{
			Amount: core.NewMonetaryInt(100),
			Asset:  "USD",
		},
	}

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", debitWalletRequest)
	rec := httptest.NewRecorder()

	var testEnv *testEnv
	testEnv = newTestEnv(
		WithRunScript(func(ctx context.Context, ledger string, script sdk.Script) (*sdk.ScriptResult, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			require.Equal(t, sdk.Script{
				Plain: numscript.BuildDebitWalletScript(),
				Vars: map[string]interface{}{
					"source":      testEnv.Chart().GetMainAccount(walletID),
					"destination": wallet.DefaultDebitDest,
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

func TestWalletsDebitWithInsufficientFund(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	debitWalletRequest := DebitWalletRequest{
		Amount: core.Monetary{
			Amount: core.NewMonetaryInt(100),
			Asset:  "USD",
		},
	}

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", debitWalletRequest)
	rec := httptest.NewRecorder()

	testEnv := newTestEnv(
		WithRunScript(func(ctx context.Context, ledger string, script sdk.Script) (*sdk.ScriptResult, error) {
			errorCode := string(sdk.INSUFFICIENT_FUND)
			return &sdk.ScriptResult{
				ErrorCode: &errorCode,
			}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Result().StatusCode)
	errorResponse := readErrorResponse(t, rec)
	require.Equal(t, ErrorCodeInsufficientFund, errorResponse.ErrorCode)
}

func TestWalletsDebitWithHold(t *testing.T) {
	t.Parallel()

	walletID := uuid.NewString()
	debitWalletRequest := DebitWalletRequest{
		Amount: core.Monetary{
			Amount: core.NewMonetaryInt(100),
			Asset:  "USD",
		},
		Pending: true,
		Metadata: map[string]any{
			"foo": "bar",
		},
		Description: "a first tx",
	}

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", debitWalletRequest)
	rec := httptest.NewRecorder()

	var (
		targetedLedger      string
		holdAccount         string
		holdAccountMetadata metadata.Metadata
		executedScript      sdk.Script
		testEnv             *testEnv
	)
	testEnv = newTestEnv(
		WithAddMetadataToAccount(func(ctx context.Context, ledger, account string, m metadata.Metadata) error {
			targetedLedger = ledger
			holdAccount = account
			holdAccountMetadata = m
			return nil
		}),
		WithRunScript(func(ctx context.Context, ledger string, script sdk.Script) (*sdk.ScriptResult, error) {
			require.Equal(t, testEnv.LedgerName(), ledger)
			executedScript = script
			return &sdk.ScriptResult{}, nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	hold := &core.DebitHold{}
	readResponse(t, rec, hold)

	require.Equal(t, testEnv.LedgerName(), targetedLedger)
	require.Equal(t, testEnv.Chart().GetHoldAccount(hold.ID), holdAccount)
	require.Equal(t, walletID, hold.WalletID)
	require.Equal(t, debitWalletRequest.Amount.Asset, hold.Asset)
	require.Equal(t, hold.LedgerMetadata(testEnv.Chart()), holdAccountMetadata)
	require.Equal(t, sdk.Script{
		Plain: numscript.BuildDebitWalletScript(),
		Vars: map[string]interface{}{
			"source":      testEnv.Chart().GetMainAccount(walletID),
			"destination": testEnv.Chart().GetHoldAccount(hold.ID),
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
	}, executedScript)
}
