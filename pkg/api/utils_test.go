package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"

	sharedapi "github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/auth"
	sharedhealth "github.com/formancehq/go-libs/v3/health"
	wallet "github.com/formancehq/wallets/pkg"
	"github.com/stretchr/testify/require"
)

func readErrorResponse(t *testing.T, rec *httptest.ResponseRecorder) *sharedapi.ErrorResponse {
	t.Helper()
	ret := &sharedapi.ErrorResponse{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(ret))
	return ret
}

func readResponse[T any](t *testing.T, rec *httptest.ResponseRecorder, to T) {
	t.Helper()
	ret := &sharedapi.BaseResponse[T]{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(ret))
	reflect.ValueOf(to).Elem().Set(reflect.ValueOf(*ret.Data).Elem())
}

func readCursor[T any](t *testing.T, rec *httptest.ResponseRecorder, to *bunpaginate.Cursor[T]) {
	t.Helper()
	ret := &sharedapi.BaseResponse[T]{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(ret))
	reflect.ValueOf(to).Elem().Set(reflect.ValueOf(ret.Cursor).Elem())
}

func bufFromObject(t *testing.T, v any) *bytes.Buffer {
	t.Helper()
	data, err := json.Marshal(v)
	require.NoError(t, err)
	return bytes.NewBuffer(data)
}

func newRequest(t *testing.T, method, path string, object any) *http.Request {
	t.Helper()
	var reader io.Reader
	if object != nil {
		reader = bufFromObject(t, object)
	}
	req := httptest.NewRequest(method, path, reader)
	req.Header.Set("Content-Type", "application/json")
	return req
}

type testEnv struct {
	router     chi.Router
	ledgerName string
	chart      *wallet.Chart
}

func (e testEnv) Router() chi.Router {
	return e.router
}

func (e testEnv) LedgerName() string {
	return e.ledgerName
}

func (e testEnv) Chart() *wallet.Chart {
	return e.chart
}

func newTestEnv(opts ...Option) *testEnv {
	ret := &testEnv{}
	ledgerMock := NewLedgerMock(opts...)
	ret.chart = wallet.NewChart("")
	ret.ledgerName = "default"
	manager := wallet.NewManager(ret.ledgerName, ledgerMock, ret.chart)
	ret.router = NewRouter(manager, &sharedhealth.HealthController{}, sharedapi.ServiceInfo{
		Version: "latest",
		Debug:   testing.Verbose(),
	}, auth.NewNoAuth())
	return ret
}

type (
	addMetadataToAccountFn func(ctx context.Context, ledger, account, ik string, metadata map[string]string) error
	getAccountFn           func(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error)
	listAccountsFn         func(ctx context.Context, ledger string, query wallet.ListAccountsQuery) (*wallet.AccountsCursorResponseCursor, error)
	listTransactionsFn     func(ctx context.Context, ledger string, query wallet.ListTransactionsQuery) (*shared.V2TransactionsCursorResponseCursor, error)
	createTransactionFn    func(ctx context.Context, ledger, ik string, postTransaction wallet.PostTransaction) (*shared.V2Transaction, error)
)

type LedgerMock struct {
	addMetadataToAccount addMetadataToAccountFn
	getAccount           getAccountFn
	listAccounts         listAccountsFn
	listTransactions     listTransactionsFn
	createTransaction    createTransactionFn
}

func (l *LedgerMock) EnsureLedgerExists(ctx context.Context, name string) error {
	return nil
}

func (l *LedgerMock) AddMetadataToAccount(ctx context.Context, ledger, account, ik string, metadata map[string]string) error {
	return l.addMetadataToAccount(ctx, ledger, account, ik, metadata)
}

func (l *LedgerMock) GetAccount(ctx context.Context, ledger, account string) (*wallet.AccountWithVolumesAndBalances, error) {
	return l.getAccount(ctx, ledger, account)
}

func (l *LedgerMock) ListAccounts(ctx context.Context, ledger string, query wallet.ListAccountsQuery) (*wallet.AccountsCursorResponseCursor, error) {
	return l.listAccounts(ctx, ledger, query)
}

func (l *LedgerMock) CreateTransaction(ctx context.Context, ledger, ik string, postTransaction wallet.PostTransaction) (*shared.V2Transaction, error) {
	return l.createTransaction(ctx, ledger, ik, postTransaction)
}

func (l *LedgerMock) ListTransactions(ctx context.Context, ledger string, query wallet.ListTransactionsQuery) (*shared.V2TransactionsCursorResponseCursor, error) {
	return l.listTransactions(ctx, ledger, query)
}

var _ wallet.Ledger = &LedgerMock{}

type Option func(mock *LedgerMock)

func WithCreateTransaction(fn createTransactionFn) Option {
	return func(mock *LedgerMock) {
		mock.createTransaction = fn
	}
}

func WithAddMetadataToAccount(fn addMetadataToAccountFn) Option {
	return func(mock *LedgerMock) {
		mock.addMetadataToAccount = fn
	}
}

func WithGetAccount(fn getAccountFn) Option {
	return func(mock *LedgerMock) {
		mock.getAccount = fn
	}
}

func WithListAccounts(fn listAccountsFn) Option {
	return func(mock *LedgerMock) {
		mock.listAccounts = fn
	}
}

func WithListTransactions(fn listTransactionsFn) Option {
	return func(mock *LedgerMock) {
		mock.listTransactions = fn
	}
}

func NewLedgerMock(opts ...Option) *LedgerMock {
	ret := &LedgerMock{}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}
