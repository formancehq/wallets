package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

// tableExpectation is the exact, ordered human table a command declares. Order
// is part of the contract: a host renders the columns in the declared sequence.
type tableExpectation struct {
	commandID string
	columns   []sdk.TableColumn
	// arguments, flags, status and body reproduce the widest documented
	// success response for the command, so the declared fields are checked
	// against a result the real adapter actually emits.
	arguments []string
	flags     []sdk.FlagOccurrence
	status    int32
	body      string
}

const (
	walletBody = `{"id":"w1","metadata":{"owner":"alice"},"name":"primary","createdAt":"2026-09-11T10:00:00Z","ledger":"default","balances":{"main":{"assets":{"USD/2":100}}}}`
	// getWallet answers with WalletWithBalances, where balances is required.
	walletWithBalancesBody = `{"id":"w1","metadata":{"owner":"alice"},"name":"primary","createdAt":"2026-09-11T10:00:00Z","ledger":"default","balances":{"main":{"assets":{"USD/2":100}}}}`
	balanceBody            = `{"name":"savings","expiresAt":"2027-01-01T00:00:00Z","priority":7}`
	balanceWithAssetsBody  = `{"name":"savings","expiresAt":"2027-01-01T00:00:00Z","priority":7,"assets":{"USD/2":100}}`
	holdBody               = `{"id":"h1","walletID":"w1","metadata":{"reason":"rent"},"asset":"USD/2","description":"rent for September","destination":{"type":"ACCOUNT","identifier":"users:42"}}`
	expandedHoldBody       = `{"id":"h1","walletID":"w1","metadata":{"reason":"rent"},"asset":"USD/2","description":"rent for September","destination":{"type":"ACCOUNT","identifier":"users:42"},"remaining":40,"originalAmount":100}`
	transactionBody        = `{"ledger":"default","timestamp":"2026-09-11T10:00:00Z","postings":[{"amount":100,"asset":"USD/2","destination":"users:42","source":"wallets:w1:main"}],"reference":"ref-1","metadata":{"reason":"rent"},"id":42,"preCommitVolumes":{"users:42":{"USD/2":{"input":0,"output":0,"balance":0}}},"postCommitVolumes":{"users:42":{"USD/2":{"input":100,"output":0,"balance":100}}}}`
)

func objectResponse(body string) string { return `{"data":` + body + `}` }
func collectionResponse(body string) string {
	return `{"cursor":{"pageSize":15,"data":[` + body + `]}}`
}

var (
	walletColumns = []sdk.TableColumn{
		{Header: "ID", Field: "id"},
		{Header: "Name", Field: "name"},
		{Header: "Ledger", Field: "ledger"},
		{Header: "Created At", Field: "createdAt"},
	}
	balanceColumns = []sdk.TableColumn{
		{Header: "Name", Field: "name"},
		{Header: "Expires At", Field: "expiresAt"},
		{Header: "Priority", Field: "priority"},
	}
	holdColumns = []sdk.TableColumn{
		{Header: "ID", Field: "id"},
		{Header: "Wallet ID", Field: "walletID"},
		{Header: "Asset", Field: "asset"},
		{Header: "Destination", Field: "destination.identifier"},
	}
	expandedHoldColumns = []sdk.TableColumn{
		{Header: "ID", Field: "id"},
		{Header: "Wallet ID", Field: "walletID"},
		{Header: "Asset", Field: "asset"},
		{Header: "Destination", Field: "destination.identifier"},
		{Header: "Original Amount", Field: "originalAmount"},
		{Header: "Remaining", Field: "remaining"},
	}
	transactionColumns = []sdk.TableColumn{
		{Header: "ID", Field: "id"},
		{Header: "Timestamp", Field: "timestamp"},
		{Header: "Ledger", Field: "ledger"},
		{Header: "Reference", Field: "reference"},
	}
)

