package core

import (
	"sort"
	"strconv"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const schemaDialect = "https://json-schema.org/draft/2020-12/schema"

var (
	objectSchema     = []byte(`{"$schema":"` + schemaDialect + `",` + emptyObjectSchema + `}`)
	collectionSchema = []byte(`{"$schema":"` + schemaDialect + `","type":"array"}`)

	// Result schemas enumerate the complete exported JSON shape of the generated
	// Wallets models. Objects remain open for forward-compatible JSON/YAML output,
	// while map values, optional fields, nullable expiry dates, and mandatory
	// fields stay typed explicitly.
	walletSchema                = []byte(`{"$schema":"` + schemaDialect + `",` + walletObjectSchema + `}`)
	walletCollectionSchema      = []byte(`{"$schema":"` + schemaDialect + `","type":"array","items":{` + walletObjectSchema + `}}`)
	walletWithBalancesSchema    = []byte(`{"$schema":"` + schemaDialect + `",` + walletWithBalancesObjectSchema + `}`)
	balanceSchema               = []byte(`{"$schema":"` + schemaDialect + `",` + balanceObjectSchema + `}`)
	balanceCollectionSchema     = []byte(`{"$schema":"` + schemaDialect + `","type":"array","items":{` + balanceObjectSchema + `}}`)
	balanceWithAssetsSchema     = []byte(`{"$schema":"` + schemaDialect + `",` + balanceWithAssetsObjectSchema + `}`)
	holdSchema                  = []byte(`{"$schema":"` + schemaDialect + `",` + expandedHoldObjectSchema + `}`)
	holdCollectionSchema        = []byte(`{"$schema":"` + schemaDialect + `","type":"array","items":{` + holdObjectSchema + `}}`)
	debitSchema                 = []byte(`{"$schema":"` + schemaDialect + `","type":"object","oneOf":[{` + holdObjectSchema + `},{` + emptyObjectSchema + `}]}`)
	transactionCollectionSchema = []byte(`{"$schema":"` + schemaDialect + `","type":"array","items":{` + transactionObjectSchema + `}}`)
)

const (
	emptyObjectSchema = `"type":"object","maxProperties":0,"additionalProperties":false`
	assetHolderSchema = `"type":"object","properties":{"assets":{"type":"object","additionalProperties":{"type":"integer"}}},"required":["assets"]`
	balancesSchema    = `"type":"object","properties":{"main":{` + assetHolderSchema + `}},"required":["main"]`
	metadataSchema    = `{"type":"object","additionalProperties":{"type":"string"}}`

	walletProperties               = `"id":{"type":"string"},"metadata":` + metadataSchema + `,"name":{"type":"string"},"createdAt":{"type":"string"},"ledger":{"type":"string"},"balances":{` + balancesSchema + `}`
	walletObjectSchema             = `"type":"object","properties":{` + walletProperties + `},"required":["id","metadata","name","createdAt","ledger"]`
	walletWithBalancesObjectSchema = `"type":"object","properties":{` + walletProperties + `},"required":["id","metadata","name","createdAt","balances","ledger"]`

	balanceProperties             = `"name":{"type":"string"},"expiresAt":{"type":["string","null"]},"priority":{"type":"integer"}`
	balanceObjectSchema           = `"type":"object","properties":{` + balanceProperties + `},"required":["name"]`
	balanceWithAssetsObjectSchema = `"type":"object","properties":{` + balanceProperties + `,"assets":{"type":"object","additionalProperties":{"type":"integer"}}},"required":["name","assets"]`

	subjectSchema            = `"type":"object","properties":{"type":{"type":"string"},"identifier":{"type":"string"},"balance":{"type":"string"}},"required":["type","identifier"]`
	holdProperties           = `"id":{"type":"string"},"walletID":{"type":"string"},"metadata":` + metadataSchema + `,"asset":{"type":"string"},"description":{"type":"string"},"destination":{` + subjectSchema + `}`
	holdObjectSchema         = `"type":"object","properties":{` + holdProperties + `},"required":["id","walletID","metadata","asset","description"]`
	expandedHoldObjectSchema = `"type":"object","properties":{` + holdProperties + `,"remaining":{"type":"integer"},"originalAmount":{"type":"integer"}},"required":["id","walletID","metadata","asset","description","remaining","originalAmount"]`

	postingSchema           = `"type":"object","properties":{"amount":{"type":"integer"},"asset":{"type":"string"},"destination":{"type":"string"},"source":{"type":"string"}},"required":["amount","asset","destination","source"]`
	volumeSchema            = `"type":"object","properties":{"input":{"type":"integer"},"output":{"type":"integer"},"balance":{"type":"integer"}},"required":["input","output","balance"]`
	aggregatedVolumesSchema = `"type":"object","additionalProperties":{"type":"object","additionalProperties":{` + volumeSchema + `}}`
	transactionObjectSchema = `"type":"object","properties":{"ledger":{"type":"string"},"timestamp":{"type":"string"},"postings":{"type":"array","items":{` + postingSchema + `}},"reference":{"type":"string"},"metadata":` + metadataSchema + `,"id":{"type":"integer"},"preCommitVolumes":{` + aggregatedVolumesSchema + `},"postCommitVolumes":{` + aggregatedVolumesSchema + `}},"required":["timestamp","postings","metadata","id"]`
)

func buildInputSchema(arguments []sdk.Argument, flags []sdk.Flag) []byte {
	type field struct {
		primitive string
		repeated  bool
		required  bool
	}
	fields := make(map[string]field, len(arguments)+len(flags))
	for _, argument := range arguments {
		fields[argument.Name] = field{primitive: argumentType(argument.Type), repeated: argument.Repeated || argument.Type == sdk.ArgumentStringArray, required: argument.Required}
	}
	for _, flag := range flags {
		fields[flag.Name] = field{primitive: flagType(flag.Type), repeated: flag.Type == sdk.FlagStringArray, required: flag.Required}
	}
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	properties := make([]string, 0, len(names))
	required := make([]string, 0, len(names))
	for _, name := range names {
		value := fields[name]
		property := `{"type":` + strconv.Quote(value.primitive)
		if value.repeated {
			property = `{"type":"array","items":{"type":` + strconv.Quote(value.primitive) + `}`
			if value.required {
				property += `,"minItems":1`
			}
		}
		property += `}`
		properties = append(properties, strconv.Quote(name)+":"+property)
		if value.required {
			required = append(required, strconv.Quote(name))
		}
	}
	out := `{"$schema":"` + schemaDialect + `","type":"object","properties":{` + strings.Join(properties, ",") + `}`
	if len(required) != 0 {
		out += `,"required":[` + strings.Join(required, ",") + `]`
	}
	out += `,"additionalProperties":false}`
	return []byte(out)
}

func argumentType(value sdk.ArgumentType) string {
	if value == sdk.ArgumentInt32 {
		return "integer"
	}
	if value == sdk.ArgumentBool {
		return "boolean"
	}
	return "string"
}

func flagType(value sdk.FlagType) string {
	if value == sdk.FlagInt32 {
		return "integer"
	}
	if value == sdk.FlagBool {
		return "boolean"
	}
	return "string"
}
