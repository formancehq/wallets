package wallet

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
)

type v3Metadata map[string]json.RawMessage

func (m v3Metadata) strings() map[string]string {
	result := make(map[string]string, len(m))
	for key, raw := range m {
		if len(raw) == 0 || raw[0] != '"' {
			continue
		}
		var value string
		if json.Unmarshal(raw, &value) == nil {
			result[key] = value
		}
	}
	return result
}

type v3Account struct {
	Address  string     `json:"address"`
	Metadata v3Metadata `json:"metadata"`
	Volumes  []struct {
		Asset   string          `json:"asset"`
		Color   string          `json:"color"`
		Volumes shared.V2Volume `json:"volumes"`
	} `json:"volumes"`
}

func (a v3Account) walletAccount(expand bool) (AccountWithVolumesAndBalances, error) {
	result := AccountWithVolumesAndBalances{
		Account:  Account{Address: a.Address, Metadata: a.Metadata.strings()},
		Balances: map[string]*big.Int{},
	}
	if !expand {
		return result, nil
	}
	result.Volumes = make(map[string]shared.V2Volume, len(a.Volumes))
	for _, entry := range a.Volumes {
		if entry.Color != "" {
			continue
		}
		if entry.Volumes.Input == nil || entry.Volumes.Output == nil {
			return result, fmt.Errorf("Ledger v3 account volume is missing input or output")
		}
		result.Volumes[entry.Asset] = entry.Volumes
		result.Balances[entry.Asset] = new(big.Int).Sub(entry.Volumes.Input, entry.Volumes.Output)
	}
	return result, nil
}

type v3Transaction struct {
	ID                *big.Int    `json:"id"`
	InsertedAt        *time.Time  `json:"insertedAt"`
	UpdatedAt         *time.Time  `json:"updatedAt"`
	Timestamp         time.Time   `json:"timestamp"`
	Reverted          bool        `json:"reverted"`
	RevertedAt        *time.Time  `json:"revertedAt"`
	Reference         *string     `json:"reference"`
	Metadata          v3Metadata  `json:"metadata"`
	Postings          []v3Posting `json:"postings"`
	PostCommitVolumes map[string][]struct {
		Asset  string   `json:"asset"`
		Color  string   `json:"color"`
		Input  *big.Int `json:"input"`
		Output *big.Int `json:"output"`
	} `json:"postCommitVolumes"`
}

type v3Posting struct {
	Amount      *big.Int `json:"amount"`
	Asset       string   `json:"asset"`
	Source      string   `json:"source"`
	Destination string   `json:"destination"`
	Color       string   `json:"color"`
}

func (t v3Transaction) walletTransaction() (shared.V2Transaction, error) {
	result := shared.V2Transaction{
		ID: t.ID, InsertedAt: t.InsertedAt, UpdatedAt: t.UpdatedAt, Timestamp: t.Timestamp,
		Reverted: t.Reverted, RevertedAt: t.RevertedAt, Reference: t.Reference,
		Metadata: t.Metadata.strings(),
		Postings: make([]shared.V2Posting, 0, len(t.Postings)),
	}
	delete(result.Metadata, v3HoldReplayKey)
	delete(result.Metadata, v3HoldReplayFingerprint)
	if t.ID == nil || t.Timestamp.IsZero() {
		return result, fmt.Errorf("Ledger v3 transaction is missing its ID or timestamp")
	}
	for _, posting := range t.Postings {
		if posting.Amount == nil || posting.Color != "" {
			return result, fmt.Errorf("Ledger v3 transaction contains an unsupported posting")
		}
		result.Postings = append(result.Postings, shared.V2Posting{
			Amount: posting.Amount, Asset: posting.Asset, Source: posting.Source, Destination: posting.Destination,
		})
	}
	for address, volumes := range t.PostCommitVolumes {
		for _, volume := range volumes {
			if volume.Color != "" {
				continue
			}
			if volume.Input == nil || volume.Output == nil {
				return result, fmt.Errorf("Ledger v3 transaction volume is missing input or output")
			}
			if result.PostCommitVolumes == nil {
				result.PostCommitVolumes = map[string]map[string]shared.V2Volume{}
			}
			if result.PostCommitVolumes[address] == nil {
				result.PostCommitVolumes[address] = map[string]shared.V2Volume{}
			}
			result.PostCommitVolumes[address][volume.Asset] = shared.V2Volume{Input: volume.Input, Output: volume.Output}
		}
	}
	return result, nil
}

