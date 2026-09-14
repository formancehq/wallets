# Wallets operation tables (generated)

Do not edit. Regenerate with `just fctl-audit` from the repository root.

Source: `openapi.yaml` (SHA-256 `715d87d7ea85344183afd1a4e16bdb67b161c1fff5665aaba74200521682a8fd`), based on product revision `a48a7d0590b8b1cfdbfbd22cba0e475713550a75`.
Legacy fctl baseline: `693c58e27865f83332e6c3199d61fed81b742f41`.

## 1. Operations by family

### wallets (7)

| operationId | method | path | scopes | success | request body | legacy fctl command | blockers |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `createWallet` | POST | `/wallets` | `wallets:write` | 201 `CreateWalletResponse` | `CreateWalletRequest` | `wallets create` | — |
| `creditWallet` | POST | `/wallets/{id}/credit` | `wallets:write` | 204 (no body) | `CreditWalletRequest` | `wallets credit` | — |
| `debitWallet` | POST | `/wallets/{id}/debit` | `wallets:write` | 201 `DebitWalletResponse` / 204 (no body) | `DebitWalletRequest` | `wallets debit` | `B2` |
| `getWallet` | GET | `/wallets/{id}` | `wallets:read` | 200 `GetWalletResponse` | — | `wallets show` | — |
| `getWalletSummary` | GET | `/wallets/{id}/summary` | `wallets:read` | 200 `GetWalletSummaryResponse` | — | — | — |
| `listWallets` | GET | `/wallets` | `wallets:read` | 200 `ListWalletsResponse` | — | `wallets list` | — |
| `updateWallet` | PATCH | `/wallets/{id}` | `wallets:write` | 204 (no body) | inline (D6) | `wallets update` | — |

### balances (3)

| operationId | method | path | scopes | success | request body | legacy fctl command | blockers |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `createBalance` | POST | `/wallets/{id}/balances` | `wallets:write` | 201 `CreateBalanceResponse` | `CreateBalanceRequest` | `wallets balances create` | — |
| `getBalance` | GET | `/wallets/{id}/balances/{balanceName}` | `wallets:read` | 200 `GetBalanceResponse` | — | `wallets balances show` | — |
| `listBalances` | GET | `/wallets/{id}/balances` | `wallets:read` | 200 `ListBalancesResponse` | — | `wallets balances list` | — |

### holds (4)

| operationId | method | path | scopes | success | request body | legacy fctl command | blockers |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `confirmHold` | POST | `/holds/{hold_id}/confirm` | `wallets:write` | 204 (no body) | `ConfirmHoldRequest` | `wallets holds confirm` | `B1` |
| `getHold` | GET | `/holds/{holdID}` | `wallets:read` | 200 `GetHoldResponse` | — | `wallets holds show` | — |
| `getHolds` | GET | `/holds` | `wallets:read` | 200 `GetHoldsResponse` | — | `wallets holds list` | — |
| `voidHold` | POST | `/holds/{hold_id}/void` | `wallets:write` | 204 (no body) | — | `wallets holds void` | `B1` |

### transactions (1)

| operationId | method | path | scopes | success | request body | legacy fctl command | blockers |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `getTransactions` | GET | `/transactions` | `wallets:read` | 200 `GetTransactionsResponse` | — | `wallets transactions list` | — |

### service (1)

| operationId | method | path | scopes | success | request body | legacy fctl command | blockers |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `getServerInfo` | GET | `/_info` | `wallets:read` | 200 `ServerInfo` | — | — | — |

## 2. Risk matrix

| operationId | mutating | moves funds | idempotency key | paginated | multiple success shapes |
| --- | --- | --- | --- | --- | --- |
| `confirmHold` | yes | yes | yes | no | no |
| `createBalance` | yes | no | yes | no | no |
| `createWallet` | yes | no | yes | no | no |
| `creditWallet` | yes | yes | yes | no | no |
| `debitWallet` | yes | yes | yes | no | yes |
| `getBalance` | no | no | no | no | no |
| `getHold` | no | no | no | no | no |
| `getHolds` | no | no | no | yes | no |
| `getServerInfo` | no | no | no | no | no |
| `getTransactions` | no | no | no | yes | no |
| `getWallet` | no | no | no | no | no |
| `getWalletSummary` | no | no | no | no | no |
| `listBalances` | no | no | no | yes | no |
| `listWallets` | no | no | no | yes | no |
| `updateWallet` | yes | no | yes | no | no |
| `voidHold` | yes | yes | yes | no | no |

No operation on this surface returns secret or display-once material, destroys caller state, or streams: those four counts are 0, 0, 0 and 0.

