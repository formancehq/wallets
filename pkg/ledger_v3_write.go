package wallet

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"time"

	"github.com/formancehq/formance-sdk-go/v3/pkg/models/shared"
	"github.com/formancehq/go-libs/v5/pkg/types/pointer"
)

var _ Ledger = (*ledgerV3)(nil)

type v3FieldInfo struct {
	Type string `json:"type"`
}

type v3LedgerInfo struct {
	Name           string                   `json:"name"`
	Mode           string                   `json:"mode"`
	AccountTypes   map[string]v3AccountType `json:"accountTypes"`
	MetadataSchema struct {
		Accounts     map[string]v3FieldInfo `json:"accountFields"`
		Transactions map[string]v3FieldInfo `json:"transactionFields"`
	} `json:"metadataSchema"`
}

func (info v3LedgerInfo) metadataField(field v3MetadataField) (v3FieldInfo, bool) {
	if field.Target == "account" {
		value, ok := info.MetadataSchema.Accounts[field.Key]
		return value, ok
	}
	value, ok := info.MetadataSchema.Transactions[field.Key]
	return value, ok
}

func (l *ledgerV3) getLedger(ctx context.Context, name string) (v3LedgerInfo, error) {
	var result struct {
		Data v3LedgerInfo `json:"data"`
	}
	err := l.request(ctx, http.MethodGet, name, nil, nil, nil, "", http.StatusOK, &result)
	if err == nil && result.Data.Name != name {
		err = fmt.Errorf("Ledger v3 returned no matching ledger")
	}
	return result.Data, err
}

func v3HasError(err error, status int, codes ...string) bool {
	failure, ok := errors.AsType[*ledgerV3Error](err)
	if !ok || failure.Status != status {
		return false
	}
	for _, code := range codes {
		if failure.Code == code {
			return true
		}
	}
	return false
}

func v3IdempotencyKey(ledger, key string) string {
	if key == "" {
		return ""
	}
	return "wallets-v3-" + hashIdempotencyKey(ledger+"\x00"+key)
}

func v3WriteError(err error) error {
	failure, ok := errors.AsType[*ledgerV3Error](err)
	if !ok {
		return err
	}
	switch failure.Code {
	case "INSUFFICIENT_FUNDS":
		return fmt.Errorf("%w: %v", ErrInsufficientFundError, failure)
	case "IDEMPOTENCY_KEY_CONFLICT":
		return fmt.Errorf("%w: %v", ErrIdempotencyConflict, failure)
	case "TRANSACTION_REFERENCE_CONFLICT":
		return fmt.Errorf("%w: %v", ErrTransactionReferenceConflict, failure)
	}
	if failure.Status == http.StatusBadRequest {
		return fmt.Errorf("%w: %v", ErrValidation, failure)
	}
	return err
}

func (l *ledgerV3) AddMetadataToAccount(ctx context.Context, ledger, account, key string, metadata map[string]string) error {
	return v3WriteError(l.request(ctx, http.MethodPost, ledger, []string{"accounts", account, "metadata"},
		nil, metadata, v3IdempotencyKey(ledger, key), http.StatusNoContent, nil))
}

type v3TransactionRequest struct {
	Metadata        map[string]string               `json:"metadata,omitempty"`
	AccountMetadata map[string]map[string]string    `json:"accountMetadata,omitempty"`
	Postings        []shared.V2Posting              `json:"postings,omitempty"`
	Reference       *string                         `json:"reference,omitempty"`
	Script          *shared.V2PostTransactionScript `json:"script,omitempty"`
	ScriptReference *v3ScriptReference              `json:"scriptReference,omitempty"`
	Timestamp       *time.Time                      `json:"timestamp,omitempty"`
}

func (l *ledgerV3) CreateTransaction(ctx context.Context, ledger, key string, tx PostTransaction) (*shared.V2Transaction, error) {
	input := v3TransactionRequest{
		Metadata: tx.Metadata, AccountMetadata: tx.accountMetadata, Postings: tx.Postings,
		Reference: tx.Reference, Script: tx.Script,
	}
	if tx.holdReplayFingerprint != "" {
		input.Metadata = make(map[string]string, len(tx.Metadata)+2)
		maps.Copy(input.Metadata, tx.Metadata)
		input.Metadata[v3HoldReplayKey] = v3IdempotencyKey(ledger, key)
		input.Metadata[v3HoldReplayFingerprint] = tx.holdReplayFingerprint
	}
	if tx.Timestamp != nil {
		input.Timestamp = &tx.Timestamp.Time
	}
	if len(tx.accountMetadata) > 0 {
		// Ledger v3 writes the hold metadata from the transaction payload.
		// The v2 script carries the same metadata in its source code.
		input.Script = &shared.V2PostTransactionScript{
			Plain: pointer.For(BuildDebitWalletScript(nil, tx.debitSources...)), Vars: tx.Script.Vars,
		}
	}
	inline := input.Script
	if tx.libraryScript != "" {
		input.ScriptReference = &v3ScriptReference{Name: tx.libraryScript, Version: walletScriptVersion, Vars: input.Script.Vars}
		input.Script = nil
	}

	var result struct {
		Data struct {
			Transaction v3Transaction `json:"transaction"`
		} `json:"data"`
	}
	path := []string{"transactions"}
	idempotencyKey := v3IdempotencyKey(ledger, key)
	err := l.request(ctx, http.MethodPost, ledger, path, nil, input, idempotencyKey, http.StatusCreated, &result)
	if idempotencyKey != "" && input.ScriptReference != nil && v3HasError(err, http.StatusConflict, "IDEMPOTENCY_KEY_CONFLICT") {
		// An earlier Wallets release may have bound this key to the exact inline
		// request. Ledger rejected the saved-script request before execution.
		input.Script, input.ScriptReference = inline, nil
		err = l.request(ctx, http.MethodPost, ledger, path, nil, input, idempotencyKey, http.StatusCreated, &result)
	}
	if err != nil {
		return nil, v3WriteError(err)
	}
	transaction, err := result.Data.Transaction.walletTransaction()
	return &transaction, err
}
