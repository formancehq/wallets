package core

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
	"github.com/formancehq/wallets/plugins/fctl/audit"
)

type expectedCommand struct {
	path      string
	aliases   [][]string
	operation string
	arguments []string
	flags     []string
	paginated bool
	mutating  bool
}

func TestCataloguePreservesWalletsV2Parity(t *testing.T) {
	expected := []expectedCommand{
		{path: "create", aliases: [][]string{{"cr"}}, operation: "createWallet", arguments: []string{"name"}, flags: []string{"metadata", "ik"}, mutating: true},
		{path: "list", aliases: [][]string{{"ls", "l"}}, operation: "listWallets", flags: []string{"metadata"}, paginated: true},
		{path: "show", aliases: [][]string{{"sh"}}, operation: "getWallet", flags: []string{"id"}},
		{path: "update", aliases: [][]string{{"up"}}, operation: "updateWallet", arguments: []string{"wallet-id"}, flags: []string{"ik", "metadata"}, mutating: true},
		{path: "credit", operation: "creditWallet", arguments: []string{"amount", "asset"}, flags: []string{"metadata", "balance", "ik", "source", "id"}, mutating: true},
		{path: "debit", aliases: [][]string{{"deb"}}, operation: "debitWallet", arguments: []string{"amount", "asset"}, flags: []string{"description", "ik", "pending", "metadata", "balance", "destination", "id"}, mutating: true},
		{path: "balances create", aliases: [][]string{{"balance", "bls", "bal"}, {"c", "cr"}}, operation: "createBalance", arguments: []string{"balance-name"}, flags: []string{"id", "expires-at", "priority", "ik"}, mutating: true},
		{path: "balances list", aliases: [][]string{{"balance", "bls", "bal"}, {"ls", "l"}}, operation: "listBalances", flags: []string{"id"}, paginated: true},
		{path: "balances show", aliases: [][]string{{"balance", "bls", "bal"}, {"sh"}}, operation: "getBalance", arguments: []string{"balance-name"}, flags: []string{"id"}},
		{path: "holds list", aliases: [][]string{{"h", "hold"}, {"ls", "l"}}, operation: "getHolds", flags: []string{"id", "metadata"}, paginated: true},
		{path: "holds show", aliases: [][]string{{"h", "hold"}, {"sh"}}, operation: "getHold", arguments: []string{"hold-id"}},
		{path: "holds confirm", aliases: [][]string{{"h", "hold"}, {"c", "conf"}}, operation: "confirmHold", arguments: []string{"hold-id"}, flags: []string{"final", "ik", "amount"}, mutating: true},
		{path: "holds void", aliases: [][]string{{"h", "hold"}, {"v"}}, operation: "voidHold", arguments: []string{"hold-id"}, flags: []string{"ik"}, mutating: true},
		{path: "transactions list", aliases: [][]string{{"transaction", "tx", "txs"}, {"ls", "l"}}, operation: "getTransactions", flags: []string{"id"}, paginated: true},
	}

	commands := Catalogue()
	if len(commands) != len(expected) {
		t.Fatalf("Catalogue() has %d commands, want %d", len(commands), len(expected))
	}
	if err := sdk.ValidateCatalogue(commands, nil); err != nil {
		t.Fatalf("ValidateCatalogue() error = %v", err)
	}

	report, err := audit.Build("../../../openapi.yaml")
	if err != nil {
		t.Fatalf("audit.Build() error = %v", err)
	}
	operations := make(map[string]audit.Record, len(report.Operations))
	for _, operation := range report.Operations {
		operations[operation.OperationID] = operation
	}

	seenOperations := make(map[string]struct{}, len(commands))
	for index, want := range expected {
		command := commands[index]
		if got := strings.Join(command.Path, " "); got != want.path {
			t.Errorf("command %d path = %q, want %q", index, got, want.path)
		}
		if !reflect.DeepEqual(command.PathAliases, want.aliases) {
			t.Errorf("%s aliases = %#v, want %#v", want.path, command.PathAliases, want.aliases)
		}
		if got := fieldNames(command.Arguments); !reflect.DeepEqual(got, want.arguments) {
			t.Errorf("%s arguments = %#v, want %#v", want.path, got, want.arguments)
		}
		if got := flagNames(command.Flags); !reflect.DeepEqual(got, want.flags) {
			t.Errorf("%s flags = %#v, want %#v", want.path, got, want.flags)
		}
		if command.ID != "wallets.v2."+strings.ReplaceAll(want.path, " ", ".") {
			t.Errorf("%s id = %q", want.path, command.ID)
		}
		if command.ExecutionKind != sdk.ExecutionKindService || command.AuthMode != sdk.AuthModeCapability || command.Target.Kind != sdk.TargetStack {
			t.Errorf("%s has wrong execution boundary", want.path)
		}
		if !reflect.DeepEqual(command.Auth, []sdk.AuthRequirement{{Capability: "auth.stack"}}) {
			t.Errorf("%s auth = %#v", want.path, command.Auth)
		}
		if !reflect.DeepEqual(command.Compatibility, []sdk.ServiceCompatibility{{Service: sdk.ServiceWallets, Majors: []uint32{2}}}) {
			t.Errorf("%s compatibility = %#v", want.path, command.Compatibility)
		}
		if len(command.Operations) != 1 {
			t.Errorf("%s operations = %d, want 1", want.path, len(command.Operations))
			continue
		}
		policy := command.Operations[0]
		contract, ok := operations[want.operation]
		if !ok {
			t.Fatalf("audit has no operation %q", want.operation)
		}
		if policy.ID != want.operation || policy.Service != sdk.ServiceWallets || !reflect.DeepEqual(policy.Scopes, contract.Scopes) {
			t.Errorf("%s operation identity/scopes drifted: %#v", want.path, policy)
		}
		if policy.HTTP == nil || policy.HTTP.Method != contract.Method || policy.HTTP.GeneratedClient == nil || policy.HTTP.GeneratedClient.PathTemplate != contract.Path {
			t.Errorf("%s HTTP policy does not match %s %s", want.path, contract.Method, contract.Path)
		}
		if command.Pagination.Supported != want.paginated {
			t.Errorf("%s pagination = %t, want %t", want.path, command.Pagination.Supported, want.paginated)
		}
		wantRequests := uint32(1)
		if want.paginated {
			wantRequests = sdk.DefaultAllPagesMaxPages
		}
		if command.ExecutionPolicy == nil || command.ExecutionPolicy.MaxHostRequests != wantRequests {
			t.Errorf("%s request budget is not %d", want.path, wantRequests)
		}
		wantRisk := sdk.RiskRead
		if want.mutating {
			wantRisk = sdk.RiskMutation
		}
		if command.Risk != wantRisk {
			t.Errorf("%s risk = %q, want %q", want.path, command.Risk, wantRisk)
		}
		seenOperations[want.operation] = struct{}{}
	}

	wantOperations := audit.BaselineTargets()
	gotOperations := make([]string, 0, len(seenOperations))
	for operation := range seenOperations {
		gotOperations = append(gotOperations, operation)
	}
	sort.Strings(gotOperations)
	if !reflect.DeepEqual(gotOperations, wantOperations) {
		t.Errorf("catalogue operations = %#v, want legacy baseline %#v", gotOperations, wantOperations)
	}
}

