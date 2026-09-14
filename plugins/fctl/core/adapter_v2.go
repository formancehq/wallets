package core

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk/producthttp"
	walletclient "github.com/formancehq/wallets/pkg/client"
	"github.com/formancehq/wallets/pkg/client/models/components"
	"github.com/formancehq/wallets/pkg/client/models/operations"
)

func executeV2(ctx context.Context, request sdk.ExecuteRequest, host sdk.Host) error {
	command, ok := commandByID(request.CommandID)
	if !ok {
		return invalidArgument("unknown command %q", request.CommandID)
	}
	if err := sdk.ValidateExecuteRequest(command, request); err != nil {
		return invalidArgument("invalid execution request: %s", err)
	}
	if err := sdk.ValidateTargetSelection(command.Target, request.Target); err != nil {
		return invalidArgument("invalid target: %s", err)
	}
	flags, err := collectFlags(command, request.Flags)
	if err != nil {
		return err
	}
	if len(command.Operations) != 1 {
		return descriptorInvalid("command %q does not declare one operation", command.ID)
	}
	bound, err := producthttp.New(host, command.Operations[0], "auth.stack")
	if err != nil {
		return fmt.Errorf("wallets: configure generated HTTP adapter: %w", err)
	}
	generated := walletclient.New(
		walletclient.WithClient(bound),
		walletclient.WithServerURL("https://product.invalid"),
	)

	switch request.CommandID {
	case "wallets.v2.create":
		if len(request.Arguments) != 1 {
			return invalidArgument("create requires one wallet name")
		}
		metadata, err := parseMetadata(flags["metadata"])
		if err != nil {
			return err
		}
		response, err := generated.Wallets.V1.CreateWallet(ctx, operations.CreateWalletRequest{
			IdempotencyKey: optionalString(first(flags["ik"])),
			CreateWalletRequest: &components.CreateWalletRequest{
				Name: request.Arguments[0], Metadata: metadata,
			},
		})
		if err != nil {
			return fmt.Errorf("wallets: create wallet: %w", err)
		}
		if response == nil || response.CreateWalletResponse == nil {
			return fmt.Errorf("wallets: create wallet returned no result")
		}
		return emitJSON(host, command.ID, sdk.ResultObject, response.CreateWalletResponse.Data, nil)
	case "wallets.v2.list":
		metadata, err := parseMetadata(flags["metadata"])
		if err != nil {
			return err
		}
		items, page, err := collectPages(request.Continuation, func(cursorValue *string) ([]components.Wallet, *string, *bool, error) {
			listRequest := operations.ListWalletsRequest{Cursor: cursorValue}
			if cursorValue == nil {
				listRequest.Metadata = metadata
			}
			response, err := generated.Wallets.V1.ListWallets(ctx, listRequest)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("wallets: list wallets: %w", err)
			}
			if response == nil || response.ListWalletsResponse == nil {
				return nil, nil, nil, fmt.Errorf("wallets: list wallets returned no result")
			}
			page := response.ListWalletsResponse.Cursor
			return page.Data, page.Next, page.HasMore, nil
		})
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultCollection, items, page)
	case "wallets.v2.show":
		response, err := generated.Wallets.V1.GetWallet(ctx, operations.GetWalletRequest{ID: first(flags["id"])})
		if err != nil {
			return fmt.Errorf("wallets: get wallet: %w", err)
		}
		if response == nil || response.GetWalletResponse == nil {
			return fmt.Errorf("wallets: get wallet returned no result")
		}
		return emitJSON(host, command.ID, sdk.ResultObject, response.GetWalletResponse.Data, nil)
	case "wallets.v2.update":
		if len(request.Arguments) != 1 {
			return invalidArgument("update requires one wallet id")
		}
		metadata, err := parseMetadata(flags["metadata"])
		if err != nil {
			return err
		}
		_, err = generated.Wallets.V1.UpdateWallet(ctx, operations.UpdateWalletRequest{
			ID: request.Arguments[0], IdempotencyKey: optionalString(first(flags["ik"])),
			RequestBody: &operations.UpdateWalletRequestBody{Metadata: metadata},
		})
		if err != nil {
			return fmt.Errorf("wallets: update wallet: %w", err)
		}
		return emitEmptyObject(host, command.ID)
	case "wallets.v2.credit":
		if len(request.Arguments) != 2 {
			return invalidArgument("credit requires amount and asset")
		}
		amount, err := parseNonNegativeAmount(request.Arguments[0])
		if err != nil {
			return err
		}
		metadata, err := parseMetadata(flags["metadata"])
		if err != nil {
			return err
		}
		sources, err := parseSubjects(flags["source"])
		if err != nil {
			return err
		}
		_, err = generated.Wallets.V1.CreditWallet(ctx, operations.CreditWalletRequest{
			ID: first(flags["id"]), IdempotencyKey: optionalString(first(flags["ik"])),
			CreditWalletRequest: &components.CreditWalletRequest{
				Amount:   components.Monetary{Amount: amount, Asset: request.Arguments[1]},
				Metadata: metadata, Sources: sources, Balance: optionalString(first(flags["balance"])),
			},
		})
		if err != nil {
			return fmt.Errorf("wallets: credit wallet: %w", err)
		}
		return emitEmptyObject(host, command.ID)
	case "wallets.v2.debit":
		if len(request.Arguments) != 2 {
			return invalidArgument("debit requires amount and asset")
		}
		amount, err := parseNonNegativeAmount(request.Arguments[0])
		if err != nil {
			return err
		}
		metadata, err := parseMetadata(flags["metadata"])
		if err != nil {
			return err
		}
		destination, err := parseOptionalSubject(first(flags["destination"]))
		if err != nil {
			return err
		}
		pending, err := optionalBool(first(flags["pending"]))
		if err != nil {
			return err
		}
		response, err := generated.Wallets.V1.DebitWallet(ctx, operations.DebitWalletRequest{
			ID: first(flags["id"]), IdempotencyKey: optionalString(first(flags["ik"])),
			DebitWalletRequest: &components.DebitWalletRequest{
				Amount:  components.Monetary{Amount: amount, Asset: request.Arguments[1]},
				Pending: pending, Metadata: metadata, Description: optionalString(first(flags["description"])),
				Destination: destination, Balances: flags["balance"],
			},
		})
		if err != nil {
			return fmt.Errorf("wallets: debit wallet: %w", err)
		}
		if response != nil && response.DebitWalletResponse != nil {
			return emitJSON(host, command.ID, sdk.ResultObject, response.DebitWalletResponse.Data, nil)
		}
		return emitEmptyObject(host, command.ID)
	case "wallets.v2.balances.create":
		if len(request.Arguments) != 1 {
			return invalidArgument("balances create requires one balance name")
		}
		expiresAt, err := optionalTime(first(flags["expires-at"]))
		if err != nil {
			return err
		}
		priority, err := optionalSignedBigInt(first(flags["priority"]))
		if err != nil {
			return err
		}
		response, err := generated.Wallets.V1.CreateBalance(ctx, operations.CreateBalanceRequest{
			ID:                   first(flags["id"]),
			IdempotencyKey:       optionalString(first(flags["ik"])),
			CreateBalanceRequest: &components.CreateBalanceRequest{Name: request.Arguments[0], ExpiresAt: expiresAt, Priority: priority},
		})
		if err != nil {
			return fmt.Errorf("wallets: create balance: %w", err)
		}
		if response == nil || response.CreateBalanceResponse == nil {
			return fmt.Errorf("wallets: create balance returned no result")
		}
		return emitJSON(host, command.ID, sdk.ResultObject, response.CreateBalanceResponse.Data, nil)
	case "wallets.v2.balances.list":
		items, page, err := collectPages(request.Continuation, func(cursorValue *string) ([]components.Balance, *string, *bool, error) {
			response, err := generated.Wallets.V1.ListBalances(ctx, operations.ListBalancesRequest{ID: first(flags["id"]), Cursor: cursorValue})
			if err != nil {
				return nil, nil, nil, fmt.Errorf("wallets: list balances: %w", err)
			}
			if response == nil || response.ListBalancesResponse == nil {
				return nil, nil, nil, fmt.Errorf("wallets: list balances returned no result")
			}
			page := response.ListBalancesResponse.Cursor
			return page.Data, page.Next, page.HasMore, nil
		})
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultCollection, items, page)
	case "wallets.v2.balances.show":
		if len(request.Arguments) != 1 {
			return invalidArgument("balances show requires one balance name")
		}
		response, err := generated.Wallets.V1.GetBalance(ctx, operations.GetBalanceRequest{ID: first(flags["id"]), BalanceName: request.Arguments[0]})
		if err != nil {
			return fmt.Errorf("wallets: get balance: %w", err)
		}
		if response == nil || response.GetBalanceResponse == nil {
			return fmt.Errorf("wallets: get balance returned no result")
		}
		return emitJSON(host, command.ID, sdk.ResultObject, response.GetBalanceResponse.Data, nil)
	case "wallets.v2.holds.list":
		metadata, err := parseMetadata(flags["metadata"])
		if err != nil {
			return err
		}
		items, page, err := collectPages(request.Continuation, func(cursorValue *string) ([]components.Hold, *string, *bool, error) {
			listRequest := operations.GetHoldsRequest{Cursor: cursorValue}
			if cursorValue == nil {
				listRequest.WalletID = optionalString(first(flags["id"]))
				listRequest.Metadata = metadata
			}
			response, err := generated.Wallets.V1.GetHolds(ctx, listRequest)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("wallets: list holds: %w", err)
			}
			if response == nil || response.GetHoldsResponse == nil {
				return nil, nil, nil, fmt.Errorf("wallets: list holds returned no result")
			}
			page := response.GetHoldsResponse.Cursor
			return page.Data, page.Next, page.HasMore, nil
		})
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultCollection, items, page)
	case "wallets.v2.holds.show":
		if len(request.Arguments) != 1 {
			return invalidArgument("holds show requires one hold id")
		}
		response, err := generated.Wallets.V1.GetHold(ctx, operations.GetHoldRequest{HoldID: request.Arguments[0]})
		if err != nil {
			return fmt.Errorf("wallets: get hold: %w", err)
		}
		if response == nil || response.GetHoldResponse == nil {
			return fmt.Errorf("wallets: get hold returned no result")
		}
		return emitJSON(host, command.ID, sdk.ResultObject, response.GetHoldResponse.Data, nil)
	case "wallets.v2.holds.confirm":
		if len(request.Arguments) != 1 {
			return invalidArgument("holds confirm requires one hold id")
		}
		amount, err := optionalNonNegativeAmount(first(flags["amount"]))
		if err != nil {
			return err
		}
		final, err := optionalBool(first(flags["final"]))
		if err != nil {
			return err
		}
		_, err = generated.Wallets.V1.ConfirmHold(ctx, operations.ConfirmHoldRequest{
			HoldID: request.Arguments[0], IdempotencyKey: optionalString(first(flags["ik"])),
			ConfirmHoldRequest: &components.ConfirmHoldRequest{Amount: amount, Final: final},
		})
		if err != nil {
			return fmt.Errorf("wallets: confirm hold: %w", err)
		}
		return emitEmptyObject(host, command.ID)
	case "wallets.v2.holds.void":
		if len(request.Arguments) != 1 {
			return invalidArgument("holds void requires one hold id")
		}
		_, err := generated.Wallets.V1.VoidHold(ctx, operations.VoidHoldRequest{HoldID: request.Arguments[0], IdempotencyKey: optionalString(first(flags["ik"]))})
		if err != nil {
			return fmt.Errorf("wallets: void hold: %w", err)
		}
		return emitEmptyObject(host, command.ID)
	case "wallets.v2.transactions.list":
		items, page, err := collectPages(request.Continuation, func(cursorValue *string) ([]components.Transaction, *string, *bool, error) {
			listRequest := operations.GetTransactionsRequest{Cursor: cursorValue}
			if cursorValue == nil {
				listRequest.WalletID = optionalString(first(flags["id"]))
			}
			response, err := generated.Wallets.V1.GetTransactions(ctx, listRequest)
			if err != nil {
				return nil, nil, nil, fmt.Errorf("wallets: list transactions: %w", err)
			}
			if response == nil || response.GetTransactionsResponse == nil {
				return nil, nil, nil, fmt.Errorf("wallets: list transactions returned no result")
			}
			page := response.GetTransactionsResponse.Cursor
			return page.Data, page.Next, page.HasMore, nil
		})
		if err != nil {
			return err
		}
		return emitJSON(host, command.ID, sdk.ResultCollection, items, page)
	default:
		return sdk.Failure{Code: string(sdk.FailureOperationNotPermitted), Message: "wallets command is not implemented"}
	}
}