func expectedTables() []tableExpectation {
	return []tableExpectation{
		{commandID: "wallets.v2.create", columns: walletColumns, arguments: []string{"primary"}, flags: []sdk.FlagOccurrence{{Name: "ik", Value: "create-1"}}, status: 201, body: objectResponse(walletBody)},
		{commandID: "wallets.v2.list", columns: walletColumns, status: 200, body: collectionResponse(walletBody)},
		{commandID: "wallets.v2.show", columns: walletColumns, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 200, body: objectResponse(walletWithBalancesBody)},
		{commandID: "wallets.v2.balances.create", columns: balanceColumns, arguments: []string{"savings"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 201, body: objectResponse(balanceBody)},
		{commandID: "wallets.v2.balances.list", columns: balanceColumns, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 200, body: collectionResponse(balanceBody)},
		{commandID: "wallets.v2.balances.show", columns: balanceColumns, arguments: []string{"savings"}, flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}}, status: 200, body: objectResponse(balanceWithAssetsBody)},
		{commandID: "wallets.v2.holds.list", columns: holdColumns, status: 200, body: collectionResponse(holdBody)},
		{commandID: "wallets.v2.holds.show", columns: expandedHoldColumns, arguments: []string{"h1"}, status: 200, body: objectResponse(expandedHoldBody)},
		{commandID: "wallets.v2.transactions.list", columns: transactionColumns, status: 200, body: collectionResponse(transactionBody)},
	}
}

// commandsWithoutStaticTables either emit a canonical empty object or have a
// success union containing one. One descriptor cannot select a table by status,
// so no static column is invented for these commands.
func commandsWithoutStaticTables() map[string]string {
	return map[string]string{
		"wallets.v2.update":        "updateWallet answers 204 with no body; the adapter emits {}",
		"wallets.v2.credit":        "creditWallet answers 204 with no body; the adapter emits {}",
		"wallets.v2.debit":         "debitWallet can answer 204 with no body; one static table would render a blank row for {}",
		"wallets.v2.holds.confirm": "confirmHold answers 204 with no body; the adapter emits {}",
		"wallets.v2.holds.void":    "voidHold answers 204 with no body; the adapter emits {}",
	}
}

func catalogueByID(t *testing.T) map[string]sdk.Command {
	t.Helper()
	commands := Catalogue()
	byID := make(map[string]sdk.Command, len(commands))
	for _, command := range commands {
		byID[command.ID] = command
	}
	if len(byID) != len(commands) {
		t.Fatalf("catalogue has duplicate command ids")
	}
	return byID
}

func TestCatalogueDeclaresTheExactOrderedTableColumns(t *testing.T) {
	byID := catalogueByID(t)
	expectations := expectedTables()
	absent := commandsWithoutStaticTables()
	if len(expectations)+len(absent) != len(byID) {
		t.Fatalf("expectations cover %d commands, catalogue has %d", len(expectations)+len(absent), len(byID))
	}
	for _, expectation := range expectations {
		command, ok := byID[expectation.commandID]
		if !ok {
			t.Errorf("command %q is not in the catalogue", expectation.commandID)
			continue
		}
		if command.Render.Table == nil {
			t.Errorf("command %q declares no table hint, want columns %v", expectation.commandID, expectation.columns)
			continue
		}
		if !reflect.DeepEqual(command.Render.Table.Columns, expectation.columns) {
			t.Errorf("command %q table columns = %v, want %v", expectation.commandID, command.Render.Table.Columns, expectation.columns)
		}
	}
}

func TestCommandsWithoutStaticTablesDeclareNoTableHint(t *testing.T) {
	for id, reason := range commandsWithoutStaticTables() {
		command, ok := catalogueByID(t)[id]
		if !ok {
			t.Errorf("command %q is not in the catalogue", id)
			continue
		}
		if command.Render.Table != nil {
			t.Errorf("command %q declares a table hint, but %s", id, reason)
		}
	}
}