## 3. Legacy fctl baseline

18 constructors under `cmd/wallets/` at `693c58e`: 4 grouping-only, 14 executable leaves.

| command | aliases | kind | operationId | source |
| --- | --- | --- | --- | --- |
| `wallets` | `wal`, `wa`, `wallet` | grouping | — | `cmd/wallets/root.go` |
| `wallets create` | `cr` | leaf | `createWallet` | `cmd/wallets/create.go` |
| `wallets list` | `ls`, `l` | leaf | `listWallets` | `cmd/wallets/list.go` |
| `wallets show` | `sh` | leaf | `getWallet` | `cmd/wallets/show.go` |
| `wallets update` | `up` | leaf | `updateWallet` | `cmd/wallets/update.go` |
| `wallets credit` | `cr` | leaf | `creditWallet` | `cmd/wallets/credit.go` |
| `wallets debit` | `deb` | leaf | `debitWallet` | `cmd/wallets/debit.go` |
| `wallets balances` | `balance`, `bls`, `bal` | grouping | — | `cmd/wallets/balances/root.go` |
| `wallets balances create` | `c`, `cr` | leaf | `createBalance` | `cmd/wallets/balances/create.go` |
| `wallets balances list` | `ls`, `l` | leaf | `listBalances` | `cmd/wallets/balances/list.go` |
| `wallets balances show` | `sh` | leaf | `getBalance` | `cmd/wallets/balances/show.go` |
| `wallets holds` | `h`, `hold` | grouping | — | `cmd/wallets/holds/root.go` |
| `wallets holds list` | `ls`, `l` | leaf | `getHolds` | `cmd/wallets/holds/list.go` |
| `wallets holds show` | `sh` | leaf | `getHold` | `cmd/wallets/holds/show.go` |
| `wallets holds confirm` | `c`, `conf` | leaf | `confirmHold` | `cmd/wallets/holds/confirm.go` |
| `wallets holds void` | `v` | leaf | `voidHold` | `cmd/wallets/holds/void.go` |
| `wallets transactions` | `transaction`, `tx`, `txs` | grouping | — | `cmd/wallets/transactions/root.go` |
| `wallets transactions list` | `ls`, `l` | leaf | `getTransactions` | `cmd/wallets/transactions/list.go` |

## 4. Operations with no legacy precedent

| operationId | method | path | family |
| --- | --- | --- | --- |
| `getServerInfo` | GET | `/_info` | service |
| `getWalletSummary` | GET | `/wallets/{id}/summary` | wallets |

## 5. Blockers

### B1 — hold resolution cannot replay a completed idempotent request

- Applies to: `confirmHold`, `voidHold`
- Evidence: pkg/manager.go checks the hold's current remaining balance before submitting the idempotency key to Ledger. After a successful confirmation or void whose response is lost, the same request reaches ErrClosedHold before Ledger can recognize the replay.
- Consequence: A caller cannot safely retry these fund-moving operations after an ambiguous response even though the portable commands require --ik.
- Fix in: Wallets hold-resolution idempotency state: recognize an exact completed replay before the mutable closed-hold precondition, then add retry-after-success tests.

### B2 — required debit idempotency excludes valid dynamic balance sources

- Applies to: `debitWallet`
- Evidence: pkg/manager.go returns ErrNonIdempotentDebit when an Idempotency-Key is combined with a wildcard source or a balance carrying an expiry, while the portable debit command requires --ik and accepts both source forms.
- Consequence: Wildcard debits and debits from expiring balances are valid API requests without a key but cannot be expressed by the portable command.
- Fix in: Product contract decision: provide a deterministic source snapshot or explicitly remove these source forms from portable debit before changing --ik policy.

### B3 — the pinned generated HTTP bridge collapses product HTTP errors

- Applies to: whole surface
- Evidence: fctl SDK 545521b producthttp.Client.readResponse maps every non-2xx response to product_response_failed before the generated Wallets client can decode status or the product error body.
- Consequence: The portable surface cannot distinguish expected product failures, including the server's 413 REQUEST_TOO_LARGE response, from an invalid response.
- Fix in: fctl public producthttp SDK: preserve a bounded, redacted non-2xx status as product_http_error and define which safe error details cross the ABI.

## 6. Spec-versus-server divergences

### D1 — /_info is served unauthenticated

- Applies to: `getServerInfo`
- Spec says: openapi.yaml declares GET /_info under Authorization with scope wallets:read.
- Server does: pkg/api/router.go registers r.Get("/_info", ...) before the r.Group that installs jwt.Middleware, so the handler is reached with no token. This is the operation the fctl target resolver uses for the product-major preflight, so the weaker real requirement matters.

