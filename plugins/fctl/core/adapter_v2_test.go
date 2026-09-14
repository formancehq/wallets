package core

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

func TestExecuteCreateWalletUsesTheGeneratedClientBridge(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.Service != sdk.ServiceWallets || request.Capability != "auth.stack" || request.Operation != "createWallet" {
			t.Fatalf("host request identity = %#v", request)
		}
		if request.HTTP == nil || request.HTTP.Method != "POST" || request.HTTP.Path != "/wallets" || request.HTTP.ContentType != "application/json" {
			t.Fatalf("HTTP request = %#v", request.HTTP)
		}
		if got := request.HTTP.Headers["idempotency-key"]; !reflect.DeepEqual(got, []string{"create-1"}) {
			t.Fatalf("idempotency header = %#v", got)
		}
		var body map[string]any
		if err := json.Unmarshal(request.HTTP.Body, &body); err != nil {
			t.Fatalf("request body is invalid JSON: %v", err)
		}
		if body["name"] != "primary" || !reflect.DeepEqual(body["metadata"], map[string]any{"owner": "alice"}) {
			t.Fatalf("request body = %#v", body)
		}
		return sdk.NewResponseStream(sdk.Response{
			Status: 201, ContentType: "application/json",
			Body: []byte(`{"data":{"id":"w1","metadata":{"owner":"alice"},"name":"primary","createdAt":"2026-09-11T10:00:00Z","ledger":"default"}}`),
		}), nil
	})

	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID:       "wallets.v2.create",
		Arguments:       []string{"primary"},
		Flags:           []sdk.FlagOccurrence{{Name: "metadata", Value: "owner=alice"}, {Name: "ik", Value: "create-1"}},
		Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
		ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
		Continuation:    sdk.SinglePageContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if len(host.Requests()) != 1 {
		t.Fatalf("host requests = %d, want 1", len(host.Requests()))
	}
	events := host.Events()
	if len(events) != 1 || events[0].Kind != sdk.EventResult || events[0].Result == nil {
		t.Fatalf("events = %#v", events)
	}
	if events[0].Result.OperationID != "wallets.v2.create" || events[0].Result.Shape != sdk.ResultObject {
		t.Fatalf("result = %#v", events[0].Result)
	}
	var result map[string]any
	if err := json.Unmarshal(events[0].Result.Data, &result); err != nil || result["id"] != "w1" || result["name"] != "primary" {
		t.Fatalf("result data = %s, err = %v", events[0].Result.Data, err)
	}
}

