// Package audit extracts a deterministic, auditable inventory of the Wallets
// HTTP surface from this repository's own OpenAPI document, and pins the legacy
// fctl command baseline it has to be measured against.
//
// The package is deliberately read-only and dependency-light: it is the
// reproducible contract inventory for the portable fctl Wallets plugin
// and must stay usable before any plugin runtime, component ABI, or transport
// exists. It contains no HTTP client, no plugin entry point, and no generated
// bindings.
//
// openapi.yaml is the authoritative contract here. pkg/client is its generated
// projection and is used as a cross-check, never as a replacement contract.
package audit

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// TagV1 is the only tag the Wallets OpenAPI document uses. Every operation
// carries it; an operation with any other tag is a document change this audit
// refuses to classify silently.
const TagV1 = "wallets.v1"

// Parameter is one resolved OpenAPI parameter of an operation.
type Parameter struct {
	Name     string `json:"name"`
	In       string `json:"in"`
	Required bool   `json:"required"`
}

// Success is one declared 2xx response of an operation.
//
// Wallets needs a list rather than a single value: debitWallet declares both
// 201 (with a Hold body) and 204 (no body), selected by the request's `pending`
// flag. A result contract that assumes one success shape per operation is wrong
// for this API.
type Success struct {
	Code string `json:"code"`
	// Body is the referenced application/json schema name, or "" when the
	// response declares no JSON body.
	Body string `json:"body"`
}

// Operation is one (method, path) OpenAPI operation of the Wallets API.
//
// Every field is read from the document; nothing is inferred. Scopes is nil
// when the operation declares no security block at all, which is a different,
// weaker fact than an empty declared scope array.
type Operation struct {
	OperationID string      `json:"operationId"`
	Method      string      `json:"method"`
	Path        string      `json:"path"`
	Tag         string      `json:"tag"`
	SDKMethod   string      `json:"sdkMethod"`
	Deprecated  bool        `json:"deprecated"`
	HasSecurity bool        `json:"hasSecurity"`
	Scopes      []string    `json:"scopes"`
	Parameters  []Parameter `json:"parameters"`
	RequestBody string      `json:"requestBody"`
	// RequestBodyInline is true when the operation declares a JSON request body
	// with an anonymous schema, so there is no reusable DTO name to bind to.
	RequestBodyInline bool      `json:"requestBodyInline"`
	Successes         []Success `json:"successes"`
	// DuplicateParameters lists "name/in" keys declared more than once in the
	// effective parameter list (path-item level plus operation level). OpenAPI
	// 3.0.3 forbids these; generators resolve them by luck, not by contract.
	DuplicateParameters []string `json:"duplicateParameters"`
}

// Paginated reports whether the operation declares the Wallets cursor
// pagination parameters. Both are required so a single stray name cannot fake
// pagination. This reads the declared contract only: the server accepts these
// parameters on more operations than the document declares them on.
func (o Operation) Paginated() bool {
	var cursor, pageSize bool
	for _, p := range o.Parameters {
		switch p.Name {
		case "cursor":
			cursor = true
		case "pageSize":
			pageSize = true
		}
	}
	return cursor && pageSize
}

// HasRequestBody reports whether the operation declares a JSON request body.
func (o Operation) HasRequestBody() bool { return o.RequestBody != "" || o.RequestBodyInline }

// Idempotent reports whether the operation declares the Idempotency-Key
// header. Every write in this document does, and every corresponding handler
// reads it (verified against pkg/api at the pinned product revision).
func (o Operation) Idempotent() bool {
	for _, p := range o.Parameters {
		if p.Name == "Idempotency-Key" && p.In == "header" {
			return true
		}
	}
	return false
}