// executeForResult runs the real adapter against the widest documented success
// response and returns the decoded public result the host would render.
func executeForResult(t *testing.T, expectation tableExpectation) any {
	t.Helper()
	host := sdk.NewMemoryHost(func(_ context.Context, _ sdk.Request) (sdk.Responses, error) {
		return sdk.NewResponseStream(sdk.Response{Status: expectation.status, ContentType: "application/json", Body: []byte(expectation.body)}), nil
	})
	err := (Plugin{}).Execute(context.Background(), sdk.ExecuteRequest{
		CommandID: expectation.commandID, Arguments: expectation.arguments, Flags: expectation.flags,
		Target:          sdk.TargetSelection{OrganizationID: "org-1", StackID: "stack-1"},
		ServiceVersions: []sdk.ServiceVersion{{Service: sdk.ServiceWallets, Version: "2.2.0", Major: 2}},
		Continuation:    sdk.SinglePageContinuationControl(),
	}, host)
	if err != nil {
		t.Fatalf("Execute(%s) error = %v", expectation.commandID, err)
	}
	events := host.Events()
	if len(events) != 1 || events[0].Result == nil {
		t.Fatalf("Execute(%s) events = %#v", expectation.commandID, events)
	}
	var decoded any
	decoder := json.NewDecoder(bytes.NewReader(events[0].Result.Data))
	decoder.UseNumber()
	if err := decoder.Decode(&decoded); err != nil {
		t.Fatalf("decode %s result: %v", expectation.commandID, err)
	}
	return decoded
}

// resultLeaves walks a decoded public result and records, for every dotted path
// a column could name, whether that path resolves to a scalar cell. A top-level
// collection contributes the union of its elements' paths, because a host
// renders one row per element. Nested arrays are recorded as non-scalar and not
// descended into: an array of objects cannot be one cell.
func resultLeaves(value any, prefix string, out map[string]bool) {
	switch typed := value.(type) {
	case map[string]any:
		if prefix != "" {
			out[prefix] = false
		}
		for key, nested := range typed {
			path := key
			if prefix != "" {
				path = prefix + "." + key
			}
			resultLeaves(nested, path, out)
		}
	case []any:
		if prefix != "" {
			out[prefix] = false
			return
		}
		for _, element := range typed {
			resultLeaves(element, prefix, out)
		}
	default:
		if prefix != "" {
			out[prefix] = true
		}
	}
}

// TestTableColumnFieldsResolveToScalarLeavesOfTheRealPublicResult proves every
// declared Field names something the adapter actually emits, and that it
// resolves to a single renderable value rather than a nested container. Dotted
// paths are traversed, so `destination.identifier` is checked against the real
// nested subject rather than assumed.
func TestTableColumnFieldsResolveToScalarLeavesOfTheRealPublicResult(t *testing.T) {
	for _, expectation := range expectedTables() {
		t.Run(expectation.commandID, func(t *testing.T) {
			leaves := map[string]bool{}
			resultLeaves(executeForResult(t, expectation), "", leaves)
			if len(leaves) == 0 {
				t.Fatalf("%s emitted no renderable result", expectation.commandID)
			}
			command, ok := catalogueByID(t)[expectation.commandID]
			if !ok || command.Render.Table == nil {
				t.Fatalf("%s declares no table hint to check", expectation.commandID)
			}
			for _, column := range command.Render.Table.Columns {
				scalar, ok := leaves[column.Field]
				if !ok {
					t.Errorf("column %q field %q is not a path of the real public result %v", column.Header, column.Field, sortedPaths(leaves))
					continue
				}
				if !scalar {
					t.Errorf("column %q field %q resolves to a nested container, not a cell", column.Header, column.Field)
				}
			}
		})
	}
}

func sortedPaths(leaves map[string]bool) []string {
	paths := make([]string, 0, len(leaves))
	for path := range leaves {
		paths = append(paths, path)
	}
	for index := 1; index < len(paths); index++ {
		for inner := index; inner > 0 && paths[inner] < paths[inner-1]; inner-- {
			paths[inner], paths[inner-1] = paths[inner-1], paths[inner]
		}
	}
	return paths
}

