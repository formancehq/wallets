package wallet

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"slices"
	"sort"

	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
	"github.com/formancehq/go-libs/v5/pkg/types/time"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v5/pkg/types/metadata"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// idempotencyNamespace seeds deterministic resource IDs derived from an
// Idempotency-Key, so that retrying a creation request resolves to the same
// wallet/hold rather than creating a duplicate.
var idempotencyNamespace = uuid.MustParse("0b6f2d6e-4e2a-4f3a-9f0a-2b9c1d8e7a31")

// deterministicID derives a stable UUID from an Idempotency-Key, scoped by a
// resource kind ("wallet", "hold", ...). The kind discriminator keeps the
// derived IDs of different resource types disjoint, so reusing the same
// Idempotency-Key across, say, a wallet creation and a pending debit cannot
// collide on the same UUID.
func deterministicID(kind, ik string) string {
	return uuid.NewSHA1(idempotencyNamespace, []byte(kind+":"+ik)).String()
}

type ListResponse[T any] struct {
	Data           []T
	Next, Previous string
	HasMore        bool
}

type Pagination struct {
	Limit           int
	PaginationToken string
}

type ListQuery[T any] struct {
	Pagination
	Payload T
}

type mapper[SRC any, DST any] func(src SRC) DST

func newListResponse[SRC any, DST any](cursor interface {
	GetData() []SRC
	GetNext() *string
	GetPrevious() *string
	GetHasMore() bool
}, mapper mapper[SRC, DST],
) *ListResponse[DST] {
	ret := make([]DST, 0)
	for _, item := range cursor.GetData() {
		ret = append(ret, mapper(item))
	}

	next := ""
	if n := cursor.GetNext(); n != nil {
		next = *n
	}

	previous := ""
	if n := cursor.GetPrevious(); n != nil {
		previous = *n
	}

	return &ListResponse[DST]{
		Data:     ret,
		Next:     next,
		Previous: previous,
		HasMore:  cursor.GetHasMore(),
	}
}

type ListHolds struct {
	WalletID string
	Metadata metadata.Metadata
}

type ListBalances struct {
	WalletID string
	Metadata metadata.Metadata
}

type ListTransactions struct {
	WalletID string
}

func BalancesMetadataFilter(walletID string) metadata.Metadata {
	return metadata.Metadata{
		MetadataKeyWalletBalance: TrueValue,
		MetadataKeyWalletID:      walletID,
	}
}

type Manager struct {
	client     Ledger
	chart      *Chart
	ledgerName string
}

func NewManager(
	ledgerName string,
	client Ledger,
	chart *Chart,
) *Manager {
	return &Manager{
		client:     client,
		chart:      chart,
		ledgerName: ledgerName,
	}
}

func (m *Manager) Init(ctx context.Context) error {
	return m.client.EnsureLedgerExists(ctx, m.ledgerName)
}

