# Wallets plugin for fctl

This module provides the portable fctl command plugin for Wallets v2. It
declares 14 commands, validates execution requests against closed schemas, and
routes every product call through the generated Wallets client and the public
fctl `producthttp` adapter.

The host owns target resolution, Stack authentication, request execution,
retry policy, pagination budgets, and rendering. The plugin does not read
credentials or construct product endpoints.

## Commands

The command tree covers wallets, balances, holds, and transactions. List
commands return one cursor page by default. The host-owned `--all` flag follows
opaque cursors and emits one aggregated JSON array, bounded to 100 pages,
10,000 items, and 4 MiB.

Nine commands with a single resource success shape declare compact, ordered table
columns. Those columns select scalar leaves only, including the nested
`destination.identifier` path for holds; arbitrary metadata, asset maps,
posting arrays, volume maps, and free-form descriptions remain available in
JSON and YAML. `update`, `credit`, `holds confirm`, and `holds void` declare no
table because their successful response is the empty object produced from an
HTTP 204 response. `debit` also declares no static table because its success is
a Hold for a pending debit but `{}` for a completed 204 debit; a static Hold
table would produce a blank row for the latter. The public schemas describe
every exported field of each generated result model — including nested maps,
optional fields, and nullability — while remaining open to forward-compatible
product fields, so table presentation never narrows machine-readable output.

Fund-moving commands (`credit`, `debit`, `holds confirm`, and `holds void`)
require `--ik` and forward it as `Idempotency-Key`. Balance priorities and
hold-confirmation amounts are accepted as decimal strings so values larger than
32-bit integers remain exact across the portable ABI. Priorities are signed;
movement amounts must be non-negative, and both priority and hold-confirmation
amounts are capped at the current server's signed 64-bit boundary before any
host request.

The OpenAPI contract labels priority as `bigint`, while the current Wallets
server binds it to Go `int` and re-reads its ledger metadata with signed 64-bit
parsing. The command therefore accepts the complete signed 64-bit range,
including values beyond 32-bit, but does not claim arbitrary-precision live
server compatibility; see divergence D8.

The four fund-moving commands require an idempotency key, but the current
Wallets server cannot replay completed hold resolution and rejects keyed debit
requests that use wildcard or expiring balance sources. These release blockers
and their resolution paths are recorded as B1 and B2 in the command inventory.
The pinned fctl HTTP bridge preserves a non-2xx product response, including
Wallets' bounded 413 response, as the `product_http_error` failure carrying its
numeric status, so the plugin does not collapse expected product rejections into
an invalid-response failure.

The exact command, flag, operation, scope, and compatibility mapping is in
[`docs/command-inventory.md`](./docs/command-inventory.md). Generated operation
tables are in [`docs/operations.generated.md`](./docs/operations.generated.md).

## Layout

| path | purpose |
| --- | --- |
| `core/` | command catalogue, schemas, execution, and generated-client mapping |
| `component/` | immutable portable descriptor |
| `entrypoints/wallets/` | component lifecycle exports |
| `wit/plugin.wit` | portable lifecycle interface used by the build |
| `audit/` | OpenAPI inventory and compatibility checks |
| `scripts/build-component.sh` | deterministic two-lane component build and admission checks |

The plugin imports only fctl public packages. It must not import fctl
`internal/` packages.

## Verify

The source-level gates need no developer-only checkout. When `FCTL_SDK_ROOT` is
unset, the wrapper materializes the exact commit recorded in
`fctl-sdk.lock.json` from the locked repository into a cache directory
(`plugins/fctl/.fctl-sdk-cache`, overridable with `FCTL_SDK_CACHE_DIR`) and
copies the locked SDK and WIT paths into a clean ephemeral projection. Set
`FCTL_SDK_ROOT` to verify a checkout you already have instead:

```sh
# ordinary CI and a clean workstation
nix develop --impure --no-write-lock-file --command just fctl-component-test

# against an existing fctl checkout
export FCTL_SDK_ROOT=/path/to/fctl-v2-poc
nix develop --impure --no-write-lock-file --command just fctl-component-test
```

Materialization never resolves a branch or a tag. The locked commit is fetched
by its exact object name, the cache's origin must be the locked repository, and
the fetched tree is rejected unless its digests match the lock.

The wrapper validates the SDK module's NAR content hash and canonical WIT hash
against `fctl-sdk.lock.json`. When the source includes Git metadata, it also
requires the locked commit and origin, then projects those exact committed SDK
and WIT paths into a clean ephemeral projection before validation. Ignored or modified working-tree files
therefore cannot affect the command. It creates an ephemeral Go workspace for
the Wallets and validated SDK modules, runs the requested command with that
workspace, and removes the whole projection afterward. No workstation path or
Nix store path is tracked. Run the provenance and temporary-module contracts
alone with `just validate-sdk`; use `just tidy-check` to prove module metadata
is already canonical without modifying it.

For a component build, combine the fctl author-tool shell with this repository's
shell, which supplies the pinned Binaryen build:

```sh
nix develop --impure --no-write-lock-file "$FCTL_SDK_ROOT#" --command \
  nix develop --impure --no-write-lock-file .# --command \
  just fctl-component-build
```

Run the complete repository gate separately with
`nix develop --impure --no-write-lock-file --command just pre-commit`.

The Wallets development shell pins the same Rust 1.91.1 authoring stack as the
fctl SDK snapshot: `componentize-go` 0.4.1, the patched `wasi-virt` revision,
`wasm-tools` 1.239.0, and Binaryen's `wasm-opt`. The component contract tests
guard those Nix definitions and their source hashes against accidental drift.

`build-component` builds the component twice, compares the artefacts, validates
the component, enforces the five admitted WASI imports, captures SHA-256 hashes,
and rejects a Wasm artefact larger than 16 MiB. Successful local output is written below
`dist/wallets/` and is intentionally not committed.

## Integration boundary

The explicit SDK source contract is portable across checkout locations and
fails closed on source or WIT drift. It remains a local source-development
boundary, not a published module release. `fctl-sdk.lock.json` is sealed at
`e9b1395f46f3100b381dbe00f5213de28e6df0e1` on the canonical
`formancehq/fctl-v2-poc` repository, and the canonical WIT hash is unchanged
from the previous pin, so the portable lifecycle interface is untouched by the
repin. A checkout carrying Git metadata must present that commit and that
origin before the wrapper runs.

That commit is reachable on the canonical remote today, but only from
`refs/heads/codex/mvp5-integration`. It is not an ancestor of
`refs/heads/main` and carries no tag, and nothing in this repository can make
it durable. Until it lands on `main`, deleting or force-moving that branch
leaves the pin unresolvable and reopens the typed `product_http_error` mapping
this plugin depends on. Landing it is release gate 1 in the command inventory.

End-to-end product scenarios additionally require a live Wallets v2 service and
the fctl real-host compatibility gates.
