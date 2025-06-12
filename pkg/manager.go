package wallet

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"sort"

	"github.com/formancehq/go-libs/v3/time"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/sdkerrors"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v3/metadata"
	"github.com/pkg/errors"
)

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
		if balance.ExpiresAt != nil && !balance.ExpiresAt.IsZero() && balance.ExpiresAt.Before(time.Now()) {
			continue
		}
		sources = append(sources, m.chart.GetBalanceAccount(debit.WalletID, balance.Name))
	}

	postTransaction := PostTransaction{
		Script: &shared.V2PostTransactionScript{
			Plain: BuildDebitWalletScript(metadata, sources...),
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
			Plain: BuildConfirmHoldScript(debit.Final, hold.Asset),
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
	if len(txs.Data) != 1 {
		return fmt.Errorf("expected 1 transaction, got %d", len(txs.Data))
	}

	account, err := m.client.GetAccount(ctx, m.ledgerName, m.chart.GetHoldAccount(void.HoldID))
	if err != nil {
		return errors.Wrap(err, "getting account")
	}

	hold := ExpandedDebitHoldFromLedgerAccount(*account)
	if hold.IsClosed() {
		return ErrClosedHold
	}

	postTransaction := PostTransaction{
		Script: &shared.V2PostTransactionScript{
			Plain: BuildCancelHoldScript(hold.Asset, txs.Data[0].Postings...),
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
			Plain: BuildCreditWalletScript(credit.Sources.ResolveAccounts(m.chart)...),
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
		switch err := err.(type) {
		case *sdkerrors.WalletsErrorResponse:
			if err.ErrorCode == sdkerrors.SchemasWalletsErrorResponseErrorCodeInsufficientFund {
				return ErrInsufficientFundError
			}
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

func (m *Manager) CreateWallet(ctx context.Context, data *CreateRequest) (*Wallet, error) {
	wallet := NewWallet(data.Name, m.ledgerName, data.Metadata)

	if err := m.client.AddMetadataToAccount(
		ctx,
		m.ledgerName,
		m.chart.GetMainBalanceAccount(wallet.ID),
		"",
		wallet.LedgerMetadata(),
	); err != nil {
		return nil, errors.Wrap(err, "adding metadata to account")
	}

	return &wallet, nil
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
		return nil, err
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