func commandByID(id string) (sdk.Command, bool) {
	for _, command := range Catalogue() {
		if command.ID == id {
			return command, true
		}
	}
	return sdk.Command{}, false
}

func collectFlags(command sdk.Command, occurrences []sdk.FlagOccurrence) (map[string][]string, error) {
	declared := make(map[string]sdk.Flag, len(command.Flags))
	for _, flag := range command.Flags {
		declared[flag.Name] = flag
	}
	values := make(map[string][]string, len(occurrences))
	for _, occurrence := range occurrences {
		flag, ok := declared[occurrence.Name]
		if !ok {
			return nil, invalidArgument("unknown flag %q", occurrence.Name)
		}
		if flag.Type != sdk.FlagStringArray && len(values[occurrence.Name]) != 0 {
			return nil, invalidArgument("flag %q is repeated", occurrence.Name)
		}
		values[occurrence.Name] = append(values[occurrence.Name], occurrence.Value)
	}
	for _, flag := range command.Flags {
		if !flag.Required {
			continue
		}
		nonEmpty := false
		for _, value := range values[flag.Name] {
			if strings.TrimSpace(value) != "" {
				nonEmpty = true
				break
			}
		}
		if !nonEmpty {
			return nil, invalidArgument("missing required flag %q", flag.Name)
		}
	}
	return values, nil
}

