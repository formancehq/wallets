#!/usr/bin/env bash
set -euo pipefail

check_version() {
  local tool="$1"
  local expected="$2"

  command -v "$tool" >/dev/null || {
    printf 'required authoring tool is unavailable: %s\n' "$tool" >&2
    exit 1
  }

  local actual
  actual="$($tool --version 2>&1)" || {
    printf 'cannot read authoring tool version: %s\n' "$tool" >&2
    exit 1
  }
  if [[ "$actual" != "$expected" ]]; then
    printf 'authoring tool version mismatch for %s: expected %s, got %s\n' \
      "$tool" "$expected" "$actual" >&2
    exit 1
  fi
}

check_version componentize-go 'componentize-go 0.4.1'
check_version wasi-virt 'wasi-virt 0.2.0'
check_version wasm-tools 'wasm-tools 1.239.0'
check_version wasm-opt 'wasm-opt version 124'
