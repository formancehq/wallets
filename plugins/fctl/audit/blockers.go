package audit

// This file records the two things the inventory must not mix with its facts:
//
//   - Blockers: something that stops a correct plugin command being built on an
//     operation today. Each one names the operations it blocks, states how it
//     was verified, and states where the fix belongs.
//   - Divergences: places where the pinned document and the pinned server
//     disagree, or where the document contradicts itself. These do not by
//     themselves block a command, but a plugin that trusts the document alone
//     will be wrong about them.
//
// Nothing here is a runtime, component, install, or dual-host claim. Those
// gates live in docs/command-inventory.md §7 and are all still open.

// Blocker is one recorded reason an operation cannot be turned into a correct
// command yet.
type Blocker struct {
	ID string `json:"id"`
	// Title is the one-line statement of the defect.
	Title string `json:"title"`
	// OperationIDs are the operations this blocker applies to, sorted. Empty
	// means the blocker is module-level and applies to the whole surface.
	OperationIDs []string `json:"operationIds"`
	// Evidence is where the claim was verified, at the pinned revisions.
	Evidence string `json:"evidence"`
	// Consequence is what goes wrong if the plugin is built anyway.
	Consequence string `json:"consequence"`
	// FixIn is the file or artefact the fix belongs in.
	FixIn string `json:"fixIn"`
}

// Blockers are the recorded blockers at the pinned revisions.
var Blockers = []Blocker{
	{
		ID:           "B1",
		Title:        "hold resolution cannot replay a completed idempotent request",
		OperationIDs: []string{"confirmHold", "voidHold"},
		Evidence: "pkg/manager.go checks the hold's current remaining balance before " +
			"submitting the idempotency key to Ledger. After a successful confirmation " +
			"or void whose response is lost, the same request reaches ErrClosedHold before " +
			"Ledger can recognize the replay.",
		Consequence: "A caller cannot safely retry these fund-moving operations after an " +
			"ambiguous response even though the portable commands require --ik.",
		FixIn: "Wallets hold-resolution idempotency state: recognize an exact completed " +
			"replay before the mutable closed-hold precondition, then add retry-after-success tests.",
	},
	{
		ID:           "B2",
		Title:        "required debit idempotency excludes valid dynamic balance sources",
		OperationIDs: []string{"debitWallet"},
		Evidence: "pkg/manager.go returns ErrNonIdempotentDebit when an Idempotency-Key is " +
			"combined with a wildcard source or a balance carrying an expiry, while the " +
			"portable debit command requires --ik and accepts both source forms.",
		Consequence: "Wildcard debits and debits from expiring balances are valid API " +
			"requests without a key but cannot be expressed by the portable command.",
		FixIn: "Product contract decision: provide a deterministic source snapshot or " +
			"explicitly remove these source forms from portable debit before changing --ik policy.",
	},
	{
		ID:    "B3",
		Title: "the pinned generated HTTP bridge collapses product HTTP errors",
		Evidence: "fctl SDK 545521b producthttp.Client.readResponse maps every non-2xx " +
			"response to product_response_failed before the generated Wallets client can " +
			"decode status or the product error body.",
		Consequence: "The portable surface cannot distinguish expected product failures, " +
			"including the server's 413 REQUEST_TOO_LARGE response, from an invalid response.",
		FixIn: "fctl public producthttp SDK: preserve a bounded, redacted non-2xx status " +
			"as product_http_error and define which safe error details cross the ABI.",
	},
}

// Divergence is one place the document and the server disagree, or the document
// contradicts itself.
type Divergence struct {
	ID string `json:"id"`
	// Title is the one-line statement.
	Title string `json:"title"`
	// OperationIDs are the affected operations, sorted. Empty means the
	// divergence is document-level.
	OperationIDs []string `json:"operationIds"`
	// Spec is what the document says.
	Spec string `json:"spec"`
	// Server is what the pinned product source does.
	Server string `json:"server"`
}