//nolint:cyclop
func (m *Manager) Debit(ctx context.Context, ik string, debit Debit) (*DebitHold, error) {
	if err := debit.Validate(); err != nil {
		return nil, err
	}

	dest := debit.getDestination()

	var (
		hold     *DebitHold
		metadata map[string]map[string]string
	)
	if debit.Pending {
		if debit.Timestamp != nil {
			return nil, errors.New("timestamp cannot be specified using pending debit")
		}

		hold = Ptr(debit.newHold())
		// Derive the hold ID from the Idempotency-Key so a retry produces an
		// identical ledger request (the ledger hashes the body to enforce
		// idempotency) and returns the same hold.
		if ik != "" {
			hold.ID = deterministicID("hold", ik)
		}
		holdAccount := m.chart.GetHoldAccount(hold.ID)
		metadata = map[string]map[string]string{
			holdAccount: hold.LedgerMetadata(m.chart),
		}

		dest = NewLedgerAccountSubject(holdAccount)
	}

	var (
		balances Balances
		err      error
	)
	switch {
	case len(debit.Balances) == 0:
		balances = Balances{{
			Name: MainBalance,
		}}
	case len(debit.Balances) == 1 && debit.Balances[0] == "*":
		// A wildcard source set is resolved from live ledger state, so it can
		// differ between two attempts. We cannot guarantee an identical ledger
		// body on retry, so refuse to pretend the call is idempotent.
		if ik != "" {
			return nil, ErrNonIdempotentDebit
		}
		balances, err = fetchAndMapAllAccounts[Balance](ctx, m, BalancesMetadataFilter(debit.WalletID), false, BalanceFromAccount)
		if err != nil {
			return nil, err
		}
		sort.Stable(balances)
	default:
		if slices.Contains(debit.Balances, "*") {
			return nil, ErrInvalidBalanceSpecified
		}

		for _, balance := range debit.Balances {
			account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetBalanceAccount(debit.WalletID, balance))
			if err != nil {
				return nil, err
			}
			balances = append(balances, BalanceFromAccount(*account))
		}
	}

	var sources []string
	// Filter expired and generate sources
	for _, balance := range balances {
		if balance.ExpiresAt != nil && !balance.ExpiresAt.IsZero() {
			if balance.ExpiresAt.Before(time.Now()) {
				continue
			}
			// The balance is live now but will expire: crossing that boundary
			// between two attempts would drop it from the source set and change
			// the ledger body. We cannot honour idempotency in that case, so
			// reject rather than offer a false guarantee.
			if ik != "" {
				return nil, ErrNonIdempotentDebit
			}
		}
		sources = append(sources, m.chart.GetBalanceAccount(debit.WalletID, balance.Name))
	}

	// All resolved balances are expired: there is nothing to debit from.
	// Return a domain error rather than building a script with an empty
	// source set, which the ledger would reject as a compile error (500).
	if len(sources) == 0 {
		return nil, ErrInsufficientFundError
	}

	postTransaction := PostTransaction{
		Script: &shared.V2PostTransactionScript{
			Plain: pointer.For(BuildDebitWalletScript(metadata, sources...)),
			Vars: map[string]string{
				"destination": dest.getAccount(m.chart),
				"amount":      fmt.Sprintf("%s %s", debit.Amount.Asset, debit.Amount.Amount),
			},
		},
		Timestamp: debit.Timestamp,
		Metadata:  TransactionMetadata(debit.Metadata),
		//nolint:godox
		// TODO: Add set account metadata for hold when released on ledger (v1.9)
	}

	if debit.Reference != "" {
		postTransaction.Reference = &debit.Reference
	}

	if err := m.CreateTransaction(ctx, ik, postTransaction); err != nil {
		return nil, err
	}

	return hold, nil
}

func (m *Manager) ConfirmHold(ctx context.Context, ik string, debit ConfirmHold) error {
	account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetHoldAccount(debit.HoldID))
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return ErrHoldNotFound
		}
		return errors.Wrap(err, "getting account")
	}
	if !IsHold(account) {
		return ErrHoldNotFound
	}

	hold := ExpandedDebitHoldFromLedgerAccount(*account)
	if hold.Remaining.Uint64() == 0 {
		return ErrClosedHold
	}

	amount, err := debit.resolveAmount(hold)
	if err != nil {
		return err
	}

	vars := map[string]string{
		"hold":   m.chart.GetHoldAccount(debit.HoldID),
		"amount": fmt.Sprintf("%s %d", hold.Asset, amount),
		"dest":   hold.Destination.getAccount(m.chart),
	}
	if debit.Final {
		vars["void_destination"] = m.chart.GetMainBalanceAccount(hold.WalletID)
	}

	postTransaction := PostTransaction{
		Script: &shared.V2PostTransactionScript{
			Plain: pointer.For(BuildConfirmHoldScript(debit.Final, hold.Asset)),
			Vars:  vars,
		},
		Metadata: TransactionMetadata(metadata.Metadata{}),
	}

	if err := m.CreateTransaction(ctx, ik, postTransaction); err != nil {
		return err
	}

	return nil
}

func (m *Manager) VoidHold(ctx context.Context, ik string, void VoidHold) error {

	txs, err := m.client.ListTransactions(ctx, m.ledgerName, ListTransactionsQuery{
		Destination: m.chart.GetHoldAccount(void.HoldID),
	})
	if err != nil {
		return fmt.Errorf("retrieving original transaction: %w", err)
	}
	if len(txs.Data) == 0 {
		return ErrHoldNotFound
	}
	if len(txs.Data) != 1 {
		return fmt.Errorf("expected 1 transaction, got %d", len(txs.Data))
	}

	account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetHoldAccount(void.HoldID))
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return ErrHoldNotFound
		}
		return errors.Wrap(err, "getting account")
	}
	if !IsHold(account) {
		return ErrHoldNotFound
	}

	hold := ExpandedDebitHoldFromLedgerAccount(*account)
	if hold.IsClosed() {
		return ErrClosedHold
	}

	postTransaction := PostTransaction{
		Script: &shared.V2PostTransactionScript{
			Plain: pointer.For(BuildCancelHoldScript(hold.Asset, txs.Data[0].Postings...)),
			Vars: map[string]string{
				"hold": m.chart.GetHoldAccount(void.HoldID),
			},
		},
		Metadata: TransactionMetadata(metadata.Metadata{}),
	}

	if err := m.CreateTransaction(ctx, ik, postTransaction); err != nil {
		return err
	}

	return nil
}