func TestExecuteCoversEveryRemainingWalletsCommand(t *testing.T) {
	tests := []struct {
		name      string
		commandID string
		operation string
		arguments []string
		flags     []sdk.FlagOccurrence
		status    int32
		body      string
		wantShape sdk.ResultShape
	}{
		{name: "list wallets", commandID: "wallets.v2.list", operation: "listWallets", flags: []sdk.FlagOccurrence{{Name: "metadata", Value: "tier=gold"}}, status: 200, body: `{"cursor":{"pageSize":15,"data":[]}}`, wantShape: sdk.ResultCollection},
		{name: "show wallet", commandID: "wallets.v2.show", operation: "getWallet", flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 200, body: `{}`, wantShape: sdk.ResultObject},
		{name: "update wallet", commandID: "wallets.v2.update", operation: "updateWallet", arguments: []string{"w1"}, flags: []sdk.FlagOccurrence{{Name: "metadata", Value: "tier=gold"}}, status: 204, wantShape: sdk.ResultObject},
		{name: "credit wallet", commandID: "wallets.v2.credit", operation: "creditWallet", arguments: []string{"42", "USD/2"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "ik", Value: "credit-1"}}, status: 204, wantShape: sdk.ResultObject},
		{name: "debit wallet", commandID: "wallets.v2.debit", operation: "debitWallet", arguments: []string{"42", "USD/2"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "ik", Value: "debit-1"}}, status: 204, wantShape: sdk.ResultObject},
		{name: "create balance", commandID: "wallets.v2.balances.create", operation: "createBalance", arguments: []string{"savings"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 201, body: `{}`, wantShape: sdk.ResultObject},
		{name: "list balances", commandID: "wallets.v2.balances.list", operation: "listBalances", flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 200, body: `{"cursor":{"pageSize":15,"data":[]}}`, wantShape: sdk.ResultCollection},
		{name: "show balance", commandID: "wallets.v2.balances.show", operation: "getBalance", arguments: []string{"savings"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 200, body: `{}`, wantShape: sdk.ResultObject},
		{name: "list holds", commandID: "wallets.v2.holds.list", operation: "getHolds", status: 200, body: `{"cursor":{"pageSize":15,"data":[]}}`, wantShape: sdk.ResultCollection},
		{name: "show hold", commandID: "wallets.v2.holds.show", operation: "getHold", arguments: []string{"h1"}, status: 200, body: `{}`, wantShape: sdk.ResultObject},
		{name: "confirm hold", commandID: "wallets.v2.holds.confirm", operation: "confirmHold", arguments: []string{"h1"}, flags: []sdk.FlagOccurrence{{Name: "ik", Value: "confirm-1"}}, status: 204, wantShape: sdk.ResultObject},
		{name: "void hold", commandID: "wallets.v2.holds.void", operation: "voidHold", arguments: []string{"h1"}, flags: []sdk.FlagOccurrence{{Name: "ik", Value: "void-1"}}, status: 204, wantShape: sdk.ResultObject},
		{name: "list transactions", commandID: "wallets.v2.transactions.list", operation: "getTransactions", status: 200, body: `{"cursor":{"pageSize":15,"data":[]}}`, wantShape: sdk.ResultCollection},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
				if request.Operation != test.operation || request.Service != sdk.ServiceWallets || request.Capability != "auth.stack" {
					t.Fatalf("host request = %#v", request)
				}
				return sdk.NewResponseStream(sdk.Response{Status: test.status, ContentType: contentTypeFor(test.body), Body: []byte(test.body)}), nil
			})
			err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
				CommandID: test.commandID, Arguments: test.arguments, Flags: test.flags,
				Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
				ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
				Continuation:    sdk.SinglePageContinuationControl(),
			}, host)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if len(host.Requests()) != 1 {
				t.Fatalf("host requests = %d, want 1", len(host.Requests()))
			}
			events := host.Events()
			if len(events) != 1 || events[0].Result == nil || events[0].Result.OperationID != test.commandID || events[0].Result.Shape != test.wantShape {
				t.Fatalf("events = %#v", events)
			}
			if test.status == 204 && string(events[0].Result.Data) != `{}` {
				t.Fatalf("empty object result = %s", events[0].Result.Data)
			}
		})
	}
}

func TestExecuteListWalletsTraversesCanonicalAllPages(t *testing.T) {
	call := 0
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		call++
		if request.Operation != "listWallets" || request.HTTP == nil {
			t.Fatalf("host request = %#v", request)
		}
		switch call {
		case 1:
			if request.HTTP.Query["cursor"] != nil {
				t.Fatalf("first cursor = %#v", request.HTTP.Query["cursor"])
			}
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":true,"next":"next-2","data":[{"id":"w1","metadata":{},"name":"one","createdAt":"2026-09-11T10:00:00Z","ledger":"default"}]}}`)}), nil
		case 2:
			if got := request.HTTP.Query; !reflect.DeepEqual(got, map[string][]string{"cursor": {"next-2"}}) {
				t.Fatalf("second-page query = %#v", got)
			}
			return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(`{"cursor":{"pageSize":1,"hasMore":false,"data":[{"id":"w2","metadata":{},"name":"two","createdAt":"2026-09-11T10:00:00Z","ledger":"default"}]}}`)}), nil
		default:
			t.Fatalf("unexpected request %d", call)
			return nil, nil
		}
	})

	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID:       "wallets.v2.list",
		Flags:           []sdk.FlagOccurrence{{Name: "metadata", Value: "tier=gold"}},
		Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
		ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
		Continuation:    sdk.AllPagesContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if call != 2 {
		t.Fatalf("host calls = %d, want 2", call)
	}
	events := host.Events()
	if len(events) != 1 || events[0].Result == nil || events[0].Result.Page != nil {
		t.Fatalf("events = %#v", events)
	}
	var wallets []map[string]any
	if err := json.Unmarshal(events[0].Result.Data, &wallets); err != nil || len(wallets) != 2 {
		t.Fatalf("result = %s, err = %v", events[0].Result.Data, err)
	}
}

