// Package core implements the product-owned fctl command surface for Wallets.
package core

import (
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const productMajor uint32 = 2

const (
	readRequestBytes  int64 = 64 << 10
	writeRequestBytes int64 = 1 << 20
	responseBytes     int64 = 512 << 10
)

type commandSpec struct {
	path      []string
	aliases   [][]string
	summary   string
	arguments []sdk.Argument
	flags     []sdk.Flag
	operation operationSpec
	paginated bool
	mutating  bool
}

type operationSpec struct {
	id          string
	method      string
	path        string
	scopes      []string
	requestBody bool
	headers     []string
}

// Catalogue returns the frozen Wallets v2 parity surface. Service discovery is
// host-owned, so GET /_info is deliberately not exposed as a product command.
func Catalogue() []sdk.Command {
	specs := catalogueSpecs()
	commands := make([]sdk.Command, 0, len(specs))
	for _, spec := range specs {
		commands = append(commands, spec.command())
	}
	return commands
}

func (spec commandSpec) command() sdk.Command {
	maxRequests := uint32(1)
	outputSchema := objectSchema
	if spec.paginated {
		maxRequests = sdk.DefaultAllPagesMaxPages
		outputSchema = collectionSchema
	}
	risk := sdk.RiskRead
	if spec.mutating {
		risk = sdk.RiskMutation
	}
	requestBytes := readRequestBytes
	contentTypes := []string(nil)
	if spec.operation.requestBody {
		requestBytes = writeRequestBytes
		contentTypes = []string{"application/json"}
	}

	arguments := normalizeArguments(spec.arguments)
	flags := normalizeFlags(spec.flags)
	return sdk.Command{
		ID:                 "wallets.v2." + strings.Join(spec.path, "."),
		ExecutionKind:      sdk.ExecutionKindService,
		AuthMode:           sdk.AuthModeCapability,
		Path:               spec.path,
		PathAliases:        spec.aliases,
		Target:             sdk.TargetRequirement{Kind: sdk.TargetStack},
		Summary:            spec.summary,
		Long:               spec.summary + ". The endpoint, credentials and transport are supplied by the fctl host.",
		Example:            exampleFor(spec.path),
		Arguments:          arguments,
		Flags:              flags,
		Auth:               []sdk.AuthRequirement{{Capability: "auth.stack"}},
		Operations:         []sdk.OperationPolicy{spec.operation.policy(requestBytes, contentTypes)},
		Compatibility:      []sdk.ServiceCompatibility{{Service: sdk.ServiceWallets, Majors: []uint32{productMajor}}},
		Risk:               risk,
		InputSchema:        buildInputSchema(arguments, flags),
		RawOutputSchema:    outputSchema,
		PublicOutputSchema: outputSchema,
		Pagination:         sdk.PaginationSpec{Supported: spec.paginated},
		OutputMediaType:    "application/json",
		ExecutionPolicy:    &sdk.CommandExecutionPolicy{MaxHostRequests: maxRequests},
	}
}

func (spec operationSpec) policy(requestBytes int64, contentTypes []string) sdk.OperationPolicy {
	return sdk.OperationPolicy{
		ID:      spec.id,
		Service: sdk.ServiceWallets,
		Scopes:  append([]string(nil), spec.scopes...),
		HTTP: &sdk.HTTPOperationPolicy{
			Method: spec.method,
			GeneratedClient: &sdk.HTTPGeneratedClientPolicy{
				PathTemplate:        spec.path,
				RequestContentTypes: contentTypes,
				RequestHeaders:      append([]string{"Accept"}, spec.headers...),
				MaxRequestBytes:     requestBytes,
				ResponseLimits: sdk.ResponseLimits{
					MaxMessageBytes: responseBytes, MaxMessages: 1, MaxAggregateBytes: responseBytes,
				},
			},
		},
	}
}

func catalogueSpecs() []commandSpec {
	write := []string{"wallets:write"}
	read := []string{"wallets:read"}
	idempotency := []string{"Idempotency-Key"}
	balanceAliases := []string{"balance", "bls", "bal"}
	holdAliases := []string{"h", "hold"}
	transactionAliases := []string{"transaction", "tx", "txs"}
	return []commandSpec{
		{path: []string{"create"}, aliases: [][]string{{"cr"}}, summary: "Create a wallet", arguments: []sdk.Argument{stringArgument("name", "Wallet name")}, flags: []sdk.Flag{stringArrayFlag("metadata", "Metadata entries as key=value"), idempotencyKeyFlag(false)}, operation: operationSpec{id: "createWallet", method: "POST", path: "/wallets", scopes: write, requestBody: true, headers: idempotency}, mutating: true},
		{path: []string{"list"}, aliases: [][]string{{"ls", "l"}}, summary: "List wallets", flags: []sdk.Flag{stringArrayFlag("metadata", "Metadata entries as key=value")}, operation: operationSpec{id: "listWallets", method: "GET", path: "/wallets", scopes: read}, paginated: true},
		{path: []string{"show"}, aliases: [][]string{{"sh"}}, summary: "Show a wallet", flags: []sdk.Flag{walletIDFlag(true)}, operation: operationSpec{id: "getWallet", method: "GET", path: "/wallets/{id}", scopes: read}},
		{path: []string{"update"}, aliases: [][]string{{"up"}}, summary: "Update wallet metadata", arguments: []sdk.Argument{stringArgument("wallet-id", "Wallet id")}, flags: []sdk.Flag{idempotencyKeyFlag(false), stringArrayFlag("metadata", "Metadata entries as key=value")}, operation: operationSpec{id: "updateWallet", method: "PATCH", path: "/wallets/{id}", scopes: write, requestBody: true, headers: idempotency}, mutating: true},
		{path: []string{"credit"}, summary: "Credit a wallet", arguments: []sdk.Argument{stringArgument("amount", "Decimal amount"), stringArgument("asset", "Asset code")}, flags: append([]sdk.Flag{stringArrayFlag("metadata", "Metadata entries as key=value"), stringFlag("balance", "Balance to credit"), idempotencyKeyFlag(true), stringArrayFlag("source", "Source account or wallet by id")}, walletIDFlag(true)), operation: operationSpec{id: "creditWallet", method: "POST", path: "/wallets/{id}/credit", scopes: write, requestBody: true, headers: idempotency}, mutating: true},
		{path: []string{"debit"}, aliases: [][]string{{"deb"}}, summary: "Debit a wallet", arguments: []sdk.Argument{stringArgument("amount", "Decimal amount"), stringArgument("asset", "Asset code")}, flags: append([]sdk.Flag{stringFlag("description", "Debit description"), idempotencyKeyFlag(true), boolFlag("pending", "Create a pending debit"), stringArrayFlag("metadata", "Metadata entries as key=value"), stringArrayFlag("balance", "Balances to debit"), stringFlag("destination", "Destination account or wallet by id")}, walletIDFlag(true)), operation: operationSpec{id: "debitWallet", method: "POST", path: "/wallets/{id}/debit", scopes: write, requestBody: true, headers: idempotency}, mutating: true},
		{path: []string{"balances", "create"}, aliases: [][]string{balanceAliases, {"c", "cr"}}, summary: "Create a wallet balance", arguments: []sdk.Argument{stringArgument("balance-name", "Balance name")}, flags: []sdk.Flag{walletIDFlag(true), stringFlag("expires-at", "Balance expiration date"), bigIntFlag("priority", "Balance priority"), idempotencyKeyFlag(false)}, operation: operationSpec{id: "createBalance", method: "POST", path: "/wallets/{id}/balances", scopes: write, requestBody: true, headers: idempotency}, mutating: true},
		{path: []string{"balances", "list"}, aliases: [][]string{balanceAliases, {"ls", "l"}}, summary: "List wallet balances", flags: []sdk.Flag{walletIDFlag(true)}, operation: operationSpec{id: "listBalances", method: "GET", path: "/wallets/{id}/balances", scopes: read}, paginated: true},
		{path: []string{"balances", "show"}, aliases: [][]string{balanceAliases, {"sh"}}, summary: "Show a wallet balance", arguments: []sdk.Argument{stringArgument("balance-name", "Balance name")}, flags: []sdk.Flag{walletIDFlag(true)}, operation: operationSpec{id: "getBalance", method: "GET", path: "/wallets/{id}/balances/{balanceName}", scopes: read}},
		{path: []string{"holds", "list"}, aliases: [][]string{holdAliases, {"ls", "l"}}, summary: "List wallet holds", flags: []sdk.Flag{walletIDFlag(false), stringArrayFlag("metadata", "Metadata entries as key=value")}, operation: operationSpec{id: "getHolds", method: "GET", path: "/holds", scopes: read}, paginated: true},
		{path: []string{"holds", "show"}, aliases: [][]string{holdAliases, {"sh"}}, summary: "Show a hold", arguments: []sdk.Argument{stringArgument("hold-id", "Hold id")}, operation: operationSpec{id: "getHold", method: "GET", path: "/holds/{holdID}", scopes: read}},
		{path: []string{"holds", "confirm"}, aliases: [][]string{holdAliases, {"c", "conf"}}, summary: "Confirm a hold", arguments: []sdk.Argument{stringArgument("hold-id", "Hold id")}, flags: []sdk.Flag{boolFlag("final", "Close the hold after confirmation"), idempotencyKeyFlag(true), bigIntFlag("amount", "Amount to confirm")}, operation: operationSpec{id: "confirmHold", method: "POST", path: "/holds/{hold_id}/confirm", scopes: write, requestBody: true, headers: idempotency}, mutating: true},
		{path: []string{"holds", "void"}, aliases: [][]string{holdAliases, {"v"}}, summary: "Void a hold", arguments: []sdk.Argument{stringArgument("hold-id", "Hold id")}, flags: []sdk.Flag{idempotencyKeyFlag(true)}, operation: operationSpec{id: "voidHold", method: "POST", path: "/holds/{hold_id}/void", scopes: write, headers: idempotency}, mutating: true},
		{path: []string{"transactions", "list"}, aliases: [][]string{transactionAliases, {"ls", "l"}}, summary: "List wallet transactions", flags: []sdk.Flag{walletIDFlag(false)}, operation: operationSpec{id: "getTransactions", method: "GET", path: "/transactions", scopes: read}, paginated: true},
	}
}

func exampleFor(path []string) string { return strings.Join(path, " ") + " --help" }

func stringArgument(name, usage string) sdk.Argument {
	return sdk.Argument{Name: name, Usage: usage, Type: sdk.ArgumentString, Required: true}
}

func stringFlag(name, usage string) sdk.Flag {
	return sdk.Flag{Name: name, Usage: usage, Type: sdk.FlagString}
}
func stringArrayFlag(name, usage string) sdk.Flag {
	return sdk.Flag{Name: name, Usage: usage, Type: sdk.FlagStringArray}
}
func boolFlag(name, usage string) sdk.Flag {
	return sdk.Flag{Name: name, Usage: usage, Type: sdk.FlagBool}
}
func bigIntFlag(name, usage string) sdk.Flag {
	return stringFlag(name, usage)
}

func idempotencyKeyFlag(required bool) sdk.Flag {
	value := stringFlag("ik", "Idempotency key")
	value.Required = required
	return value
}

func walletIDFlag(required bool) sdk.Flag {
	value := stringFlag("id", "Wallet id")
	value.Required = required
	return value
}

func normalizeArguments(values []sdk.Argument) []sdk.Argument {
	for index := range values {
		values[index].Completion.Kind = sdk.CompletionNone
	}
	return values
}

func normalizeFlags(values []sdk.Flag) []sdk.Flag {
	for index := range values {
		values[index].Completion.Kind = sdk.CompletionNone
	}
	return values
}