// TestTableColumnsExcludeNestedContainersFreeTextAndBlobs pins the deliberate
// omissions. Everything listed here stays available in --output json and
// --output yaml; it is only kept out of the compact human table.
func TestTableColumnsExcludeNestedContainersFreeTextAndBlobs(t *testing.T) {
	excluded := []string{
		"metadata",          // arbitrary key/value blob
		"balances",          // nested asset holder
		"assets",            // asset/amount blob
		"postings",          // array of objects
		"preCommitVolumes",  // nested volume blob
		"postCommitVolumes", // nested volume blob
		"description",       // unbounded free text
		"destination",       // nested subject; only its scalar identifier leaf is a column
	}
	for _, command := range Catalogue() {
		if len(command.SensitiveOutputs) != 0 {
			t.Errorf("command %q declares sensitive outputs, which a table column must never reach: %#v", command.ID, command.SensitiveOutputs)
		}
		if command.Render.Table == nil {
			continue
		}
		for _, column := range command.Render.Table.Columns {
			for _, name := range excluded {
				if column.Field == name {
					t.Errorf("command %q column %q renders excluded field %q", command.ID, column.Header, name)
				}
			}
			if strings.TrimSpace(column.Header) == "" || strings.TrimSpace(column.Field) == "" {
				t.Errorf("command %q declares an empty column: %#v", command.ID, column)
			}
		}
	}
}

// declaredSchemaLeaf resolves a dotted field against a declared public output
// schema. It reports whether the schema declares properties at all, whether the
// field is declared, and whether it resolves to a scalar type.
func declaredSchemaLeaf(schema []byte, field string) (declares, found, scalar bool) {
	var node map[string]any
	if json.Unmarshal(schema, &node) != nil {
		return false, false, false
	}
	if items, ok := node["items"].(map[string]any); ok {
		node = items
	}
	properties, ok := node["properties"].(map[string]any)
	if !ok {
		return false, false, false
	}
	declares = true
	segments := strings.Split(field, ".")
	for index, segment := range segments {
		next, ok := properties[segment].(map[string]any)
		if !ok {
			return declares, false, false
		}
		if index == len(segments)-1 {
			kind := schemaKind(next)
			return declares, true, kind != "" && !strings.Contains(kind, "object") && !strings.Contains(kind, "array")
		}
		properties, ok = next["properties"].(map[string]any)
		if !ok {
			return declares, false, false
		}
	}
	return declares, false, false
}

// schemaKeepsAdditionalProperties verifies that the row object remains open.
// This is what lets the compact table coexist with an exhaustive JSON/YAML
// result even though only the selected scalar columns are named explicitly.
func schemaKeepsAdditionalProperties(schema []byte) bool {
	var node map[string]any
	if json.Unmarshal(schema, &node) != nil {
		return false
	}
	if items, ok := node["items"].(map[string]any); ok {
		node = items
	}
	additional, declared := node["additionalProperties"]
	return !declared || additional == true
}

func TestDeclaredSchemaLeafTraversesNestedProperties(t *testing.T) {
	object := []byte(`{"type":"object","properties":{"id":{"type":"string"},"destination":{"type":"object","properties":{"identifier":{"type":"string"}}},"metadata":{"type":"object"}}}`)
	collection := []byte(`{"type":"array","items":{"type":"object","properties":{"id":{"type":"string"}}}}`)
	tests := []struct {
		name                    string
		schema                  []byte
		field                   string
		declares, found, scalar bool
	}{
		{name: "scalar property", schema: object, field: "id", declares: true, found: true, scalar: true},
		{name: "nested scalar leaf", schema: object, field: "destination.identifier", declares: true, found: true, scalar: true},
		{name: "nested container", schema: object, field: "metadata", declares: true, found: true, scalar: false},
		{name: "undeclared property", schema: object, field: "absent", declares: true},
		{name: "undeclared nested leaf", schema: object, field: "destination.absent", declares: true},
		{name: "descent through a scalar", schema: object, field: "id.nope", declares: true},
		{name: "collection element property", schema: collection, field: "id", declares: true, found: true, scalar: true},
		{name: "permissive schema declares nothing", schema: objectSchema, field: "id"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			declares, found, scalar := declaredSchemaLeaf(test.schema, test.field)
			if declares != test.declares || found != test.found || scalar != test.scalar {
				t.Fatalf("declaredSchemaLeaf(%q) = %v, %v, %v, want %v, %v, %v", test.field, declares, found, scalar, test.declares, test.found, test.scalar)
			}
		})
	}
}