func TestExecuteCreateBalanceForwardsIdempotencyKey(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.Operation != "createBalance" || request.HTTP == nil {
			t.Fatalf("host request = %#v", request)
		}
		if got := request.HTTP.Headers["idempotency-key"]; !reflect.DeepEqual(got, []string{"balance-1"}) {
			t.Fatalf("idempotency header = %#v", got)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 201, ContentType: "application/json", Body: []byte(`{}`)}), nil
	})

	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "wallets.v2.balances.create", Arguments: []string{"savings"},
		Flags:           []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "ik", Value: "balance-1"}},
		Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
		ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
		Continuation:    sdk.SinglePageContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecutePreservesServerSizedBigIntegerFlagsExactly(t *testing.T) {
	tests := []struct {
		name      string
		commandID string
		operation string
		arguments []string
		flags     []sdk.FlagOccurrence
		field     string
		value     string
		status    int32
		body      string
	}{
		{
			name: "balance priority", commandID: "wallets.v2.balances.create", operation: "createBalance",
			arguments: []string{"savings"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "priority", Value: "-2147483649"}, {Name: "ik", Value: "balance-1"}},
			field: "priority", value: "-2147483649", status: 201, body: `{}`,
		},
		{
			name: "hold confirmation amount", commandID: "wallets.v2.holds.confirm", operation: "confirmHold",
			arguments: []string{"h1"}, flags: []sdk.FlagOccurrence{{Name: "amount", Value: "2147483648"}, {Name: "ik", Value: "confirm-1"}},
			field: "amount", value: "2147483648", status: 204,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
				if request.Operation != test.operation || request.HTTP == nil {
					t.Fatalf("host request = %#v", request)
				}
				decoder := json.NewDecoder(strings.NewReader(string(request.HTTP.Body)))
				decoder.UseNumber()
				var body map[string]any
				if err := decoder.Decode(&body); err != nil {
					t.Fatalf("request body is invalid JSON: %v", err)
				}
				value, ok := body[test.field].(json.Number)
				if !ok || value.String() != test.value {
					t.Fatalf("%s = %#v, want exact integer %s", test.field, body[test.field], test.value)
				}
				return sdk.NewResponseStream(sdk.Response{Status: test.status, ContentType: contentTypeFor(test.body), Body: []byte(test.body)}), nil
			})
			err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
				CommandID: test.commandID, Arguments: test.arguments, Flags: test.flags,
				Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
				ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
				Continuation:    sdk.SinglePageContinuationControl(),
			}, host)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
		})
	}
}

func TestExecuteCarriesBeyondInt64PriorityAcrossPortableTransport(t *testing.T) {
	const priority = "922337203685477580812345678901234567890"
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.Operation != "createBalance" || request.HTTP == nil {
			t.Fatalf("host request = %#v", request)
		}
		decoder := json.NewDecoder(strings.NewReader(string(request.HTTP.Body)))
		decoder.UseNumber()
		var body map[string]any
		if err := decoder.Decode(&body); err != nil {
			t.Fatalf("request body is invalid JSON: %v", err)
		}
		value, ok := body["priority"].(json.Number)
		if !ok || value.String() != priority {
			t.Fatalf("priority = %#v, want exact portable value %s", body["priority"], priority)
		}
		return sdk.NewResponseStream(sdk.Response{Status: 201, ContentType: "application/json", Body: []byte(`{}`)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "wallets.v2.balances.create", Arguments: []string{"savings"},
		Flags:           []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "priority", Value: priority}, {Name: "ik", Value: "balance-1"}},
		Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
		ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
		Continuation:    sdk.SinglePageContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
}

func TestExecuteRejectsNegativeHoldAmountBeforeHostAccess(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		t.Fatalf("host received request with negative hold amount: %#v", request)
		return nil, nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "wallets.v2.holds.confirm", Arguments: []string{"h1"},
		Flags:           []sdk.FlagOccurrence{{Name: "amount", Value: "-1"}, {Name: "ik", Value: "confirm-1"}},
		Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
		ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
		Continuation:    sdk.SinglePageContinuationControl(),
	}, host)
	if err == nil || !strings.Contains(err.Error(), "non-negative integer") {
		t.Fatalf("Execute() error = %v, want negative amount rejection", err)
	}
	if len(host.Requests()) != 0 {
		t.Fatalf("host requests = %d, want 0", len(host.Requests()))
	}
}

