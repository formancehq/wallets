# fctl Wallets plugin — operation inventory

## 1. Scope

This document is the reproducible operation and command inventory for the
portable Wallets v2 plugin implemented under `plugins/fctl`.

It contains:

- a reproducible inventory of every operation the Wallets API declares;
- the exact provenance mapping from the historical `fctl wallets` command
  leaves, without treating every old alias as current executable UX;
- the functional groupings used by the current command catalogue;
- method, path, request body, success shapes and declared scopes, recorded only
  where the pinned sources prove them;
- the secret, display-once, destructive, idempotence, pagination and streaming
  risk position of each operation;
- blockers and spec-versus-server divergences, kept strictly separate from the
  facts above; and
- the external integration gates that remain after the local component build.

## 2. Pinned revisions and method

| what | revision |
| --- | --- |
| Wallets base revision (`origin/main`, `v2.2.0-6-ga48a7d0`) | `a48a7d0590b8b1cfdbfbd22cba0e475713550a75` |
| exact OpenAPI bytes used by the audit and `pkg/client` | SHA-256 `715d87d7ea85344183afd1a4e16bdb67b161c1fff5665aaba74200521682a8fd` |
| legacy fctl baseline (`cmd/wallets/`) | `693c58e27865f83332e6c3199d61fed81b742f41` |
| fctl public SDK development snapshot | `e9b1395f46f3100b381dbe00f5213de28e6df0e1` |

**`openapi.yaml` is the authoritative contract for this inventory.** The
generated `pkg/client` is now a buildable projection of that pinned document
and agrees on the operation set: `pkg/client/v1.go` declares exactly 16 `V1`
methods, one per `operationId`.

Server-side claims were read from `pkg/api/` and `pkg/manager.go` at the pinned
product revision. Legacy command claims were read from each controller's `Run`
method under `cmd/wallets/`, where the call reaches the API through
`stackClient.Wallets.V1.<Method>`.

Generated companion tables live in
[`operations.generated.md`](./operations.generated.md). Regenerate with `just
fctl-audit`; `just fctl-audit-check` fails the build when they drift.

## 3. Operation inventory

The Wallets document declares **16 operations, 16 unique `operationId`s, all
carrying the single tag `wallets.v1`**. None is deprecated. All 16 declare a
security block: **9 under `wallets:read`, 7 under `wallets:write`** — but see
divergence **D2** before treating those scopes as enforced.

Per-operation method, path, declared scopes, success shapes and request body are
in [`operations.generated.md`](./operations.generated.md) §1.

**Two operations have no legacy CLI precedent:**

| operationId | method | path | note |
| --- | --- | --- | --- |
| `getWalletSummary` | GET | `/wallets/{id}/summary` | wallet-level aggregate; the legacy CLI never exposed it, and `WalletSummary` appears nowhere under `cmd/`. A candidate addition, not a parity obligation. |
| `getServerInfo` | GET | `/_info` | service metadata. The legacy CLI has no `wallets versions` command (only `payments` had one, via `GetServerInfoPayments`). This is the operation fctl target resolution needs for the product-major preflight — see **D1** and **D3**. |

## 4. Legacy command mapping

`cmd/wallets/` at the pinned fctl revision declares **18 command constructors:
4 grouping-only and 14 executable leaves**. The grouping-only commands
(`wallets`, `wallets balances`, `wallets holds`, `wallets transactions`) carry
child commands and execute nothing, so they are excluded from the leaf count and
from the mapping.

**All 14 executable leaves map onto 14 distinct operations. There are no
exclusions and no deprecations to justify.** This surface is unusually clean:
each leaf calls exactly one `Wallets.V1` method, and no two leaves share one.

The full historical table, including the aliases every legacy leaf carried, is
in [`operations.generated.md`](./operations.generated.md) §3. It is provenance,
not an instruction to reproduce ambiguous legacy spelling. RFC 0013 retains an
alias only when the active digest-bound descriptor selects it as current UX,
and RFC 0010 rejects ambiguous command trees.

Two historical alias collisions therefore require an explicit current choice:

- `cr` is an alias of both `wallets create` and `wallets credit`, and also of
  `wallets balances create`.