// TestPublicOutputSchemaStaysExhaustiveAlongsideTheTableHints keeps the machine
// contract complete. A table hint selects what a human sees; it must never
// narrow --output json. The assertion is on the exact schema bytes, so erasing
// or trimming a schema fails here rather than somewhere downstream.
func TestPublicOutputSchemaStaysExhaustiveAlongsideTheTableHints(t *testing.T) {
	hinted := 0
	for _, command := range Catalogue() {
		want := objectSchema
		if command.ID == "wallets.v2.debit" {
			want = debitSchema
		} else if expectation, ok := tableExpectationByID()[command.ID]; ok {
			want = schemaForTableExpectation(expectation)
		} else if command.Pagination.Supported {
			want = collectionSchema
		}
		if !reflect.DeepEqual(command.PublicOutputSchema, want) {
			t.Errorf("command %q public output schema = %s, want the exhaustive %s", command.ID, command.PublicOutputSchema, want)
		}
		if !reflect.DeepEqual(command.RawOutputSchema, want) {
			t.Errorf("command %q raw output schema = %s, want the exhaustive %s", command.ID, command.RawOutputSchema, want)
		}
		if command.Render.Table == nil {
			continue
		}
		hinted++
		if !schemaKeepsAdditionalProperties(command.PublicOutputSchema) {
			t.Errorf("command %q public output schema narrows fields that remain available in JSON/YAML", command.ID)
		}
		for _, column := range command.Render.Table.Columns {
			declares, found, scalar := declaredSchemaLeaf(command.PublicOutputSchema, column.Field)
			if !declares {
				t.Errorf("command %q public output schema declares no properties for table field %q", command.ID, column.Field)
				continue
			}
			if !found || !scalar {
				t.Errorf("command %q column %q field %q is not a declared scalar of the public output schema", command.ID, column.Header, column.Field)
			}
		}
	}
	if hinted != len(expectedTables()) {
		t.Fatalf("catalogue declares %d table hints, want %d", hinted, len(expectedTables()))
	}
}

type schemaFieldContract struct {
	kind     string
	required bool
}

func schemaContractsByCommand() map[string]map[string]schemaFieldContract {
	wallet := map[string]schemaFieldContract{
		"id": {kind: "string", required: true}, "metadata": {kind: "object", required: true}, "metadata.*": {kind: "string"},
		"name": {kind: "string", required: true}, "createdAt": {kind: "string", required: true}, "ledger": {kind: "string", required: true},
		"balances": {kind: "object"}, "balances.main": {kind: "object", required: true},
		"balances.main.assets": {kind: "object", required: true}, "balances.main.assets.*": {kind: "integer"},
	}
	walletWithBalances := cloneSchemaContract(wallet)
	walletWithBalances["balances"] = schemaFieldContract{kind: "object", required: true}
	balance := map[string]schemaFieldContract{
		"name": {kind: "string", required: true}, "expiresAt": {kind: "null|string"}, "priority": {kind: "integer"},
	}
	balanceWithAssets := cloneSchemaContract(balance)
	balanceWithAssets["assets"] = schemaFieldContract{kind: "object", required: true}
	balanceWithAssets["assets.*"] = schemaFieldContract{kind: "integer"}
	hold := map[string]schemaFieldContract{
		"id": {kind: "string", required: true}, "walletID": {kind: "string", required: true},
		"metadata": {kind: "object", required: true}, "metadata.*": {kind: "string"},
		"asset": {kind: "string", required: true}, "description": {kind: "string", required: true},
		"destination": {kind: "object"}, "destination.type": {kind: "string", required: true},
		"destination.identifier": {kind: "string", required: true}, "destination.balance": {kind: "string"},
	}
	expandedHold := cloneSchemaContract(hold)
	expandedHold["remaining"] = schemaFieldContract{kind: "integer", required: true}
	expandedHold["originalAmount"] = schemaFieldContract{kind: "integer", required: true}
	transaction := map[string]schemaFieldContract{
		"ledger": {kind: "string"}, "timestamp": {kind: "string", required: true},
		"postings": {kind: "array", required: true}, "postings[].amount": {kind: "integer", required: true},
		"postings[].asset": {kind: "string", required: true}, "postings[].destination": {kind: "string", required: true},
		"postings[].source": {kind: "string", required: true}, "reference": {kind: "string"},
		"metadata": {kind: "object", required: true}, "metadata.*": {kind: "string"}, "id": {kind: "integer", required: true},
		"preCommitVolumes": {kind: "object"}, "preCommitVolumes.*": {kind: "object"}, "preCommitVolumes.*.*": {kind: "object"},
		"preCommitVolumes.*.*.input": {kind: "integer", required: true}, "preCommitVolumes.*.*.output": {kind: "integer", required: true},
		"preCommitVolumes.*.*.balance": {kind: "integer", required: true},
		"postCommitVolumes":            {kind: "object"}, "postCommitVolumes.*": {kind: "object"}, "postCommitVolumes.*.*": {kind: "object"},
		"postCommitVolumes.*.*.input": {kind: "integer", required: true}, "postCommitVolumes.*.*.output": {kind: "integer", required: true},
		"postCommitVolumes.*.*.balance": {kind: "integer", required: true},
	}
	return map[string]map[string]schemaFieldContract{
		"wallets.v2.create": wallet, "wallets.v2.list": wallet, "wallets.v2.show": walletWithBalances,
		"wallets.v2.balances.create": balance, "wallets.v2.balances.list": balance,
		"wallets.v2.balances.show": balanceWithAssets, "wallets.v2.holds.list": hold,
		"wallets.v2.holds.show": expandedHold, "wallets.v2.transactions.list": transaction,
	}
}