func (m *Manager) Credit(ctx context.Context, ik string, credit Credit) error {
	if err := credit.Validate(); err != nil {
		return err
	}

	if credit.Balance != "" {
		if _, err := m.GetBalance(ctx, credit.WalletID, credit.Balance); err != nil {
			return err
		}
	}

	postTransaction := PostTransaction{
		Script: &shared.V2PostTransactionScript{
			Plain: pointer.For(BuildCreditWalletScript(credit.Sources.ResolveAccounts(m.chart)...)),
			Vars: map[string]string{
				"destination": credit.destinationAccount(m.chart),
				"amount":      fmt.Sprintf("%s %s", credit.Amount.Asset, credit.Amount.Amount),
			},
		},
		Timestamp: credit.Timestamp,
		Metadata:  TransactionMetadata(credit.Metadata),
	}
	if credit.Reference != "" {
		postTransaction.Reference = &credit.Reference
	}

	if err := m.CreateTransaction(ctx, ik, postTransaction); err != nil {
		return err
	}

	return nil
}

func (m *Manager) CreateTransaction(ctx context.Context, ik string, postTransaction PostTransaction) error {
	if _, err := m.client.CreateTransaction(ctx, m.ledgerName, ik, postTransaction); err != nil {
		// The ledger SDK returns a *sdkerrors.V2ErrorResponse, which the
		// ledger layer already translates to ErrInsufficientFundError.
		if errors.Is(err, ErrInsufficientFundError) {
			return ErrInsufficientFundError
		}
		return errors.Wrap(err, "creating transaction")
	}

	return nil
}

func (m *Manager) ListWallets(ctx context.Context, query ListQuery[ListWallets]) (*ListResponse[Wallet], error) {
	return mapAccountList(ctx, m, mapAccountListQuery{
		Pagination: query.Pagination,
		Metadata: func() metadata.Metadata {
			metadata := metadata.Metadata{
				MetadataKeyWalletSpecType: PrimaryWallet,
			}
			if len(query.Payload.Metadata) > 0 {
				for k, v := range query.Payload.Metadata {
					metadata[MetadataKeyWalletCustomDataPrefix+k] = v
				}
			}
			if query.Payload.Name != "" {
				metadata[MetadataKeyWalletName] = query.Payload.Name
			}
			return metadata
		},
		ExpandVolumes: query.Payload.ExpandBalances,
	}, func(account AccountWithVolumesAndBalances) Wallet {
		return WithBalancesFromAccount(m.ledgerName, account)
	})
}

func (m *Manager) ListHolds(ctx context.Context, query ListQuery[ListHolds]) (*ListResponse[DebitHold], error) {
	return mapAccountList(ctx, m, mapAccountListQuery{
		Pagination: query.Pagination,
		Metadata: func() metadata.Metadata {
			metadata := metadata.Metadata{
				MetadataKeyWalletSpecType: HoldWallet,
			}
			if query.Payload.WalletID != "" {
				metadata[MetadataKeyHoldWalletID] = query.Payload.WalletID
			}
			if len(query.Payload.Metadata) > 0 {
				for k, v := range query.Payload.Metadata {
					metadata[MetadataKeyWalletCustomDataPrefix+k] = v
				}
			}
			return metadata
		},
	}, DebitHoldFromLedgerAccount)
}

func (m *Manager) ListBalances(ctx context.Context, query ListQuery[ListBalances]) (*ListResponse[Balance], error) {
	return mapAccountList(ctx, m, mapAccountListQuery{
		Metadata: func() metadata.Metadata {
			metadata := BalancesMetadataFilter(query.Payload.WalletID)
			if len(query.Payload.Metadata) > 0 {
				for k, v := range query.Payload.Metadata {
					metadata[MetadataKeyWalletCustomDataPrefix+k] = v
				}
			}
			return metadata
		},
		Pagination: query.Pagination,
	}, BalanceFromAccount)
}