func TestExecuteFundMovingCommandsRequireAndForwardIdempotencyKey(t *testing.T) {
	tests := []struct {
		name      string
		commandID string
		operation string
		arguments []string
		flags     []sdk.FlagOccurrence
		status    int32
		body      string
	}{
		{name: "credit", commandID: "wallets.v2.credit", operation: "creditWallet", arguments: []string{"42", "USD/2"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 204},
		{name: "debit", commandID: "wallets.v2.debit", operation: "debitWallet", arguments: []string{"42", "USD/2"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 204},
		{name: "confirm hold", commandID: "wallets.v2.holds.confirm", operation: "confirmHold", arguments: []string{"h1"}, status: 204},
		{name: "void hold", commandID: "wallets.v2.holds.void", operation: "voidHold", arguments: []string{"h1"}, status: 204},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Run("rejects missing key before host access", func(t *testing.T) {
				host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
					t.Fatalf("host received request without idempotency key: %#v", request)
					return nil, nil
				})
				err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
					CommandID: test.commandID, Arguments: test.arguments, Flags: test.flags,
					Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
					ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
					Continuation:    sdk.SinglePageContinuationControl(),
				}, host)
				if err == nil || !strings.Contains(err.Error(), `missing required flag "ik"`) {
					t.Fatalf("Execute() error = %v, want missing required idempotency key", err)
				}
				if len(host.Requests()) != 0 {
					t.Fatalf("host requests = %d, want 0", len(host.Requests()))
				}
			})

			t.Run("rejects empty key before host access", func(t *testing.T) {
				host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
					t.Fatalf("host received request with empty idempotency key: %#v", request)
					return nil, nil
				})
				flags := append(append([]sdk.FlagOccurrence(nil), test.flags...), sdk.FlagOccurrence{Name: "ik", Value: " "})
				err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
					CommandID: test.commandID, Arguments: test.arguments, Flags: flags,
					Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
					ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
					Continuation:    sdk.SinglePageContinuationControl(),
				}, host)
				if err == nil || !strings.Contains(err.Error(), `missing required flag "ik"`) {
					t.Fatalf("Execute() error = %v, want empty required idempotency key rejection", err)
				}
				if len(host.Requests()) != 0 {
					t.Fatalf("host requests = %d, want 0", len(host.Requests()))
				}
			})

			t.Run("forwards key", func(t *testing.T) {
				host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
					if request.Operation != test.operation || request.HTTP == nil {
						t.Fatalf("host request = %#v", request)
					}
					if got := request.HTTP.Headers["idempotency-key"]; !reflect.DeepEqual(got, []string{"move-1"}) {
						t.Fatalf("idempotency header = %#v", got)
					}
					return sdk.NewResponseStream(sdk.Response{Status: test.status, ContentType: contentTypeFor(test.body), Body: []byte(test.body)}), nil
				})
				flags := append(append([]sdk.FlagOccurrence(nil), test.flags...), sdk.FlagOccurrence{Name: "ik", Value: "move-1"})
				err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
					CommandID: test.commandID, Arguments: test.arguments, Flags: flags,
					Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
					ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
					Continuation:    sdk.SinglePageContinuationControl(),
				}, host)
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
			})
		})
	}
}

func TestExecuteListWalletsEmitsAnEmptyJSONArray(t *testing.T) {
	for _, continuation := range []sdk.ContinuationControl{
		sdk.SinglePageContinuationControl(),
		sdk.AllPagesContinuationControl(),
	} {
		host := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
			return sdk.NewResponseStream(sdk.Response{
				Status: 200, ContentType: "application/json",
				Body: []byte(`{"cursor":{"pageSize":15,"hasMore":false}}`),
			}), nil
		})
		err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
			CommandID:       "wallets.v2.list",
			Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
			ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
			Continuation:    continuation,
		}, host)
		if err != nil {
			t.Fatalf("Execute() error = %v", err)
		}
		events := host.Events()
		if len(events) != 1 || events[0].Result == nil || string(events[0].Result.Data) != "[]" {
			t.Fatalf("events = %#v", events)
		}
	}
}