func cloneSchemaContract(source map[string]schemaFieldContract) map[string]schemaFieldContract {
	clone := make(map[string]schemaFieldContract, len(source))
	for path, field := range source {
		clone[path] = field
	}
	return clone
}

func schemaKind(node map[string]any) string {
	switch value := node["type"].(type) {
	case string:
		return value
	case []any:
		values := make([]string, 0, len(value))
		for _, item := range value {
			if typed, ok := item.(string); ok {
				values = append(values, typed)
			}
		}
		sort.Strings(values)
		return strings.Join(values, "|")
	default:
		return ""
	}
}

func collectSchemaContract(node map[string]any, prefix string, out map[string]schemaFieldContract) {
	required := map[string]bool{}
	if values, ok := node["required"].([]any); ok {
		for _, value := range values {
			if name, ok := value.(string); ok {
				required[name] = true
			}
		}
	}
	properties, _ := node["properties"].(map[string]any)
	for name, raw := range properties {
		child, _ := raw.(map[string]any)
		path := name
		if prefix != "" {
			path = prefix + "." + name
		}
		kind := schemaKind(child)
		out[path] = schemaFieldContract{kind: kind, required: required[name]}
		if kind == "object" {
			collectSchemaContract(child, path, out)
		}
		if kind == "array" {
			if items, ok := child["items"].(map[string]any); ok {
				collectSchemaContract(items, path+"[]", out)
			}
		}
	}
	if additional, ok := node["additionalProperties"].(map[string]any); ok {
		path := "*"
		if prefix != "" {
			path = prefix + ".*"
		}
		kind := schemaKind(additional)
		out[path] = schemaFieldContract{kind: kind}
		if kind == "object" {
			collectSchemaContract(additional, path, out)
		}
	}
}

func decodedSchema(t *testing.T, encoded []byte) map[string]any {
	t.Helper()
	var schema map[string]any
	if err := json.Unmarshal(encoded, &schema); err != nil {
		t.Fatalf("decode schema: %v", err)
	}
	if items, ok := schema["items"].(map[string]any); ok {
		return items
	}
	return schema
}

func TestPublicOutputSchemasDescribeEveryExportedTypedProperty(t *testing.T) {
	byID := catalogueByID(t)
	for id, want := range schemaContractsByCommand() {
		got := map[string]schemaFieldContract{}
		collectSchemaContract(decodedSchema(t, byID[id].PublicOutputSchema), "", got)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s complete public schema contract = %#v, want %#v", id, got, want)
		}
	}
}

func assertEmptyObjectSchema(t *testing.T, commandID string, schema map[string]any) {
	t.Helper()
	if schemaKind(schema) != "object" || schema["maxProperties"] != float64(0) || schema["additionalProperties"] != false {
		t.Errorf("%s empty success schema = %#v, want a closed object with no properties", commandID, schema)
	}
}