func parseMetadata(values []string) (map[string]string, error) {
	if len(values) == 0 {
		return map[string]string{}, nil
	}
	metadata := make(map[string]string, len(values))
	for _, value := range values {
		key, item, ok := strings.Cut(value, "=")
		if !ok || key == "" {
			return nil, invalidArgument("metadata must use key=value")
		}
		if _, exists := metadata[key]; exists {
			return nil, invalidArgument("metadata key %q is repeated", key)
		}
		metadata[key] = item
	}
	return metadata, nil
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func first(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func emitJSON(host sdk.Host, commandID string, shape sdk.ResultShape, value any, page *sdk.PageInfo) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("wallets: encode %s result: %w", commandID, err)
	}
	return host.Emit(sdk.Event{Kind: sdk.EventResult, Result: &sdk.ResultEnvelope{
		OperationID: commandID, Shape: shape, MediaType: "application/json", Data: encoded, Page: page,
	}})
}

func emitEmptyObject(host sdk.Host, commandID string) error {
	return host.Emit(sdk.Event{Kind: sdk.EventResult, Result: &sdk.ResultEnvelope{
		OperationID: commandID, Shape: sdk.ResultObject, MediaType: "application/json", Data: []byte(`{}`),
	}})
}

func pageInfo(next *string, hasMore *bool) *sdk.PageInfo {
	page := &sdk.PageInfo{}
	if next != nil {
		page.NextCursor = *next
	}
	if hasMore != nil {
		page.HasMore = *hasMore
	} else {
		page.HasMore = page.NextCursor != ""
	}
	return page
}

