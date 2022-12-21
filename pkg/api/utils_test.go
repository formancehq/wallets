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

	sdk "github.com/formancehq/formance-sdk-go"
	"github.com/formancehq/go-libs/sharedapi"
	"github.com/formancehq/wallets/pkg/core"
	"github.com/formancehq/wallets/pkg/wallet"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func readResponse[T any](t *testing.T, rec *httptest.ResponseRecorder, to T) {
	t.Helper()
	ret := &sharedapi.BaseResponse[T]{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(ret))
	reflect.ValueOf(to).Elem().Set(reflect.ValueOf(*ret.Data).Elem())
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
	chart      *core.Chart
}

func (e testEnv) Router() chi.Router {
	return e.router
}

func (e testEnv) LedgerName() string {
	return e.ledgerName
}

func (e testEnv) Chart() *core.Chart {
	return e.chart
}

func newTestEnv(opts ...Option) *testEnv {
	ret := &testEnv{}
	ledgerMock := NewLedgerMock(opts...)
	ret.chart = core.NewChart("")
	ret.ledgerName = uuid.NewString()
	fundingService := wallet.NewFundingService(ret.ledgerName, ledgerMock, ret.chart)
	repository := wallet.NewRepository(ret.ledgerName, ledgerMock, ret.chart)
	ret.router = NewRouter(fundingService, repository)
	return ret
}

type (
	addMetadataToAccountFn     func(ctx context.Context, ledger, account string, metadata core.Metadata) error
	getAccountFn               func(ctx context.Context, ledger, account string) (*sdk.AccountWithVolumesAndBalances, error)
	listAccountsWithMetadataFn func(ctx context.Context, name string, m map[string]any) ([]sdk.Account, error)
	createTransactionFn        func(ctx context.Context, name string, transaction sdk.TransactionData) error
	runScriptFn                func(ctx context.Context, name string, script sdk.Script) error
)

type LedgerMock struct {
	addMetadataToAccount     addMetadataToAccountFn
	getAccount               getAccountFn
	listAccountsWithMetadata listAccountsWithMetadataFn
	createTransaction        createTransactionFn
	runScript                runScriptFn
}

func (l *LedgerMock) AddMetadataToAccount(ctx context.Context, ledger, account string, metadata core.Metadata) error {
	return l.addMetadataToAccount(ctx, ledger, account, metadata)
}

func (l *LedgerMock) GetAccount(ctx context.Context, ledger, account string) (*sdk.AccountWithVolumesAndBalances, error) {
	return l.getAccount(ctx, ledger, account)
}

func (l *LedgerMock) ListAccountsWithMetadata(ctx context.Context, name string, m map[string]any) ([]sdk.Account, error) {
	return l.listAccountsWithMetadata(ctx, name, m)
}

func (l *LedgerMock) CreateTransaction(ctx context.Context, name string, transaction sdk.TransactionData) error {
	return l.createTransaction(ctx, name, transaction)
}

func (l *LedgerMock) RunScript(ctx context.Context, name string, script sdk.Script) error {
	return l.runScript(ctx, name, script)
}

var _ wallet.Ledger = &LedgerMock{}

type Option func(mock *LedgerMock)

func WithRunScript(fn runScriptFn) Option {
	return func(mock *LedgerMock) {
		mock.runScript = fn
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

func WithCreateTransaction(fn createTransactionFn) Option {
	return func(mock *LedgerMock) {
		mock.createTransaction = fn
	}
}

func WithListAccountsWithMetadata(fn listAccountsWithMetadataFn) Option {
	return func(mock *LedgerMock) {
		mock.listAccountsWithMetadata = fn
	}
}

func NewLedgerMock(opts ...Option) *LedgerMock {
	ret := &LedgerMock{}
	for _, opt := range opts {
		opt(ret)
	}
	return ret
}
