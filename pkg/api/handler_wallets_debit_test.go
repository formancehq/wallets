package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/wallets/pkg/core"
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

	var (
		ledger          string
		transactionData sdk.TransactionData
	)
	testEnv := newTestEnv(
		WithCreateTransaction(func(ctx context.Context, l string, t sdk.TransactionData) error {
			ledger = l
			transactionData = t
			return nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusNoContent, rec.Result().StatusCode)
	require.Equal(t, testEnv.LedgerName(), ledger)
	require.Equal(t, sdk.TransactionData{
		Timestamp: nil,
		Postings: []sdk.Posting{{
			Amount:      100,
			Asset:       "USD",
			Source:      testEnv.Chart().GetMainAccount(walletID),
			Destination: "world",
		}},
	}, transactionData)
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
	}

	req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/debit", debitWalletRequest)
	rec := httptest.NewRecorder()

	var (
		ledger          string
		account         string
		metadata        core.Metadata
		transactionData sdk.TransactionData
	)
	testEnv := newTestEnv(
		WithAddMetadataToAccount(func(ctx context.Context, l, a string, m core.Metadata) error {
			ledger = l
			account = a
			metadata = m
			return nil
		}),
		WithCreateTransaction(func(ctx context.Context, l string, td sdk.TransactionData) error {
			require.Equal(t, ledger, l)
			transactionData = td
			return nil
		}),
	)
	testEnv.Router().ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Result().StatusCode)
	hold := &core.DebitHold{}
	readResponse(t, rec, hold)

	require.Equal(t, testEnv.LedgerName(), ledger)
	require.Equal(t, testEnv.Chart().GetHoldAccount(hold.ID), account)
	require.Equal(t, hold.LedgerMetadata(testEnv.Chart()), metadata)
	require.Equal(t, sdk.TransactionData{
		Postings: []sdk.Posting{{
			Amount:      100,
			Asset:       "USD",
			Source:      testEnv.Chart().GetMainAccount(walletID),
			Destination: testEnv.Chart().GetHoldAccount(hold.ID),
		}},
	}, transactionData)
}