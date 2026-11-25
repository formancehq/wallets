# Ledger v3 support in Wallets

**Status** Implemented on the Wallets 3.0 development stack. Release qualification remains a separate gate. The scope is one Ledger v3 adapter, one provisioning routine, and coverage for both Ledger backends.

## Product contract

- A Wallets deployment uses one Ledger backend. Keep v2 as the default and preserve existing v2 configuration, including `--stack-url`, `--ledger`, and `--account-prefix`. Propose `--ledger-api-version=v2|v3|auto` and `--ledger-url` as the independent Ledger service base. For a direct local instance the base is `http://127.0.0.1:9000`. For a gateway that exposes Ledger v3, the base includes its Ledger mount. The adapter appends `/v3`. OAuth token acquisition continues to use the Stack URL.
- `auto` is opt-in. Resolve the backend once at startup using a read-only Ledger API check before provisioning or money movement. Qualify the v3 `GET /_info` response and the supported v2 and gateway info routes. An unrecognized response, auth failure, timeout, or unresolved routing mismatch fails startup with an actionable diagnostic. Explicit selection is authoritative. Keep the chosen adapter for the process lifetime.
- Keep the Wallets v1 HTTP API and SDK contract for both backends. Preserve wallet and hold IDs, address rendering, balances, references, timestamps, metadata, and existing replay and conflict behavior. Preserve source ordering, balance priority and expiry, partial and final confirmation, and refunds to the original sources when voiding a hold.
- Support fresh Ledger v3 ledgers in NORMAL mode and explicitly provisioned existing NORMAL ledgers. Initial qualification covers uncolored wallet balances. Report spendable uncolored amounts in Wallets and leave colored balances outside the Wallets API contract. A transfer initiated by Wallets must retain the same minor-unit amounts and asset precision.
- An existing v2 deployment can keep its current backend during a Wallets upgrade or rollback. Moving data from v2 to v3 requires a separate cutover with balance and hold reconciliation, an idempotency transition, and a rollback plan. Changing the backend selector does not move data.