// Mutating reports whether the operation uses a state-changing HTTP method.
func (o Operation) Mutating() bool {
	switch o.Method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

// SuccessCodes returns the declared 2xx codes, sorted.
func (o Operation) SuccessCodes() []string {
	out := make([]string, 0, len(o.Successes))
	for _, s := range o.Successes {
		out = append(out, s.Code)
	}
	return out
}

// yaml shapes: only the fields this audit reads are modelled.

type yamlSchema struct {
	Ref string `yaml:"$ref"`
}

type yamlMediaType struct {
	Schema yamlSchema `yaml:"schema"`
}

type yamlBody struct {
	Ref     string                   `yaml:"$ref"`
	Content map[string]yamlMediaType `yaml:"content"`
}

type yamlResponse struct {
	Ref     string                   `yaml:"$ref"`
	Content map[string]yamlMediaType `yaml:"content"`
}

type yamlParameter struct {
	Ref      string `yaml:"$ref"`
	Name     string `yaml:"name"`
	In       string `yaml:"in"`
	Required bool   `yaml:"required"`
}

type yamlOperation struct {
	OperationID  string                  `yaml:"operationId"`
	Tags         []string                `yaml:"tags"`
	Deprecated   bool                    `yaml:"deprecated"`
	NameOverride string                  `yaml:"x-speakeasy-name-override"`
	Security     []map[string][]string   `yaml:"security"`
	Parameters   []yamlParameter         `yaml:"parameters"`
	RequestBody  *yamlBody               `yaml:"requestBody"`
	Responses    map[string]yamlResponse `yaml:"responses"`
}

type yamlPathItem struct {
	Parameters []yamlParameter `yaml:"parameters"`
	Get        *yamlOperation  `yaml:"get"`
	Put        *yamlOperation  `yaml:"put"`
	Post       *yamlOperation  `yaml:"post"`
	Delete     *yamlOperation  `yaml:"delete"`
	Patch      *yamlOperation  `yaml:"patch"`
	Head       *yamlOperation  `yaml:"head"`
	Options    *yamlOperation  `yaml:"options"`
	Trace      *yamlOperation  `yaml:"trace"`
}

type yamlFlow struct {
	Scopes map[string]string `yaml:"scopes"`
}

type yamlSecurityScheme struct {
	Type  string              `yaml:"type"`
	Flows map[string]yamlFlow `yaml:"flows"`
}

type yamlDocument struct {
	Openapi string `yaml:"openapi"`
	Info    struct {
		Title   string `yaml:"title"`
		Version string `yaml:"version"`
	} `yaml:"info"`
	Servers []struct {
		URL string `yaml:"url"`
	} `yaml:"servers"`
	Paths      map[string]yamlPathItem `yaml:"paths"`
	Components struct {
		Parameters      map[string]yamlParameter      `yaml:"parameters"`
		RequestBodies   map[string]yamlBody           `yaml:"requestBodies"`
		Responses       map[string]yamlResponse       `yaml:"responses"`
		SecuritySchemes map[string]yamlSecurityScheme `yaml:"securitySchemes"`
	} `yaml:"components"`
}

// Document is the whole parsed contract: the operations plus the document-level
// facts the inventory has to state, most of which are the evidence behind a
// recorded divergence.
type Document struct {
	// SpecVersion is info.version, which at the pinned revision does not track
	// the released product major. The runtime product major comes from the
	// /_info response, never from this field.
	SpecVersion string `json:"specVersion"`
	Title       string `json:"title"`
	// Servers is the declared server list. The plugin never uses it: the fctl
	// target resolver supplies the stack gateway route.
	Servers []string `json:"servers"`
	// SchemeScopes is the union of scope names declared by the security
	// schemes. When operations reference scopes absent from this set, the
	// document is internally inconsistent.
	SchemeScopes []string    `json:"schemeScopes"`
	Operations   []Operation `json:"-"`
}

// Load parses the Wallets OpenAPI document at specPath. Operations are sorted
// by operationId so the result is byte-stable for a given document.
func Load(specPath string) (*Document, error) {
	raw, err := os.ReadFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("read openapi document: %w", err)
	}

	var doc yamlDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("parse openapi document: %w", err)
	}

	out := &Document{SpecVersion: doc.Info.Version, Title: doc.Info.Title}
	for _, s := range doc.Servers {
		out.Servers = append(out.Servers, s.URL)
	}
	scopes := map[string]struct{}{}
	for _, scheme := range doc.Components.SecuritySchemes {
		for _, flow := range scheme.Flows {
			for name := range flow.Scopes {
				scopes[name] = struct{}{}
			}
		}
	}
	for name := range scopes {
		out.SchemeScopes = append(out.SchemeScopes, name)
	}
	sort.Strings(out.SchemeScopes)

	for path, item := range doc.Paths {
		for _, entry := range methodsOf(item) {
			if entry.op == nil || entry.op.OperationID == "" {
				continue
			}
			op, err := convert(&doc, path, entry.method, item.Parameters, *entry.op)
			if err != nil {
				return nil, err
			}
			out.Operations = append(out.Operations, op)
		}
	}
	sort.Slice(out.Operations, func(i, j int) bool {
		return out.Operations[i].OperationID < out.Operations[j].OperationID
	})
	return out, nil
}

// methodsOf returns the path item's declared operations in a fixed method
// order, so extraction output is stable for a given document.
func methodsOf(item yamlPathItem) []struct {
	method string
	op     *yamlOperation
} {
	return []struct {
		method string
		op     *yamlOperation
	}{
		{"GET", item.Get},
		{"PUT", item.Put},
		{"POST", item.Post},
		{"DELETE", item.Delete},
		{"PATCH", item.Patch},
		{"HEAD", item.Head},
		{"OPTIONS", item.Options},
		{"TRACE", item.Trace},
	}
}