- `c` is an alias of both `wallets balances create` and `wallets holds confirm`.

The v2 catalogue keeps `cr` for root `create` and omits it from root `credit`.
The nested `balances create` and `holds confirm` aliases remain valid because
their parent segments make the routes distinct. Root plugin aliases are host
installation concerns and are not command-descriptor path aliases.

The active catalogue also selects `--id`, not historical `--name`, for wallet
targeting. Resolving a name would require an undeclared `listWallets` call; on a
paginated command that would violate RFC 0010's one-operation pagination
boundary. The historical flag remains recorded here but is not executable UX.

## 5. Functional groupings

| family | count | operations |
| --- | --- | --- |
| `wallets` | 7 | the wallet resource plus the two fund movements that target it |
| `balances` | 3 | named sub-balances of a wallet |
| `holds` | 4 | pending debits and their resolution |
| `transactions` | 1 | read-only transaction listing |
| `service` | 1 | service metadata, not wallet data |

The groupings follow the resource each operation acts on, which is also how the
legacy command tree was shaped, so a command catalogue can adopt them without
re-litigating the taxonomy. The assignment is a total map in `audit/classify.go`
on purpose: a new operation fails the audit until it is classified, rather than
being grouped by a prefix heuristic.

## 6. Risk register

Per-operation marks are in [`operations.generated.md`](./operations.generated.md)
§2. The position of the whole surface:

**Secret material: none.** No operation returns a credential, key or token. The
only non-wallet response body is `ServerInfo`, whose single field is a version
string. This is a material difference from Payments, whose
`v3GetConnectorConfig` returns decrypted PSP credentials.

**Display-once values: none.** Nothing is returned exactly once.

**Destructive operations: none.** The only update is `updateWallet`, and
`pkg.Manager.UpdateWallet` *merges* the supplied metadata into the existing
metadata (`newCustomMetadata.Merge(...)` twice, then `meta.Merge(...)`) rather
than replacing it. A caller cannot drop a metadata key it did not supply, so
there is no unlisted state to lose.

**Fund movement: 4 operations** — `creditWallet`, `debitWallet`, `confirmHold`,
`voidHold`. These are the operations a host must never replay speculatively.
`createBalance` and `updateWallet` write state but move no funds.

**Idempotency-key propagation: complete; end-to-end replay semantics: not yet
complete.** All 7 mutating operations declare `Idempotency-Key`, and all 7
handlers pass it through to the manager via `api.IdempotencyKeyFromRequest(r)` —
verified handler by handler. No read operation declares it. The portable
command contract requires `--ik` on the 4 fund-moving operations and forwards
it unchanged; the other 3 mutations continue to accept an optional key. B1
records that completed hold resolutions fail before Ledger can recognize a
retry, and B2 records that keyed debit cannot use wildcard or expiring sources.

**Integer boundary.** `priority` is a signed ordering hint and the OpenAPI
schema declares it as `bigint`; `holds confirm --amount` is a non-negative
fund amount. Both cross the portable ABI as decimal strings and are converted
to `big.Int` only inside the generated-client adapter. The current confirmation
handler binds `amount` to Go `int64`, so the command rejects values above
`MaxInt64` before transport. The current balance model uses Go `int` and signed
64-bit parsing when rebuilding priority from metadata, so the command accepts
the complete signed 64-bit priority range but rejects `MaxInt64+1` and
`MinInt64-1` before host access. D8 records the OpenAPI/server mismatch.

**Pagination: all 4 listings.** `listWallets`, `listBalances`, `getHolds` and
`getTransactions` declare `cursor` and `pageSize`, matching the server cursor
contract.

**Streaming: none.** Every listing is a bounded cursor page; no operation
returns an unbounded stream. No streaming transport is required.

**Multiple success shapes: 1 operation.** `debitWallet` returns either a `201`
with a Hold or an empty `204`. The portable result contract can represent that
union with JSON Schema `oneOf`, while the HTTP bridge preserves the status code.

**Request body size: capped at 1 MiB** by both the portable operation policies
and `maxRequestBodyBytes` in `pkg/api/router.go`. The server returns `413` with
error code `REQUEST_TOO_LARGE` above that limit. The pinned fctl `producthttp`
bridge preserves that status as the bounded `product_http_error` failure with an
`httpStatus` detail, so the portable surface distinguishes an expected product
rejection from an invalid response. `core` pins the mapping for `413` and `500`
in `TestExecuteSurfacesProductHTTPStatusAsABoundedFailure`.

