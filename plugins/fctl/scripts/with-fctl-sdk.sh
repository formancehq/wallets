#!/usr/bin/env bash
set -euo pipefail

readonly plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
readonly lock_path="$plugin_root/fctl-sdk.lock.json"
readonly lock_reader="$plugin_root/scripts/read-fctl-sdk-lock.go"
readonly materializer="$plugin_root/scripts/materialize-fctl-sdk.sh"

[[ "$#" -gt 0 ]] || { printf 'usage: with-fctl-sdk.sh COMMAND [ARG...]\n' >&2; exit 2; }

IFS=$'\t' read -r module_path repository expected_commit sdk_path expected_nar_hash wit_path expected_wit_hash < <(
  GOWORK=off go run "$lock_reader" "$lock_path"
)
readonly module_path repository expected_commit sdk_path expected_nar_hash wit_path expected_wit_hash

workspace_directory="$(mktemp -d "${TMPDIR:-/tmp}/fctl-sdk-work.XXXXXXXX")"
readonly workspace_directory
trap 'rm -rf -- "$workspace_directory"' EXIT

project_locked_commit() {
  local git_directory="$1"
  source_root="$workspace_directory/source"
  mkdir -p "$source_root"
  git -C "$git_directory" archive "$expected_commit" "$sdk_path" "$wit_path" | tar -x -C "$source_root"
}

if [[ -n "${FCTL_SDK_ROOT:-}" ]]; then
  # An explicit source root always wins, so a developer can verify a checkout
  # they already have without any network access.
  [[ -d "$FCTL_SDK_ROOT" ]] || { printf 'FCTL_SDK_ROOT is not a directory: %s\n' "$FCTL_SDK_ROOT" >&2; exit 2; }
  sdk_root="$(cd "$FCTL_SDK_ROOT" && pwd -P)"
  readonly sdk_root

  source_root="$sdk_root"
  if [[ -e "$sdk_root/.git" ]]; then
    actual_commit="$(git -C "$sdk_root" rev-parse HEAD)"
    actual_repository="$(git -C "$sdk_root" remote get-url origin)"
    readonly actual_commit actual_repository
    if [[ "$actual_commit" != "$expected_commit" ]]; then
      printf 'fctl SDK checkout revision mismatch: got %s, want %s\n' "$actual_commit" "$expected_commit" >&2
      exit 1
    fi
    if [[ "$actual_repository" != "$repository" ]]; then
      printf 'fctl SDK checkout remote mismatch: got %s, want %s\n' "$actual_repository" "$repository" >&2
      exit 1
    fi
    project_locked_commit "$sdk_root"
  fi
else
  # Ordinary CI has no fctl checkout. Materialize the locked commit — by its
  # exact object name, never a branch or tag — into a cache the run owns, then
  # project and validate it exactly like a caller-supplied source.
  cache_directory="${FCTL_SDK_CACHE_DIR:-$plugin_root/.fctl-sdk-cache}"
  readonly cache_directory
  materialized_root="$("$materializer" "$repository" "$expected_commit" "$cache_directory")"
  readonly materialized_root
  project_locked_commit "$materialized_root"
fi
readonly source_root

sdk_directory="$source_root/$sdk_path"
wit_file="$source_root/$wit_path"
readonly sdk_directory wit_file
[[ -f "$sdk_directory/go.mod" ]] || { printf 'fctl SDK module is missing: %s\n' "$sdk_directory/go.mod" >&2; exit 1; }
[[ -f "$wit_file" ]] || { printf 'fctl SDK WIT is missing: %s\n' "$wit_file" >&2; exit 1; }

actual_module="$(cd "$sdk_directory" && GOWORK=off go list -m -f '{{.Path}}')"
readonly actual_module
if [[ "$actual_module" != "$module_path" ]]; then
  printf 'fctl SDK module path mismatch: got %s, want %s\n' "$actual_module" "$module_path" >&2
  exit 1
fi

command -v nix >/dev/null || { printf 'nix is required to validate the fctl SDK content hash\n' >&2; exit 1; }
actual_nar_hash="$(nix --extra-experimental-features nix-command hash path --type sha256 --sri "$sdk_directory")"
readonly actual_nar_hash
if [[ "$actual_nar_hash" != "$expected_nar_hash" ]]; then
  printf 'fctl SDK content hash mismatch: got %s, want %s\n' "$actual_nar_hash" "$expected_nar_hash" >&2
  exit 1
fi

actual_wit_hash="$(shasum -a 256 "$wit_file" | awk '{print $1}')"
readonly actual_wit_hash
if [[ "$actual_wit_hash" != "$expected_wit_hash" ]]; then
  printf 'fctl SDK WIT hash mismatch: got %s, want %s\n' "$actual_wit_hash" "$expected_wit_hash" >&2
  exit 1
fi

(
  cd "$workspace_directory"
  GOWORK=off go work init "$plugin_root"
)
readonly workspace_file="$workspace_directory/go.work"
GOWORK="$workspace_file" go work edit -replace "$module_path=$sdk_directory"
FCTL_SDK_ROOT="$source_root" GOWORK="$workspace_file" "$@"