func collectPages[T any](control sdk.ContinuationControl, fetch func(*string) ([]T, *string, *bool, error)) ([]T, *sdk.PageInfo, error) {
	maxPages := uint32(1)
	allPages := control.Mode == sdk.ContinuationAllPages
	if allPages {
		maxPages = control.MaxPages
	}
	collected := make([]T, 0)
	var cursorValue *string
	seen := map[string]struct{}{}
	for page := uint32(0); page < maxPages; page++ {
		items, next, hasMore, err := fetch(cursorValue)
		if err != nil {
			return nil, nil, err
		}
		if !allPages {
			if items == nil {
				items = make([]T, 0)
			}
			return items, pageInfo(next, hasMore), nil
		}
		collected = append(collected, items...)
		if err := validateCollectionBudget(collected, control); err != nil {
			return nil, nil, err
		}
		more := hasMore != nil && *hasMore || next != nil && *next != ""
		if !more {
			return collected, nil, nil
		}
		if next == nil || *next == "" {
			return nil, nil, fmt.Errorf("wallets: paginated response returned hasMore without a next cursor")
		}
		if _, duplicate := seen[*next]; duplicate {
			return nil, nil, fmt.Errorf("wallets: paginated response returned a repeated cursor")
		}
		seen[*next] = struct{}{}
		cursorValue = next
	}
	return nil, nil, budgetExhausted("collection exceeded the page limit")
}

