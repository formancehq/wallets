#!/usr/bin/env bash
# Contract for scripts/materialize-fctl-sdk.sh: ordinary CI must be able to
# obtain the exact pinned fctl SDK commit without a developer-only checkout,
# and must never be able to obtain anything else. Runs against real Git and a
# local repository; it needs no network and no Nix.
set -euo pipefail

readonly plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
readonly materialize="$plugin_root/scripts/materialize-fctl-sdk.sh"

fail() {
  printf 'FAIL: %s\n' "$*" >&2
  exit 1
}

expect_failure() {
  local expected_message="$1"
  shift
  if "$@" >"$test_root/stdout" 2>"$test_root/stderr"; then
    fail "command unexpectedly succeeded: $*"
  fi
  grep -F "$expected_message" "$test_root/stderr" >/dev/null || {
    sed -n '1,40p' "$test_root/stderr" >&2
    fail "missing expected diagnostic: $expected_message"
  }
  [[ ! -s "$test_root/stdout" ]] || fail "failed materialization left a result on stdout: $*"
}

test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT

[[ -x "$materialize" ]] || fail "materializer is missing or not executable: $materialize"

# An upstream repository standing in for formancehq/fctl-v2-poc. The pinned
# commit is deliberately left on a side branch that is not the default branch,
# which is exactly the shape of the real pin.
origin_root="$test_root/upstream"
mkdir -p "$origin_root/pkg/plugin" "$origin_root/wit/formance/fctl/plugin/v1"
git -C "$origin_root" init --quiet --initial-branch=main
git -C "$origin_root" config user.email fctl@example.invalid
git -C "$origin_root" config user.name 'fctl contract'
printf 'module github.com/formancehq/fctl-v2-poc/pkg/plugin\n\ngo 1.25.0\n' >"$origin_root/pkg/plugin/go.mod"
cp "$plugin_root/wit/plugin.wit" "$origin_root/wit/formance/fctl/plugin/v1/plugin.wit"
git -C "$origin_root" add -A
git -C "$origin_root" commit --quiet -m 'base'
git -C "$origin_root" checkout --quiet -b integration
printf 'package sdkfixture\n' >"$origin_root/pkg/plugin/fixture.go"
git -C "$origin_root" add -A
git -C "$origin_root" commit --quiet -m 'pinned'
pinned_commit="$(git -C "$origin_root" rev-parse HEAD)"
git -C "$origin_root" checkout --quiet main
readonly origin_root pinned_commit
readonly repository="file://$origin_root"

# A floating revision must never be accepted: only an exact object name is a pin.
expect_failure 'fctl SDK commit must be a full 40-character object name' \
  "$materialize" "$repository" 'integration' "$test_root/cache-floating"
expect_failure 'fctl SDK commit must be a full 40-character object name' \
  "$materialize" "$repository" "${pinned_commit:0:12}" "$test_root/cache-short"
expect_failure 'usage: materialize-fctl-sdk.sh' "$materialize" "$repository" "$pinned_commit"

cache="$test_root/cache"
resolved="$("$materialize" "$repository" "$pinned_commit" "$cache")"
[[ "$resolved" == "$cache" ]] || fail "materializer printed $resolved, want $cache"
git -C "$cache" cat-file -e "$pinned_commit^{commit}" || fail 'materialized cache does not contain the pinned commit'
[[ "$(git -C "$cache" remote get-url origin)" == "$repository" ]] || fail 'materialized cache has the wrong origin'

# The projected tree is the pinned commit's tree, not a branch tip.
projected="$test_root/projected"
mkdir -p "$projected"
git -C "$cache" archive "$pinned_commit" pkg/plugin wit/formance/fctl/plugin/v1/plugin.wit | tar -x -C "$projected"
[[ -f "$projected/pkg/plugin/fixture.go" ]] || fail 'projected tree is missing the pinned content'

# Replaying the same intent converges without another fetch: the upstream is
# moved out of reach and materialization must still succeed from the cache.
mv "$origin_root" "$test_root/upstream-offline"
replayed="$("$materialize" "$repository" "$pinned_commit" "$cache")"
[[ "$replayed" == "$cache" ]] || fail "replayed materialization printed $replayed, want $cache"
mv "$test_root/upstream-offline" "$origin_root"

# Repositories that refuse an unadvertised object name must still resolve the
# pin: the fallback fetches the published branches and the exact object name is
# still the only thing consumed. A shim refuses exactly that one request and
# delegates every other Git call to the real binary.
shim_bin="$test_root/bin"
mkdir -p "$shim_bin"
real_git="$(command -v git)"
cat >"$shim_bin/git" <<SHIM
#!/usr/bin/env bash
set -euo pipefail
for argument in "\$@"; do
  if [[ "\$argument" == '$pinned_commit' ]]; then
    printf 'error: Server does not allow request for unadvertised object %s\n' "\$argument" >&2
    exit 128
  fi
done
exec '$real_git' "\$@"
SHIM
chmod +x "$shim_bin/git"
fallback_cache="$test_root/cache-fallback"
fallback_resolved="$(PATH="$shim_bin:$PATH" "$materialize" "$repository" "$pinned_commit" "$fallback_cache")"
[[ "$fallback_resolved" == "$fallback_cache" ]] || fail "fallback materialization printed $fallback_resolved, want $fallback_cache"
git -C "$fallback_cache" cat-file -e "$pinned_commit^{commit}" || fail 'fallback materialization did not obtain the pinned commit'

# A cache bound to a different repository is not silently re-pointed.
wrong_cache="$test_root/cache-wrong-origin"
git init --quiet --bare "$wrong_cache"
git -C "$wrong_cache" remote add origin 'https://example.invalid/other.git'
expect_failure 'fctl SDK cache origin mismatch' \
  "$materialize" "$repository" "$pinned_commit" "$wrong_cache"

# An object the locked repository does not publish fails closed.
expect_failure 'fctl SDK commit is not available' \
  "$materialize" "$repository" '0123456789012345678901234567890123456789' "$test_root/cache-unknown"

printf 'fctl SDK materialization contract: ok\n'