func TestExecuteTraversesAllPagesForEveryPaginatedFamily(t *testing.T) {
	tests := []struct {
		name      string
		commandID string
		operation string
		flags     []sdk.FlagOccurrence
		firstBody string
		lastBody  string
	}{
		{name: "balances", commandID: "wallets.v2.balances.list", operation: "listBalances", flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, firstBody: `{"cursor":{"pageSize":1,"hasMore":true,"next":"next-2","data":[{}]}}`, lastBody: `{"cursor":{"pageSize":1,"hasMore":false,"data":[{}]}}`},
		{name: "holds", commandID: "wallets.v2.holds.list", operation: "getHolds", flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "metadata", Value: "tier=gold"}}, firstBody: `{"cursor":{"pageSize":1,"hasMore":true,"next":"next-2","data":[{}]}}`, lastBody: `{"cursor":{"pageSize":1,"hasMore":false,"data":[{}]}}`},
		{name: "transactions", commandID: "wallets.v2.transactions.list", operation: "getTransactions", flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, firstBody: `{"cursor":{"pageSize":1,"hasMore":true,"next":"next-2","data":[{}]}}`, lastBody: `{"cursor":{"pageSize":1,"hasMore":false,"data":[{}]}}`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			call := 0
			host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
				call++
				if request.Operation != test.operation || request.HTTP == nil {
					t.Fatalf("host request = %#v", request)
				}
				body := test.firstBody
				if call == 2 {
					if got := request.HTTP.Query; !reflect.DeepEqual(got, map[string][]string{"cursor": {"next-2"}}) {
						t.Fatalf("second-page query = %#v", got)
					}
					body = test.lastBody
				}
				return sdk.NewResponseStream(sdk.Response{Status: 200, ContentType: "application/json", Body: []byte(body)}), nil
			})
			err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
				CommandID: test.commandID, Flags: test.flags,
				Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
				ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
				Continuation:    sdk.AllPagesContinuationControl(),
			}, host)
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if call != 2 {
				t.Fatalf("host calls = %d, want 2", call)
			}
			var items []any
			events := host.Events()
			if len(events) != 1 || events[0].Result == nil || events[0].Result.Page != nil {
				t.Fatalf("events = %#v", events)
			}
			if err := json.Unmarshal(events[0].Result.Data, &items); err != nil || len(items) != 2 {
				t.Fatalf("result = %s, err = %v", events[0].Result.Data, err)
			}
		})
	}
}

func TestExecuteDebitWalletMapsMutationFlagsAndEmitsPendingHold(t *testing.T) {
	host := sdk.NewMemoryHost(func(_ context.Context, request sdk.Request) (sdk.Responses, error) {
		if request.Operation != "debitWallet" || request.HTTP == nil || request.HTTP.Path != "/wallets/w1/debit" {
			t.Fatalf("host request = %#v", request)
		}
		if got := request.HTTP.Headers["idempotency-key"]; !reflect.DeepEqual(got, []string{"debit-1"}) {
			t.Fatalf("idempotency header = %#v", got)
		}
		var body map[string]any
		if err := json.Unmarshal(request.HTTP.Body, &body); err != nil {
			t.Fatalf("request body is invalid JSON: %v", err)
		}
		amount, ok := body["amount"].(map[string]any)
		if !ok || amount["amount"] != float64(42) || amount["asset"] != "USD/2" {
			t.Fatalf("amount = %#v", body["amount"])
		}
		if body["pending"] != true || body["description"] != "reservation" {
			t.Fatalf("request body = %#v", body)
		}
		if got := body["balances"]; !reflect.DeepEqual(got, []any{"primary", "backup"}) {
			t.Fatalf("balances = %#v", got)
		}
		if got := body["metadata"]; !reflect.DeepEqual(got, map[string]any{"order": "o1"}) {
			t.Fatalf("metadata = %#v", got)
		}
		return sdk.NewResponseStream(sdk.Response{
			Status: 201, ContentType: "application/json",
			Body: []byte(`{"data":{"id":"h1","walletID":"w1","metadata":{"order":"o1"},"asset":"USD/2","description":"reservation"}}`),
		}), nil
	})

	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: "wallets.v2.debit", Arguments: []string{"42", "USD/2"},
		Flags: []sdk.FlagOccurrence{
			{Name: "id", Value: "w1"}, {Name: "ik", Value: "debit-1"},
			{Name: "pending", Value: "true"}, {Name: "description", Value: "reservation"},
			{Name: "metadata", Value: "order=o1"},
			{Name: "balance", Value: "primary"}, {Name: "balance", Value: "backup"},
			{Name: "destination", Value: "wallet=id:w2/savings"},
		},
		Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
		ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
		Continuation:    sdk.SinglePageContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	events := host.Events()
	if len(events) != 1 || events[0].Result == nil || events[0].Result.Shape != sdk.ResultObject {
		t.Fatalf("events = %#v", events)
	}
	var result map[string]any
	if err := json.Unmarshal(events[0].Result.Data, &result); err != nil || result["id"] != "h1" {
		t.Fatalf("result = %s, err = %v", events[0].Result.Data, err)
	}
}

func contentTypeFor(body string) string {
	if body == "" {
		return ""
	}
	return "application/json"
}
