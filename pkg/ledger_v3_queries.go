package wallet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
)

const (
	queryBalancesByWallet = "wallets-balances-by-wallet-v1"
	queryHoldsByWallet    = "wallets-holds-by-wallet-v1"
	queryHoldFunding      = "wallets-hold-funding-v1"
	queryWalletsByName    = "wallets-by-name-v1"
	queryWalletHistory    = "wallets-transactions-by-wallet-v1"
)

type v3PreparedQuery struct {
	Name   string          `json:"name"`
	Target string          `json:"target"`
	Filter json.RawMessage `json:"filter"`
}

var v3PreparedQueries = []v3PreparedQuery{
	{queryBalancesByWallet, "ACCOUNTS", json.RawMessage(
		`{"$and":[{"$match":{"metadata[` + MetadataKeyWalletBalance + `]":"true"}},{"$match":{"metadata[` +
			MetadataKeyWalletID + `]":{"$param":"walletId"}}}]}`)},
	{queryHoldsByWallet, "ACCOUNTS", json.RawMessage(
		`{"$match":{"metadata[` + MetadataKeyHoldWalletID + `]":{"$param":"walletId"}}}`)},
	{queryHoldFunding, "TRANSACTIONS", json.RawMessage(`{"$match":{"destination":{"$param":"holdAddress"}}}`)},
	{queryWalletsByName, "ACCOUNTS", json.RawMessage(
		`{"$and":[{"$match":{"metadata[` + MetadataKeyWalletSpecType + `]":"` + PrimaryWallet + `"}},{"$match":{"metadata[` +
			MetadataKeyWalletName + `]":{"$param":"name"}}}]}`)},
	{queryWalletHistory, "TRANSACTIONS", json.RawMessage(
		`{"$and":[{"$match":{"metadata[` + MetadataKeyWalletTransaction + `]":"true"}},{"$match":{"address":{"$param":"walletAddress"}}}]}`)},
}

func (q v3PreparedQuery) matches(actual v3PreparedQuery) bool {
	if q.Target != actual.Target {
		return false
	}
	var expectedFilter, actualFilter any
	return json.Unmarshal(q.Filter, &expectedFilter) == nil && json.Unmarshal(actual.Filter, &actualFilter) == nil &&
		reflect.DeepEqual(expectedFilter, actualFilter)
}

func (l *ledgerV3) preparedQueries(ctx context.Context, ledger string) (map[string]v3PreparedQuery, error) {
	var result struct {
		Data []v3PreparedQuery `json:"data"`
	}
	if err := l.request(ctx, http.MethodGet, ledger, []string{"prepared-queries"}, nil, nil, "", http.StatusOK, &result); err != nil {
		return nil, err
	}
	queries := make(map[string]v3PreparedQuery, len(result.Data))
	for _, query := range result.Data {
		queries[query.Name] = query
	}
	return queries, nil
}

func (l *ledgerV3) ensurePreparedQueries(ctx context.Context, ledger string) error {
	existing, err := l.preparedQueries(ctx, ledger)
	if err != nil {
		return fmt.Errorf("list Ledger v3 prepared queries: %w", err)
	}
	for _, query := range v3PreparedQueries {
		actual, found := existing[query.Name]
		if !found && !l.provision.verifyOnly {
			err := l.request(ctx, http.MethodPost, ledger, []string{"prepared-queries"}, nil, query, "", http.StatusNoContent, nil)
			if err != nil && !v3HasError(err, http.StatusConflict, "PREPARED_QUERY_ALREADY_EXISTS") {
				return fmt.Errorf("store Ledger v3 prepared query %q: %w", query.Name, err)
			}
			// Refresh after a create or a concurrent create before accepting it.
			existing, err = l.preparedQueries(ctx, ledger)
			if err != nil {
				return fmt.Errorf("read back Ledger v3 prepared query %q: %w", query.Name, err)
			}
			actual, found = existing[query.Name]
		}
		if !found || !query.matches(actual) {
			return fmt.Errorf("Ledger v3 prepared query %q is missing or differs from the Wallets definition", query.Name)
		}
	}
	return nil
}

func (q v3ListQuery) prepared(collection string) (string, map[string]string) {
	if collection == "accounts" && q.Address == "" && q.Destination == "" && q.Source == "" {
		switch {
		case len(q.Metadata) == 2 && q.Metadata[MetadataKeyWalletBalance] == TrueValue && q.Metadata[MetadataKeyWalletID] != "":
			return queryBalancesByWallet, map[string]string{"walletId": q.Metadata[MetadataKeyWalletID]}
		case len(q.Metadata) == 1 && q.Metadata[MetadataKeyHoldWalletID] != "":
			return queryHoldsByWallet, map[string]string{"walletId": q.Metadata[MetadataKeyHoldWalletID]}
		case len(q.Metadata) == 2 && q.Metadata[MetadataKeyWalletSpecType] == PrimaryWallet && q.Metadata[MetadataKeyWalletName] != "":
			return queryWalletsByName, map[string]string{"name": q.Metadata[MetadataKeyWalletName]}
		}
	}
	if collection == "transactions" && q.Source == "" {
		switch {
		case q.Destination != "" && q.Address == "" && len(q.Metadata) == 0:
			return queryHoldFunding, map[string]string{"holdAddress": q.Destination}
		case q.Destination == "" && q.Address != "" && len(q.Metadata) == 1 && q.Metadata[MetadataKeyWalletTransaction] == TrueValue:
			return queryWalletHistory, map[string]string{"walletAddress": q.Address}
		}
	}
	return "", nil
}

func v3WalletList[T any](ctx context.Context, l *ledgerV3, collection string, cursor *v3Cursor) (v3Page[T], error) {
	name, params := cursor.Query.prepared(collection)
	if cursor.Token == "" {
		cursor.Prepared = name
		if name == queryHoldFunding {
			cursor.Query.Size = 2
		}
	}
	if cursor.Prepared == "" {
		return v3List[T](ctx, l, collection, *cursor)
	}
	if cursor.Prepared != name {
		return v3Page[T]{}, fmt.Errorf("%w: invalid Ledger v3 prepared query cursor", ErrValidation)
	}

	body := struct {
		Parameters map[string]string `json:"parameters"`
		PageSize   int               `json:"pageSize"`
		Cursor     string            `json:"cursor,omitempty"`
		Reverse    bool              `json:"reverse,omitempty"`
	}{
		Parameters: params, PageSize: cursor.Query.Size, Cursor: cursor.Token,
		// The prepared query uses ascending transaction IDs by default.
		Reverse: collection == "transactions",
	}
	var result struct {
		Cursor *struct {
			v3Page[T]
			AccountData     []T `json:"accountData"`
			TransactionData []T `json:"transactionData"`
		} `json:"cursor"`
	}
	err := l.request(ctx, http.MethodPost, cursor.Ledger, []string{"prepared-queries", name, "execute"},
		nil, body, "", http.StatusOK, &result)
	if err != nil {
		if failure, ok := errors.AsType[*ledgerV3Error](err); ok && failure.Code == "INVALID_REQUEST" {
			err = fmt.Errorf("%w: %s", ErrValidation, failure.Message)
		}
		return v3Page[T]{}, err
	}
	if result.Cursor == nil {
		return v3Page[T]{}, fmt.Errorf("Ledger v3 prepared query %q response is missing its cursor", name)
	}
	page := result.Cursor.v3Page
	if collection == "accounts" {
		page.Data = result.Cursor.AccountData
	} else {
		page.Data = result.Cursor.TransactionData
	}
	return page, nil
}