type v3Page[T any] struct {
	Data     []T    `json:"data"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
	HasMore  bool   `json:"hasMore"`
}

type v3ListQuery struct {
	Size        int               `json:"size"`
	Expand      bool              `json:"expand,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Address     string            `json:"address,omitempty"`
	Source      string            `json:"source,omitempty"`
	Destination string            `json:"destination,omitempty"`
}

type v3Cursor struct {
	Backend  string      `json:"backend"`
	Ledger   string      `json:"ledger"`
	Scope    string      `json:"scope"`
	Query    v3ListQuery `json:"query"`
	Token    string      `json:"token"`
	Prepared string      `json:"prepared,omitempty"`
}

const v3CursorPrefix = "wallets-v3."

func (c v3Cursor) encode(token string, key []byte) *string {
	if token == "" {
		return nil
	}
	c.Token = token
	data, _ := json.Marshal(c)
	return pointer.For(v3CursorPrefix + base64.RawURLEncoding.EncodeToString(data) + "." +
		base64.RawURLEncoding.EncodeToString(signV3Cursor(data, key)))
}

func signV3Cursor(data, key []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(data)
	return mac.Sum(nil)
}

func (c *v3Cursor) restore(token string, key []byte) error {
	encoded, ok := strings.CutPrefix(token, v3CursorPrefix)
	if !ok || len(encoded) > 64<<10 {
		return fmt.Errorf("%w: invalid Ledger v3 cursor", ErrValidation)
	}
	payload, signature, ok := strings.Cut(encoded, ".")
	data, err := base64.RawURLEncoding.DecodeString(payload)
	mac, macErr := base64.RawURLEncoding.DecodeString(signature)
	var saved v3Cursor
	if !ok || err != nil || macErr != nil || !hmac.Equal(mac, signV3Cursor(data, key)) || json.Unmarshal(data, &saved) != nil || saved.Token == "" ||
		saved.Backend != c.Backend || saved.Scope != c.Scope || saved.Ledger != c.Ledger ||
		saved.Query.Size < 1 || saved.Query.Size > maxPageSize {
		return fmt.Errorf("%w: invalid Ledger v3 cursor", ErrValidation)
	}
	for key, value := range c.Query.Metadata {
		if actual, present := saved.Query.Metadata[key]; !present || actual != value {
			return fmt.Errorf("%w: cursor filter changed", ErrValidation)
		}
	}
	if c.Query.Expand && !saved.Query.Expand {
		return fmt.Errorf("%w: cursor expansion changed", ErrValidation)
	}
	if (c.Query.Address != "" && saved.Query.Address != c.Query.Address) ||
		(c.Query.Source != "" && saved.Query.Source != c.Query.Source) ||
		(c.Query.Destination != "" && saved.Query.Destination != c.Query.Destination) {
		return fmt.Errorf("%w: cursor account selector changed", ErrValidation)
	}
	*c = saved
	return nil
}

func v3Match(field, value string) map[string]any {
	return map[string]any{"$match": map[string]string{field: value}}
}