func (m *Manager) ListTransactions(ctx context.Context, query ListQuery[ListTransactions]) (*ListResponse[Transaction], error) {
	var (
		response *shared.V2TransactionsCursorResponseCursor
		err      error
	)
	if query.PaginationToken == "" {
		response, err = m.client.ListTransactions(ctx, m.ledgerName, ListTransactionsQuery{
			Limit: query.Limit,
			Account: func() string {
				if query.Payload.WalletID != "" {
					return m.chart.GetMainBalanceAccount(query.Payload.WalletID)
				}
				return ""
			}(),
			Metadata: TransactionBaseMetadataFilter(),
		})
	} else {
		response, err = m.client.ListTransactions(ctx, m.ledgerName, ListTransactionsQuery{
			Cursor: query.PaginationToken,
		})
	}
	if err != nil {
		return nil, errors.Wrap(err, "listing transactions")
	}

	return newListResponse[shared.V2Transaction, Transaction](response, func(tx shared.V2Transaction) Transaction {
		return Transaction{
			V2Transaction: tx,
			Ledger:        m.ledgerName,
		}
	}), nil
}

func (m *Manager) CreateWallet(ctx context.Context, ik string, data *CreateRequest) (*Wallet, error) {
	wallet := NewWallet(data.Name, m.ledgerName, data.Metadata)

	// Without an Idempotency-Key there is no replay contract: create with a
	// random ID and a fresh CreatedAt.
	if ik == "" {
		if err := m.client.AddMetadataToAccount(ctx, m.ledgerName, m.chart.GetMainBalanceAccount(wallet.ID), ik, wallet.LedgerMetadata()); err != nil {
			return nil, errors.Wrap(err, "adding metadata to account")
		}
		return &wallet, nil
	}

	// Derive the wallet ID from the Idempotency-Key so a retry targets the same
	// account instead of creating a duplicate wallet.
	wallet.ID = deterministicID("wallet", ik)

	// NewWallet stamps CreatedAt with time.Now(), which LedgerMetadata()
	// serialises into the ledger body; the ledger hashes that body to enforce
	// idempotency, so a retry cannot re-send it verbatim. Resolve idempotency
	// against the persisted wallet instead, matching the retry against an
	// immutable fingerprint of the original create request (stored at creation),
	// never against the wallet's live metadata which UpdateWallet can mutate.
	fingerprint := walletCreateRequestFingerprint(data.Name, data.Metadata)
	body := wallet.LedgerMetadata()
	body[MetadataKeyWalletCreateRequestHash] = fingerprint

	if existing, err := m.existingWalletAccount(ctx, wallet.ID); err != nil {
		return nil, err
	} else if existing != nil {
		return replayOrConflict(m.ledgerName, existing, fingerprint)
	}

	if err := m.client.AddMetadataToAccount(ctx, m.ledgerName, m.chart.GetMainBalanceAccount(wallet.ID), ik, body); err != nil {
		// A concurrent attempt may have created the wallet between our existence
		// check and this write. The ledger then rejects our body because its
		// CreatedAt differs — reported as a validation or a conflict error
		// depending on timing — so we don't classify the error: re-check
		// existence and, if a wallet now exists, replay it (or report a conflict
		// when the persisted request differs); otherwise surface the error.
		existing, gerr := m.existingWalletAccount(ctx, wallet.ID)
		if gerr != nil {
			return nil, gerr
		}
		if existing == nil {
			return nil, errors.Wrap(err, "adding metadata to account")
		}
		return replayOrConflict(m.ledgerName, existing, fingerprint)
	}

	return &wallet, nil
}

// existingWalletAccount returns the persisted primary wallet account stored at
// the main balance account for id, or (nil, nil) when no wallet exists there
// yet.
func (m *Manager) existingWalletAccount(ctx context.Context, id string) (*AccountWithVolumesAndBalances, error) {
	account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetMainBalanceAccount(id))
	switch {
	case errors.Is(err, ErrAccountNotFound):
		return nil, nil
	case err != nil:
		return nil, errors.Wrap(err, "getting account")
	case !IsPrimary(account):
		return nil, nil
	default:
		return account, nil
	}
}

// replayOrConflict returns the persisted wallet when the incoming request
// matches the one that originally created it (an idempotent replay), or
// ErrIdempotencyConflict when the same key was reused with a different request.
// The comparison is against the create-request fingerprint stored at creation,
// which is immutable: UpdateWallet never rewrites it, so the replay/conflict
// outcome does not depend on later wallet mutations. CreatedAt is not part of
// the fingerprint, since it legitimately differs between attempts.
func replayOrConflict(ledger string, existing *AccountWithVolumesAndBalances, fingerprint string) (*Wallet, error) {
	if GetMetadata(existing, MetadataKeyWalletCreateRequestHash) != fingerprint {
		return nil, ErrIdempotencyConflict
	}
	w := WithBalancesFromAccount(ledger, existing)
	return &w, nil
}

