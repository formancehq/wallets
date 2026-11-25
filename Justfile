set dotenv-load

default:
  @just --list

pre-commit: generate tidy lint
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m

tidy:
  @go mod tidy

generate:
  @go generate ./...

tests:
  @go test -race -covermode=atomic \
    -coverprofile coverage.txt \
    -tags it \
    ./...

generate-client:
  @speakeasy generate sdk -s openapi.yaml -o ./pkg/client -l go

release-local:
  @goreleaser release --nightly --skip=publish --clean

release-ci:
  @goreleaser release --nightly --clean

release:
  @goreleaser release --clean

tests-ledger-v3:
  @test -n "${WALLETS_LEDGER_V3_URL:-}" || { echo 'Set WALLETS_LEDGER_V3_URL to the Ledger v3 service base.' >&2; exit 1; }
  @go test -race -count=1 -tags it -v ./test/e2e/ledgerv3

tests-ledger-v3-ci:
  #!/usr/bin/env bash
  set -euo pipefail
  ledger_container=$(docker run --detach --publish 127.0.0.1::9000 \
    ghcr.io/formancehq/ledger:v3.0.0-rc.3@sha256:467b632d1b234c1d064558249fe55fec8843a1535bd2a40302546b1dc55cfd07 \
    run --bootstrap --node-id 1 --cluster-id wallets-ci --bind-addr 127.0.0.1:7777 \
    --data-dir /tmp/wallets-ci/data --wal-dir /tmp/wallets-ci/wal)
  trap 'docker rm --force "$ledger_container" >/dev/null' EXIT
  ledger_url="http://$(docker port "$ledger_container" 9000/tcp)"
  for attempt in {1..30}; do
    if curl --fail --silent --max-time 2 "$ledger_url/clusterz" >/dev/null; then
      if WALLETS_LEDGER_V3_URL="$ledger_url" just tests-ledger-v3; then
        exit 0
      fi
      break
    fi
    sleep 1
  done
  docker logs --tail 80 "$ledger_container"
  exit 1