func TestCommandsWithoutTablesStillDescribeEverySuccessResult(t *testing.T) {
	byID := catalogueByID(t)
	for _, id := range []string{"wallets.v2.update", "wallets.v2.credit", "wallets.v2.holds.confirm", "wallets.v2.holds.void"} {
		assertEmptyObjectSchema(t, id, decodedSchema(t, byID[id].PublicOutputSchema))
	}

	debit := decodedSchema(t, byID["wallets.v2.debit"].PublicOutputSchema)
	alternatives, ok := debit["oneOf"].([]any)
	if !ok || len(alternatives) != 2 {
		t.Fatalf("wallets.v2.debit schema alternatives = %#v, want Hold and empty object", debit["oneOf"])
	}
	hold, ok := alternatives[0].(map[string]any)
	if !ok {
		t.Fatalf("wallets.v2.debit Hold alternative = %#v", alternatives[0])
	}
	got := map[string]schemaFieldContract{}
	collectSchemaContract(hold, "", got)
	if want := schemaContractsByCommand()["wallets.v2.holds.list"]; !reflect.DeepEqual(got, want) {
		t.Errorf("wallets.v2.debit Hold schema contract = %#v, want %#v", got, want)
	}
	empty, ok := alternatives[1].(map[string]any)
	if !ok {
		t.Fatalf("wallets.v2.debit empty alternative = %#v", alternatives[1])
	}
	assertEmptyObjectSchema(t, "wallets.v2.debit 204", empty)
}

func TestDebitTypedSuccessUnionValidatesAgainstItsMatchingPublicSchemaAlternative(t *testing.T) {
	command := catalogueByID(t)["wallets.v2.debit"]
	root := decodedSchema(t, command.PublicOutputSchema)
	alternatives, ok := root["oneOf"].([]any)
	if !ok || len(alternatives) != 2 {
		t.Fatalf("wallets.v2.debit schema alternatives = %#v, want Hold and empty object", root["oneOf"])
	}
	holdSchema, _ := alternatives[0].(map[string]any)
	emptySchema, _ := alternatives[1].(map[string]any)

	pending := executeForResult(t, tableExpectation{
		commandID: "wallets.v2.debit", arguments: []string{"100", "USD/2"},
		flags:  []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "ik", Value: "debit-1"}, {Name: "pending", Value: "true"}},
		status: 201, body: objectResponse(holdBody),
	})
	if err := validatePopulatedResult(holdSchema, pending, "wallets.v2.debit.201"); err != nil {
		t.Errorf("pending debit typed result: %v", err)
	}

	completed := executeForResult(t, tableExpectation{
		commandID: "wallets.v2.debit", arguments: []string{"100", "USD/2"},
		flags: []sdk.FlagOccurrence{{Name: "id", Value: "w1"}, {Name: "ik", Value: "debit-2"}}, status: 204,
	})
	if err := validatePopulatedResult(emptySchema, completed, "wallets.v2.debit.204"); err != nil {
		t.Errorf("completed debit typed result: %v", err)
	}
}

func schemaTypeAccepts(kind string, value any) bool {
	for _, candidate := range strings.Split(kind, "|") {
		switch candidate {
		case "null":
			if value == nil {
				return true
			}
		case "string":
			_, ok := value.(string)
			if ok {
				return true
			}
		case "integer":
			if number, ok := value.(json.Number); ok && !strings.ContainsAny(number.String(), ".eE") {
				return true
			}
		case "boolean":
			_, ok := value.(bool)
			if ok {
				return true
			}
		case "object":
			_, ok := value.(map[string]any)
			if ok {
				return true
			}
		case "array":
			_, ok := value.([]any)
			if ok {
				return true
			}
		}
	}
	return false
}