## 7. Blockers and divergences

Both are recorded in `audit/blockers.go` with their evidence, and rendered in
[`operations.generated.md`](./operations.generated.md) §5 and §6.

**2 blockers remain.** B1 applies to `confirmHold` and `voidHold`: both inspect
mutable hold state before Ledger can recognize a completed retry under the same
idempotency key. The recommended product fix is an exact completed-replay lookup
before the closed-hold precondition, backed by retry-after-success tests.

B2 applies to `debitWallet`: the portable command requires `--ik`, while the
server rejects keyed wildcard and expiring balance sources because their
resolved Ledger request can change between attempts. Resolve the contract by
making the source set deterministic, or explicitly remove those source forms
from portable debit; do not weaken the key requirement implicitly.

The previously recorded module-level blocker B3 is closed by the SDK pin above.
fctl `e9b1395f` maps a non-2xx product response to `product_http_error`, carrying
the numeric status and marking `5xx` retryable, instead of collapsing it into
`product_response_failed`. No module-level blocker remains.

The generated client still builds standalone, models `Hold.asset` as required,
exposes pagination for `listBalances`, and represents the `debitWallet` success
union. Those facts do not close B1 or B2.

**8 divergences.** `D1` `/_info` is served unauthenticated though declared under
`wallets:read`. `D2` declared scopes are neither defined in the security scheme
(`scopes: {}`) nor asserted by the server (`jwt.Middleware` only). `D3`
`info.version` is `0.1.0` while the released product is `v2.2.0`, so the product
major must come from `/_info` at runtime. `D4` `Idempotency-Key` is declared
twice on three operations. `D5` the hold path parameter is spelled `{holdID}` on
one operation and `{hold_id}` on two others. `D6` `updateWallet` declares an
anonymous request body. `D7` the only declared server is
`http://localhost:8080/`, so the endpoint must come from fctl target resolution.
`D8` records that OpenAPI declares balance priority as arbitrary-precision
`bigint` while the server's portable cross-platform storage boundary is signed
64-bit.

`D1`, `D2`, `D4` and `D8` are fixable in this repository's `openapi.yaml` or
server contract. `D3`, `D5` and `D6` are contract-hygiene items. `D7` is expected
and is recorded so nobody wires the document's server list into a plugin.

## 8. Integration status

The product-local command catalogue, closed schemas, execution dispatch,
generated-client HTTP mapping, component descriptor, lifecycle entrypoint, and
deterministic component build are implemented under `plugins/fctl`. All 14
commands declare exactly one OpenAPI operation and the catalogue is checked
against this inventory.

After B1 and B2 are resolved, the remaining release gates are external to the
command implementation:

1. replace the local fctl SDK development path with a published immutable SDK
   version;
2. validate `/_info` against a running Stack 3.2 Wallets service and record the
   observed product-major mapping (see **D3**);
3. run real read and mutation scenarios against that service through both
   admitted fctl hosts, including browser coverage;
4. retain the resulting component hashes, WIT imports, size, and host receipts
   as release evidence.

## 9. Determinism

- `just fctl-audit` regenerates `audit/testdata/report.json` and
  `docs/operations.generated.md` from `openapi.yaml`. Review the diff: a change
  there is a change to this plugin's source of truth.
- `just fctl-audit-check` fails when either artefact is stale. It runs as part
  of `just pre-commit`.
- `audit/audit_test.go` checks structural invariants against the live document,
  not against the golden report: unique ids, total family coverage, the scope
  contract, the single multi-success operation, the idempotency contract, the
  pagination position, the duplicate-parameter set, and baseline well-formedness.
- `audit/documented_test.go` pins every number and every claim §1–§7 states in
  prose. Prose cannot be regenerated, so when the contract moves a test fails
  with the exact quantity that changed.
- Two assertions pin defects on purpose — `D2` and `D4`. When the
  document is fixed those tests fail. That failure is the signal to retire the
  record, not to relax the test.