// walletCreateRequestFingerprint is a stable hash of the idempotency-relevant
// fields of a create-wallet request (name and custom metadata). It is stored
// with the wallet so retries can be distinguished from key reuse with a
// different body, independently of any later metadata changes.
func walletCreateRequestFingerprint(name string, md metadata.Metadata) string {
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	h := sha256.New()
	_, _ = h.Write([]byte(name))
	for _, k := range keys {
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(k))
		_, _ = h.Write([]byte{0})
		_, _ = h.Write([]byte(md[k]))
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (m *Manager) UpdateWallet(ctx context.Context, id, ik string, data *PatchRequest) error {
	account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetMainBalanceAccount(id))
	if err != nil {
		return ErrWalletNotFound
	}

	if !IsPrimary(account) {
		return ErrWalletNotFound
	}

	newCustomMetadata := metadata.Metadata{}
	newCustomMetadata = newCustomMetadata.Merge(ExtractCustomMetadata(account))
	newCustomMetadata = newCustomMetadata.Merge(data.Metadata)

	meta := metadata.Metadata(account.GetMetadata())
	meta = meta.Merge(EncodeCustomMetadata(newCustomMetadata))

	if err := m.client.AddMetadataToAccount(ctx, m.ledgerName, m.chart.GetMainBalanceAccount(id), ik, meta); err != nil {
		return errors.Wrap(err, "adding metadata to account")
	}

	return nil
}

func (m *Manager) GetWallet(ctx context.Context, id string) (*Wallet, error) {
	account, err := m.client.GetAccount(
		ctx,
		m.ledgerName,
		m.chart.GetMainBalanceAccount(id),
	)
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return nil, ErrWalletNotFound
		}
		return nil, errors.Wrap(err, "getting account")
	}

	if !IsPrimary(account) {
		return nil, ErrWalletNotFound
	}

	return Ptr(WithBalancesFromAccount(m.ledgerName, account)), nil
}

type Summary struct {
	Balances       []ExpandedBalance   `json:"balances"`
	AvailableFunds map[string]*big.Int `json:"availableFunds"`
	ExpiredFunds   map[string]*big.Int `json:"expiredFunds"`
	ExpirableFunds map[string]*big.Int `json:"expirableFunds"`
	HoldFunds      map[string]*big.Int `json:"holdFunds"`
}

func (m *Manager) GetWalletSummary(ctx context.Context, id string) (*Summary, error) {
	balances, err := fetchAndMapAllAccounts(ctx, m, metadata.Metadata{
		MetadataKeyWalletID: id,
	}, true, func(src AccountWithVolumesAndBalances) ExpandedBalance {
		return ExpandedBalanceFromAccount(src)
	})
	if err != nil {
		return nil, err
	}

	s := &Summary{
		Balances:       balances,
		AvailableFunds: map[string]*big.Int{},
		ExpiredFunds:   map[string]*big.Int{},
		ExpirableFunds: map[string]*big.Int{},
		HoldFunds:      map[string]*big.Int{},
	}

	for _, balance := range balances {
		for asset, amount := range balance.Assets {
			switch {
			case balance.ExpiresAt != nil && balance.ExpiresAt.Before(time.Now()):
				if s.ExpiredFunds[asset] == nil {
					s.ExpiredFunds[asset] = new(big.Int)
				}
				s.ExpiredFunds[asset].Add(s.ExpiredFunds[asset], amount)
			case balance.ExpiresAt != nil && !balance.ExpiresAt.Before(time.Now()):
				if s.ExpirableFunds[asset] == nil {
					s.ExpirableFunds[asset] = new(big.Int)
				}
				s.ExpirableFunds[asset].Add(s.ExpirableFunds[asset], amount)
				if s.AvailableFunds[asset] == nil {
					s.AvailableFunds[asset] = new(big.Int)
				}
				s.AvailableFunds[asset].Add(s.AvailableFunds[asset], amount)
			case balance.ExpiresAt == nil:
				if s.AvailableFunds[asset] == nil {
					s.AvailableFunds[asset] = new(big.Int)
				}
				s.AvailableFunds[asset].Add(s.AvailableFunds[asset], amount)
			}
		}
	}

	holds, err := fetchAndMapAllAccounts(ctx, m, metadata.Metadata{
		MetadataKeyHoldWalletID: id,
	}, true, func(src AccountWithVolumesAndBalances) ExpandedDebitHold {
		return ExpandedDebitHoldFromLedgerAccount(src)
	})
	if err != nil {
		return nil, err
	}

	for _, hold := range holds {
		if s.HoldFunds[hold.Asset] == nil {
			s.HoldFunds[hold.Asset] = new(big.Int)
		}
		s.HoldFunds[hold.Asset].Add(s.HoldFunds[hold.Asset], hold.Remaining)
	}

	return s, nil
}