func TestBigIntegerFlagsUseLosslessStringContract(t *testing.T) {
	tests := []struct {
		commandID string
		flagName  string
	}{
		{commandID: "wallets.v2.balances.create", flagName: "priority"},
		{commandID: "wallets.v2.holds.confirm", flagName: "amount"},
	}

	for _, test := range tests {
		t.Run(test.commandID+"/"+test.flagName, func(t *testing.T) {
			command, ok := commandByID(test.commandID)
			if !ok {
				t.Fatalf("command %q is missing", test.commandID)
			}
			for _, flag := range command.Flags {
				if flag.Name == test.flagName {
					if flag.Type != sdk.FlagString {
						t.Fatalf("flag %q type = %q, want lossless string", test.flagName, flag.Type)
					}
					return
				}
			}
			t.Fatalf("flag %q is missing", test.flagName)
		})
	}
}

func TestFundMovingCommandsRequireIdempotencyKey(t *testing.T) {
	for _, commandID := range []string{
		"wallets.v2.credit",
		"wallets.v2.debit",
		"wallets.v2.holds.confirm",
		"wallets.v2.holds.void",
	} {
		t.Run(commandID, func(t *testing.T) {
			command, ok := commandByID(commandID)
			if !ok {
				t.Fatalf("command %q is missing", commandID)
			}
			for _, flag := range command.Flags {
				if flag.Name == "ik" {
					if !flag.Required {
						t.Fatal("idempotency key flag is optional")
					}
					return
				}
			}
			t.Fatal("idempotency key flag is missing")
		})
	}
}

func fieldNames(fields []sdk.Argument) []string {
	if len(fields) == 0 {
		return nil
	}
	out := make([]string, len(fields))
	for index := range fields {
		out[index] = fields[index].Name
	}
	return out
}

func flagNames(flags []sdk.Flag) []string {
	if len(flags) == 0 {
		return nil
	}
	out := make([]string, len(flags))
	for index := range flags {
		out[index] = flags[index].Name
	}
	return out
}
