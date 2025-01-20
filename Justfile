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