func (m *Manager) GetHold(ctx context.Context, id string) (*ExpandedDebitHold, error) {
	account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetHoldAccount(id))
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return nil, ErrHoldNotFound
		}
		return nil, err
	}

	if !IsHold(account) {
		return nil, ErrHoldNotFound
	}

	return Ptr(ExpandedDebitHoldFromLedgerAccount(*account)), nil
}

func (m *Manager) CreateBalance(ctx context.Context, data *CreateBalance) (*Balance, error) {
	if err := data.Validate(); err != nil {
		return nil, err
	}
	ret, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetBalanceAccount(data.WalletID, data.Name))
	switch {
	case errors.Is(err, ErrAccountNotFound):
	case err == nil:
		if ret.Metadata != nil &&
			ret.Metadata[MetadataKeyWalletBalance] == TrueValue {
			return nil, ErrBalanceAlreadyExists
		}
	default:
		return nil, err
	}

	balance := NewBalance(data.Name, data.ExpiresAt)
	balance.Priority = data.Priority

	if err := m.client.AddMetadataToAccount(
		ctx,
		m.ledgerName,
		m.chart.GetBalanceAccount(data.WalletID, balance.Name),
		"",
		balance.LedgerMetadata(data.WalletID),
	); err != nil {
		return nil, errors.Wrap(err, "adding metadata to account")
	}

	return &balance, nil
}

func (m *Manager) GetBalance(ctx context.Context, walletID string, balanceName string) (*ExpandedBalance, error) {
	account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetBalanceAccount(walletID, balanceName))
	if err != nil {
		if errors.Is(err, ErrAccountNotFound) {
			return nil, ErrBalanceNotExists
		}
		return nil, err
	}

	if account.Metadata[MetadataKeyWalletBalance] != TrueValue {
		return nil, ErrBalanceNotExists
	}

	return Ptr(ExpandedBalanceFromAccount(*account)), nil
}

type mapAccountListQuery struct {
	Pagination
	Metadata      func() metadata.Metadata
	ExpandVolumes bool
}

func mapAccountList[TO any](ctx context.Context, r *Manager, query mapAccountListQuery, mapper mapper[AccountWithVolumesAndBalances, TO]) (*ListResponse[TO], error) {
	var (
		cursor *AccountsCursorResponseCursor
		err    error
	)
	if query.PaginationToken == "" {
		cursor, err = r.client.ListAccounts(ctx, r.ledgerName, ListAccountsQuery{
			Limit:         query.Limit,
			Metadata:      query.Metadata(),
			ExpandVolumes: query.ExpandVolumes,
		})
	} else {
		cursor, err = r.client.ListAccounts(ctx, r.ledgerName, ListAccountsQuery{
			Cursor: query.PaginationToken,
		})
	}
	if err != nil {
		return nil, err
	}

	return newListResponse[AccountWithVolumesAndBalances, TO](cursor, func(item AccountWithVolumesAndBalances) TO {
		return mapper(item)
	}), nil
}

const maxPageSize = 100

func fetchAndMapAllAccounts[TO any](ctx context.Context, r *Manager, md metadata.Metadata, expandVolumes bool, mapper mapper[AccountWithVolumesAndBalances, TO]) ([]TO, error) {

	ret := make([]TO, 0)
	query := mapAccountListQuery{
		Metadata: func() metadata.Metadata {
			return md
		},
		Pagination: Pagination{
			Limit: maxPageSize,
		},
		ExpandVolumes: expandVolumes,
	}
	for {
		listResponse, err := mapAccountList(ctx, r, query, mapper)
		if err != nil {
			return nil, err
		}
		ret = append(ret, listResponse.Data...)
		if listResponse.Next == "" {
			return ret, nil
		}
		query = mapAccountListQuery{
			Pagination: Pagination{
				PaginationToken: listResponse.Next,
			},
		}
	}
}