### D2 — declared scopes are neither defined in the scheme nor enforced by the server

- Applies to: document-level
- Spec says: Every one of the 16 operations declares a security block naming wallets:read or wallets:write, but components.securitySchemes.Authorization.flows.clientCredentials.scopes is the empty map, so the referenced scope names are undeclared in the scheme they are referenced through.
- Server does: pkg/api/router.go applies jwt.Middleware(authenticator) and nothing else: the token is authenticated, no scope is asserted. The per-operation scope arrays are readable and are recorded as declared, but they are a documentation-level contract this service does not check.

### D3 — info.version does not track the released product major

- Applies to: document-level
- Spec says: openapi.yaml declares info.version 0.1.0.
- Server does: The latest release tag on the pinned origin/main is v2.2.0, and /_info returns cmd.Version, injected at build time (cmd/serve.go passes Version into sharedapi.ServiceInfo). The product major must be read from the /_info response at runtime; the document cannot supply it.

### D4 — Idempotency-Key is declared twice on three operations

- Applies to: `creditWallet`, `debitWallet`, `voidHold`
- Spec says: The header is declared at both the path-item level and the operation level, so it appears twice in the effective parameter list. OpenAPI 3.0.3 requires parameters to be unique by name and location.
- Server does: Harmless on the wire, and pkg/client happens to collapse it to a single IdempotencyKey field, but the resolution is generator behaviour rather than contract. Recorded so a future generator change is not read as a product change.

### D5 — the hold path parameter is spelled two different ways

- Applies to: `confirmHold`, `getHold`, `voidHold`
- Spec says: getHold declares /holds/{holdID} while confirmHold and voidHold declare /holds/{hold_id}. Wallet paths declare {id} throughout.
- Server does: pkg/api/router.go routes on {holdID} and {walletID}. Path parameter names are local to each document and never appear on the wire, so nothing breaks, but the generated request DTOs inherit the inconsistency: GetHoldRequest.HoldID binds name=holdID and VoidHoldRequest.HoldID binds name=hold_id.

### D6 — updateWallet declares an anonymous request body

- Applies to: `updateWallet`
- Spec says: PATCH /wallets/{id} declares an inline application/json schema with no $ref, so the contract exposes no reusable DTO name for the body.
- Server does: pkg/api/handler_wallets_patch.go binds wallet.PatchRequest. The generated client synthesises its own operation-scoped type instead of a shared component, so any hand-written binding has no stable name to target.

### D7 — the only declared server is a local development address

- Applies to: document-level
- Spec says: servers declares exactly one entry, http://localhost:8080/.
- Server does: In a stack, Wallets is reached through the gateway route /api/wallets. The plugin must take its endpoint from fctl target resolution and never from the document's server list.

### D8 — balance priority bigint exceeds the server's signed 64-bit storage boundary

- Applies to: `createBalance`
- Spec says: CreateBalanceRequest.priority is declared as bigint and the generated client therefore represents it as an arbitrary-precision integer.
- Server does: pkg.CreateBalance binds priority to Go int and BalanceFromAccount later parses the stored metadata with strconv.ParseInt(..., 64). The portable command follows the current cross-platform server boundary and rejects values outside signed 64-bit before host access.

## 7. Derived totals

| quantity | value |
| --- | --- |
| operations declared | 16 |
| unique operationIds | 16 |
| deprecated | 0 |
| declaring a security block | 16 |
| declaring wallets:read | 9 |
| declaring wallets:write | 7 |
| legacy fctl constructors | 18 |
| legacy grouping-only commands | 4 |
| legacy executable leaves | 14 |
| leaves mapped to an operation | 14 |
| leaves excluded with evidence | 0 |
| distinct operations the baseline reaches | 14 |
| operations with a legacy precedent | 14 |
| operations without a legacy precedent | 2 |
| operations carrying an operation-scoped blocker | 3 |
| operations with no operation-scoped blocker | 13 |
| module-level blockers | 1 |
| recorded divergences | 8 |
| mutating operations | 7 |
| operations accepting an idempotency key | 7 |
| operations declaring cursor pagination | 4 |
| operations that move funds | 4 |
| operations with multiple success shapes | 1 |
| operations returning secret material | 0 |
| display-once operations | 0 |
| destructive operations | 0 |
| streaming operations | 0 |

The last column of §7 is a count of proven facts. None of it is an
acceptance claim: no runtime, component, OCI install or dual-host gate is
satisfied by this inventory.