func (q v3ListQuery) filter() any {
	predicates := make([]map[string]any, 0, len(q.Metadata)+3)
	keys := make([]string, 0, len(q.Metadata))
	for key := range q.Metadata {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	for _, key := range keys {
		predicates = append(predicates, v3Match("metadata["+key+"]", q.Metadata[key]))
	}
	for _, selector := range []struct{ field, value string }{
		{"address", q.Address}, {"source", q.Source}, {"destination", q.Destination},
	} {
		if selector.value != "" {
			predicates = append(predicates, v3Match(selector.field, selector.value))
		}
	}
	switch len(predicates) {
	case 0:
		return nil
	case 1:
		return predicates[0]
	default:
		return map[string]any{"$and": predicates}
	}
}

func v3List[T any](ctx context.Context, l *ledgerV3, collection string, cursor v3Cursor) (v3Page[T], error) {
	params := url.Values{"pageSize": {strconv.Itoa(cursor.Query.Size)}}
	if filter := cursor.Query.filter(); filter != nil {
		raw, _ := json.Marshal(filter)
		params.Set("filter", string(raw))
	}
	if cursor.Token != "" {
		params.Set("cursor", cursor.Token)
	}
	var page v3Page[T]
	err := l.request(ctx, http.MethodGet, cursor.Ledger, []string{collection}, params, nil, "", http.StatusOK, &page)
	if failure, ok := errors.AsType[*ledgerV3Error](err); ok && failure.Code == "INVALID_REQUEST" {
		err = fmt.Errorf("%w: %s", ErrValidation, failure.Message)
	}
	if err == nil && page.Data == nil {
		err = fmt.Errorf("Ledger v3 list response is missing its data array")
	}
	return page, err
}

func (l *ledgerV3) GetAccount(ctx context.Context, ledger, account string) (*AccountWithVolumesAndBalances, error) {
	// The point read synthesizes an empty account for an unknown address.
	page, err := v3List[v3Account](ctx, l, "accounts", v3Cursor{
		Ledger: ledger, Query: v3ListQuery{Size: 1, Address: account},
	})
	if err != nil {
		return nil, err
	}
	switch {
	case page.HasMore || len(page.Data) > 1:
		return nil, fmt.Errorf("Ledger v3 returned more than one result for an exact account query")
	case len(page.Data) == 0:
		return nil, ErrAccountNotFound
	case page.Data[0].Address != account:
		return nil, fmt.Errorf("Ledger v3 returned an unexpected address for an exact account query")
	}
	result, err := page.Data[0].walletAccount(true)
	return &result, err
}

func (l *ledgerV3) cursorBackend() string {
	return hashIdempotencyKey(l.base.JoinPath("v3").String())
}

func (l *ledgerV3) ListAccounts(ctx context.Context, ledger string, q ListAccountsQuery) (*AccountsCursorResponseCursor, error) {
	cursor := v3Cursor{Backend: l.cursorBackend(), Ledger: ledger, Scope: "accounts:" + q.scope, Query: v3ListQuery{
		Size: v3PageSize(q.Limit), Expand: q.ExpandVolumes, Metadata: q.Metadata,
	}}
	if q.Cursor != "" {
		if err := cursor.restore(q.Cursor, l.cursorKey); err != nil {
			return nil, err
		}
	}
	if err := l.validateCustomFilters(cursor.Query.Metadata); err != nil {
		return nil, err
	}
	page, err := v3WalletList[v3Account](ctx, l, "accounts", &cursor)
	if err != nil {
		return nil, err
	}
	result := &AccountsCursorResponseCursor{
		Data: make([]AccountWithVolumesAndBalances, 0, len(page.Data)), HasMore: page.HasMore,
		PageSize: int64(cursor.Query.Size), Next: cursor.encode(page.Next, l.cursorKey), Previous: cursor.encode(page.Previous, l.cursorKey),
	}
	for _, account := range page.Data {
		item, err := account.walletAccount(cursor.Query.Expand)
		if err != nil {
			return nil, err
		}
		result.Data = append(result.Data, item)
	}
	return result, nil
}

func (l *ledgerV3) ListTransactions(ctx context.Context, ledger string, q ListTransactionsQuery) (*shared.V2TransactionsCursorResponseCursor, error) {
	cursor := v3Cursor{Backend: l.cursorBackend(), Ledger: ledger, Scope: "transactions", Query: v3ListQuery{
		Size: v3PageSize(q.Limit), Metadata: q.Metadata, Address: q.Account, Source: q.Source, Destination: q.Destination,
	}}
	if q.Cursor != "" {
		if err := cursor.restore(q.Cursor, l.cursorKey); err != nil {
			return nil, err
		}
	}
	page, err := v3WalletList[v3Transaction](ctx, l, "transactions", &cursor)
	if err != nil {
		return nil, err
	}
	result := &shared.V2TransactionsCursorResponseCursor{
		Data: make([]shared.V2Transaction, 0, len(page.Data)), HasMore: page.HasMore,
		PageSize: int64(cursor.Query.Size), Next: cursor.encode(page.Next, l.cursorKey), Previous: cursor.encode(page.Previous, l.cursorKey),
	}
	for _, transaction := range page.Data {
		item, err := transaction.walletTransaction()
		if err != nil {
			return nil, err
		}
		result.Data = append(result.Data, item)
	}
	return result, nil
}

func v3PageSize(size int) int {
	if size <= 0 {
		return maxPageSize
	}
	return min(size, maxPageSize)
}
