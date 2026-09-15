#!/usr/bin/env bash
set -euo pipefail

readonly component_limit_bytes=$((16 * 1024 * 1024))
plugin_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
entrypoint="$plugin_root/entrypoints/wallets/implementation.go"
wit_root="$plugin_root/wit"
module_path="github.com/formancehq/wallets/plugins/fctl"

for tool in cmp go; do
  command -v "$tool" >/dev/null || {
    printf 'required build tool is unavailable: %s\n' "$tool" >&2
    exit 1
  }
done
"$plugin_root/scripts/check-authoring-toolchain.sh"

export GOFLAGS="${GOFLAGS:-} -trimpath -tags=fctl_component_guest"
mkdir -p "$plugin_root/build" "$plugin_root/dist"
staging="$plugin_root/build/component-staging"
rm -rf "$staging"
mkdir "$staging"
cleanup() { rm -rf "$staging"; }
trap cleanup EXIT

build_lane() {
  local lane="$1"
  local lane_root="$staging/source"
  local lane_result="$staging/result-$lane"
  local bindings_path="$module_path/build/component-staging/source/bindings"
  local implementation_dir="$lane_root/bindings/export_formance_fctl_plugin_lifecycle"
  rm -rf "$lane_root"
  mkdir -p "$implementation_dir" "$lane_result"
  cp "$entrypoint" "$implementation_dir/implementation.go"
  componentize-go --ignore-toml-files -d "$wit_root" -w plugin bindings \
    --format --pkg-name "$bindings_path" --output "$lane_root/bindings"
  sed "s|github.com/formancehq/wallets/plugins/fctl/bindings|$bindings_path|g" \
    "$plugin_root/main.go.in" > "$lane_root/main.go"
  (
    cd "$lane_root"
    componentize-go --ignore-toml-files -d "$wit_root" -w plugin build \
      --go "$plugin_root/scripts/go-component-build.sh" --output wallets.raw.wasm
    wasi-virt --allow-clocks --allow-random --allow-env --stdio=ignore \
      --out wallets.wasm wallets.raw.wasm
    wasm-tools strip --all --output wallets.stripped.wasm wallets.wasm
    mv wallets.stripped.wasm wallets.wasm
    wasm-tools validate wallets.wasm
    wasm-tools component wit wallets.wasm > wallets.wit
    sed -n -E 's/^[[:space:]]*import ([^[:space:];]+).*/\1/p' wallets.wit > imports.txt
    [[ "$(wc -l < imports.txt | tr -d ' ')" == 5 ]]
    grep -Fx 'wasi:clocks/wall-clock@0.2.12' imports.txt >/dev/null
    grep -Fx 'wasi:random/random@0.2.12' imports.txt >/dev/null
    grep -Fx 'wasi:cli/environment@0.2.12' imports.txt >/dev/null
    grep -Fx 'wasi:io/poll@0.2.12' imports.txt >/dev/null
    grep -Fx 'wasi:clocks/monotonic-clock@0.2.12' imports.txt >/dev/null
    size="$(wc -c < wallets.wasm | tr -d ' ')"
    [[ "$size" -le "$component_limit_bytes" ]] || {
      printf 'component exceeds %d-byte admission limit: %d\n' "$component_limit_bytes" "$size" >&2
      exit 1
    }
    shasum -a 256 wallets.wasm wallets.wit imports.txt > artifact.sha256
  )
  cp "$lane_root/wallets.wasm" "$lane_root/wallets.wit" "$lane_root/imports.txt" "$lane_root/artifact.sha256" "$lane_result/"
}

build_lane one
build_lane two
for artifact in wallets.wasm wallets.wit imports.txt artifact.sha256; do
  cmp "$staging/result-one/$artifact" "$staging/result-two/$artifact"
done

destination="$plugin_root/dist/wallets"
rm -rf "$destination"
mkdir "$destination"
install -m 0444 "$staging/result-one/wallets.wasm" "$staging/result-one/wallets.wit" \
  "$staging/result-one/imports.txt" "$staging/result-one/artifact.sha256" "$destination/"
printf '%s\n' "$destination/wallets.wasm"