[Wallets service wiring](../cmd/serve.go) · [Wallets HTTP contract](../openapi.yaml) · [Ledger v3 RC.3 HTTP contract](https://raw.githubusercontent.com/formancehq/ledger/v3.0.0-rc.3/openapi.yml)

## Configuration and validation

The default Wallets command continues to use Ledger v2 through the Stack URL. To start Wallets with the local v3 service, choose a disposable ledger name and run the v3 adapter.

```sh
go run . serve --ledger-api-version=auto --ledger-url=http://127.0.0.1:9000 --ledger=wallets-dev
```

| Flag | Default | Behavior |
| --- | --- | --- |
| `--ledger-api-version` | `v2` | Select `v2`, `v3`, or `auto`. The version check runs once at startup in `auto` mode. |
| `--ledger-url` | Empty | Service base, including the Ledger mount for a gateway. With v3 or auto and a Stack URL, the default service base is the Stack's `/api/ledger` mount. |
| `--ledger-v3-provisioning` | `managed` | `managed` reconciles missing Wallets resources. `verify` checks operator-provisioned definitions and index readiness using read-only requests. Runtime writes still require Ledger write scopes. |
| `--ledger-v3-cursor-key` | Random per process | Shared signing secret of at least 32 bytes. Configure the same secret on Wallets replicas to preserve cursors across replicas and restarts. |
| `--ledger-v3-index-status-urls` | Empty | Comma-separated Ledger replica service URLs for additional index readiness checks. Set these to the serving replicas when the main URL routes through a load balancer. |
| `--ledger-v3-indexed-metadata-keys` | Empty | Comma-separated Wallets custom metadata keys available for v3 filtering. |

Use `just tests` for the v2 baseline. Use the same smoke target as CI against a disposable Ledger v3 instance.

```sh
WALLETS_LEDGER_V3_URL=http://127.0.0.1:9000 just tests-ledger-v3
```

The smoke creates test ledgers and synthetic balances. `just tests-ledger-v3-ci` launches the pinned image on a temporary Docker port and removes the container after the test. Wallets checks the configured Ledger endpoint and any additional replica URLs at startup. In a deployment where Wallets cannot reach replica endpoints, the operator checks each serving replica before routing traffic.

## Adapter boundary

Wire the v3 implementation behind the existing `Ledger` interface in `pkg/module.go`. Keep `DefaultLedger` and its v2 SDK behavior available to existing callers. Keep v3 HTTP transport and wire types inside the adapter. Convert to the types the Manager already consumes. Reuse the injectable HTTP client for authentication, cancellation, and observability. Select a supported Ledger v3 client if it covers the operations below. A narrow HTTP client for the same operations is sufficient otherwise. [Ledger interface](../pkg/ledger_interface.go) · [Module wiring](../pkg/module.go)

| Wallets operation | Ledger v3 requirement |
| --- | --- |
| Ensure ledger exists | Read `GET /v3/{ledger}`. Only a ledger-specific `NOT_FOUND` permits managed creation. Ledger v3 returns `201` for creation and accepts the chart and initial schema in the request. A concurrent create requires a read back and comparison before continuing. |
| Get account | Use an exact `address` filter on `GET /v3/{ledger}/accounts` to establish existence. The point read can return HTTP 200 with empty volumes for an absent address. An exact-list result distinguishes an absent account from a metadata-only account. Keep the Wallets conflict when a requested creation address belongs to an existing non-wallet account. |
| Add account metadata | Use `POST /v3/{ledger}/accounts/{address}/metadata` and accept `204`. Retain string metadata, including JSON stored as strings and replay snapshots. |
| List accounts and transactions | Send v3 `filter` with the supported structured DSL. Map the transaction `account` selector to the v3 any-role `address` selector. Omit `expand=volumes` on the Ledger request and honor the caller's expansion choice when mapping the response. Decode top-level `data`, `next`, `previous`, and `hasMore` for ordinary lists. Decode `cursor.accountData` or `cursor.transactionData` for prepared queries. Preserve the Wallets response envelope. |
| Read balances and transaction volumes | Map v3 asset-and-color rows to the uncolored Wallets volume model. Account rows carry nested `volumes`. Transaction post-commit volumes group their rows by account. Preserve integer precision through decoding and Wallets JSON serialization, with regression coverage beyond JavaScript's safe integer range. When adding the `ledger` response field, preserve numeric JSON tokens through `Transaction.MarshalJSON`, which currently decodes them through `float64`. |
| Create transaction | Accept `201` and decode `data.transaction`. Send exactly one source of postings, inline Numscript, or `scriptReference`. Preserve transaction metadata, timestamp, reference and key. Put hold account metadata in `accountMetadata` in the same transaction. |
| Map errors | Preserve existing Wallets validation, not-found, conflict, and insufficient-funds responses. Map v3 `INSUFFICIENT_FUNDS` and malformed-cursor `INVALID_REQUEST` at the adapter boundary. Surface missing or building required indexes as an operational condition with the affected index in diagnostics. |

Pagination across either v3 list form must retain the original filter or prepared query and parameters, page size, direction, expansion, Ledger identity, and operation when callers send only `cursor`. Use a versioned Wallets token authenticated with HMAC-SHA-256. Validate the Ledger service URL, ledger name, route, and supplied bindings. Reject incompatible tokens as validation errors. Preserve existing v2 cursor handling. For prepared transaction history, set `reverse: true` on RC.3 to preserve newest-first order across pages. RC.3 returned oldest-first by default on the prepared route and newest-first on the ordinary transaction list. [Ledger v3 RC.3 HTTP contract](https://raw.githubusercontent.com/formancehq/ledger/v3.0.0-rc.3/openapi.yml)

Ledger v3 uses a cluster-wide idempotency namespace. Use a deterministic Ledger key derived from the Wallets namespace, ledger identity, and caller key. Keep the existing per-ledger conflict domain and leave absent keys absent. Keep domain IDs and replay snapshots derived from the original caller key. A retry after a timeout or lost response must submit the same payload under the same Ledger key. Changing script versions or write representation during a rollout needs a plan for in-flight keys. [Ledger v3 idempotency model](https://github.com/formancehq/ledger/blob/v3.0.0-rc.3/docs/drafts/advanced-concepts.md)

## Ledger v3 provisioning

Wallets publishes its structure through native Ledger resources. Use one versioned set of definitions owned by Wallets. On a Wallets-managed ledger, the startup path provisions the definitions and checks readiness. For an operator-managed ledger, the operator applies the same definitions and Wallets verifies them before serving. Run one Wallets initializer in `managed` mode per ledger and start the other Wallets replicas in `verify` mode after provisioning. RC.3 has no conditional metadata schema write. Use the selected Ledger's API and scopes for ledger creation, schema, account types, indexes, script library, and prepared queries. Deployment qualification must verify those permissions and the runtime read and transaction-write scopes separately on an authenticated Stack.

1. Read the target and create a missing managed ledger with `mode=NORMAL`, `accountTypes`, and `initialSchema`. On a managed existing ledger, read and add missing Wallets definitions where supported. Check effective definitions on restart. Preserve the operator's existing enforcement setting and unrelated resources.
2. Create the required indexes with `POST /v3/{ledger}/indexes`. A successful create registers the index. Wait for the serving replicas to report the usable version via `GET /v3/{ledger}/indexes/{canonicalId}/status` and verify a representative indexed query. The status is per replica. Pending or missing indexes keep the deployment from readiness after a bounded wait and identify the resource that needs attention.
3. Register the saved scripts and prepared queries below and read them back. Treat a create conflict as success only after the existing definition matches. Resume missing steps after partial provisioning or another replica's concurrent startup. Report conflicting definitions with the resource name and expected version. Keep provisioning additive so older replicas can continue using their definitions during a rolling upgrade.

[Ledger creation and status endpoints](https://raw.githubusercontent.com/formancehq/ledger/v3.0.0-rc.3/openapi.yml) · [Ledger index ownership](https://github.com/formancehq/ledger/blob/v3.0.0-rc.3/docs/technical/architecture/subsystems/indexer/operator-ownership.md)

### Chart of accounts

Derive the base from `Chart.BasePath()` and existing address rendering. With no prefix it is `wallets`. With a prefix it is the rendered `<prefix>:wallets`. Wallets-generated wallet and hold IDs are UUIDs. The chart removes dashes in the Ledger address and keeps the original IDs in metadata. Register the following types and retain metadata as the source of Wallets identity and balance role.

| Account type | Relative address pattern | Purpose and constraints |
| --- | --- | --- |
| `wallets-balance` | `{walletId}:{balanceName}` | Main and named spendable balances. Use a 32-character lowercase hexadecimal segment for generated wallet IDs and `[0-9A-Za-z_]+` for the rendered balance name. Wallet metadata identifies the primary record at `main`. |
| `wallets-hold` | `holds:{holdId}` | Pending debit funds. Use a 32-character lowercase hexadecimal segment for generated hold IDs. Retain input and output when the balance reaches zero. |

Set persistence to `NORMAL` for both types. RC.3 rejects adding a third literal `main` account type alongside the hold type on an existing ledger. One balance type covers main and named balances with the same Wallets address and metadata contract. Recommend `AUDIT` for a new managed ledger so caller-provided `ACCOUNT` counterparties remain usable. Treat `world` and caller-provided account addresses as counterparties outside the Wallets chart. An existing `STRICT` ledger needs compatible types for its external counterparties before it can serve that traffic. Existing wallet addresses, including the dash normalization, are part of the compatibility contract. [Wallets chart](../pkg/chart.go) · [Wallets subject resolution](../pkg/subject.go) · [Ledger account-type contract](https://raw.githubusercontent.com/formancehq/ledger/v3.0.0-rc.3/openapi.yml)

### Metadata and built-in indexes

Declare each fixed metadata field as `string` in `initialSchema` on creation. Values such as `"true"` already have string semantics in Wallets and existing data. Use `PUT /metadata-schema/{targetType}/{key}` for additive provisioning on a managed ledger. Register the index after the schema declaration. Encode a key containing slashes as one path segment on detail endpoints.

| Target | Required canonical index ID | Query served |
| --- | --- | --- |
| Account | `metadata:TARGET_TYPE_ACCOUNT:wallets/spec/type` | Wallet and hold listings |
| Account | `metadata:TARGET_TYPE_ACCOUNT:wallets/name` | Exact wallet-name lookup within primary wallets |
| Account | `metadata:TARGET_TYPE_ACCOUNT:wallets/id` | Balances by wallet and wallet summaries |
| Account | `metadata:TARGET_TYPE_ACCOUNT:wallets/balances` | Balance membership and wildcard debit source discovery |
| Account | `metadata:TARGET_TYPE_ACCOUNT:wallets/holds/wallet_id` | Holds by wallet and held-funds summaries |
| Transaction | `metadata:TARGET_TYPE_TRANSACTION:wallets/transaction` | Wallets transaction history |
| Transaction | `metadata:TARGET_TYPE_TRANSACTION:wallets/holdReplayKey` | Keyed hold confirm and void retries, including retries after the hold closes |
| Transaction | `tx_builtin:TX_BUILTIN_INDEX_ADDRESS` | Wallet-filtered transaction history, matching either posting role |
| Transaction | `tx_builtin:TX_BUILTIN_INDEX_DESTINATION_ADDRESS` | The original hold funding transaction used by void |

The exact account lookup uses the account store directly. The table covers the indexes for current Wallets queries. The hold confirm and void transaction records a hashed replay key and request fingerprint atomically with the postings. The key is indexed. The fingerprint, `wallets/createResponse`, and balance replay snapshots need storage only.

For v3 custom metadata filters, propose an operator-configured list of queryable keys. Declare and index each as an account string field under `wallets/custom_data_<key>`. Continue storing and reading custom keys generally. An unconfigured filter key returns the existing validation envelope with the missing key named. This is a v3 deployment requirement and a release documentation item. Keep index creation out of request handling. Check the configured keys and index quota before qualifying an existing tenant. [Wallets metadata and filter keys](../pkg/metadata.go) · [Wallets query construction](../pkg/manager.go) · [Ledger RC.3 index API](https://raw.githubusercontent.com/formancehq/ledger/v3.0.0-rc.3/openapi.yml)

### Numscript library

Save and execute stable script shapes in bootstrap and runtime respectively. Keep the source in the repository, append immutable versions with `PUT /numscripts/{name}`, and read back exact content on a duplicate version. Call through `scriptReference` with the exact semver. Treat any content change as a new version. Use the same hold metadata payload for either saved or inline scripts.

| Wallets path | Proposed v3 execution |
| --- | --- |
| Credit with the default source resolved to `world` | Save `wallets-credit-world@1.0.0`. Pass amount and destination as variables. |
| Confirm a hold without `final` | Save `wallets-confirm-hold@1.0.0`. Pass hold, destination and resolved amount. |
| Other credits and debits | Keep the ordered source templates inline for the first release. Explicit or wildcard balance queries can return a variable number of sources and expiration checks change eligibility. A generic single-source debit may be saved later if it removes code duplication. |
| Final confirm | Keep the current asset-specific send-all inline so Ledger calculates the remainder as part of execution. |
| Void | Keep the inline refund generated from the original funding postings. Preserve each source, amount cap and order. |

Keep v2 script rendering in the existing path. Pass the static script choice, variables, and hold account metadata explicitly from the operation that has the domain inputs. Use pinned versions for saved execution. If Ledger rejects the saved request with an idempotency conflict, try the earlier inline form with the original key. Ledger still rejects changed input. Keep the previous script version available for in-flight retries. For v3, send `accountMetadata` in the same Ledger write as the script. A probe on RC.3 committed a hold with inline `set_account_meta` but returned an escaped JSON string that Wallets could not decode. The `accountMetadata` path returned the expected value and passed create, confirm, and void. [Wallets script builders](../pkg/scripts.go) · [Ledger Numscript library](https://github.com/formancehq/ledger/blob/v3.0.0-rc.3/docs/technical/architecture/subsystems/scripting/numscript-library.md)

### Prepared queries

Ledger v3 can store a named filter and accept runtime parameters. Use the following bounded catalog for repeatable reads, with direct v3 filters for other shapes and requests that add configured custom metadata. Persist only the fixed query definitions. Provision their indexes separately and verify execution with representative parameters. Treat query names as versioned because Ledger lets a saved definition change in place. If the shape changes, append a new name so active cursors and older replicas can use the original.

| Prepared query | Target and filter | Caller |
| --- | --- | --- |
| `wallets-balances-by-wallet-v1` | ACCOUNTS, balance marker equals `"true"` and wallet ID equals parameter | Balance list and wildcard debit. The available-funds summary retains its existing wallet-ID-only filter. |
| `wallets-holds-by-wallet-v1` | ACCOUNTS, hold owner equals wallet ID parameter | Held-funds summary. The public hold list adds its existing hold-type predicate through the direct filter path. |
| `wallets-hold-funding-v1` | TRANSACTIONS, destination equals hold address parameter | Void. Use a page of two or equivalent cardinality check to distinguish one match from multiple matches. |
| `wallets-by-name-v1` | ACCOUNTS, primary wallet type and exact name parameter | Wallet list with a name and no custom filters |
| `wallets-transactions-by-wallet-v1` | TRANSACTIONS, Wallets transaction marker and any-role wallet address parameter | Wallet-filtered history. Preserve the current main-account selector and ordering. |

Store with `POST /prepared-queries` and use `POST /prepared-queries/{name}/execute` with the same parameters on every page. The structured filter uses `{"$param":"walletId"}` for a parameter value. Use LIST mode for the first release. Wallet summaries still need the per-account expiry, priority and hold metadata that aggregation alone cannot express. [Ledger prepared queries](https://github.com/formancehq/ledger/blob/v3.0.0-rc.3/docs/technical/architecture/subsystems/read-path/prepared-queries.md)

## Implementation and release

| Change | Reviewable scope and acceptance |
| --- | --- |
| 1. Restore the baseline build | Repair the Uptrace dependency mismatch independently. Run the current unit tests and the v2 integration suite against the pinned v2 backend. |
| 2. Add the v3 HTTP adapter | Add explicit backend selection, opt-in API detection, and adapter contract tests. Run the Wallets flow against a preprovisioned v3 ledger. Cover ledger and account absence, committed writes with a lost response, error mapping, opaque pagination, full integer precision, and v2 regression. |
| 3. Provision the chart and indexes | Exercise creation from an empty ledger, concurrent startup, restart after partial provisioning, schema drift, an index still building, and an operator-provisioned ledger. Verify named and custom metadata filters with actual results. |
| 4. Save and execute the static scripts and queries | Read back the registry and verify exact script versions, parameters, replay, posting order, and forward and backward pagination. Compare the existing v2 flow and the v3 flow, including a secondary balance with expiry, an external counterparty, partial and final confirmation, and a multi-source void. |

Qualify a pinned supported Ledger v3 release, the pinned v2 build, and authenticated Stack routing before publishing the support matrix. The RC.3 local result applies to that prerelease. CI should use the existing v2 suite and the same externally observable Wallets smoke assertions on v3, with the v3 case starting from a fresh ledger. Wallets 3.0 is a proposed release target. Ledger v3 support can be delivered with the Wallets v1 contract intact. The module version and Stack compatibility line are separate release decisions.

[Wallets 3.0 Jira epic](https://formance-team.atlassian.net/browse/EN-3409)

## Pre-implementation evidence on 25 November 2026

Ledger `3.0.0-rc.3` at `127.0.0.1:9000` reported commit `0e044f130` and protocol `40`. The Wallets checkout was `b634d23`.

| Check | Result |
| --- | --- |
| Wallets directly against rc.3 | Startup failed on the hard-coded v2 route. A prefix-only relay created the ledger but failed startup because Wallets expected `204`. After a restart against the existing ledger, the relay produced an HTTP 500 for a credit that Ledger had already committed. |
| Same Wallets smoke with temporary request and response translation | Passed on `wallets-v3-http-f67427d5`. It covered keyed wallet creation and replay, a secondary balance, credit and replay, debit, hold read, confirm and void, insufficient funds, filtered transaction pagination, and balance listing and summary. Final balance was `700` minor units of `USD/2` with zero held. The relay used supplied indexes and simulated the adapter. |
| Native Ledger primitives | Passed on `wallets-v3-primitives-cb4701e7`. Probes verified initial schema and three account types, all fixed indexes and one custom-account-metadata index, an exact account lookup, two saved script calls, a closed hold with retained volumes, five prepared queries, and newest-first prepared transaction pagination. Reusing a raw key from the other ledger caused HTTP 409. |
| Wallets build, unit tests and v2 integration suite | The checkout required a temporary module file with compatible Uptrace dependencies because the pinned OpenTelemetry update broke compilation. Race-enabled tests passed for `./pkg`, `./pkg/api`, and `./cmd`. The existing `-race -tags it ./test/e2e` suite passed against its pinned v2 backend. Both runs used the temporary module file. |

The prototype used an unauthenticated local instance and synthetic funds. The two disposable ledgers hold the prototype evidence.

## Native validation on 25 November 2026

- `just tests` passed with the checked-in dependency repair, the existing pinned Ledger v2 integration suite, and the v3 smoke connected to `127.0.0.1:9000`.
- `just tests-ledger-v3-ci` passed with the pinned Ledger v3.0.0-rc.3 container. Wallets created the ledger and registered its account types, indexes, scripts, and queries through the native API.
- The reference flow left 700 minor units of `USD/2` available and zero held. The smoke compared the Wallets summary with the Ledger volumes, then tested expiry, a multi-source void, partial and final confirmation, and an external counterparty.
- The local gateway fixture verified client-credentials token acquisition and forwarding before v3 provisioning and writes. The supported-production release matrix and hosted Stack rollout require release validation.