func validateCollectionBudget[T any](values []T, control sdk.ContinuationControl) error {
	if uint32(len(values)) > control.MaxItems {
		return budgetExhausted("collection exceeded the item limit")
	}
	encoded, err := json.Marshal(values)
	if err != nil {
		return fmt.Errorf("wallets: encode collection budget: %w", err)
	}
	if uint64(len(encoded)) > control.MaxBytes {
		return budgetExhausted("collection exceeded the byte limit")
	}
	return nil
}

func budgetExhausted(message string) error {
	return sdk.Failure{Code: string(sdk.FailureBudgetExhausted), Message: "wallets: " + message}
}

func parseSignedBigInt(value string) (*big.Int, error) {
	integer, ok := new(big.Int).SetString(value, 10)
	if !ok {
		return nil, invalidArgument("integer %q is invalid", value)
	}
	return integer, nil
}

func optionalSignedBigInt(value string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	return parseSignedBigInt(value)
}

func parseNonNegativeAmount(value string) (*big.Int, error) {
	amount, err := parseSignedBigInt(value)
	if err != nil || amount.Sign() < 0 {
		return nil, invalidArgument("amount %q is not a non-negative integer", value)
	}
	return amount, nil
}

func optionalNonNegativeAmount(value string) (*big.Int, error) {
	if value == "" {
		return nil, nil
	}
	return parseNonNegativeAmount(value)
}

func optionalBool(value string) (*bool, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return nil, invalidArgument("boolean value %q is invalid", value)
	}
	return &parsed, nil
}

func optionalTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, invalidArgument("date-time value %q is invalid", value)
	}
	return &parsed, nil
}

func parseSubjects(values []string) ([]components.Subject, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]components.Subject, 0, len(values))
	for _, value := range values {
		parsed, err := parseOptionalSubject(value)
		if err != nil {
			return nil, err
		}
		out = append(out, *parsed)
	}
	return out, nil
}

func parseOptionalSubject(value string) (*components.Subject, error) {
	if value == "" {
		return nil, nil
	}
	kind, identifier, ok := strings.Cut(value, "=")
	if !ok || identifier == "" {
		return nil, invalidArgument("subject %q is invalid", value)
	}
	switch kind {
	case "account":
		subject := components.CreateSubjectAccount(components.LedgerAccountSubject{Identifier: identifier})
		return &subject, nil
	case "wallet":
		if !strings.HasPrefix(identifier, "id:") {
			return nil, invalidArgument("wallet subject must use wallet=id:<wallet-id>[/<balance>]")
		}
		identifier = strings.TrimPrefix(identifier, "id:")
		walletID, balance, _ := strings.Cut(identifier, "/")
		if walletID == "" {
			return nil, invalidArgument("wallet subject id is empty")
		}
		wallet := components.WalletSubject{Identifier: walletID, Balance: optionalString(balance)}
		subject := components.CreateSubjectWallet(wallet)
		return &subject, nil
	default:
		return nil, invalidArgument("subject %q has an unsupported kind", value)
	}
}

func invalidArgument(format string, arguments ...any) error {
	return sdk.Failure{Code: string(sdk.FailureInvalidArgument), Message: fmt.Sprintf("wallets: "+format, arguments...)}
}

func descriptorInvalid(format string, arguments ...any) error {
	return sdk.Failure{Code: string(sdk.FailureDescriptorInvalid), Message: fmt.Sprintf("wallets: "+format, arguments...)}
}