// Divergences are the recorded spec-versus-server and spec-internal
// disagreements at the pinned revisions.
var Divergences = []Divergence{
	{
		ID:           "D1",
		Title:        "/_info is served unauthenticated",
		OperationIDs: []string{"getServerInfo"},
		Spec:         "openapi.yaml declares GET /_info under Authorization with scope wallets:read.",
		Server: "pkg/api/router.go registers r.Get(\"/_info\", ...) before the " +
			"r.Group that installs jwt.Middleware, so the handler is reached with no " +
			"token. This is the operation the fctl target resolver uses for the " +
			"product-major preflight, so the weaker real requirement matters.",
	},
	{
		ID:    "D2",
		Title: "declared scopes are neither defined in the scheme nor enforced by the server",
		Spec: "Every one of the 16 operations declares a security block naming " +
			"wallets:read or wallets:write, but " +
			"components.securitySchemes.Authorization.flows.clientCredentials.scopes " +
			"is the empty map, so the referenced scope names are undeclared in the " +
			"scheme they are referenced through.",
		Server: "pkg/api/router.go applies jwt.Middleware(authenticator) and nothing " +
			"else: the token is authenticated, no scope is asserted. The per-operation " +
			"scope arrays are readable and are recorded as declared, but they are a " +
			"documentation-level contract this service does not check.",
	},
	{
		ID:    "D3",
		Title: "info.version does not track the released product major",
		Spec:  "openapi.yaml declares info.version 0.1.0.",
		Server: "The latest release tag on the pinned origin/main is v2.2.0, and " +
			"/_info returns cmd.Version, injected at build time (cmd/serve.go passes " +
			"Version into sharedapi.ServiceInfo). The product major must be read from " +
			"the /_info response at runtime; the document cannot " +
			"supply it.",
	},
	{
		ID:           "D4",
		Title:        "Idempotency-Key is declared twice on three operations",
		OperationIDs: []string{"creditWallet", "debitWallet", "voidHold"},
		Spec: "The header is declared at both the path-item level and the operation " +
			"level, so it appears twice in the effective parameter list. OpenAPI 3.0.3 " +
			"requires parameters to be unique by name and location.",
		Server: "Harmless on the wire, and pkg/client happens to collapse it to a " +
			"single IdempotencyKey field, but the resolution is generator behaviour " +
			"rather than contract. Recorded so a future generator change is not read " +
			"as a product change.",
	},
	{
		ID:           "D5",
		Title:        "the hold path parameter is spelled two different ways",
		OperationIDs: []string{"confirmHold", "getHold", "voidHold"},
		Spec: "getHold declares /holds/{holdID} while confirmHold and voidHold " +
			"declare /holds/{hold_id}. Wallet paths declare {id} throughout.",
		Server: "pkg/api/router.go routes on {holdID} and {walletID}. Path parameter " +
			"names are local to each document and never appear on the wire, so nothing " +
			"breaks, but the generated request DTOs inherit the inconsistency: " +
			"GetHoldRequest.HoldID binds name=holdID and VoidHoldRequest.HoldID binds " +
			"name=hold_id.",
	},
	{
		ID:           "D6",
		Title:        "updateWallet declares an anonymous request body",
		OperationIDs: []string{"updateWallet"},
		Spec: "PATCH /wallets/{id} declares an inline application/json schema with no " +
			"$ref, so the contract exposes no reusable DTO name for the body.",
		Server: "pkg/api/handler_wallets_patch.go binds wallet.PatchRequest. The " +
			"generated client synthesises its own operation-scoped type instead of a " +
			"shared component, so any hand-written binding has no stable name to target.",
	},
	{
		ID:    "D7",
		Title: "the only declared server is a local development address",
		Spec:  "servers declares exactly one entry, http://localhost:8080/.",
		Server: "In a stack, Wallets is reached through the gateway route " +
			"/api/wallets. The plugin must take its endpoint from fctl target " +
			"resolution and never from the document's server list.",
	},
	{
		ID:           "D8",
		Title:        "balance priority bigint exceeds the server's signed 64-bit storage boundary",
		OperationIDs: []string{"createBalance"},
		Spec: "CreateBalanceRequest.priority is declared as bigint and the generated client " +
			"therefore represents it as an arbitrary-precision integer.",
		Server: "pkg.CreateBalance binds priority to Go int and BalanceFromAccount later parses " +
			"the stored metadata with strconv.ParseInt(..., 64). The portable command follows the " +
			"current cross-platform server boundary and rejects values outside signed 64-bit before host access.",
	},
}

// blockersByOperation indexes Blockers by the operations they name. Module-level
// blockers, which name no operation, are deliberately absent: they are not
// attributed to individual operations and so never inflate the blocked count.
func blockersByOperation() map[string][]string {
	out := map[string][]string{}
	for _, b := range Blockers {
		for _, op := range b.OperationIDs {
			out[op] = append(out[op], b.ID)
		}
	}
	return out
}

// ModuleBlockers returns the blockers that apply to the whole surface rather
// than to named operations.
func ModuleBlockers() []Blocker {
	var out []Blocker
	for _, b := range Blockers {
		if len(b.OperationIDs) == 0 {
			out = append(out, b)
		}
	}
	return out
}
