set dotenv-load

default:
  @just --list

pre-commit: generate tidy lint fctl-audit-check
pc: pre-commit

lint:
  @golangci-lint run --fix --build-tags it --timeout 5m
  @cd {{justfile_directory()}}/plugins/fctl && ./scripts/with-fctl-sdk.sh golangci-lint run --fix --timeout 5m

tidy:
  @go mod tidy
  @cd {{justfile_directory()}}/plugins/fctl && ./scripts/with-fctl-sdk.sh ./scripts/tidy-with-fctl-sdk.sh

generate:
  @go generate ./...

tests:
  @go test -race -covermode=atomic -count=1 \
    -coverprofile coverage.txt \
    -tags it \
    ./...
  @cd {{justfile_directory()}}/plugins/fctl && ./scripts/with-fctl-sdk.sh go test -race -covermode=atomic -count=1 -coverprofile "{{justfile_directory()}}/coverage-fctl.txt" ./...
  @tail -n +2 coverage-fctl.txt >> coverage.txt && rm coverage-fctl.txt

# Regenerates the committed fctl Wallets plugin operation inventory from
# openapi.yaml. Review the diff: these artefacts are the plugin's source of
# truth for the operation set, the legacy command mapping, and the recorded
# blockers.
fctl-audit:
  @cd {{justfile_directory()}}/plugins/fctl && ./scripts/with-fctl-sdk.sh go run ./cmd/specaudit -spec ../../openapi.yaml -out .

# Fails when the committed inventory no longer matches openapi.yaml.
fctl-audit-check:
  @cd {{justfile_directory()}}/plugins/fctl && ./scripts/with-fctl-sdk.sh go run ./cmd/specaudit -spec ../../openapi.yaml -out . -check

fctl-audit-tidy-check:
  @cd {{justfile_directory()}}/plugins/fctl && ./scripts/with-fctl-sdk.sh ./scripts/tidy-with-fctl-sdk.sh --check

fctl-component-test:
  @cd {{justfile_directory()}}/plugins/fctl && just test

fctl-component-build:
  @cd {{justfile_directory()}}/plugins/fctl && just build-component

generate-client:
  @speakeasy generate sdk -s openapi.yaml -o ./pkg/client -l go

release-local:
  @goreleaser release --nightly --skip=publish --clean

release-ci:
  @goreleaser release --nightly --clean

release:
  @goreleaser release --clean
