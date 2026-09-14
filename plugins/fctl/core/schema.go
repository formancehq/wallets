package core

import (
	"sort"
	"strconv"
	"strings"

	"github.com/formancehq/fctl-v2-poc/pkg/plugin/sdk"
)

const schemaDialect = "https://json-schema.org/draft/2020-12/schema"

var (
	objectSchema     = []byte(`{"$schema":"` + schemaDialect + `","type":"object"}`)
	collectionSchema = []byte(`{"$schema":"` + schemaDialect + `","type":"array"}`)
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
