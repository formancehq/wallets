#!/usr/bin/env bash
set -euo pipefail

readonly plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
readonly guard="$plugin_root/scripts/check-authoring-toolchain.sh"

fail() {
  printf 'check-authoring-toolchain test failed: %s\n' "$1" >&2
  exit 1
}

[[ -x "$guard" ]] || fail "$guard is not executable"

temporary_root="$(mktemp -d)"
cleanup() { rm -rf "$temporary_root"; }
trap cleanup EXIT
fake_bin="$temporary_root/bin"
mkdir "$fake_bin"

write_fake() {
  local name="$1"
  local version="$2"
  printf '#!/bin/sh\nprintf "%%s\\n" %s\n' "'${version}'" >"$fake_bin/$name"
  chmod +x "$fake_bin/$name"
}

write_fake componentize-go 'componentize-go 0.4.1'
write_fake wasi-virt 'wasi-virt 0.2.0'
write_fake wasm-tools 'wasm-tools 1.239.0'
write_fake wasm-opt 'wasm-opt version 124'

PATH="$fake_bin:/usr/bin:/bin" "$guard" || fail "the exact pinned fake toolchain was rejected"

assert_mismatch() {
  local name="$1"
  local expected="$2"
  local mismatch="$3"
  write_fake "$name" "$mismatch"
  if PATH="$fake_bin:/usr/bin:/bin" "$guard" >"$temporary_root/stdout" 2>"$temporary_root/stderr"; then
    fail "a mismatched $name version was accepted"
  fi
  grep -F "authoring tool version mismatch for $name: expected $expected, got $mismatch" \
    "$temporary_root/stderr" >/dev/null || fail "$name version-mismatch diagnostic is missing"
  write_fake "$name" "$expected"
}

assert_mismatch componentize-go 'componentize-go 0.4.1' 'componentize-go 0.4.0'
assert_mismatch wasi-virt 'wasi-virt 0.2.0' 'wasi-virt 0.1.0'
assert_mismatch wasm-tools 'wasm-tools 1.239.0' 'wasm-tools 1.238.0'
assert_mismatch wasm-opt 'wasm-opt version 124' 'wasm-opt version 123'
