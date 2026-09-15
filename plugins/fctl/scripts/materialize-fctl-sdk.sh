#!/usr/bin/env bash
# Materializes the exact fctl SDK commit recorded in fctl-sdk.lock.json into a
# local cache so ordinary CI needs no developer-supplied checkout. The commit is
# always an exact object name: this script resolves no branch, no tag and no
# default ref, so the pin cannot float. Prints the cache directory on stdout;
# every diagnostic goes to stderr.
set -euo pipefail

[[ "$#" -eq 3 ]] || { printf 'usage: materialize-fctl-sdk.sh REPOSITORY COMMIT CACHE_DIR\n' >&2; exit 2; }

readonly repository="$1"
readonly commit="$2"
readonly cache="$3"

[[ -n "$repository" ]] || { printf 'fctl SDK repository is required\n' >&2; exit 2; }
[[ -n "$cache" ]] || { printf 'fctl SDK cache directory is required\n' >&2; exit 2; }
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || {
  printf 'fctl SDK commit must be a full 40-character object name: %s\n' "$commit" >&2
  exit 2
}

command -v git >/dev/null || { printf 'git is required to materialize the fctl SDK\n' >&2; exit 1; }

mkdir -p "$cache"
git -C "$cache" init --bare --quiet >&2

if existing_origin="$(git -C "$cache" remote get-url origin 2>/dev/null)"; then
  if [[ "$existing_origin" != "$repository" ]]; then
    printf 'fctl SDK cache origin mismatch: got %s, want %s\n' "$existing_origin" "$repository" >&2
    exit 1
  fi
else
  git -C "$cache" remote add origin "$repository"
fi

has_commit() { git -C "$cache" cat-file -e "$commit^{commit}" 2>/dev/null; }

if ! has_commit; then
  # Exact-object fetch first; repositories that do not advertise unreachable
  # object names fall back to fetching the published branches, after which the
  # same exact object name is still the only thing consumed.
  git -C "$cache" fetch --quiet --no-tags --depth 1 origin "$commit" >/dev/null 2>&1 ||
    git -C "$cache" fetch --quiet --no-tags origin '+refs/heads/*:refs/remotes/origin/*' >&2 ||
    true
fi

has_commit || {
  printf 'fctl SDK commit is not available from %s: %s\n' "$repository" "$commit" >&2
  exit 1
}

printf '%s\n' "$cache"
