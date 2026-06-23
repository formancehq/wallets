package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/formancehq/go-libs/v5/pkg/types/time"

	wallet "github.com/formancehq/wallets/pkg"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func ptr[V any](v V) *V {
	return &v
}

type balanceCreateTestCase struct {
	name               string
	request            wallet.CreateBalance
	expectedStatusCode int
	expectedErrorCode  string
}

var balanceCreateTestCases = []balanceCreateTestCase{
	{
		name: "nominal",
		request: wallet.CreateBalance{
			Name: "balance1",
		},
	},
	{
		name: "with invalid name",
		request: wallet.CreateBalance{
			Name: "!!!!!!!",
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
	{
		// The name contains valid characters but also an account separator;
		// an unanchored regex would have accepted it, allowing address/script injection.
		name: "with name containing an account separator",
		request: wallet.CreateBalance{
			Name: "balance:injected",
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
	{
		name: "with name containing whitespace and numscript tokens",
		request: wallet.CreateBalance{
			Name: "x\n@world",
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
	{
		// Dashes are allowed: dashed/UUID balance names must keep working.
		// (They still alias under Address.String(); see chart.go.)
		name: "with name containing a dash",
		request: wallet.CreateBalance{
			Name: "foo-bar",
		},
	},
	{
		name: "with reserved name",
		request: wallet.CreateBalance{
			Name: wallet.MainBalance,
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
	{
		name: "with expiration",
		request: wallet.CreateBalance{
			Name:      wallet.MainBalance,
			ExpiresAt: ptr(time.Now().Add(10 * time.Second)),
		},
		expectedStatusCode: http.StatusBadRequest,
		expectedErrorCode:  ErrorCodeValidation,
	},
}

func TestBalancesCreate(t *testing.T) {
	t.Parallel()

	for _, testCase := range balanceCreateTestCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			walletID := uuid.NewString()
			req := newRequest(t, http.MethodPost, "/wallets/"+walletID+"/balances", testCase.request)
			rec := httptest.NewRecorder()

			var (
				targetedLedger  string
				targetedAccount string
				appliedMetadata map[string]string
			)
			testEnv := newTestEnv(
				WithAddMetadataToAccount(func(ctx context.Context, ledger, account, ik string, metadata map[string]string) error {
					targetedLedger = ledger
					targetedAccount = account
					appliedMetadata = metadata
					return nil
				}),
				WithGetAccount(func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
					return &wallet.AccountWithVolumesAndBalances{}, nil
				}),
			)
			testEnv.Router().ServeHTTP(rec, req)

			expectedStatusCode := testCase.expectedStatusCode
			if expectedStatusCode == 0 {
				expectedStatusCode = http.StatusCreated
			}
			require.Equal(t, expectedStatusCode, rec.Result().StatusCode)

			if expectedStatusCode == http.StatusCreated {
				balance := &wallet.Balance{}
				readResponse(t, rec, balance)
				require.Equal(t, testEnv.LedgerName(), targetedLedger)
				require.Equal(t, targetedAccount, testEnv.Chart().GetBalanceAccount(walletID, balance.Name))
				require.Equal(t, balance.LedgerMetadata(walletID), appliedMetadata)
				require.Equal(t, balance.Name, testCase.request.Name)
			} else {
				errorResponse := readErrorResponse(t, rec)
				require.Equal(t, testCase.expectedErrorCode, errorResponse.ErrorCode)
			}
		})
	}
}