func validatePopulatedResult(node map[string]any, value any, path string) error {
	kind := schemaKind(node)
	if !schemaTypeAccepts(kind, value) {
		return fmt.Errorf("%s has %T, schema requires %s", path, value, kind)
	}
	if value == nil {
		return nil
	}
	switch typed := value.(type) {
	case map[string]any:
		properties, _ := node["properties"].(map[string]any)
		required := map[string]bool{}
		if values, ok := node["required"].([]any); ok {
			for _, raw := range values {
				if name, ok := raw.(string); ok {
					required[name] = true
				}
			}
		}
		for name := range required {
			if _, ok := typed[name]; !ok {
				return fmt.Errorf("%s omits required property %s", path, name)
			}
		}
		for name, childValue := range typed {
			child, named := properties[name].(map[string]any)
			if !named {
				child, named = node["additionalProperties"].(map[string]any)
			}
			if !named {
				return fmt.Errorf("%s property %s is not described", path, name)
			}
			if err := validatePopulatedResult(child, childValue, strings.TrimPrefix(path+"."+name, ".")); err != nil {
				return err
			}
		}
	case []any:
		items, ok := node["items"].(map[string]any)
		if !ok {
			return fmt.Errorf("%s array has no item schema", path)
		}
		for index, child := range typed {
			if err := validatePopulatedResult(items, child, fmt.Sprintf("%s[%d]", path, index)); err != nil {
				return err
			}
		}
	}
	return nil
}

func TestPopulatedTypedAdapterResultsValidateRecursivelyAgainstPublicSchemas(t *testing.T) {
	byID := catalogueByID(t)
	for _, expectation := range expectedTables() {
		root := map[string]any{}
		if err := json.Unmarshal(byID[expectation.commandID].PublicOutputSchema, &root); err != nil {
			t.Fatal(err)
		}
		if err := validatePopulatedResult(root, executeForResult(t, expectation), expectation.commandID); err != nil {
			t.Errorf("%s: %v", expectation.commandID, err)
		}
	}
}

func TestCompleteSchemaContractDetectsNonTableOptionalPropertyMutations(t *testing.T) {
	byID := catalogueByID(t)
	for _, test := range []struct{ id, path, badKind string }{
		{id: "wallets.v2.create", path: "balances", badKind: ""},
		{id: "wallets.v2.balances.create", path: "expiresAt", badKind: "string"},
		{id: "wallets.v2.balances.show", path: "assets", badKind: "array"},
		{id: "wallets.v2.transactions.list", path: "reference", badKind: "integer"},
	} {
		schema := decodedSchema(t, byID[test.id].PublicOutputSchema)
		properties, ok := schema["properties"].(map[string]any)
		if !ok {
			t.Fatalf("%s schema declares no properties", test.id)
		}
		if test.badKind == "" {
			delete(properties, test.path)
		} else {
			property, ok := properties[test.path].(map[string]any)
			if !ok {
				t.Fatalf("%s schema declares no property %s", test.id, test.path)
			}
			property["type"] = test.badKind
		}
		got := map[string]schemaFieldContract{}
		collectSchemaContract(schema, "", got)
		if reflect.DeepEqual(got, schemaContractsByCommand()[test.id]) {
			t.Fatalf("%s mutation of non-table property %s escaped the complete schema contract", test.id, test.path)
		}
	}
}

func tableExpectationByID() map[string]tableExpectation {
	byID := make(map[string]tableExpectation, len(expectedTables()))
	for _, expectation := range expectedTables() {
		byID[expectation.commandID] = expectation
	}
	return byID
}

func schemaForTableExpectation(expectation tableExpectation) []byte {
	switch expectation.commandID {
	case "wallets.v2.create", "wallets.v2.show":
		if expectation.commandID == "wallets.v2.show" {
			return walletWithBalancesSchema
		}
		return walletSchema
	case "wallets.v2.list":
		return walletCollectionSchema
	case "wallets.v2.holds.show":
		return holdSchema
	case "wallets.v2.balances.create":
		return balanceSchema
	case "wallets.v2.balances.show":
		return balanceWithAssetsSchema
	case "wallets.v2.balances.list":
		return balanceCollectionSchema
	case "wallets.v2.holds.list":
		return holdCollectionSchema
	case "wallets.v2.transactions.list":
		return transactionCollectionSchema
	default:
		panic("missing schema expectation for " + expectation.commandID)
	}
}
