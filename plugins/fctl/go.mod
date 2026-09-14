module github.com/formancehq/wallets/plugins/fctl

go 1.26.0

require (
	github.com/formancehq/fctl-v2-poc/pkg/plugin v0.0.0
	github.com/formancehq/wallets/pkg/client v0.0.0
	go.bytecodealliance.org/pkg v0.2.2
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/cenkalti/backoff/v4 v4.2.0 // indirect
	github.com/ericlagergren/decimal v0.0.0-20240411145413-00de7ca16731 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/formancehq/wallets/pkg/client => ../../pkg/client