func convert(doc *yamlDocument, path, method string, shared []yamlParameter, raw yamlOperation) (Operation, error) {
	op := Operation{
		OperationID: raw.OperationID,
		Method:      method,
		Path:        path,
		Deprecated:  raw.Deprecated,
		SDKMethod:   raw.NameOverride,
	}
	if op.SDKMethod == "" {
		op.SDKMethod = raw.OperationID
	}
	if len(raw.Tags) != 1 {
		return Operation{}, fmt.Errorf("operation %s: expected exactly one tag, got %v", raw.OperationID, raw.Tags)
	}
	op.Tag = raw.Tags[0]

	if raw.Security != nil {
		op.HasSecurity = true
		op.Scopes = []string{}
		for _, scheme := range raw.Security {
			for _, scopes := range scheme {
				op.Scopes = append(op.Scopes, scopes...)
			}
		}
		sort.Strings(op.Scopes)
	}

	seen := map[string]int{}
	for _, group := range [][]yamlParameter{shared, raw.Parameters} {
		for _, p := range group {
			resolved, err := resolveParameter(doc, p)
			if err != nil {
				return Operation{}, fmt.Errorf("operation %s: %w", raw.OperationID, err)
			}
			key := resolved.Name + "/" + resolved.In
			seen[key]++
			if seen[key] == 2 {
				op.DuplicateParameters = append(op.DuplicateParameters, key)
				continue
			}
			if seen[key] > 2 {
				continue
			}
			op.Parameters = append(op.Parameters, resolved)
		}
	}
	sort.Strings(op.DuplicateParameters)
	sort.Slice(op.Parameters, func(i, j int) bool {
		if op.Parameters[i].In != op.Parameters[j].In {
			return op.Parameters[i].In < op.Parameters[j].In
		}
		return op.Parameters[i].Name < op.Parameters[j].Name
	})

	if raw.RequestBody != nil {
		body, err := resolveBody(doc, *raw.RequestBody)
		if err != nil {
			return Operation{}, fmt.Errorf("operation %s: %w", raw.OperationID, err)
		}
		media, ok := body.Content["application/json"]
		if ok {
			op.RequestBody = schemaName(body.Content)
			op.RequestBodyInline = op.RequestBody == "" && media.Schema.Ref == ""
		}
	}

	successes, err := successResponses(doc, raw.Responses)
	if err != nil {
		return Operation{}, fmt.Errorf("operation %s: %w", raw.OperationID, err)
	}
	op.Successes = successes

	return op, nil
}

func resolveParameter(doc *yamlDocument, p yamlParameter) (Parameter, error) {
	if p.Ref != "" {
		name, err := refName(p.Ref, "parameters")
		if err != nil {
			return Parameter{}, err
		}
		target, ok := doc.Components.Parameters[name]
		if !ok {
			return Parameter{}, fmt.Errorf("unresolved parameter ref %q", p.Ref)
		}
		p = target
	}
	return Parameter{Name: p.Name, In: p.In, Required: p.Required}, nil
}

func resolveBody(doc *yamlDocument, b yamlBody) (yamlBody, error) {
	if b.Ref == "" {
		return b, nil
	}
	name, err := refName(b.Ref, "requestBodies")
	if err != nil {
		return yamlBody{}, err
	}
	target, ok := doc.Components.RequestBodies[name]
	if !ok {
		return yamlBody{}, fmt.Errorf("unresolved requestBody ref %q", b.Ref)
	}
	return target, nil
}

func resolveResponse(doc *yamlDocument, r yamlResponse) (yamlResponse, error) {
	if r.Ref == "" {
		return r, nil
	}
	name, err := refName(r.Ref, "responses")
	if err != nil {
		return yamlResponse{}, err
	}
	target, ok := doc.Components.Responses[name]
	if !ok {
		return yamlResponse{}, fmt.Errorf("unresolved response ref %q", r.Ref)
	}
	return target, nil
}

// successResponses returns every declared 2xx response, sorted by code. At
// least one is required; more than one is a legitimate shape in this document
// and is preserved rather than collapsed.
func successResponses(doc *yamlDocument, responses map[string]yamlResponse) ([]Success, error) {
	var codes []string
	for code := range responses {
		if strings.HasPrefix(code, "2") {
			codes = append(codes, code)
		}
	}
	if len(codes) == 0 {
		return nil, fmt.Errorf("no 2xx response declared")
	}
	sort.Strings(codes)

	out := make([]Success, 0, len(codes))
	for _, code := range codes {
		resolved, err := resolveResponse(doc, responses[code])
		if err != nil {
			return nil, err
		}
		out = append(out, Success{Code: code, Body: schemaName(resolved.Content)})
	}
	return out, nil
}

func schemaName(content map[string]yamlMediaType) string {
	media, ok := content["application/json"]
	if !ok {
		return ""
	}
	if media.Schema.Ref == "" {
		return ""
	}
	name, err := refName(media.Schema.Ref, "schemas")
	if err != nil {
		return ""
	}
	return name
}

func refName(ref, kind string) (string, error) {
	prefix := "#/components/" + kind + "/"
	if !strings.HasPrefix(ref, prefix) {
		return "", fmt.Errorf("ref %q is not a local %s ref", ref, kind)
	}
	return strings.TrimPrefix(ref, prefix), nil
}

// Index returns the operations keyed by operationId.
func Index(ops []Operation) map[string]Operation {
	out := make(map[string]Operation, len(ops))
	for _, op := range ops {
		out[op.OperationID] = op
	}
	return out
}
